package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// HadolintTool runs Hadolint Dockerfile linting.
type HadolintTool struct{}

func (t *HadolintTool) Name() string { return "hadolint" }

func (t *HadolintTool) Run(dir string, runner CommandRunner, output io.Writer) error {
	dockerfiles := findDockerfiles(dir)
	if len(dockerfiles) == 0 {
		logf(output, "no Dockerfiles found, skipping hadolint")
		return nil
	}

	reportDir, err := ensureReportDir(dir, "hadolint")
	if err != nil {
		return err
	}

	tempCreated, err := writeDefaultConfig(dir, ".hadolint.yaml", "hadolint.yaml")
	if err != nil {
		return err
	}
	if tempCreated {
		defer cleanupDefaultConfig(dir, ".hadolint.yaml")
	}

	reportFile := filepath.Join(reportDir, "hadolint.sarif")
	fileArgs := strings.Join(dockerfiles, " ")

	cmd := fmt.Sprintf("hadolint --format sarif --output %s %s", reportFile, fileArgs)

	logf(output, "linting %d Dockerfile(s)", len(dockerfiles))
	logf(output, "report: %s", reportFile)
	return runner.RunInteractive(dir, cmd)
}

// findDockerfiles searches for Dockerfile* files in the given directory, excluding common vendor dirs.
func findDockerfiles(dir string) []string {
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, ".codeql-db": true,
	}

	var results []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && skipDirs[info.Name()] {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "Dockerfile") {
			results = append(results, path)
		}
		return nil
	})
	return results
}
