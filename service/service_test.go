package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/railzwaylabs/macro/service"
)

func TestNewAppliesOptions(t *testing.T) {
	t.Parallel()

	svc := service.New(service.Name("billing"))

	if got, want := svc.Name(), "billing"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}

	if got, want := svc.Options().Name, "billing"; got != want {
		t.Fatalf("Options().Name = %q, want %q", got, want)
	}
}

func TestNewUsesDefaultName(t *testing.T) {
	t.Parallel()

	svc := service.New()

	if got, want := svc.Name(), "macro"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestLifecycleIsIdempotent(t *testing.T) {
	t.Parallel()

	srv := &fakeServer{}
	svc := service.New(service.WithServer(srv))
	ctx := context.Background()
	for range 2 {
		if err := svc.Start(ctx); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	}
	for range 2 {
		if err := svc.Stop(ctx); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
	}

	if got, want := srv.starts, 1; got != want {
		t.Fatalf("server starts = %d, want %d", got, want)
	}
	if got, want := srv.stops, 1; got != want {
		t.Fatalf("server stops = %d, want %d", got, want)
	}
}

func TestShutdownTimeout(t *testing.T) {
	t.Parallel()

	const timeout = 3 * time.Second
	svc := service.New(service.ShutdownTimeout(timeout))

	if got := svc.Options().ShutdownTimeout; got != timeout {
		t.Fatalf("ShutdownTimeout = %s, want %s", got, timeout)
	}
}

func TestDefaultShutdownTimeout(t *testing.T) {
	t.Parallel()

	svc := service.New()

	if got, want := svc.Options().ShutdownTimeout, 10*time.Second; got != want {
		t.Fatalf("ShutdownTimeout = %s, want %s", got, want)
	}
}

type fakeServer struct {
	starts int
	stops  int
}

func (s *fakeServer) Start(context.Context) error {
	s.starts++
	return nil
}

func (s *fakeServer) Stop(context.Context) error {
	s.stops++
	return nil
}
