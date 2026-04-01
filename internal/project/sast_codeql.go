package project

import (
	"fmt"
	"io"
	"path/filepath"
)

// CodeQLTool runs CodeQL static analysis.
type CodeQLTool struct {
	Language string
}

func (t *CodeQLTool) Name() string { return "codeql" }

func (t *CodeQLTool) Run(dir string, runner CommandRunner, output io.Writer) error {
	if t.Language == "" {
		logf(output, "no CodeQL language mapping, skipping")
		return nil
	}

	reportDir, err := ensureReportDir(dir, "codeql")
	if err != nil {
		return err
	}

	tempCreated, err := writeDefaultConfig(dir, ".codeql-false-positives", "codeql-false-positives")
	if err != nil {
		return err
	}
	if tempCreated {
		defer cleanupDefaultConfig(dir, ".codeql-false-positives")
	}

	dbPath := filepath.Join(dir, ".codeql-db")
	reportFile := filepath.Join(reportDir, "codeql.sarif")

	createCmd := fmt.Sprintf(
		"codeql database create --language=%s --source-root=%s %s",
		t.Language, dir, dbPath,
	)
	err = runner.RunInteractive(dir, createCmd)
	if err != nil {
		return fmt.Errorf("database creation failed: %w", err)
	}

	analyzeCmd := fmt.Sprintf(
		"codeql database analyze --format=sarifv2.1.0 --output=%s %s %s-security-and-quality.qls",
		reportFile, dbPath, t.Language,
	)
	err = runner.RunInteractive(dir, analyzeCmd)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	cleanupCmd := fmt.Sprintf("rm -rf %s", dbPath)
	_ = runner.RunInteractive(dir, cleanupCmd)

	logf(output, "report: %s", reportFile)
	return nil
}
