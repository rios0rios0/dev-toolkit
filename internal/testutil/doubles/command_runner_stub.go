package doubles

// CommandRunnerStub is a test double for project.CommandRunner.
type CommandRunnerStub struct {
	RunInteractiveFunc func(dir, command string) error
	Calls              []string
}

func NewCommandRunnerStub() *CommandRunnerStub {
	return &CommandRunnerStub{
		RunInteractiveFunc: func(_, _ string) error { return nil },
	}
}

func (s *CommandRunnerStub) WithError(err error) *CommandRunnerStub {
	s.RunInteractiveFunc = func(_, _ string) error { return err }
	return s
}

func (s *CommandRunnerStub) RunInteractive(dir, command string) error {
	s.Calls = append(s.Calls, command)
	return s.RunInteractiveFunc(dir, command)
}
