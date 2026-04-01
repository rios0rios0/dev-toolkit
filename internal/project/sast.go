package project

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed sast_defaults/*
var sastDefaults embed.FS

// SASTTool represents a single SAST analysis tool.
type SASTTool interface {
	Name() string
	Run(dir string, runner CommandRunner, output io.Writer) error
}

// DefaultSASTTools returns the standard set of SAST tools for a given language.
func DefaultSASTTools(language string) []SASTTool {
	codeqlLanguageMap := map[string]string{
		"go":     "go",
		"python": "python",
		"java":   "java",
		"node":   "javascript",
		"csharp": "csharp",
	}

	semgrepLanguageMap := map[string]string{
		"go":        "golang",
		"python":    "python",
		"java":      "java",
		"node":      "javascript",
		"csharp":    "csharp",
		"terraform": "terraform",
	}

	semgrepLang := semgrepLanguageMap[language]
	codeqlLang := codeqlLanguageMap[language]

	tools := []SASTTool{
		&SemgrepTool{Language: semgrepLang},
		&TrivyTool{},
		&HadolintTool{},
		&GitleaksTool{},
	}

	if codeqlLang != "" {
		tools = append(tools, &CodeQLTool{Language: codeqlLang})
	}

	return tools
}

// RunSAST detects the project language and runs all SAST tools.
// Individual tool failures are collected and reported at the end; all tools run regardless.
func RunSAST(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	logf(cfg.Output, "detected %s project, running SAST suite", info.SDKName)

	tools := DefaultSASTTools(info.Language)
	var toolErrors []error

	for _, tool := range tools {
		logf(cfg.Output, "running %s...", tool.Name())
		if toolErr := tool.Run(repoPath, cfg.Runner, cfg.Output); toolErr != nil {
			logf(cfg.Output, "%s failed: %s", tool.Name(), toolErr)
			toolErrors = append(toolErrors, fmt.Errorf("%s: %w", tool.Name(), toolErr))
		} else {
			logf(cfg.Output, "%s completed successfully", tool.Name())
		}
	}

	if len(toolErrors) > 0 {
		logf(cfg.Output, "SAST suite finished with %d failure(s)", len(toolErrors))
		return errors.Join(toolErrors...)
	}

	logf(cfg.Output, "SAST suite completed successfully")
	return nil
}

// ensureReportDir creates the report directory for a tool under build/reports/<tool>.
func ensureReportDir(repoPath, toolName string) (string, error) {
	dir := filepath.Join(repoPath, "build", "reports", toolName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create report directory: %w", err)
	}
	return dir, nil
}

// writeDefaultConfig writes an embedded default config file to the project directory
// if the project does not already have one. Returns true if a temp file was written
// (and should be cleaned up), false if the project already had the file.
func writeDefaultConfig(repoPath, projectFileName, embeddedFileName string) (bool, error) {
	target := filepath.Join(repoPath, projectFileName)
	if _, err := os.Stat(target); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to stat %s: %w", target, err)
	}

	data, err := sastDefaults.ReadFile("sast_defaults/" + embeddedFileName)
	if err != nil {
		return false, fmt.Errorf("failed to read embedded default %s: %w", embeddedFileName, err)
	}

	err = os.WriteFile(target, data, 0o644) // #nosec G306
	if err != nil {
		return false, fmt.Errorf("failed to write default %s: %w", projectFileName, err)
	}
	return true, nil
}

// cleanupDefaultConfig removes a temporarily written config file.
func cleanupDefaultConfig(repoPath, fileName string) {
	_ = os.Remove(filepath.Join(repoPath, fileName))
}
