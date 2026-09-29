package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "manifest.yaml")

	if err := WriteFileAtomic(path, []byte("first"), 0o640); err != nil {
		t.Fatalf("WriteFileAtomic() first write error = %v", err)
	}
	if err := WriteFileAtomic(path, []byte("second"), 0o640); err != nil {
		t.Fatalf("WriteFileAtomic() replacement error = %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "second" {
		t.Fatalf("contents = %q, want %q", contents, "second")
	}
}
