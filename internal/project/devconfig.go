package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DevConfig represents the contents of a .dev.yaml file.
type DevConfig struct {
	Dependencies []string `yaml:"dependencies"`
}

// ConfigReader abstracts reading and parsing of .dev.yaml files.
type ConfigReader interface {
	Read(repoPath string) (*DevConfig, error)
}

// FileConfigReader reads .dev.yaml from the filesystem.
type FileConfigReader struct{}

func (r *FileConfigReader) Read(repoPath string) (*DevConfig, error) {
	filePath := filepath.Join(repoPath, ".dev.yaml")
	data, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read .dev.yaml: %w", err)
	}

	var cfg DevConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse .dev.yaml: %w", err)
	}
	return &cfg, nil
}

// ResolveDependencyOrder resolves the full start order for a project and its transitive dependencies.
// Returns absolute paths in start order (dependencies first, target project last).
func ResolveDependencyOrder(rootPath string, reader ConfigReader) ([]string, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	var result []string
	visited := map[string]bool{}
	visiting := map[string]bool{}

	if err := dfsResolve(absRoot, reader, visited, visiting, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func dfsResolve(
	absPath string,
	reader ConfigReader,
	visited, visiting map[string]bool,
	result *[]string,
) error {
	if visited[absPath] {
		return nil
	}
	if visiting[absPath] {
		return fmt.Errorf("dependency cycle detected at %s", absPath)
	}

	visiting[absPath] = true

	cfg, err := reader.Read(absPath)
	if err != nil {
		return fmt.Errorf("failed to read config for %s: %w", absPath, err)
	}

	if cfg != nil {
		for _, dep := range cfg.Dependencies {
			depAbs, absErr := filepath.Abs(filepath.Join(absPath, dep))
			if absErr != nil {
				return fmt.Errorf("failed to resolve dependency path %s: %w", dep, absErr)
			}
			if err := dfsResolve(depAbs, reader, visited, visiting, result); err != nil {
				if strings.Contains(err.Error(), "dependency cycle detected") {
					return fmt.Errorf("%w -> %s", err, absPath)
				}
				return err
			}
		}
	}

	delete(visiting, absPath)
	visited[absPath] = true
	*result = append(*result, absPath)
	return nil
}
