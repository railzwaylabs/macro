package grpc

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
)

const DefaultAddress = ":8000"

type Server struct {
	address  string
	server   *grpc.Server
	listener net.Listener
}

func New(address string) *Server {
	if strings.TrimSpace(address) == "" {
		address = DefaultAddress
	}

	return &Server{
		address: address,
		server:  grpc.NewServer(),
	}
}

// GRPC returns the underlying gRPC server for generated service registration.
// Register services before starting the Macro service.
func (s *Server) GRPC() *grpc.Server {
	return s.server
}

func (s *Server) Start(context.Context) error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.listener = listener

	go func() {
		// Serve returns when the server is stopped. Runtime error propagation can
		// be added to the Server contract when lifecycle supervision is needed.
		_ = s.server.Serve(listener)
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	}
}
