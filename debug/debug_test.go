package debug_test

import (
	"testing"

	"github.com/railzwaylabs/macro/debug"
)

func TestNewUsesDefaultAddress(t *testing.T) {
	t.Parallel()

	server := debug.New(debug.Config{}, nil)
	if got, want := server.Address(), debug.DefaultAddress; got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}
}

func TestNewUsesConfiguredAddress(t *testing.T) {
	t.Parallel()

	server := debug.New(debug.Config{Address: "127.0.0.1:6061"}, nil)
	if got, want := server.Address(), "127.0.0.1:6061"; got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}
}
