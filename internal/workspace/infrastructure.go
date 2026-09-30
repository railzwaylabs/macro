package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/railzwaylabs/macro/internal/fsutil"
	"github.com/railzwaylabs/macro/internal/project"
)

const InfrastructureDirectory = ".macro/infra"

var validComponents = map[string]bool{"postgres": true, "redis": true, "observability": true}

type GeneratedFile struct {
	Path     string
	Contents []byte
}

func ValidateProfile(profile Profile) error {
	if profile != ProfileNginxCompose && profile != ProfileTraefikNomad {
		return fmt.Errorf("invalid infrastructure profile %q: use nginx-compose or traefik-nomad", profile)
	}
	return nil
}

func AddInfrastructure(directory, component string, profile Profile) error {
	component = strings.ToLower(strings.TrimSpace(component))
	if !validComponents[component] {
		return fmt.Errorf("invalid infrastructure component %q: use postgres, redis, or observability", component)
	}
	manifest, err := Read(directory)
	if err != nil {
		return err
	}
	if manifest.Infrastructure == nil {
		manifest.Infrastructure = &Infrastructure{Profile: profile}
	}
	if err := ValidateProfile(manifest.Infrastructure.Profile); err != nil {
		return err
	}
	for _, existing := range manifest.Infrastructure.Components {
		if existing == component {
			return nil
		}
	}
	manifest.Infrastructure.Components = append(manifest.Infrastructure.Components, component)
	sort.Strings(manifest.Infrastructure.Components)
	return write(filepath.Join(directory, ManifestName), manifest)
}

func ValidateInfrastructure(directory string, manifest Manifest) error {
	infra := manifest.Infrastructure
	if infra == nil {
		return nil
	}
	if err := ValidateProfile(infra.Profile); err != nil {
		return err
	}
	seenComponents := map[string]bool{}
	for _, component := range infra.Components {
		if !validComponents[component] {
			return fmt.Errorf("invalid component %q", component)
		}
		if seenComponents[component] {
			return fmt.Errorf("duplicate component %q", component)
		}
		seenComponents[component] = true
	}
	projects := map[string]project.Manifest{}
	for _, ref := range manifest.Projects {
		value, err := project.Read(filepath.Join(directory, filepath.FromSlash(ref.Path)))
		if err != nil {
			return fmt.Errorf("project %s: %w", ref.Path, err)
		}
		projects[value.Name] = value
	}
	seenRoutes := map[string]bool{}
	for _, route := range infra.Routes {
		p, ok := projects[route.Project]
		if !ok {
			return fmt.Errorf("route references unknown project %q", route.Project)
		}
		if p.Type != project.TypeService || !p.Runtime.HTTP.Enabled {
			return fmt.Errorf("route project %q must be an HTTP-enabled service", route.Project)
		}
		if strings.TrimSpace(route.Host) == "" || strings.ContainsAny(route.Host, "/ ") {
			return fmt.Errorf("route for %q has invalid host %q", route.Project, route.Host)
		}
		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route for %q path must start with /", route.Project)
		}
		key := strings.ToLower(route.Host) + " " + route.Path
		if seenRoutes[key] {
			return fmt.Errorf("conflicting route %s%s", route.Host, route.Path)
		}
		seenRoutes[key] = true
	}
	for _, target := range infra.Metrics {
		p, ok := projects[target.Project]
		if !ok {
			return fmt.Errorf("metrics references unknown project %q", target.Project)
		}
		if !p.Runtime.Metrics.Enabled {
			return fmt.Errorf("metrics are not enabled for project %q", target.Project)
		}
		if !strings.HasPrefix(target.Path, "/") {
			return fmt.Errorf("metrics path for %q must start with /", target.Project)
		}
	}
	return nil
}

