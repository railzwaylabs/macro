package workspace

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/railzwaylabs/macro/internal/project"
)

var ErrAlreadyRegistered = errors.New("project already registered")

//go:embed templates
var templateFiles embed.FS

type Entry struct {
	Name string       `json:"name"`
	Type project.Type `json:"type"`
	Path string       `json:"path"`
}

type Registration struct {
	ProjectName   string
	WorkspaceName string
	Path          string
}

func Init(parent, name string) (destination string, returnErr error) {
	if err := project.ValidateName(name); err != nil {
		return "", err
	}

	createdDirectory := filepath.Join(parent, name)
	if _, err := os.Stat(createdDirectory); err == nil {
		return "", fmt.Errorf("destination %s already exists", createdDirectory)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect destination %s: %w", createdDirectory, err)
	}

	if err := os.Mkdir(createdDirectory, 0o755); err != nil {
		return "", fmt.Errorf("create workspace %s: %w", createdDirectory, err)
	}
	defer func() {
		if returnErr == nil {
			return
		}

		if cleanupErr := os.RemoveAll(createdDirectory); cleanupErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove incomplete workspace %s: %w", createdDirectory, cleanupErr))
		}
	}()

	manifest := Manifest{Name: name, Projects: []ProjectRef{}}
	if err := write(filepath.Join(createdDirectory, ManifestName), manifest); err != nil {
		return "", fmt.Errorf("create workspace manifest in %s: %w", createdDirectory, err)
	}
	for _, file := range []struct{ source, target string }{
		{"templates/README.md.tmpl", "README.md"},
		{"templates/Makefile.tmpl", "Makefile"},
		{"templates/gitignore.tmpl", ".gitignore"},
	} {
		if err := renderWorkspaceFile(createdDirectory, file.source, file.target, struct{ Name string }{name}); err != nil {
			return "", err
		}
	}

	return createdDirectory, nil
}

func renderWorkspaceFile(directory, source, target string, data any) error {
	parsed, err := template.ParseFS(templateFiles, source)
	if err != nil {
		return fmt.Errorf("parse workspace template %s: %w", source, err)
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return fmt.Errorf("render workspace template %s: %w", source, err)
	}
	path := filepath.Join(directory, target)
	if err := os.WriteFile(path, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write workspace file %s: %w", path, err)
	}
	return nil
}

func Add(workspaceDirectory, projectPath string) (Registration, error) {
	manifest, err := Read(workspaceDirectory)
	if err != nil {
		return Registration{}, err
	}

	absoluteProjectPath, projectManifest, err := loadProject(projectPath)
	if err != nil {
		return Registration{}, err
	}

	absoluteWorkspacePath, err := filepath.Abs(workspaceDirectory)
	if err != nil {
		return Registration{}, fmt.Errorf("resolve workspace path %s: %w", workspaceDirectory, err)
	}

	relativePath, err := filepath.Rel(absoluteWorkspacePath, absoluteProjectPath)
	if err != nil {
		return Registration{}, fmt.Errorf("make project path relative to workspace %s: %w", absoluteWorkspacePath, err)
	}
	relativePath = normalizePath(relativePath)

	if containsProject(manifest.Projects, relativePath) {
		return Registration{}, fmt.Errorf("%s: %w", relativePath, ErrAlreadyRegistered)
	}

	manifest.Projects = append(manifest.Projects, ProjectRef{Path: relativePath})
	manifestPath := filepath.Join(absoluteWorkspacePath, ManifestName)
	if err := write(manifestPath, manifest); err != nil {
		return Registration{}, err
	}

	registration := Registration{
		ProjectName:   projectManifest.Name,
		WorkspaceName: manifest.Name,
		Path:          relativePath,
	}

	return registration, nil
}

func loadProject(projectPath string) (string, project.Manifest, error) {
	absolutePath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", project.Manifest{}, fmt.Errorf("resolve project path %s: %w", projectPath, err)
	}

	projectInfo, err := os.Stat(absolutePath)
	if err != nil {
		return "", project.Manifest{}, fmt.Errorf("inspect project %s: %w", absolutePath, err)
	}
	if !projectInfo.IsDir() {
		return "", project.Manifest{}, fmt.Errorf("project path %s is not a directory", absolutePath)
	}

	manifest, err := project.Read(absolutePath)
	if err != nil {
		return "", project.Manifest{}, fmt.Errorf("validate Macro project %s: %w", absolutePath, err)
	}

	return absolutePath, manifest, nil
}

func containsProject(projects []ProjectRef, relativePath string) bool {
	for _, registeredProject := range projects {
		if normalizePath(registeredProject.Path) == relativePath {
			return true
		}
	}

	return false
}

func List(workspaceDirectory string) ([]Entry, error) {
	manifest, err := Read(workspaceDirectory)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(manifest.Projects))
	var issues []error
	for _, reference := range manifest.Projects {
		projectDirectory := filepath.Join(workspaceDirectory, filepath.FromSlash(reference.Path))
		projectManifest, err := project.Read(projectDirectory)
		if err != nil {
			issues = append(issues, fmt.Errorf("project %s: %w", reference.Path, err))
			continue
		}

		entry := Entry{
			Name: projectManifest.Name,
			Type: projectManifest.Type,
			Path: normalizePath(reference.Path),
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	return entries, errors.Join(issues...)
}

func normalizePath(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "./") {
		return clean
	}

	return "./" + clean
}
