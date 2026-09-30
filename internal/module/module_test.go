package module

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/railzwaylabs/macro/internal/project"
)

func TestAddCreatesMinimalModuleAndMigrations(t *testing.T) {
	directory := createProject(t)
	when := time.Date(2026, 9, 29, 22, 0, 1, 0, time.UTC)
	generated, err := add(directory, "invoice", when)
	if err != nil {
		t.Fatalf("add() error = %v", err)
	}

	servicePath := filepath.Join(generated.Directory, "application", "service.go")
	handlerPath := filepath.Join(generated.Directory, "transport", "grpc", "handler.go")
	service := parseGoFile(t, servicePath)
	handler := parseGoFile(t, handlerPath)
	assertDeclarations(t, service, "Service", "NewService")
	assertDeclarations(t, handler, "Handler", "NewHandler")
	modulePath := filepath.Join(generated.Directory, "module.go")
	moduleFile := parseGoFile(t, modulePath)
	assertDeclarations(t, moduleFile, "Module")

	for _, absent := range []string{
		filepath.Join(generated.Directory, "domain", "domain.go"),
		filepath.Join(generated.Directory, "infrastructure", "infrastructure.go"),
	} {
		if _, err := os.Stat(absent); !os.IsNotExist(err) {
			t.Errorf("unexpected scaffold %s", absent)
		}
	}

	if got := filepath.Base(generated.MigrationUp); got != "20260929220001_invoice.up.sql" {
		t.Fatalf("up migration = %q", got)
	}
	if got := filepath.Base(generated.MigrationDown); got != "20260929220001_invoice.down.sql" {
		t.Fatalf("down migration = %q", got)
	}
	assertFileEquals(t, generated.MigrationUp, "-- invoice module migration\n")
	assertFileEquals(t, generated.MigrationDown, "-- invoice module rollback\n")

	manifest, err := project.Read(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := manifest.Modules["invoice"]; !exists {
		t.Fatal("module was not registered")
	}

	forbidden := []string{"Repository", "FindByID", "Create(", "Update(", "Delete(", "List("}
	for _, path := range []string{servicePath, handlerPath, modulePath} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range forbidden {
			if strings.Contains(string(contents), fragment) {
				t.Errorf("generated code contains guessed behavior %q", fragment)
			}
		}
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	directory := createProject(t)
	if _, err := Add(directory, "invoice"); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(directory, "invoice"); err == nil {
		t.Fatal("duplicate Add() error = nil")
	}
}

func TestAddRefusesDeveloperOwnedRegistry(t *testing.T) {
	directory := createProject(t)
	registry := filepath.Join(directory, "internal", "modules", "modules.go")
	const developerSource = "package modules\n\n// developer owned\n"
	if err := os.WriteFile(registry, []byte(developerSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(directory, "invoice"); err == nil || !strings.Contains(err.Error(), "developer-owned") {
		t.Fatalf("Add() error = %v", err)
	}
	contents, err := os.ReadFile(registry)
	if err != nil || string(contents) != developerSource {
		t.Fatalf("registry was overwritten: %q, %v", contents, err)
	}
}

func TestAddUpdatesDeterministicRegistryAndGeneratedProjectCompiles(t *testing.T) {
	directory := createProject(t)
	if _, err := Add(directory, "product"); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(directory, "category"); err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(directory, "internal", "modules", "modules.go")
	contents, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	registry := string(contents)
	if !strings.Contains(registry, "category.Module") || !strings.Contains(registry, "product.Module") || strings.Index(registry, "category.Module") > strings.Index(registry, "product.Module") {
		t.Fatalf("registry is not complete and deterministic:\n%s", registry)
	}
	macroRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	runGo(t, directory, "mod", "edit", "-replace=github.com/railzwaylabs/macro="+macroRoot)
	runGo(t, directory, "mod", "tidy")
	runGo(t, directory, "test", "./...")
}

func runGo(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("go", arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
}

func createProject(t *testing.T) string {
	t.Helper()
	generated, err := project.Generate(project.GenerateOptions{Parent: t.TempDir(), Name: "billing", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	return generated.Directory
}

func parseGoFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}

func assertDeclarations(t *testing.T, file *ast.File, names ...string) {
	t.Helper()
	found := make(map[string]bool)
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					found[typeSpec.Name.Name] = true
				}
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, identifier := range valueSpec.Names {
						found[identifier.Name] = true
					}
				}
			}
		case *ast.FuncDecl:
			found[value.Name.Name] = true
		}
	}
	for _, name := range names {
		if !found[name] {
			t.Errorf("declaration %s missing", name)
		}
	}
}

func assertFileEquals(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Errorf("%s = %q, want %q", path, contents, want)
	}
}
