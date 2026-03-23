package project

import "fmt"

// RunBuild detects the project language and runs its build commands in order.
func RunBuild(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	if len(info.BuildCommands) == 0 {
		return fmt.Errorf("no build commands available for %s", info.Language)
	}

	logf(cfg.Output, "detected %s project", info.SDKName)
	for _, cmd := range info.BuildCommands {
		logf(cfg.Output, "running: %s", cmd)
		if cmdErr := cfg.Runner.RunInteractive(repoPath, cmd); cmdErr != nil {
			return fmt.Errorf("build command %q failed: %w", cmd, cmdErr)
		}
	}
	logf(cfg.Output, "build completed successfully")
	return nil
}
