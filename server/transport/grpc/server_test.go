package grpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewUsesDefaultAddress(t *testing.T) {
	t.Parallel()

	server := New("")
	if got, want := server.address, DefaultAddress; got != want {
		t.Fatalf("address = %q, want %q", got, want)
	}
}

func TestHealthServiceWithBufconn(t *testing.T) {
	t.Parallel()
	listener := bufconn.Listen(1024 * 1024)
	server := New("")
	go func() { _ = server.GRPC().Serve(listener) }()
	t.Cleanup(server.GRPC().Stop)

	connection, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	response, err := grpc_health_v1.NewHealthClient(connection).Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("health status = %s", response.Status)
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
