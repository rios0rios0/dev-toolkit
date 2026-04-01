package project

import "fmt"

// RunTest detects the project language and runs its test commands in order.
func RunTest(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	if len(info.TestCommands) == 0 {
		return fmt.Errorf("no test commands available for %s", info.Language)
	}

	logf(cfg.Output, "detected %s project", info.SDKName)
	for _, cmd := range info.TestCommands {
		logf(cfg.Output, "running: %s", cmd)
		if cmdErr := cfg.Runner.RunInteractive(repoPath, cmd); cmdErr != nil {
			return fmt.Errorf("test command %q failed: %w", cmd, cmdErr)
		}
	}
	logf(cfg.Output, "tests completed successfully")
	return nil
}
