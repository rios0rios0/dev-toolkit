package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func detectLanguage(cwd string) (string, error) {
	var detected string

	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}

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

		// Check for special config files and file extensions
		for language, config := range Globalconf.ProjectsConfig {
			for _, pattern := range config.SpecialPatterns {
				if info.Name() == pattern {
					detected = language
					return filepath.SkipDir
				}
			}
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

	if detected == "" {
		return "", errors.New("project language not found")
	}

	return detected, nil
}

func RunStart(cwd string) {
	language, err := detectLanguage(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	startCmd := Globalconf.ProjectsConfig[language].Start
	RunCommand(startCmd)
}

func RunBuild(cwd string) {
	language, err := detectLanguage(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	buildCmd := Globalconf.ProjectsConfig[language].Build
	RunCommand(buildCmd)
}

func RunCommand(cmdStr string) {
	cmdParts := strings.Fields(cmdStr)
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
