package grpc

import (
	"context"
	"errors"
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

type RegisterFunc func(*grpc.Server)

func New(address string, register RegisterFunc) *Server {
	if strings.TrimSpace(address) == "" {
		address = DefaultAddress
	}

	srv := grpc.NewServer()

	if register != nil {
		register(srv)
	}

	return &Server{
		address: address,
		server:  srv,
	}
}

func (s *Server) Start(context.Context) error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.listener = listener

	go func() {
		if err := s.server.Serve(listener); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			// Kirim error ke service error channel/logger.
		}
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
