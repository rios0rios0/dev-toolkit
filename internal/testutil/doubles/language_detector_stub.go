package doubles

import "github.com/rios0rios0/devforge/internal/project"

// LanguageDetectorStub is a test double for project.LanguageDetector.
type LanguageDetectorStub struct {
	Info *project.LanguageInfo
	Err  error
}

func NewLanguageDetectorStub(info *project.LanguageInfo) *LanguageDetectorStub {
	return &LanguageDetectorStub{Info: info}
}

func (s *LanguageDetectorStub) WithError(err error) *LanguageDetectorStub {
	s.Err = err
	s.Info = nil
	return s
}

func (s *LanguageDetectorStub) Detect(_ string) (*project.LanguageInfo, error) {
	return s.Info, s.Err
}
