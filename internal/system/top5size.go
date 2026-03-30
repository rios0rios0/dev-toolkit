package system

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RunTop5Size shows the top 5 largest items in the given directory.
func RunTop5Size(runner Runner, fs FileSystem, dir string, useSudo bool, output io.Writer) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}

	if len(entries) == 0 {
		logf(output, "directory is empty")
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, filepath.Join(dir, e.Name()))
	}

	args := append([]string{"-sb"}, names...)
	bin, cmdArgs := buildCommand(useSudo, "du", args...)

	raw, err := runner.Output(bin, cmdArgs...)
	if err != nil {
		return fmt.Errorf("running du: %w", err)
	}

	items, err := parseDuOutput(raw)
	if err != nil {
		return fmt.Errorf("parsing du output: %w", err)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].bytes > items[j].bytes
	})

	limit := min(5, len(items))

	for _, item := range items[:limit] {
		fmt.Fprintf(output, "%-10s %s\n", formatBytes(item.bytes), item.name)
	}

	return nil
}

type duEntry struct {
	bytes int64
	name  string
}

const duFieldCount = 2

func parseDuOutput(raw string) ([]duEntry, error) {
	var items []duEntry
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", duFieldCount)
		if len(parts) != duFieldCount {
			continue
		}
		b, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %w", parts[0], err)
		}
		items = append(items, duEntry{bytes: b, name: strings.TrimSpace(parts[1])})
	}
	return items, nil
}

func formatBytes(b int64) string {
	units := []string{"B", "K", "M", "G", "T"}
	size := float64(b)
	idx := 0
	for size >= 1024 && idx < len(units)-1 {
		size /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d%s", b, units[idx])
	}
	return fmt.Sprintf("%.1f%s", size, units[idx])
}
