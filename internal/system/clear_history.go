package system

import (
	"fmt"
	"io"
	"path/filepath"
)

var historyFiles = []string{
	".bash_history",
	".bashrc.original",
	".lesshst",
	".python_history",
	".shell.pre-oh-my-zsh",
	".sudo_as_admin_successful",
	".zshrc.pre-oh-my-zsh",
}

var historyGlobs = []string{
	".zcompdump*",
}

// RunClearHistory removes shell history files and leftover dotfiles from the home directory.
func RunClearHistory(fs FileSystem, dryRun bool, output io.Writer) error {
	homeDir, err := fs.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	if dryRun {
		logf(output, "(dry-run mode)")
	}

	for _, name := range historyFiles {
		path := filepath.Join(homeDir, name)
		if dryRun {
			logf(output, "would remove %s", path)
		} else {
			logf(output, "removing %s", path)
			if removeErr := fs.Remove(path); removeErr != nil {
				logf(output, "warning: %v", removeErr)
			}
		}
	}

	for _, pattern := range historyGlobs {
		matches, globErr := fs.Glob(filepath.Join(homeDir, pattern))
		if globErr != nil {
			logf(output, "warning: glob %s: %v", pattern, globErr)
			continue
		}
		for _, match := range matches {
			if dryRun {
				logf(output, "would remove %s", match)
			} else {
				logf(output, "removing %s", match)
				if removeErr := fs.Remove(match); removeErr != nil {
					logf(output, "warning: %v", removeErr)
				}
			}
		}
	}

	if !dryRun {
		logf(output, "history cleared")
	}
	return nil
}
