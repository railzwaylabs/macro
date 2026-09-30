package grpc

import (
	"context"
	"net"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/railzwaylabs/macro/logger"
)

const DefaultAddress = ":4318"

type Option func(*options)

type options struct {
	logger     *logger.Logger
	reflection bool
	server     []grpc.ServerOption
}

func WithLogger(log *logger.Logger) Option {
	return func(options *options) { options.logger = log }
}

func WithReflection() Option {
	return func(options *options) { options.reflection = true }
}

func WithServerOptions(serverOptions ...grpc.ServerOption) Option {
	return func(options *options) { options.server = append(options.server, serverOptions...) }
}

type Server struct {
	address  string
	server   *grpc.Server
	listener net.Listener
}

func New(address string, configured ...Option) *Server {
	if strings.TrimSpace(address) == "" {
		address = DefaultAddress
	}

	settings := options{}
	for _, configure := range configured {
		if configure != nil {
			configure(&settings)
		}
	}
	unary := []grpc.UnaryServerInterceptor{recoveryInterceptor()}
	if settings.logger != nil {
		unary = append(unary, requestLogger(settings.logger))
	}
	settings.server = append(settings.server, grpc.ChainUnaryInterceptor(unary...))
	native := grpc.NewServer(settings.server...)
	grpc_health_v1.RegisterHealthServer(native, health.NewServer())
	if settings.reflection {
		reflection.Register(native)
	}
	return &Server{
		address: address,
		server:  native,
	}
}

// GRPC returns the underlying gRPC server for generated service registration.
// Register services before starting the Macro service.
func (s *Server) GRPC() *grpc.Server {
	return s.server
}

func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.address
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

func recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, request)
	}
}

func requestLogger(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		response, err := handler(ctx, request)
		log.Zap().Info("grpc request", zap.String("method", info.FullMethod), zap.String("status", status.Code(err).String()))
		return response, err
	}
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
