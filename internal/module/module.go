package module

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/railzwaylabs/macro/internal/project"
)

//go:embed templates
var templates embed.FS

var ErrAlreadyExists = errors.New("module already exists")

type Result struct {
	Directory     string
	MigrationUp   string
	MigrationDown string
}

type Plan struct {
	Directory     string
	CreateFiles   []string
	UpdateFiles   []string
	manifest      project.Manifest
	migrationUp   string
	migrationDown string
}

func BuildPlan(projectDirectory, name string, timestamp time.Time) (Plan, error) {
	if err := project.ValidateName(name); err != nil {
		return Plan{}, fmt.Errorf("invalid module name: %w", err)
	}
	manifest, err := project.Read(projectDirectory)
	if err != nil {
		return Plan{}, err
	}
	if _, exists := manifest.Modules[name]; exists {
		return Plan{}, fmt.Errorf("module %q already exists in %s: choose another module name or remove the existing module first: %w", name, project.ManifestName, ErrAlreadyExists)
	}
	destination := filepath.Join(projectDirectory, "internal", name)
	if err := ensureDestinationAvailable(destination); err != nil {
		return Plan{}, err
	}
	migrationName := normalizeMigrationName(name)
	prefix := timestamp.UTC().Format("20060102150405")
	migrationDirectory := filepath.Join(projectDirectory, "migrations")
	up := filepath.Join(migrationDirectory, prefix+"_"+migrationName+".up.sql")
	down := filepath.Join(migrationDirectory, prefix+"_"+migrationName+".down.sql")
	return Plan{
		Directory: destination,
		CreateFiles: []string{
			filepath.Join(destination, "application", "service.go"),
			filepath.Join(destination, "transport", "grpc", "handler.go"), up, down,
		},
		UpdateFiles: []string{filepath.Join(projectDirectory, project.ManifestName)},
		manifest:    manifest, migrationUp: up, migrationDown: down,
	}, nil
}

func Add(projectDirectory, name string) (Result, error) {
	return add(projectDirectory, name, time.Now().UTC())
}

func add(projectDirectory, name string, timestamp time.Time) (generated Result, returnErr error) {
	plan, err := BuildPlan(projectDirectory, name, timestamp)
	if err != nil {
		return Result{}, err
	}
	manifest := plan.manifest
	destination := plan.Directory
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return Result{}, fmt.Errorf("create module directory %s: %w", destination, err)
	}

	migrationName := normalizeMigrationName(name)
	migrationDirectory := filepath.Join(projectDirectory, "migrations")
	generated = Result{
		Directory:     destination,
		MigrationUp:   plan.migrationUp,
		MigrationDown: plan.migrationDown,
	}
	defer func() {
		if returnErr == nil {
			return
		}
		for _, path := range []string{destination, generated.MigrationUp, generated.MigrationDown} {
			if cleanupErr := os.RemoveAll(path); cleanupErr != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("remove incomplete scaffold %s: %w", path, cleanupErr))
			}
		}
	}()

	for _, file := range []struct{ source, target string }{
		{"templates/service.go.tmpl", filepath.Join(destination, "application", "service.go")},
		{"templates/handler.go.tmpl", filepath.Join(destination, "transport", "grpc", "handler.go")},
	} {
		if err := render(file.source, file.target, nil); err != nil {
			return generated, err
		}
	}
	if err := os.MkdirAll(migrationDirectory, 0o755); err != nil {
		return generated, fmt.Errorf("create migrations directory: %w", err)
	}
	data := map[string]string{"Name": migrationName}
	if err := render("templates/migration.up.sql.tmpl", generated.MigrationUp, data); err != nil {
		return generated, err
	}
	if err := render("templates/migration.down.sql.tmpl", generated.MigrationDown, data); err != nil {
		return generated, err
	}

	if manifest.Modules == nil {
		manifest.Modules = make(map[string]project.Module)
	}
	manifest.Modules[name] = project.Module{}
	if err := project.Write(filepath.Join(projectDirectory, project.ManifestName), manifest); err != nil {
		return generated, fmt.Errorf("register module %q: %w", name, err)
	}
	return generated, nil
}

func render(source, target string, data any) error {
	parsed, err := template.ParseFS(templates, source)
	if err != nil {
		return fmt.Errorf("parse %s: %w", source, err)
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return fmt.Errorf("render %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", target, err)
	}
	if err := os.WriteFile(target, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

func ensureDestinationAvailable(destination string) error {
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("module directory %s: %w", destination, ErrAlreadyExists)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect module directory %s: %w", destination, err)
	}
	return nil
}

func normalizeMigrationName(name string) string {
	var result strings.Builder
	previousUnderscore := false
	for _, character := range strings.ToLower(name) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			result.WriteRune(character)
			previousUnderscore = false
		} else if !previousUnderscore && result.Len() > 0 {
			result.WriteByte('_')
			previousUnderscore = true
		}
	}
	return strings.Trim(result.String(), "_")
}
