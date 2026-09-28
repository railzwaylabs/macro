package grpc

import "testing"

func TestNewUsesDefaultAddress(t *testing.T) {
	t.Parallel()

	server := New("", nil)
	if got, want := server.address, DefaultAddress; got != want {
		t.Fatalf("address = %q, want %q", got, want)
	}
}

func TestNewUsesConfiguredAddress(t *testing.T) {
	t.Parallel()

	server := New(":9000", nil)
	if got, want := server.address, ":9000"; got != want {
		t.Fatalf("address = %q, want %q", got, want)
	}
}
