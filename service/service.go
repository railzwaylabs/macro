package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/railzwaylabs/macro/logger"
	"go.uber.org/zap"
)

// Service represents the lifecycle of a Macro service.
type Service interface {
	Name() string
	Options() Options

	Logger() *logger.Logger

	// Start starts the service. Repeated calls are safe.
	Start(context.Context) error

	// Stop gracefully stops the service. Repeated calls are safe.
	Stop(context.Context) error

	// Run starts the service, waits for an interrupt, and gracefully stops it.
	Run() error
}

type serviceImpl struct {
	mu      sync.RWMutex
	options Options
	started bool
}

// New creates a service and applies all options in declaration order.
func New(opts ...Option) Service {
	return &serviceImpl{
		options: NewOptions(opts...),
	}
}

func (s *serviceImpl) Name() string {
	return s.options.Name
}

func (s *serviceImpl) Options() Options {
	return s.options
}

func (s *serviceImpl) Logger() *logger.Logger {
	return s.options.Logger
}

func (s *serviceImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	for i, srv := range s.options.Servers {
		if err := srv.Start(ctx); err != nil {
			// Roll back servers that were already started.
			for j := i - 1; j >= 0; j-- {
				_ = s.options.Servers[j].Stop(ctx)
			}
			return fmt.Errorf("start server: %w", err)
		}
	}

	s.started = true
	return nil
}

func (s *serviceImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	var errs []error

	for i := len(s.options.Servers) - 1; i >= 0; i-- {
		if err := s.options.Servers[i].Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	s.started = false
	return errors.Join(errs...)
}

func (s *serviceImpl) Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("start %s service: %w", s.Name(), err)
	}

	if s.Logger() != nil {
		s.Logger().Zap().Info(
			"service started",
			zap.String("service", s.Name()),
		)
	}

	<-ctx.Done()

	if s.Logger() != nil {
		s.Logger().Zap().Info(
			"stopping service",
			zap.String("service", s.Name()),
		)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		s.options.ShutdownTimeout,
	)
	defer shutdownCancel()

	if err := s.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("stop %s service: %w", s.Name(), err)
	}

	if s.Logger() != nil {
		s.Logger().Zap().Info(
			"service stopped",
			zap.String("service", s.Name()),
		)
	}

	return nil
}
