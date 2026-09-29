package macro

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/railzwaylabs/macro/logger"
)

// Worker is a long-running application workload. Implementations should stop
// and return when the context is canceled.
type Worker interface {
	Run(context.Context) error
}

// RunWorker runs a worker until it exits or the process receives SIGINT or
// SIGTERM. The worker owns its processing loop and should return after
// observing context cancellation.
func RunWorker(name string, worker Worker) error {
	if worker == nil {
		return errors.New("macro: worker is required")
	}

	workerLogger, err := logger.New(logger.Config{Service: name, Mode: "info"})
	if err != nil {
		return fmt.Errorf("create worker logger: %w", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	return runWorker(ctx, name, worker, workerLogger)
}

func runWorker(
	ctx context.Context,
	name string,
	worker Worker,
	workerLogger *logger.Logger,
) error {
	logWorkerEvent(workerLogger, "worker started", name)
	err := worker.Run(ctx)
	logWorkerEvent(workerLogger, "worker stopped", name)

	return normalizeWorkerError(ctx, name, err)
}

func normalizeWorkerError(ctx context.Context, name string, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return nil
	}

	return fmt.Errorf("run %s worker: %w", name, err)
}

func logWorkerEvent(workerLogger *logger.Logger, message, name string) {
	if workerLogger == nil {
		return
	}

	workerLogger.Zap().Info(message, zap.String("worker", name))
}
