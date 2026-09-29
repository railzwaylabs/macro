package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNotFound = errors.New("macro project not found")

// Find returns the nearest project directory at or above start.
func Find(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve project search path: %w", err)
	}

	for {
		manifestPath := filepath.Join(directory, ManifestName)
		if _, err := os.Stat(manifestPath); err == nil {
			return directory, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect %s: %w", manifestPath, err)
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", ErrNotFound
		}

		directory = parent
	}
}
