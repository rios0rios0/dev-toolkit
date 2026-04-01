package project

import (
	"fmt"
	"io"
	"path/filepath"
)

// SemgrepTool runs Semgrep static analysis via Docker.
type SemgrepTool struct {
	Language string
}

func (t *SemgrepTool) Name() string { return "semgrep" }

func (t *SemgrepTool) Run(dir string, runner CommandRunner, output io.Writer) error {
	_, err := ensureReportDir(dir, "semgrep")
	if err != nil {
		return err
	}

	tempCreated, err := writeDefaultConfig(dir, ".semgrepignore", "semgrepignore")
	if err != nil {
		return err
	}
	if tempCreated {
		defer cleanupDefaultConfig(dir, ".semgrepignore")
	}

	containerPath := "/src"
	reportFile := filepath.Join(containerPath, "build", "reports", "semgrep", "semgrep.json")

	configs := fmt.Sprintf(
		"--config p/docker --config p/dockerfile --config p/secrets --config p/owasp-top-ten --config p/r2c-best-practices",
	)
	if t.Language != "" {
		configs = fmt.Sprintf("--config p/%s %s", t.Language, configs)
	}

	cmd := fmt.Sprintf(
		"docker run --rm -v %s:%s --workdir %s returntocorp/semgrep:1.80.0 semgrep "+
			"--metrics=off %s --enable-version-check --force-color --error --json --output %s",
		dir, containerPath, containerPath, configs, reportFile,
	)

	logf(output, "report: %s", filepath.Join(dir, "build", "reports", "semgrep", "semgrep.json"))
	return runner.RunInteractive(dir, cmd)
}
