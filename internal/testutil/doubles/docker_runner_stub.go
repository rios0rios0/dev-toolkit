package doubles

// DockerRunnerStub is a test double for docker.Runner with configurable behavior.
type DockerRunnerStub struct {
	RunFunc    func(args ...string) error
	OutputFunc func(args ...string) (string, error)
}

func NewDockerRunnerStub() *DockerRunnerStub {
	return &DockerRunnerStub{
		RunFunc:    func(_ ...string) error { return nil },
		OutputFunc: func(_ ...string) (string, error) { return "", nil },
	}
}

func (s *DockerRunnerStub) WithOutput(args []string, output string) *DockerRunnerStub {
	prev := s.OutputFunc
	s.OutputFunc = func(a ...string) (string, error) {
		if matchArgs(args, a) {
			return output, nil
		}
		return prev(a...)
	}
	return s
}

func (s *DockerRunnerStub) WithOutputError(args []string, err error) *DockerRunnerStub {
	prev := s.OutputFunc
	s.OutputFunc = func(a ...string) (string, error) {
		if matchArgs(args, a) {
			return "", err
		}
		return prev(a...)
	}
	return s
}

func (s *DockerRunnerStub) WithRunError(args []string, err error) *DockerRunnerStub {
	prev := s.RunFunc
	s.RunFunc = func(a ...string) error {
		if matchArgs(args, a) {
			return err
		}
		return prev(a...)
	}
	return s
}

func (s *DockerRunnerStub) Run(args ...string) error {
	return s.RunFunc(args...)
}

func (s *DockerRunnerStub) Output(args ...string) (string, error) {
	return s.OutputFunc(args...)
}
