package project

import "fmt"

// RunStop detects the project language and runs its stop command.
func RunStop(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	if info.StopCommand == "" {
		return fmt.Errorf("no stop command available for %s", info.Language)
	}

	logf(cfg.Output, "detected %s project", info.SDKName)
	logf(cfg.Output, "running: %s", info.StopCommand)
	return cfg.Runner.RunInteractive(repoPath, info.StopCommand)
}
