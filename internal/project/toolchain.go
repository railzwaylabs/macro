package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrBufNotFound = errors.New("buf executable not found")

type CommandRunner interface {
	LookPath(string) (string, error)
	Run(context.Context, string, string, ...string) error
}

type ExecRunner struct{}

func (ExecRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (ExecRunner) Run(ctx context.Context, directory, name string, arguments ...string) error {
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("run %s: %w: %s", name, err, message)
		}
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}

func GenerateProto(ctx context.Context, directory string, runner CommandRunner) error {
	if _, err := runner.LookPath("buf"); err != nil {
		return fmt.Errorf("%w: %v", ErrBufNotFound, err)
	}
	return runner.Run(ctx, directory, "buf", "generate")
}

func Tidy(ctx context.Context, directory string, runner CommandRunner) error {
	return runner.Run(ctx, directory, "go", "mod", "tidy")
}
