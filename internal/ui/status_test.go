package ui

import (
	"bytes"
	"errors"
	"testing"
)

func TestStatusRun(t *testing.T) {
	var output bytes.Buffer
	status := NewStatus(&output)

	called := false
	err := status.Run(Step{
		Start:   "Creating project",
		Success: "Created billing",
		Failure: "Failed to create billing",
		Run: func() error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("Run() did not execute step")
	}
	if got, want := output.String(), "✓ Created billing\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestStatusRunFailure(t *testing.T) {
	var output bytes.Buffer
	status := NewStatus(&output)
	wantErr := errors.New("boom")

	err := status.Run(Step{
		Start:   "Creating project",
		Success: "Created billing",
		Failure: "Failed to create billing",
		Run:     func() error { return wantErr },
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if got, want := output.String(), "✗ Failed to create billing\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
