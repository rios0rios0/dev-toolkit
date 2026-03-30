package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner abstracts command execution for testability.
type Runner interface {
	Output(name string, args ...string) (string, error)
	Run(name string, args ...string) error
}

// FileSystem abstracts filesystem operations for testability.
type FileSystem interface {
	Remove(path string) error
	Glob(pattern string) ([]string, error)
	UserHomeDir() (string, error)
	ReadDir(dir string) ([]os.DirEntry, error)
}

// DefaultRunner executes real shell commands via [exec.CommandContext].
type DefaultRunner struct{}

func (r *DefaultRunner) Output(name string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), name, args...) // #nosec G204
	cmd.Stdin = nil
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *DefaultRunner) Run(name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...) // #nosec G204
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

// DefaultFileSystem uses the real OS filesystem.
type DefaultFileSystem struct{}

func (f *DefaultFileSystem) Remove(path string) error                    { return os.RemoveAll(path) }
func (f *DefaultFileSystem) Glob(pattern string) ([]string, error)       { return filepath.Glob(pattern) }
func (f *DefaultFileSystem) UserHomeDir() (string, error)                { return os.UserHomeDir() }
func (f *DefaultFileSystem) ReadDir(dir string) ([]os.DirEntry, error)   { return os.ReadDir(dir) }

func logf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "[dev] "+format+"\n", args...)
}

func buildCommand(useSudo bool, name string, args ...string) (string, []string) {
	if useSudo {
		return "sudo", append([]string{name}, args...)
	}
	return name, args
}
