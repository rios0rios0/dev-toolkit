package repo

import (
	"os"
	"path/filepath"
)

// ScanLocalRepos scans the root directory for git repositories at the given depth.
func ScanLocalRepos(rootDir string, depth int) []string {
	if depth == ScanDepthNested {
		return ScanNestedRepos(rootDir)
	}
	return ScanFlatRepos(rootDir)
}

// ScanFlatRepos scans a single level for directories containing .git.
func ScanFlatRepos(rootDir string) []string {
	var repos []string
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return repos
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(rootDir, e.Name(), ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			repos = append(repos, e.Name())
		}
	}
	return repos
}

// ScanNestedRepos scans two levels deep (project/repo) for directories containing .git.
func ScanNestedRepos(rootDir string) []string {
	var repos []string
	projects, err := os.ReadDir(rootDir)
	if err != nil {
		return repos
	}
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		repos = append(repos, scanProjectRepos(rootDir, p.Name())...)
	}
	return repos
}

func scanProjectRepos(rootDir, projectName string) []string {
	var repos []string
	subEntries, err := os.ReadDir(filepath.Join(rootDir, projectName))
	if err != nil {
		return repos
	}
	for _, e := range subEntries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(rootDir, projectName, e.Name(), ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			repos = append(repos, projectName+"/"+e.Name())
		}
	}
	return repos
}

// FindAllRepos walks the directory tree and returns all paths containing .git directories.
func FindAllRepos(rootDir string) []string {
	var repos []string
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, _ error) error {
		if info == nil {
			return filepath.SkipDir
		}
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			if repoPath != rootDir {
				repos = append(repos, repoPath)
			}
			return filepath.SkipDir
		}
		return nil
	})
	return repos
}
