package main

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configPath string
	mainCmd    = &cobra.Command{
		Version: "1.0.0",
		Use:     "vinit",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Starts the application",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommandFromConfig(cmd.Flag("path").Value.String(), "Start")
		},
	}
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stops the application",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommandFromConfig(cmd.Flag("path").Value.String(), "Stop")
		},
	}
	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Builds the application",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommandFromConfig(cmd.Flag("path").Value.String(), "Build")
		},
	}
)

func main() {
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
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		if configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("Error getting current working directory: %s", err)
				os.Exit(1)
			}
			configPath = filepath.Join(cwd, "configs", "versainit.yaml")
		}

		err := InitConfig(configPath)
		if err != nil {
			log.Fatalf("Error initializing config: %s", err)
			os.Exit(1)
		}
	})
}
