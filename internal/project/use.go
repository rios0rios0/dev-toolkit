package project

import "fmt"

// RunUse detects the project language and prints shell commands to switch to the required SDK version.
// Status messages go to cfg.Output (stderr), eval-able commands go to cfg.Stdout.
func RunUse(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	if info.RequiredVersion == "" {
		logf(cfg.Output, "no version constraint found for %s", info.Language)
		return nil
	}

	if info.CurrentVersion == info.RequiredVersion {
		logf(cfg.Output, "%s %s already active", info.SDKName, info.CurrentVersion)
		return nil
	}

	if info.UseCommand == "" {
		return fmt.Errorf("no use command available for %s (version manager: %s)", info.Language, info.VersionManager)
	}

	logf(cfg.Output, "%s: switching %s -> %s", info.SDKName, valueOrNone(info.CurrentVersion), info.RequiredVersion)

	if info.CurrentVersion == "" && info.InstallCommand != "" {
		fmt.Fprintln(cfg.Stdout, info.InstallCommand)
	}
	fmt.Fprintln(cfg.Stdout, info.UseCommand)

	return nil
}
