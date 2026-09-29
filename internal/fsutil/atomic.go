package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic replaces path only after all data has been written and synced.
func WriteFileAtomic(path string, contents []byte, mode os.FileMode) (returnErr error) {
	directory := filepath.Dir(path)
	temporaryFile, err := os.CreateTemp(directory, ".macro-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}

	temporaryPath := temporaryFile.Name()
	defer func() {
		cleanupErr := os.Remove(temporaryPath)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove temporary file %s: %w", temporaryPath, cleanupErr))
		}
	}()

	if err := temporaryFile.Chmod(mode); err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("set permissions on temporary file for %s: %w", path, err)
	}

	if _, err := temporaryFile.Write(contents); err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("write temporary file for %s: %w", path, err)
	}

	if err := temporaryFile.Sync(); err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("sync temporary file for %s: %w", path, err)
	}

	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", path, err)
	}

	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}

	return nil
}
