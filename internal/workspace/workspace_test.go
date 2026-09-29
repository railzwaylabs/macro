package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
)

func TestInitCreatesValidWorkspace(t *testing.T) {
	parent := t.TempDir()
	directory, err := Init(parent, "commerce")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	manifest, err := Read(directory)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if manifest.Name != "commerce" || len(manifest.Projects) != 0 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if _, err := Init(parent, "commerce"); err == nil {
		t.Fatal("second Init() error = nil")
	}
}

func TestReadRejectsMalformedWorkspace(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ManifestName), []byte("name: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(directory); err == nil {
		t.Fatal("Read() error = nil")
	}
}

func TestFindWalksUpward(t *testing.T) {
	parent := t.TempDir()
	root, err := Init(parent, "commerce")
	if err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(nested)
	if err != nil || got != root {
		t.Fatalf("Find() = %q, %v; want %q", got, err, root)
	}
}

func TestAddAndListProject(t *testing.T) {
	parent := t.TempDir()
	root, err := Init(parent, "commerce")
	if err != nil {
		t.Fatal(err)
	}
	result, err := project.Generate(project.GenerateOptions{Parent: root, Name: "billing", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	registration, err := Add(root, result.Directory)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if registration.ProjectName != "billing" || registration.WorkspaceName != "commerce" || registration.Path != "./billing" {
		t.Fatalf("registration = %#v", registration)
	}
	if _, err := Add(root, result.Directory); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("duplicate Add() error = %v", err)
	}
	entries, err := List(root)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "billing" || entries[0].Path != "./billing" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestAddRejectsNonMacroDirectory(t *testing.T) {
	parent := t.TempDir()
	root, err := Init(parent, "commerce")
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(parent, "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(root, other); err == nil {
		t.Fatal("Add() error = nil")
	}
}

func TestListReportsMissingAndMalformedProjects(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(string) ProjectRef
	}{
		{name: "missing", setup: func(string) ProjectRef { return ProjectRef{Path: "./missing"} }},
		{name: "malformed", setup: func(root string) ProjectRef {
			path := filepath.Join(root, "broken")
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, project.ManifestName), []byte("name: ["), 0o644); err != nil {
				t.Fatal(err)
			}
			return ProjectRef{Path: "./broken"}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()
			root, err := Init(parent, "commerce")
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := Read(root)
			if err != nil {
				t.Fatal(err)
			}
			manifest.Projects = []ProjectRef{test.setup(root)}
			if err := write(filepath.Join(root, ManifestName), manifest); err != nil {
				t.Fatal(err)
			}
			if _, err := List(root); err == nil {
				t.Fatal("List() error = nil")
			}
		})
	}
}
