package project

import (
	"fmt"
	"io"
	"path/filepath"
)

// TrivyTool runs Trivy IaC misconfiguration scanning.
type TrivyTool struct{}

func (t *TrivyTool) Name() string { return "trivy" }

func (t *TrivyTool) Run(dir string, runner CommandRunner, output io.Writer) error {
	reportDir, err := ensureReportDir(dir, "trivy")
	if err != nil {
		return err
	}

	tempCreated, err := writeDefaultConfig(dir, ".trivyignore", "trivyignore")
	if err != nil {
		return err
	}
	if tempCreated {
		defer cleanupDefaultConfig(dir, ".trivyignore")
	}

	reportFile := filepath.Join(reportDir, "trivy.sarif")

	cmd := fmt.Sprintf(
		"trivy filesystem --scanners misconfig --format sarif --output %s --exit-code 1 %s",
		reportFile, dir,
	)

	logf(output, "report: %s", reportFile)
	return runner.RunInteractive(dir, cmd)
}
