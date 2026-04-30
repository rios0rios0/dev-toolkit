package doubles

import (
	"fmt"

	"github.com/rios0rios0/dev-toolkit/internal/project"
)

// LanguageDetectorMultiStub is a path-aware test double for project.LanguageDetector.
type LanguageDetectorMultiStub struct {
	Infos map[string]*project.LanguageInfo
	Err   error
}

func NewLanguageDetectorMultiStub() *LanguageDetectorMultiStub {
	return &LanguageDetectorMultiStub{Infos: make(map[string]*project.LanguageInfo)}
}

func (s *LanguageDetectorMultiStub) WithInfo(path string, info *project.LanguageInfo) *LanguageDetectorMultiStub {
	s.Infos[path] = info
	return s
}

func (s *LanguageDetectorMultiStub) WithError(err error) *LanguageDetectorMultiStub {
	s.Err = err
	return s
}

func (s *LanguageDetectorMultiStub) Detect(repoPath string) (*project.LanguageInfo, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	info, ok := s.Infos[repoPath]
	if !ok {
		return nil, fmt.Errorf("no language info configured for path: %s", repoPath)
	}
	return info, nil
}
