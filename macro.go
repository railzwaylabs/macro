package macro

import (
	"time"

	"google.golang.org/grpc"

	"github.com/railzwaylabs/macro/debug"
	"github.com/railzwaylabs/macro/logger"
	"github.com/railzwaylabs/macro/server"
	macrogrpc "github.com/railzwaylabs/macro/server/transport/grpc"
	"github.com/railzwaylabs/macro/service"
)

// Service is a ready-to-run Macro service with access to its native gRPC
// server for generated protobuf registration.
type Service interface {
	service.Service
	GRPC() *grpc.Server
}

// Option configures the default Macro runtime.
type Option func(*options)

type Options = service.Options

type options struct {
	name            string
	logger          *logger.Logger
	servers         []server.Server
	shutdownTimeout time.Duration
	grpcAddress     string
	debugAddress    string
	grpcEnabled     bool
	debugEnabled    bool
}

type macroService struct {
	service.Service
	grpcServer *macrogrpc.Server
}

// NewService creates a service with default logging, gRPC, diagnostics, signal
// handling, and graceful shutdown.
func NewService(name string, opts ...Option) Service {
	return newService(append([]Option{Name(name)}, opts...)...)
}

// New creates a service using the default name "macro".
// NewService is preferred when the service name is known.
func New(opts ...Option) Service {
	return newService(opts...)
}

func newService(opts ...Option) Service {
	config := options{
		name:            "macro",
		shutdownTimeout: 10 * time.Second,
		grpcEnabled:     true,
		debugEnabled:    true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}

	log := config.logger
	if log == nil {
		var err error
		log, err = logger.New(logger.Config{Service: config.name, Mode: "info"})
		if err != nil {
			panic("macro: create default logger: " + err.Error())
		}
	}

	servers := make([]server.Server, 0, len(config.servers)+2)
	var rpcServer *macrogrpc.Server
	if config.grpcEnabled {
		rpcServer = macrogrpc.New(config.grpcAddress)
		servers = append(servers, rpcServer)
	}
	if config.debugEnabled {
		servers = append(servers, debug.New(debug.Config{
			Address: config.debugAddress,
			Service: config.name,
		}, log))
	}
	servers = append(servers, config.servers...)

	serviceOptions := []service.Option{
		service.Name(config.name),
		service.WithLogger(log),
		service.ShutdownTimeout(config.shutdownTimeout),
	}
	for _, srv := range servers {
		serviceOptions = append(serviceOptions, service.WithServer(srv))
	}

	return &macroService{
		Service:    service.New(serviceOptions...),
		grpcServer: rpcServer,
	}
}

func (s *macroService) GRPC() *grpc.Server {
	if s.grpcServer == nil {
		return nil
	}
	return s.grpcServer.GRPC()
}

func Name(name string) Option {
	return func(o *options) {
		if name != "" {
			o.name = name
		}
	}
}

func WithLogger(log *logger.Logger) Option {
	return func(o *options) {
		if log != nil {
			o.logger = log
		}
	}
}

func WithServer(srv server.Server) Option {
	return func(o *options) {
		if srv != nil {
			o.servers = append(o.servers, srv)
		}
	}
}

func ShutdownTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.shutdownTimeout = timeout
		}
	}
}

// GRPCAddress overrides the default gRPC address (:8000).
func GRPCAddress(address string) Option {
	return func(o *options) {
		o.grpcAddress = address
	}
}

// DebugAddress overrides the default diagnostics address (127.0.0.1:6060).
func DebugAddress(address string) Option {
	return func(o *options) {
		o.debugAddress = address
	}
}

// WithoutGRPC disables the default gRPC server.
func WithoutGRPC() Option {
	return func(o *options) {
		o.grpcEnabled = false
	}
}

// WithoutDebug disables the default diagnostics server.
func WithoutDebug() Option {
	return func(o *options) {
		o.debugEnabled = false
	}
}
