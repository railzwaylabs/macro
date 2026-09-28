package grpc

import "testing"

func TestNewUsesDefaultAddress(t *testing.T) {
	t.Parallel()

	server := New("")
	if got, want := server.address, DefaultAddress; got != want {
		t.Fatalf("address = %q, want %q", got, want)
	}
}

func TestNewUsesConfiguredAddress(t *testing.T) {
	t.Parallel()

	server := New(":9000")
	if got, want := server.address, ":9000"; got != want {
		t.Fatalf("address = %q, want %q", got, want)
	}
}

func TestGRPCReturnsNativeServer(t *testing.T) {
	t.Parallel()

	server := New("")
	if server.GRPC() == nil {
		t.Fatal("GRPC() = nil, want native gRPC server")
	}
}
