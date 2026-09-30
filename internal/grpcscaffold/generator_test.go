package grpcscaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	modulegenerator "github.com/railzwaylabs/macro/internal/module"
	"github.com/railzwaylabs/macro/internal/project"
)

func TestAddGatewayCreatesDeterministicDualTransport(t *testing.T) {
	generated, err := project.Generate(project.GenerateOptions{Parent: t.TempDir(), Name: "catalogue", Type: project.TypeService, ModulePath: "example.com/catalogue"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := modulegenerator.Add(generated.Directory, "product"); err != nil {
		t.Fatal(err)
	}
	result, err := Add(Options{ProjectDirectory: generated.Directory, Name: "product", Gateway: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 7 {
		t.Fatalf("files = %d", len(result.Files))
	}
	assertContains(t, filepath.Join(generated.Directory, "api/proto/catalogue/v1/product.proto"),
		`import "google/api/annotations.proto";`, `post: "/v1/products"`, `get: "/v1/products/{id}"`)
	assertContains(t, filepath.Join(generated.Directory, "internal/modules/modules.go"),
		"product.Module.With(product.GRPC, product.Gateway)", "macro.WithGateway()")
	assertContains(t, filepath.Join(generated.Directory, "internal/product/transport.go"),
		"RegisterProductServiceServer", "RegisterProductServiceHandlerServer")
	assertContains(t, filepath.Join(generated.Directory, "buf.gen.yaml"), "buf.build/grpc-ecosystem/gateway")
	manifest, err := project.Read(generated.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.Modules["product"].GRPC || !manifest.Modules["product"].Gateway || !manifest.Runtime.HTTP.Enabled {
		t.Fatalf("manifest = %#v", manifest)
	}
	if _, err := Add(Options{ProjectDirectory: generated.Directory, Name: "product", Gateway: true}); err == nil {
		t.Fatal("duplicate Add() error = nil")
	}
}

func TestAddGRPCDoesNotEnableHTTP(t *testing.T) {
	generated, err := project.Generate(project.GenerateOptions{Parent: t.TempDir(), Name: "catalogue", Type: project.TypeService, ModulePath: "example.com/catalogue"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := modulegenerator.Add(generated.Directory, "product"); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(Options{ProjectDirectory: generated.Directory, Name: "product"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := project.Read(generated.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Runtime.HTTP.Enabled || manifest.Modules["product"].Gateway {
		t.Fatalf("HTTP unexpectedly enabled: %#v", manifest)
	}
	proto, err := os.ReadFile(filepath.Join(generated.Directory, "api/proto/catalogue/v1/product.proto"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(proto), "google.api.http") {
		t.Fatal("gRPC-only proto contains HTTP annotations")
	}
}

func assertContains(t *testing.T, path string, fragments ...string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(string(contents), fragment) {
			t.Errorf("%s missing %q", path, fragment)
		}
	}
}
