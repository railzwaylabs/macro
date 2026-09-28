package macro

import (
	"time"

	"github.com/railzwaylabs/macro/logger"
	"github.com/railzwaylabs/macro/server"
	"github.com/railzwaylabs/macro/service"
)

type Service = service.Service

type Option = service.Option

type Options = service.Options

func New(opts ...Option) Service {
	return service.New(opts...)
}

func Name(name string) Option {
	return service.Name(name)
}

func WithLogger(log *logger.Logger) Option {
	return service.WithLogger(log)
}

func WithServer(srv server.Server) Option {
	return service.WithServer(srv)
}

func ShutdownTimeout(timeout time.Duration) Option {
	return service.ShutdownTimeout(timeout)
}
