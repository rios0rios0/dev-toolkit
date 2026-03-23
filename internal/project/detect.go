package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	langRepos "github.com/rios0rios0/langforge/pkg/domain/repositories"
	langRegistry "github.com/rios0rios0/langforge/pkg/infrastructure/registry"
)

// LanguageInfo holds detected language metadata for display and command execution.
type LanguageInfo struct {
	Language       string
	SDKName        string
	VersionManager string
	CurrentVersion string
	StartCommand   string
	StopCommand    string
	LintCommands   []string
	BuildCommands  []string
}

// LanguageDetector abstracts language detection and metadata retrieval.
type LanguageDetector interface {
	Detect(repoPath string) (*LanguageInfo, error)
}

// Config holds all dependencies for a project operation.
type Config struct {
	RepoPath string
	Detector LanguageDetector
	Runner   CommandRunner
	Output   io.Writer
}

// DefaultLanguageDetector uses langforge's registry for detection.
type DefaultLanguageDetector struct {
	registry *langRegistry.LanguageRegistry
}

// NewDefaultLanguageDetector creates a detector with all built-in language providers.
func NewDefaultLanguageDetector() *DefaultLanguageDetector {
	return &DefaultLanguageDetector{
		registry: langRegistry.NewDefaultRegistry(),
	}
}

func (d *DefaultLanguageDetector) Detect(repoPath string) (*LanguageInfo, error) {
	provider, err := d.registry.Detect(repoPath)
	if err != nil {
		return nil, fmt.Errorf("could not detect language: %w", err)
	}

	info := &LanguageInfo{
		Language: string(provider.Language()),
	}

	if full, ok := provider.(langRepos.LanguageProviderFull); ok {
		info.SDKName = full.SDKName()
		info.VersionManager = full.VersionManager()
		info.CurrentVersion, _ = full.CurrentVersion()
		info.StartCommand = full.StartCommand()
		info.StopCommand = full.StopCommand()
		info.LintCommands = full.LintCommands()
		info.BuildCommands = full.BuildCommands()
	}

	return info, nil
}

func resolveRepoPath(path string) (string, error) {
	if path == "" {
		return os.Getwd()
	}
	return filepath.Clean(path), nil
}

func logf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "[dev] "+format+"\n", args...)
}

func valueOrNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func commandsOrNone(cmds []string) string {
	if len(cmds) == 0 {
		return "(none)"
	}
	return strings.Join(cmds, "; ")
}
