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
