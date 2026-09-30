package macro_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/railzwaylabs/macro"
	"github.com/railzwaylabs/macro/logger"
)

func TestNewFacade(t *testing.T) {
	t.Parallel()

	log, err := logger.New(logger.Config{
		Service: "billing",
		Mode:    "info",
	})
	if err != nil {
		t.Fatalf("logger.New() error = %v", err)
	}

	svc := macro.New(
		macro.Name("billing"),
		macro.WithLogger(log),
	)

	if got, want := svc.Name(), "billing"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got := svc.Logger(); got != log {
		t.Fatalf("Logger() = %p, want %p", got, log)
	}
	if got := svc.Options().Logger; got != log {
		t.Fatalf("Options().Logger = %p, want %p", got, log)
	}
	if svc.Logger().Zap().Core().Enabled(zap.DebugLevel) {
		t.Fatal("debug logging unexpectedly enabled")
	}

	if err := svc.Logger().SetMode("debug"); err != nil {
		t.Fatalf("SetMode() error = %v", err)
	}
	if !svc.Logger().Zap().Core().Enabled(zap.DebugLevel) {
		t.Fatal("debug logging was not enabled through service logger")
	}
}

func TestTransportConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		options  []macro.Option
		wantHTTP bool
		wantGRPC bool
	}{
		{name: "http only", options: []macro.Option{macro.WithoutGRPC(), macro.WithHTTP(http.NewServeMux()), macro.HTTPAddress("127.0.0.1:0")}, wantHTTP: true},
		{name: "grpc only", options: []macro.Option{macro.GRPCAddress("127.0.0.1:0")}, wantGRPC: true},
		{name: "http and grpc", options: []macro.Option{macro.GRPCAddress("127.0.0.1:0"), macro.WithHTTP(http.NewServeMux()), macro.HTTPAddress("127.0.0.1:0")}, wantHTTP: true, wantGRPC: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := macro.NewService("transport-test", append(test.options, macro.WithoutDebug())...)
			if (service.HTTP() != nil) != test.wantHTTP || (service.GRPC() != nil) != test.wantGRPC {
				t.Fatalf("HTTP=%t GRPC=%t", service.HTTP() != nil, service.GRPC() != nil)
			}
		})
	}
}

func TestDependencyWiringValidation(t *testing.T) {
	service := macro.NewService(
		"wiring-test",
		macro.WithModules(macro.NewModule("broken", fx.Invoke(func(string) {}))),
		macro.WithoutGRPC(),
		macro.WithoutDebug(),
	)
	err := service.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing type") {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestConflictingTransportAddressesFailAtStartup(t *testing.T) {
	service := macro.NewService(
		"conflict",
		macro.GRPCAddress("127.0.0.1:9999"),
		macro.WithHTTP(http.NewServeMux()),
		macro.HTTPAddress("127.0.0.1:9999"),
		macro.WithoutDebug(),
	)
	err := service.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "same listener address") {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestNewServiceProvidesRuntimeDefaults(t *testing.T) {
	t.Parallel()

	svc := macro.NewService("billing")

	if got, want := svc.Name(), "billing"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if svc.Logger() == nil {
		t.Fatal("Logger() = nil, want default logger")
	}
	if svc.GRPC() == nil {
		t.Fatal("GRPC() = nil, want default gRPC server")
	}
	if got, want := len(svc.Options().Servers), 2; got != want {
		t.Fatalf("server count = %d, want %d (gRPC and debug)", got, want)
	}
}

func TestNewServiceCanDisableDefaultServers(t *testing.T) {
	t.Parallel()

	svc := macro.NewService(
		"worker",
		macro.WithoutGRPC(),
		macro.WithoutDebug(),
	)

	if svc.GRPC() != nil {
		t.Fatal("GRPC() is non-nil with WithoutGRPC")
	}
	if got := len(svc.Options().Servers); got != 0 {
		t.Fatalf("server count = %d, want 0", got)
	}
}
