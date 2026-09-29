package project

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

//go:embed templates
var templateFiles embed.FS

type TemplateData struct {
	Name             string
	Type             string
	ModulePath       string
	ProtoName        string
	ProtoGoPackage   string
	ProtoServiceName string
}

type GenerateOptions struct {
	Parent     string
	Name       string
	Type       Type
	ModulePath string
}

type Result struct {
	Directory    string
	ManifestPath string
}

type Plan struct {
	Directory   string
	CreateFiles []string
	Commands    []string
}

func BuildPlan(options GenerateOptions) (Plan, error) {
	options, err := prepareOptions(options)
	if err != nil {
		return Plan{}, err
	}
	destination := filepath.Join(options.Parent, options.Name)
	if err := validateDestination(destination); err != nil {
		return Plan{}, err
	}
	data := TemplateData{
		Name: options.Name, Type: string(options.Type), ModulePath: options.ModulePath,
		ProtoName: normalizeProtoName(options.Name), ProtoGoPackage: normalizeProtoGoPackage(options.Name), ProtoServiceName: normalizeProtoServiceName(options.Name),
	}
	files := make([]string, 0)
	for _, generatedFile := range templatesFor(options.Type, data) {
		files = append(files, filepath.ToSlash(generatedFile.target))
	}
	files = append(files, ManifestName)
	sort.Strings(files)
	commands := []string{"go mod tidy"}
	if options.Type == TypeService {
		commands = append([]string{"buf generate"}, commands...)
	}
	return Plan{Directory: destination, CreateFiles: files, Commands: commands}, nil
}

type fileTemplate struct {
	source string
	target string
}

func Generate(options GenerateOptions) (generated Result, returnErr error) {
	options, err := prepareOptions(options)
	if err != nil {
		return Result{}, err
	}

	destination := filepath.Join(options.Parent, options.Name)
	if err := validateDestination(destination); err != nil {
		return Result{}, err
	}

	if err := os.Mkdir(destination, 0o755); err != nil {
		return Result{}, fmt.Errorf("create project directory %s: %w", destination, err)
	}

	defer func() {
		if returnErr == nil {
			return
		}

		if cleanupErr := os.RemoveAll(destination); cleanupErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove incomplete project %s: %w", destination, cleanupErr))
		}
	}()

	if err := populateProject(destination, options); err != nil {
		return Result{}, err
	}

	manifestPath := filepath.Join(destination, ManifestName)
	generated = Result{Directory: destination, ManifestPath: manifestPath}

	return generated, nil
}

func prepareOptions(options GenerateOptions) (GenerateOptions, error) {
	if err := ValidateName(options.Name); err != nil {
		return GenerateOptions{}, err
	}

	if _, err := ParseType(string(options.Type)); err != nil {
		return GenerateOptions{}, err
	}

	if options.Parent == "" {
		options.Parent = "."
	}

	if options.ModulePath == "" {
		options.ModulePath = options.Name
	}

	return options, nil
}

func validateDestination(destination string) error {
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination %s already exists", destination)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect destination %s: %w", destination, err)
	}

	return nil
}

func populateProject(destination string, options GenerateOptions) error {
	if err := createDirectories(destination, options.Type); err != nil {
		return err
	}

	templateData := TemplateData{
		Name:             options.Name,
		Type:             string(options.Type),
		ModulePath:       options.ModulePath,
		ProtoName:        normalizeProtoName(options.Name),
		ProtoGoPackage:   normalizeProtoGoPackage(options.Name),
		ProtoServiceName: normalizeProtoServiceName(options.Name),
	}
	for _, target := range templatesFor(options.Type, templateData) {
		if err := renderFile(destination, target, templateData); err != nil {
			return err
		}
	}

	manifestPath := filepath.Join(destination, ManifestName)
	if err := Write(manifestPath, NewManifest(options.Name, options.Type)); err != nil {
		return err
	}

	return nil
}

func createDirectories(destination string, kind Type) error {
	directories := []string{
		"internal",
		"tests",
	}

	if kind == TypeService {
		directories = append(directories, filepath.Join("api", "proto"), "migrations")
	}

	for _, directory := range directories {
		if err := os.MkdirAll(filepath.Join(destination, directory), 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", directory, err)
		}
	}

	return nil
}

func templatesFor(kind Type, data TemplateData) []fileTemplate {
	common := []fileTemplate{
		{source: "templates/common/go.mod.tmpl", target: "go.mod"},
		{source: "templates/common/Dockerfile.tmpl", target: "Dockerfile"},
		{source: "templates/common/gitignore.tmpl", target: ".gitignore"},
		{source: "templates/common/golangci.yaml.tmpl", target: ".golangci.yaml"},
	}
	if kind != TypeWorker {
		common = append(common, fileTemplate{source: "templates/common/README.md.tmpl", target: "README.md"})
	}

	switch kind {
	case TypeService:
		serviceFiles := []fileTemplate{
			{source: "templates/service/main.go.tmpl", target: filepath.Join("cmd", "service", "main.go")},
			{source: "templates/service/Makefile.tmpl", target: "Makefile"},
			{source: "templates/service/buf.yaml.tmpl", target: "buf.yaml"},
			{source: "templates/service/buf.gen.yaml.tmpl", target: "buf.gen.yaml"},
			{source: "templates/service/service.proto.tmpl", target: filepath.Join("api", "proto", data.ProtoName, "v1", data.ProtoName+".proto")},
		}
		return append(common, serviceFiles...)
	case TypeWorker:
		workerFiles := []fileTemplate{
			{
				source: "templates/worker/README.md.tmpl",
				target: "README.md",
			},
			{
				source: "templates/worker/main.go.tmpl",
				target: filepath.Join("cmd", "worker", "main.go"),
			},
			{
				source: "templates/worker/worker.go.tmpl",
				target: filepath.Join("internal", "worker", "worker.go"),
			},
			{
				source: "templates/worker/worker_test.go.tmpl",
				target: filepath.Join("internal", "worker", "worker_test.go"),
			},
			{
				source: "templates/worker/Makefile.tmpl",
				target: "Makefile",
			},
		}
		return append(common, workerFiles...)
	case TypeJob:
		jobFiles := []fileTemplate{
			{source: "templates/job/main.go.tmpl", target: filepath.Join("cmd", "job", "main.go")},
			{source: "templates/job/Makefile.tmpl", target: "Makefile"},
		}
		return append(common, jobFiles...)
	default:
		return common
	}
}

func renderFile(destination string, file fileTemplate, data TemplateData) error {
	tmpl, err := template.ParseFS(templateFiles, file.source)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", file.source, err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return fmt.Errorf("render template %s: %w", file.source, err)
	}

	target := filepath.Join(destination, file.target)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", target, err)
	}

	if err := os.WriteFile(target, rendered.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}

	return nil
}
