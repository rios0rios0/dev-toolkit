package gist

import (
	"os"
	"path/filepath"
)

// ScanLocalGists scans rootDir/<owner>/<slug> two levels deep for directories
// containing a .git directory and returns "<owner>/<slug>" keys.
func ScanLocalGists(rootDir string) []string {
	var gists []string
	owners, err := os.ReadDir(rootDir)
	if err != nil {
		return gists
	}
	for _, o := range owners {
		if !o.IsDir() {
			continue
		}
		gists = append(gists, scanOwnerGists(rootDir, o.Name())...)
	}
	return gists
}

func scanOwnerGists(rootDir, ownerName string) []string {
	var gists []string
	entries, err := os.ReadDir(filepath.Join(rootDir, ownerName))
	if err != nil {
		return gists
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(rootDir, ownerName, e.Name(), ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			gists = append(gists, ownerName+"/"+e.Name())
		}
	}
	return gists
}
