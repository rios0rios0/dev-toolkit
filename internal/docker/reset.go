package docker

import (
	"fmt"
	"io"
	"strings"
)

// RunReset stops all running containers and prunes all Docker resources.
func RunReset(runner Runner, dryRun bool, output io.Writer) error {
	type pruneStep struct {
		label string
		args  []string
	}
	steps := []pruneStep{
		{"containers", []string{"container", "prune", "--force"}},
		{"volumes", []string{"volume", "prune", "--force"}},
		{"networks", []string{"network", "prune", "--force"}},
		{"build cache", []string{"builder", "prune", "-f"}},
	}
	ids, err := runner.Output("container", "ls", "-aq")
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	if dryRun {
		logf(output, "(dry-run mode)")
		if ids == "" {
			logf(output, "no containers to stop")
		} else {
			logf(output, "would stop all containers")
		}
		for _, step := range steps {
			logf(output, "would prune %s", step.label)
		}
		return nil
	}

	if ids != "" {
		logf(output, "stopping all containers...")
		stopArgs := append([]string{"container", "stop", "-t", "5"}, strings.Fields(ids)...)
		if stopErr := runner.Run(stopArgs...); stopErr != nil {
			logf(output, "warning: some containers could not be stopped: %v", stopErr)
		}
	} else {
		logf(output, "no containers to stop")
	}

	for _, step := range steps {
		logf(output, "pruning %s...", step.label)
		if pruneErr := runner.Run(step.args...); pruneErr != nil {
			logf(output, "warning: prune %s failed: %v", step.label, pruneErr)
		}
	}

	logf(output, "docker environment reset complete")
	return nil
}
