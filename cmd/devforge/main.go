package main

import (
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
