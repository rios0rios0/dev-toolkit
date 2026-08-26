package doubles

import (
	"fmt"
	"strings"
)

// CLIRunnerStub is a configurable test double for repo.CLIRunner. A binary is only
// visible to LookPath once it has been registered, so the "CLI is not installed" path
// is the default.
type CLIRunnerStub struct {
	Installed map[string]bool
	Tokens    map[string]string
	Errors    map[string]error
	Calls     []string
}

// NewCLIRunnerStub creates a stub where no CLI is installed.
func NewCLIRunnerStub() *CLIRunnerStub {
	return &CLIRunnerStub{
		Installed: make(map[string]bool),
		Tokens:    make(map[string]string),
		Errors:    make(map[string]error),
	}
}

// WithToken registers an installed, authenticated CLI that prints the given token.
func (s *CLIRunnerStub) WithToken(binary, token string) *CLIRunnerStub {
	s.Installed[binary] = true
	s.Tokens[binary] = token
	return s
}

// WithError registers an installed CLI whose token command fails.
func (s *CLIRunnerStub) WithError(binary string, err error) *CLIRunnerStub {
	s.Installed[binary] = true
	s.Errors[binary] = err
	return s
}

// WithInstalled registers an installed CLI that prints nothing.
func (s *CLIRunnerStub) WithInstalled(binary string) *CLIRunnerStub {
	s.Installed[binary] = true
	return s
}

func (s *CLIRunnerStub) LookPath(binary string) (string, error) {
	if !s.Installed[binary] {
		return "", fmt.Errorf("exec: %q: executable file not found in $PATH", binary)
	}
	return "/usr/bin/" + binary, nil
}

func (s *CLIRunnerStub) Output(binary string, args ...string) (string, error) {
	s.Calls = append(s.Calls, strings.TrimSpace(binary+" "+strings.Join(args, " ")))
	if err, ok := s.Errors[binary]; ok {
		return "", err
	}
	return s.Tokens[binary], nil
}
