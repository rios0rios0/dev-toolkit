package project

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
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
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	cmd := exec.CommandContext(context.Background(), parts[0], parts[1:]...) // #nosec G204
	cmd.Dir = dir
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}
