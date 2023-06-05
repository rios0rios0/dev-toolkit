package main

import (
	"fmt"
	"os"
	"path/filepath"

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
			RunStart(cmd.Flag("path").Value.String())
		},
	}
	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Builds the application",
		Run: func(cmd *cobra.Command, args []string) {
			RunBuild(cmd.Flag("path").Value.String())
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		if configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			configPath = filepath.Join(cwd, "configs", "versainit.yaml")
		}

		err := InitConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	})
}
