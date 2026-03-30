package doubles

// SystemRunnerStub is a test double for system.Runner with configurable behavior.
type SystemRunnerStub struct {
	RunFunc    func(name string, args ...string) error
	OutputFunc func(name string, args ...string) (string, error)
}

func NewSystemRunnerStub() *SystemRunnerStub {
	return &SystemRunnerStub{
		RunFunc:    func(_ string, _ ...string) error { return nil },
		OutputFunc: func(_ string, _ ...string) (string, error) { return "", nil },
	}
}

func (s *SystemRunnerStub) WithOutput(name string, args []string, output string) *SystemRunnerStub {
	prev := s.OutputFunc
	s.OutputFunc = func(n string, a ...string) (string, error) {
		if n == name && matchArgs(args, a) {
			return output, nil
		}
		return prev(n, a...)
	}
	return s
}

func (s *SystemRunnerStub) WithOutputError(name string, args []string, err error) *SystemRunnerStub {
	prev := s.OutputFunc
	s.OutputFunc = func(n string, a ...string) (string, error) {
		if n == name && matchArgs(args, a) {
			return "", err
		}
		return prev(n, a...)
	}
	return s
}

func (s *SystemRunnerStub) WithRunError(name string, args []string, err error) *SystemRunnerStub {
	prev := s.RunFunc
	s.RunFunc = func(n string, a ...string) error {
		if n == name && matchArgs(args, a) {
			return err
		}
		return prev(n, a...)
	}
	return s
}

func (s *SystemRunnerStub) Run(name string, args ...string) error {
	return s.RunFunc(name, args...)
}

func (s *SystemRunnerStub) Output(name string, args ...string) (string, error) {
	return s.OutputFunc(name, args...)
}
