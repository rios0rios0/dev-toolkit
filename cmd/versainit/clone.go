package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	log "github.com/sirupsen/logrus"
)

// findDependencyPath searches for the dependency directory in parent domains
func findDependencyPath(dependencyURL string, cwd string) (string, error) {
	var domainAndPath []string
	if strings.HasPrefix(dependencyURL, "git@") {
		depURL := strings.Replace(dependencyURL, ":", "/", 1)
		depURL = strings.TrimPrefix(depURL, "git@")
		depURL = strings.TrimSuffix(depURL, ".git")
		domainAndPath = strings.SplitN(depURL, "/", 2)
	} else {
		u, err := url.Parse(dependencyURL)
		if err != nil {
			return "", err
		}
		domainAndPath = strings.SplitN(u.Host+u.Path, "/", 2)
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
	return "", fmt.Errorf("dependency path not found")
}

func launchDependencies(localConfig *GlobalConfig, cmdType string) error {
	for _, dependency := range localConfig.Dependencies {
		repoURL := dependency.URL
		repoParent := localConfig.CacheDir
		repoName := filepath.Base(repoURL)
		repoPath := filepath.Join(repoParent, repoName)

		// Check for custom dependency path
		if dependency.Path != "" {
			repoParent = dependency.Path
			log.Infof("Cloning %s into %s\n", repoURL, repoParent)

			cloneOptions := &git.CloneOptions{
				URL:   repoURL,
				Depth: 1,
			}

			// Clone the repository
			if _, err := git.PlainClone(repoPath, false, cloneOptions); err != nil {
				if err == git.ErrRepositoryAlreadyExists {
					log.Warnf("Repository already exists: %s\n", repoPath)
				} else {
					log.Errorf("Error cloning: %s\n", err)
					os.RemoveAll(repoPath)
					return err
				}
			}
		} else {
			// If no custom path, search for dependency in parent domains
			cwd, err := os.Getwd()
			if err != nil {
				log.Errorf("Error getting current working directory: %s\n", err)
				return err
			}
			depPath, err := findDependencyPath(repoURL, cwd)
			if err != nil {
				log.Errorf("Error finding dependency path: %s\n", err)
				return err
			}
			repoParent = depPath
		}

		// Execute commands in the cloned repository
		cwdOld, err := os.Getwd()
		if err != nil {
			log.Errorf("Error getting current working directory: %s\n", err)
			return err
		}

		os.Chdir(repoPath)
		executeCommandFromConfig(repoPath, cmdType)
		os.Chdir(cwdOld)
	}

	return nil
}