func PlanInfrastructure(directory string) ([]GeneratedFile, error) {
	manifest, err := Read(directory)
	if err != nil {
		return nil, err
	}
	if manifest.Infrastructure == nil {
		manifest.Infrastructure = &Infrastructure{Profile: ProfileNginxCompose}
	}
	if err := ValidateInfrastructure(directory, manifest); err != nil {
		return nil, err
	}
	projects := map[string]project.Manifest{}
	paths := map[string]string{}
	for _, ref := range manifest.Projects {
		p, _ := project.Read(filepath.Join(directory, filepath.FromSlash(ref.Path)))
		projects[p.Name] = p
		paths[p.Name] = ref.Path
	}
	files := generateFiles(manifest, projects, paths)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func GenerateInfrastructure(directory string, force bool) ([]GeneratedFile, error) {
	files, err := PlanInfrastructure(directory)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(directory, filepath.FromSlash(InfrastructureDirectory))
	if _, err := os.Stat(target); err == nil && !force {
		return nil, fmt.Errorf("generated infrastructure already exists at %s; use --force to replace it", target)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(parent, ".infra-stage-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	for _, file := range files {
		path := filepath.Join(stage, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := fsutil.WriteFileAtomic(path, file.Contents, 0o644); err != nil {
			return nil, err
		}
	}
	backup := target + ".previous"
	if force {
		_ = os.RemoveAll(backup)
		if _, err := os.Stat(target); err == nil {
			if err := os.Rename(target, backup); err != nil {
				return nil, err
			}
		}
	}
	if err := os.Rename(stage, target); err != nil {
		if force {
			_ = os.Rename(backup, target)
		}
		return nil, err
	}
	if force {
		_ = os.RemoveAll(backup)
	}
	return files, nil
}

func has(infra *Infrastructure, name string) bool {
	for _, item := range infra.Components {
		if item == name {
			return true
		}
	}
	return false
}

func add(files *[]GeneratedFile, path, body string) {
	*files = append(*files, GeneratedFile{Path: path, Contents: []byte(strings.TrimSpace(body) + "\n")})
}

func generateFiles(m Manifest, projects map[string]project.Manifest, paths map[string]string) []GeneratedFile {
	var files []GeneratedFile
	i := m.Infrastructure
	add(&files, "README.md", "# Generated infrastructure\n\nManaged by Macro v0.5.0. Regenerate with `macro workspace infra generate --force`. Do not edit files in this directory; keep user-owned overrides outside `.macro/infra`.\n\nThis is a development baseline, not a production deployment specification.")
	add(&files, ".env.example", "# Copy values to your own environment; never commit real secrets.\nPOSTGRES_PASSWORD=change-me\nVAULT_DEV_ROOT_TOKEN_ID=change-me")
	if i.Profile == ProfileNginxCompose {
		generateCompose(&files, m, projects, paths)
	} else {
		generateNomad(&files, m, projects)
	}
	return files
}

func generateCompose(files *[]GeneratedFile, m Manifest, projects map[string]project.Manifest, paths map[string]string) {
	i := m.Infrastructure
	var services bytes.Buffer
	services.WriteString("services:\n  gateway:\n    image: nginx:1.27.5-alpine\n    ports: [\"8080:80\"]\n    volumes: [\"./nginx/nginx.conf:/etc/nginx/nginx.conf:ro\"]\n")
	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		context := filepath.ToSlash(filepath.Join("../..", paths[name]))
		fmt.Fprintf(&services, "  %s:\n    build:\n      context: %s\n", name, context)
	}

	if has(i, "postgres") {
		services.WriteString("  postgres:\n    image: postgres:17.4-alpine\n    environment:\n      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}\n    volumes: [\"postgres-data:/var/lib/postgresql/data\"]\n")
	}

	if has(i, "redis") {
		services.WriteString("  redis:\n    image: redis:7.4.2-alpine\n    command: [\"redis-server\", \"--appendonly\", \"yes\"]\n    volumes: [\"redis-data:/data\"]\n")
	}

	if has(i, "observability") {
		services.WriteString("  prometheus:\n    image: prom/prometheus:v3.2.1\n    volumes: [\"./observability/prometheus.yml:/etc/prometheus/prometheus.yml:ro\"]\n  loki:\n    image: grafana/loki:3.4.2\n    command: [\"-config.file=/etc/loki/config.yml\"]\n    volumes: [\"./observability/loki.yml:/etc/loki/config.yml:ro\", \"loki-data:/loki\"]\n  alloy:\n    image: grafana/alloy:v1.7.5\n    command: [\"run\", \"/etc/alloy/config.alloy\"]\n    volumes: [\"./observability/config.alloy:/etc/alloy/config.alloy:ro\", \"/var/run/docker.sock:/var/run/docker.sock:ro\"]\n")
	}

	services.WriteString("volumes:\n")
	if has(i, "postgres") {
		services.WriteString("  postgres-data: {}\n")
	}

	if has(i, "redis") {
		services.WriteString("  redis-data: {}\n")
	}

	if has(i, "observability") {
		services.WriteString("  loki-data: {}\n")
	}

	add(files, "compose.yaml", services.String())
	var nginx bytes.Buffer
	nginx.WriteString("events {}\nhttp {\n")
	for _, r := range i.Routes {
		p := projects[r.Project]
		fmt.Fprintf(&nginx, "  server { listen 80; server_name %s; location %s { proxy_pass http://%s:%d; proxy_set_header Host $host; } }\n", r.Host, r.Path, r.Project, p.Runtime.HTTP.Port)
	}

	nginx.WriteString("}\n")
	add(files, "nginx/nginx.conf", nginx.String())
	if has(i, "observability") {
		addObservability(files, m, projects, false)
	}
}

func generateNomad(files *[]GeneratedFile, m Manifest, projects map[string]project.Manifest) {
	i := m.Infrastructure
	add(files, "bootstrap/consul.hcl", "datacenter = \"dc1\"\ndata_dir = \"/opt/consul\"\nserver = true\nbootstrap_expect = 1\nclient_addr = \"0.0.0.0\"")
	add(files, "bootstrap/nomad.hcl", "data_dir = \"/opt/nomad\"\nbind_addr = \"0.0.0.0\"\ndatacenter = \"dc1\"\nserver {\n  enabled = true\n  bootstrap_expect = 1\n}\nclient {\n  enabled = true\n}")
	add(files, "bootstrap/vault.hcl", "storage \"file\" {\n  path = \"/opt/vault/data\"\n}\nlistener \"tcp\" {\n  address = \"0.0.0.0:8200\"\n  tls_disable = 1\n}\napi_addr = \"http://127.0.0.1:8200\"\n# Development baseline: initialize and unseal Vault separately.")
	add(files, "jobs/gateway.nomad.hcl", "job \"macro-gateway\" {\n  datacenters = [\"dc1\"]\n  group \"traefik\" {\n    network { port \"http\" { static = 80 } }\n    service {\n      name = \"traefik\"\n      port = \"http\"\n    }\n    task \"traefik\" {\n      driver = \"docker\"\n      config {\n        image = \"traefik:v3.3.4\"\n        ports = [\"http\"]\n        args = [\"--providers.consulcatalog=true\", \"--entrypoints.web.address=:80\"]\n      }\n    }\n  }\n}")
	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := projects[name]
		var tags []string
		for _, r := range i.Routes {
			if r.Project == name {
				tags = append(tags, "\"traefik.enable=true\"", fmt.Sprintf("\"traefik.http.routers.%s.rule=Host(`%s`) && PathPrefix(`%s`)\"", name, r.Host, r.Path))
			}
		}
		service, network := "", ""
		if p.Type == project.TypeService && p.Runtime.HTTP.Enabled {
			service = fmt.Sprintf("service { name = \"%s\"; port = \"http\"; tags = [%s] }", name, strings.Join(tags, ", "))
			network = fmt.Sprintf("network { port \"http\" { to = %d } }", p.Runtime.HTTP.Port)
		}
		add(files, "jobs/projects/"+name+".nomad.hcl", fmt.Sprintf("job \"%s\" {\n  datacenters = [\"dc1\"]\n  group \"%s\" {\n    %s\n    %s\n    task \"%s\" {\n      driver = \"docker\"\n      config { image = \"${NOMAD_VAR_%s_image}\" }\n    }\n  }\n}", name, name, network, service, name, name))
	}

	if has(i, "postgres") {
		add(files, "jobs/postgres.nomad.hcl", "job \"postgres\" { datacenters = [\"dc1\"]; group \"postgres\" { volume \"data\" { type = \"host\"; source = \"postgres-data\" } task \"postgres\" { driver = \"docker\"; config { image = \"postgres:17.4-alpine\" } env { POSTGRES_PASSWORD = \"${NOMAD_VAR_postgres_password}\" } volume_mount { volume = \"data\"; destination = \"/var/lib/postgresql/data\" } } } }\n# Create a distinct database/schema and credentials per project; sharing is not assumed.")
	}

	if has(i, "redis") {
		add(files, "jobs/redis.nomad.hcl", "job \"redis\" { datacenters = [\"dc1\"]; group \"redis\" { task \"redis\" { driver = \"docker\"; config { image = \"redis:7.4.2-alpine\"; args = [\"redis-server\", \"--appendonly\", \"yes\"] } } } }")
	}

	if has(i, "observability") {
		addObservability(files, m, projects, true)
		add(files, "jobs/observability.nomad.hcl", "job \"observability\" { datacenters = [\"dc1\"] # Mount generated configs and persistent host volumes before submission.\n group \"monitoring\" { task \"prometheus\" { driver = \"docker\"; config { image = \"prom/prometheus:v3.2.1\" } } task \"loki\" { driver = \"docker\"; config { image = \"grafana/loki:3.4.2\" } } task \"alloy\" { driver = \"docker\"; config { image = \"grafana/alloy:v1.7.5\" } } } }")
	}

}

func addObservability(files *[]GeneratedFile, m Manifest, projects map[string]project.Manifest, nomad bool) {
	var scrape bytes.Buffer
	scrape.WriteString("global:\n  scrape_interval: 15s\nscrape_configs:\n")
	for _, v := range m.Infrastructure.Metrics {
		p := projects[v.Project]
		fmt.Fprintf(&scrape, "  - job_name: %s\n    metrics_path: %s\n    static_configs:\n      - targets: [\"%s:%d\"]\n", v.Project, v.Path, v.Project, p.Runtime.Metrics.Port)
	}

	add(files, "observability/prometheus.yml", scrape.String())
	add(files, "observability/loki.yml", "auth_enabled: false\nserver: { http_listen_port: 3100 }\ncommon:\n  path_prefix: /loki\n  replication_factor: 1\n  ring: { kvstore: { store: inmemory } }\nschema_config:\n  configs:\n    - { from: 2024-01-01, store: tsdb, object_store: filesystem, schema: v13, index: { prefix: index_, period: 24h } }\nstorage_config: { filesystem: { directory: /loki/chunks } }")

	if nomad {
		add(files, "observability/config.alloy", "local.file_match \"nomad_logs\" { path_targets = [{ __path__ = \"/alloc/logs/*.stdout.*\", environment = \"nomad\" }] }\nloki.source.file \"nomad\" { targets = local.file_match.nomad_logs.targets; forward_to = [loki.write.local.receiver] }\nloki.write \"local\" { endpoint { url = \"http://loki.service.consul:3100/loki/api/v1/push\" } }")
	} else {
		add(files, "observability/config.alloy", "discovery.docker \"containers\" { host = \"unix:///var/run/docker.sock\" }\ndiscovery.relabel \"containers\" { targets = discovery.docker.containers.targets; rule { target_label = \"environment\"; replacement = \"compose\" } }\nloki.source.docker \"containers\" { host = \"unix:///var/run/docker.sock\"; targets = discovery.relabel.containers.output; forward_to = [loki.write.local.receiver] }\nloki.write \"local\" { endpoint { url = \"http://loki:3100/loki/api/v1/push\" } }")
	}
}
