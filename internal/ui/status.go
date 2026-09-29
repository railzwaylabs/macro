package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type Step struct {
	Start   string
	Success string
	Failure string
	Run     func() error
}

type Status struct {
	output      io.Writer
	interactive bool
}

func NewStatus(output io.Writer) Status {
	return Status{
		output:      output,
		interactive: isTerminal(output),
	}
}

func (status Status) Heading(message string) error {
	_, err := fmt.Fprintf(status.output, "%s\n\n", message)
	return err
}

func (status Status) Run(step Step) error {
	if step.Run == nil {
		return errors.New("status step operation is required")
	}

	if status.interactive {
		if _, err := fmt.Fprintf(status.output, "⠋ %s...\r", step.Start); err != nil {
			return fmt.Errorf("render step start: %w", err)
		}
	}

	operationErr := step.Run()
	if operationErr != nil {
		renderErr := status.finish("✗", step.Failure)
		return errors.Join(operationErr, renderErr)
	}

	return status.finish("✓", step.Success)
}

func (status Status) Success(message string) error {
	_, err := fmt.Fprintf(status.output, "✓ %s\n", message)
	return err
}

func (status Status) Failure(message string) error {
	_, err := fmt.Fprintf(status.output, "✗ %s\n", message)
	return err
}

func (status Status) Warning(message string) error {
	_, err := fmt.Fprintf(status.output, "! %s\n", message)
	return err
}

func (status Status) finish(symbol, message string) error {
	if status.interactive {
		if _, err := fmt.Fprint(status.output, "\r\x1b[2K"); err != nil {
			return fmt.Errorf("clear step status: %w", err)
		}
	}

	if _, err := fmt.Fprintf(status.output, "%s %s\n", symbol, message); err != nil {
		return fmt.Errorf("render step result: %w", err)
	}

	return nil
}

func isTerminal(output io.Writer) bool {
	file, ok := output.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
