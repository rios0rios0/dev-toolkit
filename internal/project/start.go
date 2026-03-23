package project

import "fmt"

// RunStart detects the project language and runs its start command.
func RunStart(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	if info.StartCommand == "" {
		return fmt.Errorf("no start command available for %s", info.Language)
	}

	logf(cfg.Output, "detected %s project", info.SDKName)
	logf(cfg.Output, "running: %s", info.StartCommand)
	return cfg.Runner.RunInteractive(repoPath, info.StartCommand)
}
