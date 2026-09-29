package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNotFound = errors.New("macro workspace not found")

func Find(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve workspace search path: %w", err)
	}

	for {
		path := filepath.Join(directory, ManifestName)
		if _, err := os.Stat(path); err == nil {
			return directory, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect %s: %w", path, err)
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", ErrNotFound
		}

		directory = parent
	}
}
