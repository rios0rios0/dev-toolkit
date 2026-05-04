package project

import (
	"errors"
	"fmt"
	"slices"
)

// RunStartWithDeps resolves project dependencies from .dev.yaml and starts them in order.
func RunStartWithDeps(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	order, err := resolveDeps(repoPath, cfg.ConfigReader)
	if err != nil {
		return err
	}

	if len(order) == 1 {
		return RunStart(cfg)
	}

	for i, path := range order {
		derived := cfg
		derived.RepoPath = path
		if i < len(order)-1 {
			logf(cfg.Output, "starting dependency: %s", path)
		} else {
			logf(cfg.Output, "starting project: %s", path)
		}
		if startErr := RunStart(derived); startErr != nil {
			return fmt.Errorf("failed to start %s: %w", path, startErr)
		}
	}
	return nil
}

// RunStopWithDeps resolves project dependencies from .dev.yaml and stops them in reverse order.
func RunStopWithDeps(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	order, err := resolveDeps(repoPath, cfg.ConfigReader)
	if err != nil {
		return err
	}

	if len(order) == 1 {
		return RunStop(cfg)
	}

	var errs []error
	for i, path := range slices.Backward(order) {
		derived := cfg
		derived.RepoPath = path
		if i == len(order)-1 {
			logf(cfg.Output, "stopping project: %s", path)
		} else {
			logf(cfg.Output, "stopping dependency: %s", path)
		}
		if stopErr := RunStop(derived); stopErr != nil {
			logf(cfg.Output, "failed to stop %s: %v", path, stopErr)
			errs = append(errs, fmt.Errorf("failed to stop %s: %w", path, stopErr))
		}
	}
	return errors.Join(errs...)
}

func resolveDeps(repoPath string, reader ConfigReader) ([]string, error) {
	if reader == nil {
		return []string{repoPath}, nil
	}
	return ResolveDependencyOrder(repoPath, reader)
}
