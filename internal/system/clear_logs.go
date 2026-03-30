package system

import (
	"fmt"
	"io"
)

// RunClearLogs removes log files older than 5 days from /var/log using sudo.
func RunClearLogs(runner Runner, dryRun bool, output io.Writer) error {
	if dryRun {
		logf(output, "(dry-run mode)")
		logf(output, "would remove log files older than 5 days from /var/log")
		return nil
	}

	logf(output, "removing log files older than 5 days from /var/log...")
	bin, args := buildCommand(true, "find", "/var/log",
		"-name", "*.log", "-type", "f", "-mtime", "+5",
		"-exec", "rm", "-f", "{}", ";")
	if err := runner.Run(bin, args...); err != nil {
		return fmt.Errorf("clearing logs: %w", err)
	}

	logf(output, "old log files cleared")
	return nil
}
