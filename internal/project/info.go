package project

import "fmt"

// RunInfo detects the project language and displays all available metadata.
func RunInfo(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	logf(cfg.Output, "language:        %s", info.Language)
	logf(cfg.Output, "SDK:             %s", valueOrNone(info.SDKName))
	logf(cfg.Output, "version manager: %s", valueOrNone(info.VersionManager))
	if info.CurrentVersion != "" {
		logf(cfg.Output, "current version: %s", info.CurrentVersion)
	} else {
		logf(cfg.Output, "current version: (not installed)")
	}
	logf(cfg.Output, "start command:   %s", valueOrNone(info.StartCommand))
	logf(cfg.Output, "stop command:    %s", valueOrNone(info.StopCommand))
	logf(cfg.Output, "lint commands:   %s", commandsOrNone(info.LintCommands))
	logf(cfg.Output, "build commands:  %s", commandsOrNone(info.BuildCommands))

	if cfg.ConfigReader != nil {
		devCfg, readErr := cfg.ConfigReader.Read(repoPath)
		if readErr != nil {
			return fmt.Errorf("failed to read .dev.yaml: %w", readErr)
		}
		if devCfg != nil && len(devCfg.Dependencies) > 0 {
			logf(cfg.Output, "dependencies:")
			for _, dep := range devCfg.Dependencies {
				logf(cfg.Output, "  %s", dep)
			}
		}
	}

	return nil
}
