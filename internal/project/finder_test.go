package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFindWalksUpToProject(t *testing.T) {
	projectDirectory := t.TempDir()
	manifest := NewManifest("billing", TypeService)
	if err := Write(filepath.Join(projectDirectory, ManifestName), manifest); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(projectDirectory, "internal", "billing")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := Find(nested)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found != projectDirectory {
		t.Fatalf("Find() = %q, want %q", found, projectDirectory)
	}
}

func TestFindReturnsNotFound(t *testing.T) {
	_, err := Find(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find() error = %v, want ErrNotFound", err)
	}
}
