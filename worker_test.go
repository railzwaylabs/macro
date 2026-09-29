package macro

import (
	"context"
	"errors"
	"testing"
	"time"
)

type contextWorker struct {
	started chan struct{}
}

func TestRunWorkerRejectsNilWorker(t *testing.T) {
	if err := RunWorker("rating", nil); err == nil {
		t.Fatal("RunWorker() error = nil")
	}
}

func (worker *contextWorker) Run(ctx context.Context) error {
	close(worker.started)
	<-ctx.Done()
	return ctx.Err()
}

func TestRunWorkerStopsAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &contextWorker{started: make(chan struct{})}
	done := make(chan error, 1)

	go func() {
		done <- runWorker(ctx, "rating", worker, nil)
	}()

	<-worker.started
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runWorker() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runWorker() did not stop after context cancellation")
	}
}

func TestRunWorkerReturnsApplicationError(t *testing.T) {
	wantErr := errors.New("consumer unavailable")
	worker := workerFunc(func(context.Context) error { return wantErr })

	err := runWorker(context.Background(), "rating", worker, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("runWorker() error = %v, want wrapped %v", err, wantErr)
	}
}

type workerFunc func(context.Context) error

func (run workerFunc) Run(ctx context.Context) error {
	return run(ctx)
}
