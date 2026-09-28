// Package debug provides the internal HTTP management server.
package debug

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"

	macrologger "github.com/railzwaylabs/macro/logger"
	"go.uber.org/zap"
)

const DefaultAddress = "127.0.0.1:6060"

var profiles = []string{
	"allocs",
	"block",
	"goroutine",
	"heap",
	"mutex",
	"threadcreate",
}

type Config struct {
	Address string
	Service string
}

// Server serves pprof and runtime logger controls over HTTP.
type Server struct {
	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	logger   *zap.Logger
}

// New creates a management server. An empty address uses DefaultAddress.
func New(cfg Config, log *macrologger.Logger) *Server {
	if strings.TrimSpace(cfg.Address) == "" {
		cfg.Address = DefaultAddress
	}
	if strings.TrimSpace(cfg.Service) == "" {
		cfg.Service = "macro"
	}

	mux := http.NewServeMux()
	registerPprof(mux)

	zapLogger := zap.NewNop()
	if log != nil {
		log.RegisterHandler(mux)
		zapLogger = log.Zap()
	}

	return &Server{
		server: &http.Server{
			Addr:    cfg.Address,
			Handler: mux,
		},
		logger: zapLogger.Named(fmt.Sprintf("%s_management", cfg.Service)),
	}
}

// Address returns the configured management server address.
func (s *Server) Address() string {
	return s.server.Addr
}

// Start starts the management HTTP listener.
func (s *Server) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return nil
	}

	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.server.Addr, err)
	}
	s.listener = listener

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("management server stopped", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the management HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown management server: %w", err)
	}
	s.listener = nil
	return nil
}

func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("POST /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	for _, profile := range profiles {
		mux.Handle("GET /debug/pprof/"+profile, pprof.Handler(profile))
	}
}
