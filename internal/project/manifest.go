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
	Telemetry  Telemetry         `yaml:"telemetry"`
}

type Module struct {
	GRPC    bool `yaml:"grpc,omitempty"`
	Gateway bool `yaml:"gateway,omitempty"`
}

type Runtime struct {
	HTTP    Listener `yaml:"http"`
	GRPC    Listener `yaml:"grpc"`
	Debug   Debug    `yaml:"debug"`
	Metrics Listener `yaml:"metrics"`
}

type Listener struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type Debug struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
}

type Telemetry struct {
	Enabled  bool   `yaml:"enabled"`
	Exporter string `yaml:"exporter"`
	Protocol string `yaml:"protocol"`
	Endpoint string `yaml:"endpoint"`
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
			HTTP:    Listener{Enabled: false, Port: 8080},
			GRPC:    Listener{Enabled: kind == TypeService, Port: 9000},
			Debug:   Debug{Enabled: true, Host: "127.0.0.1", Port: 6060},
			Metrics: Listener{Enabled: true, Port: 9090},
		},
		Telemetry: Telemetry{Enabled: false, Exporter: "otlp", Protocol: "grpc", Endpoint: "localhost:4317"},
		Modules:   map[string]Module{},
		Deployment: Deployment{
			Docker:     true,
			Kubernetes: false,
			Nomad:      false,
		},
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
