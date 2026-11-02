package doubles

import "fmt"

// GitRunnerStub is a test double for repo.GitRunner with configurable behavior.
type GitRunnerStub struct {
	RunFunc    func(dir string, args ...string) error
	OutputFunc func(dir string, args ...string) string
	CloneFunc  func(url, target string) error
}

func NewGitRunnerStub() *GitRunnerStub {
	return &GitRunnerStub{
		RunFunc:    func(_ string, _ ...string) error { return nil },
		OutputFunc: func(_ string, _ ...string) string { return "" },
		CloneFunc:  func(_, _ string) error { return nil },
	}
}

func (s *GitRunnerStub) WithRunError(args []string, err error) *GitRunnerStub {
	prev := s.RunFunc
	s.RunFunc = func(dir string, a ...string) error {
		if matchArgs(args, a) {
			return err
		}
		return prev(dir, a...)
	}
	return s
}

func (s *GitRunnerStub) WithOutput(args []string, output string) *GitRunnerStub {
	prev := s.OutputFunc
	s.OutputFunc = func(dir string, a ...string) string {
		if matchArgs(args, a) {
			return output
		}
		return prev(dir, a...)
	}
	return s
}

// WithOutputForDir configures output for a command only when it runs in a specific
// directory, so callers that fan the same command across sibling checkouts (e.g.
// worktrees) can be given per-directory answers.
func (s *GitRunnerStub) WithOutputForDir(dir string, args []string, output string) *GitRunnerStub {
	prev := s.OutputFunc
	s.OutputFunc = func(d string, a ...string) string {
		if d == dir && matchArgs(args, a) {
			return output
		}
		return prev(d, a...)
	}
	return s
}

// WithRef makes a fully qualified ref resolve, as `git rev-parse --verify <ref>` does for a
// branch that exists.
func (s *GitRunnerStub) WithRef(ref string) *GitRunnerStub {
	return s.WithOutput([]string{"rev-parse", "--verify", ref}, "abc123")
}

// WithRemoteHEAD records branch as the default branch of remote the way git clone does:
// refs/remotes/<remote>/HEAD points at refs/remotes/<remote>/<branch>, which exists.
func (s *GitRunnerStub) WithRemoteHEAD(remote, branch string) *GitRunnerStub {
	ref := "refs/remotes/" + remote + "/" + branch
	return s.
		WithOutput([]string{"symbolic-ref", "refs/remotes/" + remote + "/HEAD"}, ref).
		WithRef(ref)
}

func (s *GitRunnerStub) Run(dir string, args ...string) error {
	return s.RunFunc(dir, args...)
}

func (s *GitRunnerStub) Output(dir string, args ...string) string {
	return s.OutputFunc(dir, args...)
}

func (s *GitRunnerStub) Clone(url, target string) error {
	return s.CloneFunc(url, target)
}

func (s *GitRunnerStub) WithCloneError(err error) *GitRunnerStub {
	s.CloneFunc = func(_, _ string) error { return err }
	return s
}

func matchArgs(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}
	return true
}

// RunErrorf creates a formatted error matching git command output style.
func RunErrorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
