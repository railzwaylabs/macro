package project

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateLayouts(t *testing.T) {
	for _, test := range []struct {
		kind          Type
		command       string
		wantServiceDB bool
	}{
		{kind: TypeService, command: "service", wantServiceDB: true},
		{kind: TypeWorker, command: "worker"},
		{kind: TypeJob, command: "job"},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			parent := t.TempDir()
			result, err := Generate(GenerateOptions{Parent: parent, Name: "billing", Type: test.kind})
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(result.Directory, "cmd", test.command, "main.go")); err != nil {
				t.Fatalf("main.go: %v", err)
			}
			if _, err := parser.ParseFile(
				token.NewFileSet(),
				filepath.Join(result.Directory, "cmd", test.command, "main.go"),
				nil,
				parser.AllErrors,
			); err != nil {
				t.Fatalf("generated main.go is invalid: %v", err)
			}
			for _, path := range []string{
				"internal",
				"tests",
				"Dockerfile",
				"Makefile",
				"README.md",
				".gitignore",
				".golangci.yaml",
			} {
				if _, err := os.Stat(filepath.Join(result.Directory, path)); err != nil {
					t.Errorf("required path %s: %v", path, err)
				}
			}
			_, migrationErr := os.Stat(filepath.Join(result.Directory, "migrations"))
			_, protoErr := os.Stat(filepath.Join(result.Directory, "api", "proto", "billing", "v1", "billing.proto"))
			if test.wantServiceDB {
				if migrationErr != nil || protoErr != nil {
					t.Fatalf("service directories: migrations=%v proto=%v", migrationErr, protoErr)
				}
			} else if !os.IsNotExist(migrationErr) || !os.IsNotExist(protoErr) {
				t.Fatalf("non-service unexpectedly has service directories")
			}
			for _, deployment := range []string{filepath.Join("deploy", "kubernetes"), filepath.Join("deploy", "nomad")} {
				if _, err := os.Stat(filepath.Join(result.Directory, deployment)); !os.IsNotExist(err) {
					t.Errorf("unexpected deployment placeholder %s", deployment)
				}
			}
		})
	}
}

