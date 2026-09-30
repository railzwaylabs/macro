package http

import (
	"context"
	"errors"
	"net"
	stdhttp "net/http"
	"strings"
)

const DefaultAddress = ":8080"

type Server struct {
	address  string
	server   *stdhttp.Server
	listener net.Listener
}

func New(address string, handler stdhttp.Handler) *Server {
	if strings.TrimSpace(address) == "" {
		address = DefaultAddress
	}

	if handler == nil {
		handler = stdhttp.NewServeMux()
	}

	return &Server{address: address, server: &stdhttp.Server{Addr: address, Handler: handler}}
}

func (server *Server) HTTP() *stdhttp.Server { return server.server }

func (server *Server) Address() string {
	if server.listener != nil {
		return server.listener.Addr().String()
	}
	return server.address
}

func (server *Server) Start(context.Context) error {
	listener, err := net.Listen("tcp", server.address)
	if err != nil {
		return err
	}

	server.listener = listener
	go func() {
		_ = server.server.Serve(listener)
	}()

	return nil
}

func (server *Server) Stop(ctx context.Context) error {
	err := server.server.Shutdown(ctx)
	if errors.Is(err, stdhttp.ErrServerClosed) {
		return nil
	}

	return err
}
