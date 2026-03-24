package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner abstracts docker command execution for testability.
type Runner interface {
	Run(args ...string) error
	Output(args ...string) (string, error)
}

// DefaultRunner executes real docker commands via [exec.CommandContext].
type DefaultRunner struct{}

func (r *DefaultRunner) Run(args ...string) error {
	cmd := exec.CommandContext(context.Background(), "docker", args...) // #nosec G204
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

func (r *DefaultRunner) Output(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "docker", args...) // #nosec G204
	cmd.Stdin = nil
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return "", fmt.Errorf("%s: %s", strings.Join(args, " "), stderr)
		}
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return "", fmt.Errorf("%s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("%s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(output)), nil
}
