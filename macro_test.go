package macro_test

import (
	"testing"

	"github.com/railzwaylabs/macro"
	"github.com/railzwaylabs/macro/logger"
	"go.uber.org/zap"
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
