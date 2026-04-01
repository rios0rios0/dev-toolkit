package project

import "fmt"

// RunLint detects the project language and runs its lint commands in order.
func RunLint(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	if len(info.LintCommands) == 0 {
		return fmt.Errorf("no lint commands available for %s", info.Language)
	}

	logf(cfg.Output, "detected %s project", info.SDKName)
	for _, cmd := range info.LintCommands {
		logf(cfg.Output, "running: %s", cmd)
		if cmdErr := cfg.Runner.RunInteractive(repoPath, cmd); cmdErr != nil {
			return fmt.Errorf("lint command %q failed: %w", cmd, cmdErr)
		}
	}
	logf(cfg.Output, "lint completed successfully")
	return nil
}
