package main

import (
	"os"
	"path/filepath"

	"github.com/rios0rios0/devforge/internal/docker"
	"github.com/rios0rios0/devforge/internal/project"
	"github.com/rios0rios0/devforge/internal/repo"
	"github.com/rios0rios0/devforge/internal/system"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "dev",
		Version: version,
		Short:   "Developer workspace toolkit",
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	repoCmd := &cobra.Command{
		Use:   "repo",
		Short: "Repository management commands",
	}
	repoCmd.AddCommand(newCloneCmd())
	repoCmd.AddCommand(newSyncCmd())
	repoCmd.AddCommand(newPruneCmd())
	repoCmd.AddCommand(newMirrorCmd())
	repoCmd.AddCommand(newFailoverCmd())
	repoCmd.AddCommand(newRestoreCmd())

	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Project language detection and management commands",
	}
	projectCmd.AddCommand(newProjectStartCmd())
	projectCmd.AddCommand(newProjectBuildCmd())
	projectCmd.AddCommand(newProjectStopCmd())
	projectCmd.AddCommand(newProjectInfoCmd())
	projectCmd.AddCommand(newProjectUseCmd())

	dockerCmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker environment management commands",
	}
	dockerCmd.AddCommand(newDockerIPsCmd())
	dockerCmd.AddCommand(newDockerResetCmd())

	systemCmd := &cobra.Command{
		Use:   "system",
		Short: "System utility commands",
	}
	systemCmd.AddCommand(newSystemTop5SizeCmd())
	systemCmd.AddCommand(newSystemClearHistoryCmd())
	if !system.IsAndroid() {
		systemCmd.AddCommand(newSystemClearLogsCmd())
	}

	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(dockerCmd)
	rootCmd.AddCommand(systemCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

func newCloneCmd() *cobra.Command {
	var dryRun bool
	var includeArchived bool

	cmd := &cobra.Command{
		Use:   "clone <ssh-alias> [root-dir]",
		Short: "Clone missing repositories from a Git provider",
		Long: `Discovers repositories from the Git provider, compares with local directories,
clones missing repos via SSH, and optionally removes extra local repos.`,
		Args: cobra.RangeArgs(1, repo.MaxCloneArgs()),
		RunE: func(_ *cobra.Command, args []string) error {
			sshAlias := args[0]
			rootDir, _ := os.Getwd()
			if len(args) > 1 {
				rootDir = args[1]
			}
			rootDir = filepath.Clean(rootDir)

			provider, resolveErr := repo.ResolveProvider(mustDetectProvider(rootDir))
			if resolveErr != nil {
				return resolveErr
			}

			return repo.RunClone(repo.CloneConfig{
				RootDir:         rootDir,
				SSHAlias:        sshAlias,
				DryRun:          dryRun,
				IncludeArchived: includeArchived,
				Provider:        provider,
				Runner:          &repo.DefaultGitRunner{},
				Output:          os.Stderr,
				Input:           os.Stdin,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "include archived repositories")

	return cmd
}

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [root-dir]",
		Short: "Sync all repositories under a directory",
		Long: `For each repository found under the root directory, fetches all remotes,
rebases the default branch, and preserves any uncommitted work via WIP commits.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			rootDir, _ := os.Getwd()
			if len(args) > 0 {
				rootDir = args[0]
			}
			rootDir = filepath.Clean(rootDir)
			return repo.RunSync(rootDir, &repo.DefaultGitRunner{}, os.Stderr)
		},
	}
}

func newPruneCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "prune [root-dir]",
		Short: "Delete local branches that have been merged into the default branch",
		Long: `For each repository found one level deep under the root directory,
lists local branches merged into the default branch and deletes them.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			rootDir, _ := os.Getwd()
			if len(args) > 0 {
				rootDir = args[0]
			}
			rootDir = filepath.Clean(rootDir)
			return repo.RunPrune(rootDir, &repo.DefaultGitRunner{}, dryRun, os.Stderr)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show which branches would be deleted without deleting them")

	return cmd
}

func newProjectConfig(args []string) project.Config {
	repoPath := ""
	if len(args) > 0 {
		repoPath = args[0]
	}
	return project.Config{
		RepoPath:     repoPath,
		Detector:     project.NewDefaultLanguageDetector(),
		ConfigReader: &project.FileConfigReader{},
		Runner: &project.DefaultCommandRunner{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
		Output: os.Stderr,
	}
}

func newProjectStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [path]",
		Short: "Detect language and run the project start command",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return project.RunStartWithDeps(newProjectConfig(args))
		},
	}
}

func newProjectBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build [path]",
		Short: "Detect language and run the project build commands",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return project.RunBuild(newProjectConfig(args))
		},
	}
}

func newProjectStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [path]",
		Short: "Detect language and run the project stop command",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return project.RunStopWithDeps(newProjectConfig(args))
		},
	}
}

func newProjectInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [path]",
		Short: "Detect language and show project information",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return project.RunInfo(newProjectConfig(args))
		},
	}
}

func newProjectUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [path]",
		Short: "Detect language and print shell commands to switch to the required SDK version",
		Long: `Reads the required SDK version from project files (go.mod, .nvmrc, pyproject.toml),
compares with the currently installed version, and prints shell commands to stdout.

Use with eval to switch versions in the current shell:
  eval "$(dev project use .)"

Or define a shell function:
  dev-use() { eval "$(dev project use "${1:-.}")"; }`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg := newProjectConfig(args)
			cfg.Stdout = os.Stdout
			cfg.Output = os.Stderr
			return project.RunUse(cfg)
		},
	}
}

func newDockerIPsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ips",
		Short: "List IP addresses of all running Docker containers",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return docker.RunIPs(&docker.DefaultRunner{}, os.Stdout)
		},
	}
}

func newDockerResetCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Stop all containers and prune all Docker resources",
		Long: `Stops all running containers, then prunes stopped containers,
volumes, networks, and the build cache.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return docker.RunReset(&docker.DefaultRunner{}, dryRun, os.Stderr)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")

	return cmd
}

func newMirrorCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "mirror <ssh-alias> [source-dir]",
		Short: "Create Codeberg pull mirrors for all repositories",
		Long: `For each repository under the source directory, creates a pull mirror
on Codeberg via the Forgejo migration API and adds a 'codeberg' remote locally.

Requires CODEBERG_TOKEN environment variable.`,
		Args: cobra.RangeArgs(1, repo.MaxMirrorArgs()),
		RunE: func(_ *cobra.Command, args []string) error {
			sshAlias := args[0]
			sourceDir, _ := os.Getwd()
			if len(args) > 1 {
				sourceDir = args[1]
			}
			sourceDir = filepath.Clean(sourceDir)

			targetProvider, targetErr := repo.ResolveProvider("codeberg")
			if targetErr != nil {
				return targetErr
			}

			// source provider is optional (used for metadata)
			sourceProviderName, _, _ := repo.DetectProviderAndOwner(sourceDir)
			sourceProvider, _ := repo.ResolveProvider(sourceProviderName)

			return repo.RunMirror(repo.MirrorConfig{
				SourceDir:      sourceDir,
				SSHAlias:       sshAlias,
				DryRun:         dryRun,
				SourceProvider: sourceProvider,
				TargetProvider: targetProvider,
				Runner:         &repo.DefaultGitRunner{},
				Output:         repo.NewLogger(os.Stderr),
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")

	return cmd
}

func newFailoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "failover [root-dir]",
		Short: "Switch all repos from GitHub to Codeberg as primary remote",
		Long: `For each repository that has a 'codeberg' remote, renames 'origin' to 'github'
and 'codeberg' to 'origin'. This makes all tooling use Codeberg during GitHub outages.

Use 'dev repo restore' to switch back after GitHub recovers.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			rootDir, _ := os.Getwd()
			if len(args) > 0 {
				rootDir = args[0]
			}
			rootDir = filepath.Clean(rootDir)
			return repo.RunFailover(repo.FailoverConfig{
				RootDir: rootDir,
				Runner:  &repo.DefaultGitRunner{},
				Output:  repo.NewLogger(os.Stderr),
			})
		},
	}
}

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [root-dir]",
		Short: "Restore GitHub as primary remote after a failover",
		Long: `For each repository in failover state (has 'github' remote), pushes any
Codeberg-only commits to GitHub, then renames remotes back to normal
('origin' -> 'codeberg', 'github' -> 'origin').`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			rootDir, _ := os.Getwd()
			if len(args) > 0 {
				rootDir = args[0]
			}
			rootDir = filepath.Clean(rootDir)
			return repo.RunRestore(repo.RestoreConfig{
				RootDir: rootDir,
				Runner:  &repo.DefaultGitRunner{},
				Output:  repo.NewLogger(os.Stderr),
			})
		},
	}
}

func newSystemTop5SizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "top5size [dir]",
		Short: "Show the top 5 largest items in a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir, _ := os.Getwd()
			if len(args) > 0 {
				dir = args[0]
			}
			dir = filepath.Clean(dir)
			return system.RunTop5Size(
				&system.DefaultRunner{},
				&system.DefaultFileSystem{},
				dir,
				!system.IsAndroid(),
				os.Stdout,
			)
		},
	}
}

func newSystemClearHistoryCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "clear-history",
		Short: "Clear shell history and leftover dotfiles",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return system.RunClearHistory(&system.DefaultFileSystem{}, dryRun, os.Stderr)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be removed without removing")

	return cmd
}

func newSystemClearLogsCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "clear-logs",
		Short: "Remove log files older than 5 days from /var/log (requires sudo)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return system.RunClearLogs(&system.DefaultRunner{}, dryRun, os.Stderr)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be removed without removing")

	return cmd
}

func mustDetectProvider(rootDir string) string {
	providerName, _, err := repo.DetectProviderAndOwner(rootDir)
	if err != nil {
		logger.Fatal(err)
	}
	return providerName
}
