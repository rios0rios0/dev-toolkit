package main

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
// During development, it defaults to "dev".
var version = "dev"

func main() {
	var configPath string

	mainCmd := &cobra.Command{
		Version: version,
		Use:     "vinit",
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Starts the application",
		Run: func(cmd *cobra.Command, _ []string) {
			executeCommandFromConfig(cmd.Flag("path").Value.String(), "Start")
		},
	}
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds the application",
		Run: func(cmd *cobra.Command, _ []string) {
			executeCommandFromConfig(cmd.Flag("path").Value.String(), "Build")
		},
	}

	cobra.OnInitialize(func() {
		if configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("Error getting current working directory: %s", err)
			}
			configPath = filepath.Join(cwd, "configs", "versainit.yaml")
		}

		err := InitConfig(configPath)
		if err != nil {
			log.Fatalf("Error initializing config: %s", err)
		}
	})

	mainCmd.PersistentFlags().StringVarP(
		&configPath, "config", "c", "",
		"path to the configuration file",
	)
	mainCmd.AddCommand(startCmd)
	mainCmd.AddCommand(buildCmd)

	startCmd.Flags().StringP("path", "p", "", "path to the project directory")
	buildCmd.Flags().StringP("path", "p", "", "path to the project directory")

	err := mainCmd.Execute()
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
}
