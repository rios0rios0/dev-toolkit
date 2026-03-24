package docker

import (
	"fmt"
	"io"
	"strings"
)

// RunIPs lists the IP addresses of all running Docker containers.
func RunIPs(runner Runner, output io.Writer) error {
	ids, err := runner.Output("ps", "-q")
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	if ids == "" {
		logf(output, "no running containers")
		return nil
	}

	args := []string{
		"inspect",
		"--format", "{{ .Name }}: {{ range .NetworkSettings.Networks }}{{ .IPAddress }}{{ end }}",
	}
	args = append(args, strings.Fields(ids)...)

	result, err := runner.Output(args...)
	if err != nil {
		return fmt.Errorf("inspecting containers: %w", err)
	}

	for line := range strings.SplitSeq(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "/")
		fmt.Fprintln(output, line)
	}

	return nil
}

func logf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "[dev] "+format+"\n", args...)
}
