package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func detectLanguage(cwd string) (string, error) {
	var detected string

	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}

	// Check project type by special files
	for language, config := range Globalconf.LanguagesConfig {
		for _, pattern := range config.SpecialPatterns {
			_, err := os.Stat(filepath.Join(absPath, pattern))
			if !os.IsNotExist(err) {
				return language, nil
			}
		}
	}

	// Check project type by file extensions
	err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if detected != "" {
			return filepath.SkipDir
		}

		for language, config := range Globalconf.LanguagesConfig {
			for _, ext := range config.Extensions {
				if strings.HasSuffix(info.Name(), "."+ext) {
					detected = language
					return filepath.SkipDir
				}
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return "", errors.New("project language not found")
}

func executeCommandFromConfig(repoPath, cmdType string) {
	localConfig, err := loadLocalConfig(repoPath)
	if err != nil {
		log.Fatalf("Error loading local config: %s", err)
		os.Exit(1)
	}

	err = launchDependencies(localConfig, cmdType)
	if err != nil {
		log.Fatalf("Error launching dependencies: %s", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %s", err)
		os.Exit(1)
	}
	language, err := detectLanguage(cwd)
	if err != nil {
		log.Fatalf("Error detecting language: %s", err)
		os.Exit(1)
	}
	log.Infof("Detected project language: %s", language)

	var command string
	if cmdType == "Start" {
		command = localConfig.LanguagesConfig[language].Start
	} else if cmdType == "Stop" {
		command = localConfig.LanguagesConfig[language].Stop
	} else if cmdType == "Build" {
		command = localConfig.LanguagesConfig[language].Build
	} else {
		log.Fatalf("Invalid command type: %s", cmdType)
		os.Exit(1)
	}

	if command == "" {
		log.Fatalf("No command found for %s", language)
		os.Exit(1)
	}

	RunCommand(command)
}

func RunStart(cwd string) {
	executeCommandFromConfig(cwd, "Start")
}

func RunStop(cwd string) {
	executeCommandFromConfig(cwd, "Stop")
}

func RunBuild(cwd string) {
	executeCommandFromConfig(cwd, "Build")
}

func RunCommand(cmdStr string) {
	// execute using sh so the command can use &&, ||, etc.
	log.Infof("Running command: %s", cmdStr)
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error running command: %s", err)
	}
}
