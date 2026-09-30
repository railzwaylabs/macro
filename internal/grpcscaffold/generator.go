package grpcscaffold

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	modulegenerator "github.com/railzwaylabs/macro/internal/module"
	"github.com/railzwaylabs/macro/internal/project"
)

var ErrAlreadyExists = errors.New("gRPC transport already exists")

type Options struct {
	ProjectDirectory string
	Name             string
	Gateway          bool
}

type Result struct{ Files []string }

type data struct {
	Name, Package, Type, Proto, ProtoPackage, ModulePath, Route string
	Gateway                                                     bool
}

func Add(options Options) (Result, error) {
	if err := project.ValidateName(options.Name); err != nil {
		return Result{}, fmt.Errorf("invalid module name: %w", err)
	}
	manifest, err := project.Read(options.ProjectDirectory)
	if err != nil {
		return Result{}, err
	}
	if manifest.Type != project.TypeService {
		return Result{}, fmt.Errorf("gRPC transport requires a service project, got %s", manifest.Type)
	}
	if configuration, ok := manifest.Modules[options.Name]; !ok {
		return Result{}, fmt.Errorf("module %q is not registered: run macro add module %s first", options.Name, options.Name)
	} else if configuration.GRPC {
		return Result{}, fmt.Errorf("module %q: %w", options.Name, ErrAlreadyExists)
	}
	modulePath, err := readModulePath(options.ProjectDirectory)
	if err != nil {
		return Result{}, err
	}
	value := templateData(manifest.Name, options.Name, modulePath, options.Gateway)
	files := targets(options.ProjectDirectory, value)
	for _, file := range files {
		if _, err := os.Stat(file.path); err == nil {
			return Result{}, fmt.Errorf("refuse to overwrite %s", file.path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}
	created := make([]string, 0, len(files))
	rollback := func() {
		for _, path := range created {
			_ = os.Remove(path)
		}
	}
	for _, file := range files {
		if err := render(file.path, file.body, value); err != nil {
			rollback()
			return Result{}, err
		}
		created = append(created, file.path)
	}
	if options.Gateway {
		if err := enableGatewayPlugin(options.ProjectDirectory); err != nil {
			rollback()
			return Result{}, err
		}
	}
	if err := modulegenerator.ConfigureTransport(options.ProjectDirectory, options.Name, options.Gateway); err != nil {
		rollback()
		return Result{}, err
	}
	sort.Strings(created)
	return Result{Files: created}, nil
}

type generatedFile struct{ path, body string }

func targets(root string, d data) []generatedFile {
	moduleRoot := filepath.Join(root, "internal", d.Name)
	files := []generatedFile{
		{filepath.Join(root, "api", "proto", d.Proto, "v1", normalize(d.Name)+".proto"), protoTemplate},
		{filepath.Join(moduleRoot, "domain", normalize(d.Name)+".go"), domainTemplate},
		{filepath.Join(moduleRoot, "application", normalize(d.Name)+".go"), applicationTemplate},
		{filepath.Join(moduleRoot, "infrastructure", "memory", normalize(d.Name)+".go"), repositoryTemplate},
		{filepath.Join(moduleRoot, "transport", "grpc", normalize(d.Name)+"_server.go"), serverTemplate},
		{filepath.Join(moduleRoot, "transport", "grpc", normalize(d.Name)+"_server_test.go"), serverTestTemplate},
		{filepath.Join(moduleRoot, "transport.go"), transportTemplate},
	}
	return files
}

func templateData(projectName, name, modulePath string, gateway bool) data {
	proto := normalize(projectName)
	typeName := exported(name)
	return data{Name: name, Package: packageName(name), Type: typeName, Proto: proto,
		ProtoPackage: strings.ReplaceAll(proto, "_", "") + "v1", ModulePath: modulePath,
		Route: strings.ReplaceAll(normalize(name), "_", "-") + "s", Gateway: gateway}
}

func normalize(value string) string {
	var result strings.Builder
	underscore := false
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			result.WriteRune(r)
			underscore = false
		} else if result.Len() > 0 && !underscore {
			result.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(result.String(), "_")
}
func packageName(value string) string { return strings.ReplaceAll(normalize(value), "_", "") }
func exported(value string) string {
	parts := strings.Split(normalize(value), "_")
	var b strings.Builder
	for _, part := range parts {
		if part != "" {
			b.WriteString(strings.ToUpper(part[:1]))
			b.WriteString(part[1:])
		}
	}
	return b.String()
}

func readModulePath(root string) (string, error) {
	contents, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(contents))
	if len(fields) < 2 || fields[0] != "module" {
		return "", errors.New("go.mod must start with a module directive")
	}
	return fields[1], nil
}

func render(path, source string, d data) error {
	parsed, err := template.New(filepath.Base(path)).Parse(source)
	if err != nil {
		return err
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, d); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents := output.Bytes()
	if filepath.Ext(path) == ".go" {
		contents, err = format.Source(contents)
		if err != nil {
			return fmt.Errorf("format %s: %w", path, err)
		}
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func enableGatewayPlugin(root string) error {
	path := filepath.Join(root, "buf.gen.yaml")
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.Contains(contents, []byte("grpc-ecosystem/gateway")) {
		return nil
	}
	addition := []byte("  - remote: buf.build/grpc-ecosystem/gateway\n    out: gen\n    opt:\n      - paths=source_relative\n")
	return os.WriteFile(path, append(contents, addition...), 0o644)
}

const protoTemplate = `syntax = "proto3";

package {{ .Proto }}.v1;

option go_package = "{{ .ModulePath }}/gen/{{ .Proto }}/v1;{{ .ProtoPackage }}";
{{ if .Gateway }}
import "google/api/annotations.proto";
{{ end }}
service {{ .Type }}Service {
  rpc Create{{ .Type }}(Create{{ .Type }}Request) returns ({{ .Type }}) { {{ if .Gateway }}option (google.api.http) = { post: "/v1/{{ .Route }}" body: "*" };{{ end }} }
  rpc Get{{ .Type }}(Get{{ .Type }}Request) returns ({{ .Type }}) { {{ if .Gateway }}option (google.api.http) = { get: "/v1/{{ .Route }}/{id}" };{{ end }} }
}

message Create{{ .Type }}Request { string name = 1; }
message Get{{ .Type }}Request { string id = 1; }
message {{ .Type }} { string id = 1; string name = 2; }
`

const domainTemplate = `package domain

import (
  "errors"
  "strings"
)

var ErrInvalid{{ .Type }} = errors.New("{{ .Name }} name is required")

type {{ .Type }} struct { ID, Name string }

func New{{ .Type }}(id, name string) ({{ .Type }}, error) {
  name = strings.TrimSpace(name)
  if name == "" { return {{ .Type }}{}, ErrInvalid{{ .Type }} }
  return {{ .Type }}{ID: id, Name: name}, nil
}
`

const applicationTemplate = `package application

import (
  "context"
  "errors"
  "fmt"

  "{{ .ModulePath }}/internal/{{ .Name }}/domain"
)

var Err{{ .Type }}NotFound = errors.New("{{ .Name }} not found")

type {{ .Type }}Repository interface {
  Save(context.Context, domain.{{ .Type }}) error
  Get(context.Context, string) (domain.{{ .Type }}, error)
}

type {{ .Type }}Commands struct { repository {{ .Type }}Repository }
func New{{ .Type }}Commands(repository {{ .Type }}Repository) *{{ .Type }}Commands { return &{{ .Type }}Commands{repository: repository} }

func (service *{{ .Type }}Commands) Create(ctx context.Context, name string) (domain.{{ .Type }}, error) {
  entity, err := domain.New{{ .Type }}(fmt.Sprintf("{{ .Package }}-%s", name), name)
  if err != nil { return domain.{{ .Type }}{}, err }
  if err := service.repository.Save(ctx, entity); err != nil { return domain.{{ .Type }}{}, err }
  return entity, nil
}
func (service *{{ .Type }}Commands) Get(ctx context.Context, id string) (domain.{{ .Type }}, error) { return service.repository.Get(ctx, id) }
`

const repositoryTemplate = `package memory

import (
  "context"
  "sync"

  "{{ .ModulePath }}/internal/{{ .Name }}/application"
  "{{ .ModulePath }}/internal/{{ .Name }}/domain"
)

type {{ .Type }}Repository struct { mu sync.RWMutex; values map[string]domain.{{ .Type }} }
func New{{ .Type }}Repository() application.{{ .Type }}Repository { return &{{ .Type }}Repository{values: make(map[string]domain.{{ .Type }})} }
func (repository *{{ .Type }}Repository) Save(_ context.Context, value domain.{{ .Type }}) error { repository.mu.Lock(); defer repository.mu.Unlock(); repository.values[value.ID] = value; return nil }
func (repository *{{ .Type }}Repository) Get(_ context.Context, id string) (domain.{{ .Type }}, error) { repository.mu.RLock(); defer repository.mu.RUnlock(); value, ok := repository.values[id]; if !ok { return domain.{{ .Type }}{}, application.Err{{ .Type }}NotFound }; return value, nil }
`

const serverTemplate = `package grpc

import (
  "context"
  "errors"

  "google.golang.org/grpc/codes"
  "google.golang.org/grpc/status"

  {{ .ProtoPackage }} "{{ .ModulePath }}/gen/{{ .Proto }}/v1"
  "{{ .ModulePath }}/internal/{{ .Name }}/application"
  "{{ .ModulePath }}/internal/{{ .Name }}/domain"
)

type {{ .Type }}Server struct { {{ .ProtoPackage }}.Unimplemented{{ .Type }}ServiceServer; commands *application.{{ .Type }}Commands }
func New{{ .Type }}Server(commands *application.{{ .Type }}Commands) *{{ .Type }}Server { return &{{ .Type }}Server{commands: commands} }
func (server *{{ .Type }}Server) Create{{ .Type }}(ctx context.Context, request *{{ .ProtoPackage }}.Create{{ .Type }}Request) (*{{ .ProtoPackage }}.{{ .Type }}, error) {
  value, err := server.commands.Create(ctx, request.GetName()); if errors.Is(err, domain.ErrInvalid{{ .Type }}) { return nil, status.Error(codes.InvalidArgument, err.Error()) }; if err != nil { return nil, status.Error(codes.Internal, "create {{ .Name }}") }
  return &{{ .ProtoPackage }}.{{ .Type }}{Id: value.ID, Name: value.Name}, nil
}
func (server *{{ .Type }}Server) Get{{ .Type }}(ctx context.Context, request *{{ .ProtoPackage }}.Get{{ .Type }}Request) (*{{ .ProtoPackage }}.{{ .Type }}, error) {
  value, err := server.commands.Get(ctx, request.GetId()); if errors.Is(err, application.Err{{ .Type }}NotFound) { return nil, status.Error(codes.NotFound, err.Error()) }; if err != nil { return nil, status.Error(codes.Internal, "get {{ .Name }}") }
  return &{{ .ProtoPackage }}.{{ .Type }}{Id: value.ID, Name: value.Name}, nil
}
`

const serverTestTemplate = `package grpc

import (
  "context"
  "net"
  {{ if .Gateway }}"net/http"
  "net/http/httptest"
  "strings"{{ end }}
  "testing"

  {{ if .Gateway }}"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"{{ end }}
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
  "google.golang.org/grpc/test/bufconn"

  {{ .ProtoPackage }} "{{ .ModulePath }}/gen/{{ .Proto }}/v1"
  "{{ .ModulePath }}/internal/{{ .Name }}/application"
  "{{ .ModulePath }}/internal/{{ .Name }}/infrastructure/memory"
)

func newTestServer(t *testing.T) (*{{ .Type }}Server, {{ .ProtoPackage }}.{{ .Type }}ServiceClient) {
  t.Helper()
  handler := New{{ .Type }}Server(application.New{{ .Type }}Commands(memory.New{{ .Type }}Repository()))
  listener := bufconn.Listen(1024 * 1024)
  server := grpc.NewServer()
  {{ .ProtoPackage }}.Register{{ .Type }}ServiceServer(server, handler)
  go func() { _ = server.Serve(listener) }()
  t.Cleanup(func() { server.Stop(); _ = listener.Close() })
  connection, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
  if err != nil { t.Fatal(err) }
  t.Cleanup(func() { _ = connection.Close() })
  return handler, {{ .ProtoPackage }}.New{{ .Type }}ServiceClient(connection)
}

func Test{{ .Type }}GRPC(t *testing.T) {
  _, client := newTestServer(t)
  created, err := client.Create{{ .Type }}(context.Background(), &{{ .ProtoPackage }}.Create{{ .Type }}Request{Name: "Example"})
  if err != nil { t.Fatal(err) }
  found, err := client.Get{{ .Type }}(context.Background(), &{{ .ProtoPackage }}.Get{{ .Type }}Request{Id: created.GetId()})
  if err != nil || found.GetName() != "Example" { t.Fatalf("found = %v, error = %v", found, err) }
}
{{ if .Gateway }}
func Test{{ .Type }}Gateway(t *testing.T) {
  handler, _ := newTestServer(t)
  mux := runtime.NewServeMux()
  if err := {{ .ProtoPackage }}.Register{{ .Type }}ServiceHandlerServer(context.Background(), mux, handler); err != nil { t.Fatal(err) }
  request := httptest.NewRequest(http.MethodPost, "/v1/{{ .Route }}", strings.NewReader(` + "`" + `{"name":"Example"}` + "`" + `))
  request.Header.Set("Content-Type", "application/json")
  response := httptest.NewRecorder()
  mux.ServeHTTP(response, request)
  if response.Code != http.StatusOK { t.Fatalf("status = %d, body = %s", response.Code, response.Body.String()) }
}
{{ end }}`

const transportTemplate = `package {{ .Package }}

import (
  "context"

  "go.uber.org/fx"
  "google.golang.org/grpc"
  {{ if .Gateway }}"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"{{ end }}

  {{ .ProtoPackage }} "{{ .ModulePath }}/gen/{{ .Proto }}/v1"
  "{{ .ModulePath }}/internal/{{ .Name }}/application"
  "{{ .ModulePath }}/internal/{{ .Name }}/infrastructure/memory"
  transportgrpc "{{ .ModulePath }}/internal/{{ .Name }}/transport/grpc"
)

var GRPC = fx.Options(
  fx.Provide(memory.New{{ .Type }}Repository, application.New{{ .Type }}Commands, transportgrpc.New{{ .Type }}Server),
  fx.Invoke(func(server *grpc.Server, handler *transportgrpc.{{ .Type }}Server) { {{ .ProtoPackage }}.Register{{ .Type }}ServiceServer(server, handler) }),
)
{{ if .Gateway }}
// Gateway uses grpc-gateway's in-process adapter. HTTP and gRPC share the same
// application service, while HTTP calls intentionally bypass gRPC interceptors.
var Gateway = fx.Invoke(func(mux *runtime.ServeMux, handler *transportgrpc.{{ .Type }}Server) error {
  return {{ .ProtoPackage }}.Register{{ .Type }}ServiceHandlerServer(context.Background(), mux, handler)
})
{{ else }}
var _ = context.Background
{{ end }}`
