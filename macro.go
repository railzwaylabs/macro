package macro

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	"github.com/railzwaylabs/macro/debug"
	"github.com/railzwaylabs/macro/logger"
	"github.com/railzwaylabs/macro/server"
	macrogrpc "github.com/railzwaylabs/macro/server/transport/grpc"
	macrohttp "github.com/railzwaylabs/macro/server/transport/http"
	"github.com/railzwaylabs/macro/service"
)

// Service is a ready-to-run Macro service with access to its native gRPC
// server for generated protobuf registration.
type Service interface {
	service.Service
	GRPC() *grpc.Server
	HTTP() *http.Server
	Gateway() *runtime.ServeMux
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
	httpAddress     string
	debugAddress    string
	grpcEnabled     bool
	httpEnabled     bool
	debugEnabled    bool
	grpcReflection  bool
	httpHandler     http.Handler
	gatewayMux      *runtime.ServeMux
	modules         []Module
}

type macroService struct {
	service.Service
	grpcServer *macrogrpc.Server
	httpServer *macrohttp.Server
	gatewayMux *runtime.ServeMux
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
		httpEnabled:     false,
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

	servers := make([]server.Server, 0, len(config.servers)+4)
	var rpcServer *macrogrpc.Server
	if config.grpcEnabled {
		grpcOptions := []macrogrpc.Option{macrogrpc.WithLogger(log)}
		if config.grpcReflection {
			grpcOptions = append(grpcOptions, macrogrpc.WithReflection())
		}
		rpcServer = macrogrpc.New(config.grpcAddress, grpcOptions...)
	}

	var webServer *macrohttp.Server
	if config.gatewayMux != nil {
		config.httpEnabled = true
		config.httpHandler = config.gatewayMux
	}

	if config.httpEnabled {
		webServer = macrohttp.New(config.httpAddress, config.httpHandler)
	}

	if config.grpcEnabled && config.httpEnabled && rpcServer.Address() == webServer.Address() && !isEphemeralAddress(rpcServer.Address()) {
		servers = append(servers, invalidServer{err: fmt.Errorf("macro: HTTP and gRPC cannot use the same listener address %q", rpcServer.Address())})
	}

	if len(config.modules) > 0 {
		servers = append(servers, newWiringRuntime(config.modules, nativeGRPC(rpcServer), nativeHTTP(webServer), config.gatewayMux))
	}

	if rpcServer != nil {
		servers = append(servers, rpcServer)
	}

	if webServer != nil {
		servers = append(servers, webServer)
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
		httpServer: webServer,
		gatewayMux: config.gatewayMux,
	}
}

func (s *macroService) Gateway() *runtime.ServeMux { return s.gatewayMux }

func isEphemeralAddress(address string) bool {
	return address == ":0" || len(address) >= 2 && address[len(address)-2:] == ":0"
}

func (s *macroService) HTTP() *http.Server {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.HTTP()
}

func nativeGRPC(server *macrogrpc.Server) *grpc.Server {
	if server == nil {
		return nil
	}
	return server.GRPC()
}

func nativeHTTP(server *macrohttp.Server) *http.Server {
	if server == nil {
		return nil
	}
	return server.HTTP()
}

type invalidServer struct{ err error }

func (server invalidServer) Start(context.Context) error { return server.err }
func (invalidServer) Stop(context.Context) error         { return nil }

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

// GRPCAddress overrides the default gRPC address (:9000).
func GRPCAddress(address string) Option {
	return func(o *options) {
		o.grpcAddress = address
	}
}

// WithHTTP enables the standard HTTP server with an application-owned handler.
func WithHTTP(handler http.Handler) Option {
	return func(options *options) {
		options.httpEnabled = true
		options.httpHandler = handler
	}
}

// WithGateway enables HTTP/JSON routing through the official grpc-gateway mux.
func WithGateway(muxOptions ...runtime.ServeMuxOption) Option {
	return func(config *options) {
		config.gatewayMux = runtime.NewServeMux(muxOptions...)
	}
}

func HTTPAddress(address string) Option {
	return func(options *options) { options.httpAddress = address }
}

func WithGRPCReflection() Option {
	return func(options *options) { options.grpcReflection = true }
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
