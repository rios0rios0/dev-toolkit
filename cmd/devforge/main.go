package main

import (
	"os"
	"path/filepath"

	"github.com/rios0rios0/devforge/internal/repo"
	log "github.com/sirupsen/logrus"
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

	rootCmd.AddCommand(repoCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
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

func mustDetectProvider(rootDir string) string {
	providerName, _, err := repo.DetectProviderAndOwner(rootDir)
	if err != nil {
		log.Fatal(err)
	}
	return providerName
}
