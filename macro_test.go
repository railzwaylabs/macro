package macro_test

import (
	"testing"

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
