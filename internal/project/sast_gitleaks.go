package project

import (
	"fmt"
	"io"
	"path/filepath"
)

// GitleaksTool runs Gitleaks secret detection via Docker.
type GitleaksTool struct{}

func (t *GitleaksTool) Name() string { return "gitleaks" }

func (t *GitleaksTool) Run(dir string, runner CommandRunner, output io.Writer) error {
	reportDir, err := ensureReportDir(dir, "gitleaks")
	if err != nil {
		return err
	}

	containerPath := "/opt/src"
	reportFile := filepath.Join(reportDir, "gitleaks.json")
	containerReport := filepath.Join(containerPath, "build", "reports", "gitleaks", "gitleaks.json")

	cmd := fmt.Sprintf(
		"docker run --rm -v %s:%s zricethezav/gitleaks:latest detect "+
			"--source %s --report-path %s",
		dir, containerPath, containerPath, containerReport,
	)

	logf(output, "report: %s", reportFile)
	return runner.RunInteractive(dir, cmd)
}
