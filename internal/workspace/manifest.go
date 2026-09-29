package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/railzwaylabs/macro/internal/fsutil"
	"github.com/railzwaylabs/macro/internal/project"
)

const ManifestName = "macro.workspace.yaml"

type Manifest struct {
	Name     string       `yaml:"name"`
	Projects []ProjectRef `yaml:"projects"`
}

type ProjectRef struct {
	Path string `yaml:"path"`
}

func Read(directory string) (Manifest, error) {
	path := filepath.Join(directory, ManifestName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", path, err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", path, err)
	}

	if err := project.ValidateName(manifest.Name); err != nil {
		return Manifest{}, fmt.Errorf("validate %s: %w", path, err)
	}

	if manifest.Projects == nil {
		manifest.Projects = []ProjectRef{}
	}

	return manifest, nil
}

func write(path string, manifest Manifest) error {
	contents, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal workspace manifest: %w", err)
	}

	if err := fsutil.WriteFileAtomic(path, contents, 0o644); err != nil {
		return fmt.Errorf("write workspace manifest: %w", err)
	}

	return nil
}
