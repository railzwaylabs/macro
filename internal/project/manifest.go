package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/railzwaylabs/macro/internal/fsutil"
)

const ManifestName = "macro.yaml"

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

var reservedNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

type Type string

const (
	TypeService Type = "service"
	TypeWorker  Type = "worker"
	TypeJob     Type = "job"
)

type Manifest struct {
	Name       string            `yaml:"name"`
	Type       Type              `yaml:"type"`
	Runtime    Runtime           `yaml:"runtime"`
	Modules    map[string]Module `yaml:"modules"`
	Deployment Deployment        `yaml:"deployment"`
}

// Module reserves a typed manifest value for future module configuration.
type Module struct{}

type Runtime struct {
	GRPCPort  *int `yaml:"grpc_port,omitempty"`
	DebugPort int  `yaml:"debug_port"`
}

type Deployment struct {
	Docker     bool `yaml:"docker"`
	Kubernetes bool `yaml:"kubernetes"`
	Nomad      bool `yaml:"nomad"`
}

func NewManifest(name string, kind Type) Manifest {
	manifest := Manifest{
		Name: name,
		Type: kind,
		Runtime: Runtime{
			DebugPort: 6060,
		},
		Modules: map[string]Module{},
		Deployment: Deployment{
			Docker:     true,
			Kubernetes: false,
			Nomad:      false,
		},
	}
	if kind == TypeService {
		port := 8000
		manifest.Runtime.GRPCPort = &port
	}
	return manifest
}

func ParseType(value string) (Type, error) {
	kind := Type(value)
	switch kind {
	case TypeService, TypeWorker, TypeJob:
		return kind, nil
	default:
		return "", fmt.Errorf("invalid project type %q: use service, worker, or job", value)
	}
}

func ValidateName(name string) error {
	if name == "" || name == "." || name == ".." || !validName.MatchString(name) {
		return fmt.Errorf("invalid project name %q: use letters, numbers, dash, or underscore", name)
	}
	if _, reserved := reservedNames[strings.ToUpper(name)]; reserved {
		return fmt.Errorf("invalid project name %q: reserved filesystem name", name)
	}
	return nil
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
	if err := ValidateName(manifest.Name); err != nil {
		return Manifest{}, fmt.Errorf("validate %s: %w", path, err)
	}
	if _, err := ParseType(string(manifest.Type)); err != nil {
		return Manifest{}, fmt.Errorf("validate %s: %w", path, err)
	}
	return manifest, nil
}

func Write(path string, manifest Manifest) error {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal project manifest: %w", err)
	}
	if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write project manifest: %w", err)
	}
	return nil
}