func TestGenerateUsesProjectModuleInLintConfiguration(t *testing.T) {
	generated, err := Generate(GenerateOptions{
		Parent:     t.TempDir(),
		Name:       "billing",
		Type:       TypeService,
		ModulePath: "example.com/commerce/billing",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	lintConfig := filepath.Join(generated.Directory, ".golangci.yaml")
	assertFileContains(t, lintConfig, "- example.com/commerce/billing")

	gitignore := filepath.Join(generated.Directory, ".gitignore")
	assertFileContains(t, gitignore, "/bin/", "/dist/", ".env", ".DS_Store")
}

func TestGenerateServiceProtoAndBufConfiguration(t *testing.T) {
	generated, err := Generate(GenerateOptions{
		Parent: t.TempDir(), Name: "billing-api", Type: TypeService,
		ModulePath: "example.com/commerce/billing-api",
	})
	if err != nil {
		t.Fatal(err)
	}
	proto := filepath.Join(generated.Directory, "api", "proto", "billing_api", "v1", "billing_api.proto")
	assertFileContains(t, proto,
		`syntax = "proto3";`,
		"package billing_api.v1;",
		`option go_package = "example.com/commerce/billing-api/gen/billing_api/v1;billingapiv1";`,
		"service BillingApiService {}",
	)
	contents, err := os.ReadFile(proto)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "rpc ") {
		t.Fatal("initial proto contains business RPC methods")
	}
	assertFileContains(t, filepath.Join(generated.Directory, "buf.yaml"), "version: v2", "path: api/proto")
	assertFileContains(t, filepath.Join(generated.Directory, "buf.gen.yaml"), "buf.build/protocolbuffers/go", "buf.build/grpc/go")
	assertFileContains(t, filepath.Join(generated.Directory, "Makefile"), "proto:", "buf generate")

	macroRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	runGoCommand(t, generated.Directory, "mod", "edit", "-replace=github.com/railzwaylabs/macro="+macroRoot)
	runGoCommand(t, generated.Directory, "mod", "tidy")
	runGoCommand(t, generated.Directory, "test", "./...")
	runGoCommand(t, generated.Directory, "build", "./...")
}

func TestManifestDeploymentMatchesGeneratedFiles(t *testing.T) {
	manifest := NewManifest("billing", TypeService)
	if !manifest.Deployment.Docker || manifest.Deployment.Kubernetes || manifest.Deployment.Nomad {
		t.Fatalf("deployment = %#v", manifest.Deployment)
	}
}

func TestManifestRuntimeDefaults(t *testing.T) {
	service := NewManifest("billing", TypeService)
	if service.Runtime.HTTP.Enabled || service.Runtime.HTTP.Port != 8080 || !service.Runtime.GRPC.Enabled || service.Runtime.GRPC.Port != 9000 {
		t.Fatalf("service transports = %#v", service.Runtime)
	}
	if !service.Runtime.Debug.Enabled || service.Runtime.Debug.Host != "127.0.0.1" || service.Runtime.Debug.Port != 6060 {
		t.Fatalf("debug = %#v", service.Runtime.Debug)
	}
	if !service.Runtime.Metrics.Enabled || service.Runtime.Metrics.Port != 9090 {
		t.Fatalf("metrics = %#v", service.Runtime.Metrics)
	}
	if service.Telemetry.Enabled || service.Telemetry.Protocol != "grpc" || service.Telemetry.Endpoint != "localhost:4317" {
		t.Fatalf("telemetry = %#v", service.Telemetry)
	}
	for _, kind := range []Type{TypeWorker, TypeJob} {
		manifest := NewManifest("task", kind)
		if manifest.Runtime.HTTP.Enabled || manifest.Runtime.GRPC.Enabled {
			t.Fatalf("%s unexpectedly enables application transports", kind)
		}
	}
}

func TestGenerateRejectsExistingDestination(t *testing.T) {
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, "billing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(GenerateOptions{Parent: parent, Name: "billing", Type: TypeService}); err == nil {
		t.Fatal("Generate() error = nil")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	directory := t.TempDir()
	manifest := NewManifest("rating-worker", TypeWorker)
	if err := Write(filepath.Join(directory, ManifestName), manifest); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	got, err := Read(directory)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.Name != manifest.Name || got.Type != manifest.Type || got.Runtime.GRPC.Enabled {
		t.Fatalf("manifest = %#v", got)
	}
}

func TestGenerateWorkerLifecycleScaffold(t *testing.T) {
	parent := t.TempDir()
	generated, err := Generate(GenerateOptions{
		Parent:     parent,
		Name:       "rating-worker",
		Type:       TypeWorker,
		ModulePath: "example.com/rating-worker",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	workerFile := filepath.Join(generated.Directory, "internal", "worker", "worker.go")
	workerTestFile := filepath.Join(generated.Directory, "internal", "worker", "worker_test.go")
	makefile := filepath.Join(generated.Directory, "Makefile")
	for _, path := range []string{workerFile, workerTestFile, makefile} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required worker file %s: %v", path, err)
		}
	}

	assertFileContains(t, makefile,
		"go run ./cmd/worker",
		"go test ./...",
		"go build -trimpath -o ./bin/app ./cmd/worker",
		"golangci-lint run",
		"docker build -t rating-worker .",
	)
	assertFileContains(t, workerFile, "Run(ctx context.Context) error", "<-ctx.Done()")

	moduleFile := filepath.Join(generated.Directory, "go.mod")
	moduleContents, err := os.ReadFile(moduleFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, broker := range []string{"kafka", "nats", "rabbitmq", "redis", "sqs", "pubsub"} {
		if strings.Contains(strings.ToLower(string(moduleContents)), broker) {
			t.Errorf("generated go.mod contains broker-specific dependency %q", broker)
		}
	}

	macroRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	runGoCommand(t, generated.Directory, "mod", "edit", "-replace=github.com/railzwaylabs/macro="+macroRoot)
	runGoCommand(t, generated.Directory, "mod", "tidy")
	runGoCommand(t, generated.Directory, "test", "./...")
}

func assertFileContains(t *testing.T, path string, expected ...string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, fragment := range expected {
		if !strings.Contains(string(contents), fragment) {
			t.Errorf("%s does not contain %q", path, fragment)
		}
	}
}

func runGoCommand(t *testing.T, directory string, arguments ...string) {
	t.Helper()

	command := exec.Command("go", arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
}
