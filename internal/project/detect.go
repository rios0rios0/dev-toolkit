package project

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	langRepos "github.com/rios0rios0/langforge/pkg/domain/repositories"
	langRegistry "github.com/rios0rios0/langforge/pkg/infrastructure/registry"
)

// LanguageInfo holds detected language metadata for display and command execution.
type LanguageInfo struct {
	Language        string
	SDKName         string
	VersionManager  string
	CurrentVersion  string
	RequiredVersion string
	InstallCommand  string
	UseCommand      string
	StartCommand    string
	StopCommand     string
	LintCommands    []string
	BuildCommands   []string
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
	Stdout   io.Writer
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
		info.RequiredVersion = ReadRequiredSDKVersion(repoPath, info.Language)
		if info.RequiredVersion != "" {
			info.InstallCommand = full.InstallCommand(info.RequiredVersion)
			info.UseCommand = buildUseCommand(info.VersionManager, info.Language, info.RequiredVersion)
		}
	}

	return info, nil
}

// ReadRequiredSDKVersion extracts the required SDK version from language-specific project files.
func ReadRequiredSDKVersion(repoPath, language string) string {
	type extractor struct {
		file string
		fn   func(string) string
	}
	extractors := map[string]extractor{
		"go":     {"go.mod", extractGoSDKVersion},
		"node":   {".nvmrc", extractNodeSDKVersion},
		"python": {"pyproject.toml", extractPythonSDKVersion},
	}

	ext, ok := extractors[language]
	if !ok {
		return ""
	}

	filePath := filepath.Join(repoPath, ext.file)
	content, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		return ""
	}
	return ext.fn(string(content))
}

var goDirectiveRe = regexp.MustCompile(`^go\s+(\d+\.\d+(?:\.\d+)?)`)

func extractGoSDKVersion(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if m := goDirectiveRe.FindStringSubmatch(scanner.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}

func extractNodeSDKVersion(content string) string {
	return strings.TrimSpace(content)
}

var pythonRequiresRe = regexp.MustCompile(`requires-python\s*=\s*["']([><=~^!]*)(\d+\.\d+(?:\.\d+)?)`)

func extractPythonSDKVersion(content string) string {
	if m := pythonRequiresRe.FindStringSubmatch(content); m != nil {
		return m[2]
	}
	return ""
}

// buildUseCommand returns the shell command to switch to a specific SDK version.
func buildUseCommand(versionManager, language, version string) string {
	type cmdTemplate struct {
		format string
	}
	templates := map[string]cmdTemplate{
		"gvm":    {"gvm use go%s"},
		"nvm":    {"nvm use %s"},
		"pyenv":  {"pyenv local %s"},
		"sdkman": {"sdk use java %s"},
	}

	tmpl, ok := templates[versionManager]
	if !ok {
		return ""
	}
	_ = language // reserved for future use
	return fmt.Sprintf(tmpl.format, version)
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
