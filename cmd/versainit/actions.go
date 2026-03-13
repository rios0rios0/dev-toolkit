package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func detectLanguageBySpecialFiles(absPath string) (string, bool) {
	for language, config := range Globalconf.LanguagesConfig {
		for _, pattern := range config.SpecialPatterns {
			if _, err := os.Stat(filepath.Join(absPath, pattern)); !os.IsNotExist(err) {
				return language, true
			}
		}
	}
	return "", false
}

func detectLanguageByExtensions(absPath string) (string, error) {
	var detected string

	err := filepath.Walk(absPath, func(_ string, info os.FileInfo, err error) error {
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

	return detected, nil
}

func detectLanguage(cwd string) (string, error) {
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}

	// Check project type by special files.
	if language, found := detectLanguageBySpecialFiles(absPath); found {
		return language, nil
	}

	// Check project type by file extensions.
	detected, err := detectLanguageByExtensions(absPath)
	if err != nil {
		return "", err
	}
	if detected != "" {
		return detected, nil
	}

	return "", errors.New("project language not found")
}

func resolveCommand(localConfig *GlobalConfig, language, cmdType string) string {
	switch cmdType {
	case "Start":
		return localConfig.LanguagesConfig[language].Start
	case "Stop":
		return localConfig.LanguagesConfig[language].Stop
	case "Build":
		return localConfig.LanguagesConfig[language].Build
	default:
		log.Fatalf("Invalid command type: %s", cmdType)
		return ""
	}
}

func executeCommandFromConfig(repoPath, cmdType string) {
	localConfig, err := loadLocalConfig(repoPath)
	if err != nil {
		log.Fatalf("Error loading local config: %s", err)
	}

	err = launchDependencies(localConfig, cmdType)
	if err != nil {
		log.Fatalf("Error launching dependencies: %s", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %s", err)
	}
	language, err := detectLanguage(cwd)
	if err != nil {
		log.Fatalf("Error detecting language: %s", err)
	}
	log.Infof("Detected project language: %s", language)

	command := resolveCommand(localConfig, language, cmdType)
	if command == "" {
		log.Fatalf("No command found for %s", language)
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
	cmd := exec.CommandContext(context.Background(), "/bin/sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error running command: %s", err)
	}
}
