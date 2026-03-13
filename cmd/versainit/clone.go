package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	log "github.com/sirupsen/logrus"
)

const domainAndPathParts = 2

// findDependencyPath searches for the dependency directory in parent domains.
func findDependencyPath(dependencyURL string, cwd string) (string, error) {
	var domainAndPath []string
	if strings.HasPrefix(dependencyURL, "git@") {
		depURL := strings.Replace(dependencyURL, ":", "/", 1)
		depURL = strings.TrimPrefix(depURL, "git@")
		depURL = strings.TrimSuffix(depURL, ".git")
		domainAndPath = strings.SplitN(depURL, "/", domainAndPathParts)
	} else {
		u, err := url.Parse(dependencyURL)
		if err != nil {
			return "", err
		}
		domainAndPath = strings.SplitN(u.Host+u.Path, "/", domainAndPathParts)
	}

	for {
		depPath := filepath.Join(cwd, domainAndPath[0], domainAndPath[1])
		// Check if dependency directory exists
		if _, err := os.Stat(depPath); !os.IsNotExist(err) {
			log.Infof("Found existing dependency at path: %s\n", depPath)
			return depPath, nil
		}
		// Stop searching if we are at the root directory
		if cwd == filepath.Dir(cwd) {
			break
		}
		// Move upward to search in parent directory
		cwd = filepath.Dir(cwd)
	}
	log.Warnf("No existing dependency path found for %s\n", dependencyURL)
	return "", errors.New("dependency path not found")
}

func cloneDependency(repoURL, repoPath string) error {
	cloneOptions := &git.CloneOptions{
		URL:   repoURL,
		Depth: 1,
	}

	if _, err := git.PlainClone(repoPath, false, cloneOptions); err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			log.Warnf("Repository already exists: %s\n", repoPath)
		} else {
			log.Errorf("Error cloning: %s\n", err)
			if removeErr := os.RemoveAll(repoPath); removeErr != nil {
				log.Errorf("Error removing failed clone: %s\n", removeErr)
			}
			return err
		}
	}
	return nil
}

func findDependencyDir(repoURL string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Error getting current working directory: %s\n", err)
		return "", err
	}
	depPath, err := findDependencyPath(repoURL, cwd)
	if err != nil {
		log.Errorf("Error finding dependency path: %s\n", err)
		return "", err
	}
	return depPath, nil
}

func launchDependencies(localConfig *GlobalConfig, cmdType string) error {
	for _, dependency := range localConfig.Dependencies {
		repoURL := dependency.URL
		repoName := filepath.Base(repoURL)
		repoPath := filepath.Join(localConfig.CacheDir, repoName)

		if dependency.Path != "" {
			log.Infof("Cloning %s into %s\n", repoURL, dependency.Path)
			if err := cloneDependency(repoURL, repoPath); err != nil {
				return err
			}
		} else {
			if _, err := findDependencyDir(repoURL); err != nil {
				return err
			}
		}

		// Execute commands in the cloned repository
		cwdOld, err := os.Getwd()
		if err != nil {
			log.Errorf("Error getting current working directory: %s\n", err)
			return err
		}

		if err = os.Chdir(repoPath); err != nil {
			log.Errorf("Error changing directory to %s: %s\n", repoPath, err)
			return err
		}
		executeCommandFromConfig(repoPath, cmdType)
		if err = os.Chdir(cwdOld); err != nil {
			log.Errorf("Error changing directory back to %s: %s\n", cwdOld, err)
			return err
		}
	}

	return nil
}
