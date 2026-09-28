package service

import (
	"time"

	"github.com/railzwaylabs/macro/logger"
	"github.com/railzwaylabs/macro/server"
)

const (
	defaultName            = "macro"
	defaultShutdownTimeout = 10 * time.Second
)

// Options contains the configuration used to construct a Service.
type Options struct {
	Name            string
	Logger          *logger.Logger
	Servers         []server.Server
	ShutdownTimeout time.Duration
}

// Option configures a Service during construction.
type Option func(*Options)

// NewOptions returns service options after applying the supplied configuration.
func NewOptions(opts ...Option) Options {
	options := Options{
		Name:            defaultName,
		ShutdownTimeout: defaultShutdownTimeout,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return options
}

// Name sets the service name.
func Name(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

func WithServer(srv server.Server) Option {
	return func(o *Options) {
		if srv != nil {
			o.Servers = append(o.Servers, srv)
		}
	}
}

func WithLogger(log *logger.Logger) Option {
	return func(o *Options) {
		if log != nil {
			o.Logger = log
		}
	}
}

// ShutdownTimeout sets the maximum duration allowed for graceful shutdown.
func ShutdownTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		if timeout > 0 {
			o.ShutdownTimeout = timeout
		}
	}
}
