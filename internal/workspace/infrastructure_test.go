package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
)

func infrastructureWorkspace(t *testing.T, profile Profile) string {
	t.Helper()
	root, err := InitWithProfile(t.TempDir(), "commerce", profile)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := project.Generate(project.GenerateOptions{Parent: root, Name: "billing", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.Read(generated.Directory)
	if err != nil {
		t.Fatal(err)
	}
	p.Runtime.HTTP.Enabled = true
	if err := project.Write(generated.ManifestPath, p); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(root, generated.Directory); err != nil {
		t.Fatal(err)
	}
	m, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	m.Infrastructure.Routes = []Route{{Project: "billing", Host: "billing.local", Path: "/api"}}
	m.Infrastructure.Metrics = []Metrics{{Project: "billing", Path: "/internal/metrics"}}
	if err := write(filepath.Join(root, ManifestName), m); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLegacyWorkspaceDefaultsOnlyWhenInfrastructureIsUsed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ManifestName), []byte("name: legacy\nprojects: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Infrastructure != nil {
		t.Fatal("legacy manifest was mutated")
	}
	files, err := PlanInfrastructure(root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsGenerated(files, "compose.yaml", "nginx:1.27.5-alpine") {
		t.Fatal("legacy default did not use nginx-compose")
	}
}

func TestInfrastructureProfilesComponentsAndDeterminism(t *testing.T) {
	for _, profile := range []Profile{ProfileNginxCompose, ProfileTraefikNomad} {
		t.Run(string(profile), func(t *testing.T) {
			root := infrastructureWorkspace(t, profile)
			for _, component := range []string{"observability", "redis", "postgres"} {
				if err := AddInfrastructure(root, component, profile); err != nil {
					t.Fatal(err)
				}
			}
			first, err := PlanInfrastructure(root)
			if err != nil {
				t.Fatal(err)
			}
			second, err := PlanInfrastructure(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(first) != len(second) {
				t.Fatal("non-deterministic file count")
			}
			for index := range first {
				if first[index].Path != second[index].Path || !bytes.Equal(first[index].Contents, second[index].Contents) {
					t.Fatalf("non-deterministic output %s", first[index].Path)
				}
			}
			if !containsGenerated(first, "observability/prometheus.yml", "/internal/metrics") || !containsGenerated(first, "observability/config.alloy", "loki.write") || !containsGenerated(first, "observability/loki.yml", "filesystem") {
				t.Fatal("observability output incomplete")
			}
			if profile == ProfileNginxCompose {
				if !containsGenerated(first, "compose.yaml", "postgres:17.4-alpine") || !containsGenerated(first, "nginx/nginx.conf", "billing.local") {
					t.Fatal("compose output incomplete")
				}
			} else if !containsGenerated(first, "bootstrap/nomad.hcl", "bootstrap_expect") || !containsGenerated(first, "bootstrap/consul.hcl", "datacenter") || !containsGenerated(first, "bootstrap/vault.hcl", "storage") || !containsGenerated(first, "jobs/projects/billing.nomad.hcl", "traefik.http.routers.billing.rule") {
				t.Fatal("nomad output incomplete")
			}
		})
	}
}

func TestGenerateProtectsExistingOutput(t *testing.T) {
	root := infrastructureWorkspace(t, ProfileNginxCompose)
	if _, err := GenerateInfrastructure(root, false); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, InfrastructureDirectory, "user-change")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateInfrastructure(root, false); err == nil {
		t.Fatal("overwrite was allowed without --force")
	}
	contents, err := os.ReadFile(marker)
	if err != nil || string(contents) != "keep" {
		t.Fatal("existing output changed on refusal")
	}
	if _, err := GenerateInfrastructure(root, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("force did not replace managed directory")
	}
}

func TestInfrastructureValidation(t *testing.T) {
	root := infrastructureWorkspace(t, ProfileNginxCompose)
	m, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	m.Infrastructure.Routes = append(m.Infrastructure.Routes, Route{Project: "billing", Host: "billing.local", Path: "/api"})
	if err := ValidateInfrastructure(root, m); err == nil || !strings.Contains(err.Error(), "conflicting route") {
		t.Fatalf("error = %v", err)
	}
	m.Infrastructure.Routes = []Route{{Project: "missing", Host: "missing.local", Path: "/"}}
	if err := ValidateInfrastructure(root, m); err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("error = %v", err)
	}
}

func containsGenerated(files []GeneratedFile, path, part string) bool {
	for _, file := range files {
		if file.Path == path && strings.Contains(string(file.Contents), part) {
			return true
		}
	}
	return false
}
