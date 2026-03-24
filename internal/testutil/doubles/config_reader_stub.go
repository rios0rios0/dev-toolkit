package doubles

import "github.com/rios0rios0/devforge/internal/project"

// ConfigReaderStub is a test double for project.ConfigReader.
type ConfigReaderStub struct {
	Configs map[string]*project.DevConfig
	Err     error
}

func NewConfigReaderStub() *ConfigReaderStub {
	return &ConfigReaderStub{Configs: make(map[string]*project.DevConfig)}
}

func (s *ConfigReaderStub) WithConfig(path string, cfg *project.DevConfig) *ConfigReaderStub {
	s.Configs[path] = cfg
	return s
}

func (s *ConfigReaderStub) WithError(err error) *ConfigReaderStub {
	s.Err = err
	return s
}

func (s *ConfigReaderStub) Read(repoPath string) (*project.DevConfig, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Configs[repoPath], nil
}
