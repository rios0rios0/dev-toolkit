package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRunner abstracts git command execution for testability.
type GitRunner interface {
	Run(dir string, args ...string) error
	Output(dir string, args ...string) string
	Clone(url, target string) error
}

// DefaultGitRunner executes real git commands via [exec.CommandContext].
type DefaultGitRunner struct{}

func (r *DefaultGitRunner) Run(dir string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), "git", args...) // #nosec G204
	cmd.Dir = dir
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

func (r *DefaultGitRunner) Output(dir string, args ...string) string {
	cmd := exec.CommandContext(context.Background(), "git", args...) // #nosec G204
	cmd.Dir = dir
	cmd.Stdin = nil
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (r *DefaultGitRunner) Clone(url, target string) error {
	if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o750); mkdirErr != nil {
		return mkdirErr
	}

	cmd := exec.CommandContext(
		context.Background(), "git", "clone", url, target,
	) // #nosec G204
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(),
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new -o BatchMode=yes",
	)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(cmdOutput)))
	}
	return nil
}
