package gist

import (
	"os"
	"path/filepath"
)

// ScanLocalGists scans rootDir one level deep for directories containing a
// .git directory and returns their basenames as keys. The owner is implied
// by rootDir, so the keys match what Key (and AssignKeys) produce.
func ScanLocalGists(rootDir string) []string {
	var gists []string
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return gists
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(rootDir, e.Name(), ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			gists = append(gists, e.Name())
		}
	}
	return gists
}
