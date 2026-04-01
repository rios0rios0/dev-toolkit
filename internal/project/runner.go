package project

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

// CommandRunner executes shell commands with passthrough I/O for interactive usage.
type CommandRunner interface {
	RunInteractive(dir, command string) error
}

// DefaultCommandRunner executes real shell commands with passthrough I/O.
type DefaultCommandRunner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (r *DefaultCommandRunner) RunInteractive(dir, command string) error {
	if command == "" {
		return errors.New("empty command")
	}
	cmd := exec.CommandContext(context.Background(), "sh", "-c", command) // #nosec G204
	cmd.Dir = dir
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}
