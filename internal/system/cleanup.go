package system

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CleanupConfig bundles the dependencies needed by [RunCleanup].
type CleanupConfig struct {
	Runner Runner
	FS     FileSystem
	DryRun bool
	Output io.Writer
	// LookPath reports whether a binary is available on PATH. When nil,
	// [exec.LookPath] is used. Overriding this keeps unit tests free of
	// host-specific binary assumptions.
	LookPath func(bin string) bool
}

func (cfg CleanupConfig) hasBinary(bin string) bool {
	if cfg.LookPath != nil {
		return cfg.LookPath(bin)
	}
	_, err := exec.LookPath(bin)
	return err == nil
}

// RunCleanup reclaims disk space in $HOME by clearing caches, transient
// downloads, and obsolete tool-version artifacts. Credentials, configs,
// shell history, and installed SDK runtimes are never touched.
//
// Categories run sequentially and a failure in one does not abort the
// others. The function returns an error only when $HOME cannot be resolved.
func RunCleanup(cfg CleanupConfig) error {
	homeDir, err := cfg.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	if cfg.DryRun {
		logf(cfg.Output, "(dry-run mode)")
	}

	type categoryFn func(cfg CleanupConfig, homeDir string) int64

	categories := []struct {
		name string
		fn   categoryFn
	}{
		{"JetBrains caches", cleanupJetBrains},
		{"Go caches", cleanupGo},
		{"Terra/Terraform caches", cleanupTerra},
		{"Gradle", cleanupGradle},
		{"SDKMAN", cleanupSDKMAN},
		{"Node/JS caches", cleanupNode},
		{"Python caches", cleanupPython},
		{"CLI agent old versions", cleanupAgentVersions},
		{"Miscellaneous caches", cleanupMisc},
		{"Claude Code transient state", cleanupClaudeState},
		{"Misc stale files", cleanupMiscStale},
	}

	var total int64
	for _, c := range categories {
		logf(cfg.Output, "==> %s", c.name)
		total += c.fn(cfg, homeDir)
	}

	logf(cfg.Output, "==> Summary")
	if cfg.DryRun {
		logf(cfg.Output, "dry-run total (would reclaim): %s", formatBytes(total))
	} else {
		logf(cfg.Output, "reclaimed: %s", formatBytes(total))
	}
	logf(cfg.Output, "preserved: credentials, shell history, installed SDKs, user work")
	return nil
}

// -- Category: JetBrains ------------------------------------------------------

func cleanupJetBrains(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64

	// Remote IDE backend distributions — re-downloaded by Gateway on next connect.
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/JetBrains/RemoteDev/dist"),
		".cache/JetBrains/RemoteDev/dist")

	// Per-product cache/log/tmp subfolders. Product dirs themselves remain so
	// the IDE can write fresh indexes on launch.
	productsDir := filepath.Join(homeDir, ".cache/JetBrains")
	entries, err := cfg.FS.ReadDir(productsDir)
	if err != nil {
		return reclaimed
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "RemoteDev" || strings.HasPrefix(name, "RemoteDev-") {
			continue // already handled above
		}
		base := filepath.Join(productsDir, name)
		reclaimed += removePath(cfg, filepath.Join(base, "caches"), ".cache/JetBrains/"+name+"/caches")
		reclaimed += removePath(cfg, filepath.Join(base, "log"), ".cache/JetBrains/"+name+"/log")
		reclaimed += removePath(cfg, filepath.Join(base, "tmp"), ".cache/JetBrains/"+name+"/tmp")
	}
	return reclaimed
}

// -- Category: Go -------------------------------------------------------------

func cleanupGo(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64

	// Tool-native cleaners handle read-only permissions in the module cache.
	if cfg.hasBinary("go") {
		goCachePath := filepath.Join(homeDir, ".cache/go-build")
		goModCachePath := filepath.Join(homeDir, "go/pkg/mod")
		before := pathSize(cfg, goCachePath) + pathSize(cfg, goModCachePath)
		runTool(cfg, "go clean -cache", "go", "clean", "-cache")
		runTool(cfg, "go clean -modcache", "go", "clean", "-modcache")
		runTool(cfg, "go clean -testcache", "go", "clean", "-testcache")
		if !cfg.DryRun {
			reclaimed += before - pathSize(cfg, goCachePath) - pathSize(cfg, goModCachePath)
		} else {
			reclaimed += before
		}
	} else {
		// Fallback: direct removal if `go` is not on PATH.
		reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/go-build"), ".cache/go-build")
		reclaimed += removePath(cfg, filepath.Join(homeDir, "go/pkg/mod"), "go/pkg/mod")
	}

	reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/golangci-lint"), ".cache/golangci-lint")
	return reclaimed
}

// -- Category: Terra ----------------------------------------------------------

func cleanupTerra(cfg CleanupConfig, homeDir string) int64 {
	if cfg.hasBinary("terra") {
		terraCachePath := filepath.Join(homeDir, ".cache/terra")
		before := pathSize(cfg, terraCachePath)
		runTool(cfg, "terra clear --global", "terra", "clear", "--global")
		if cfg.DryRun {
			return before
		}
		return before - pathSize(cfg, terraCachePath)
	}
	return removePath(cfg, filepath.Join(homeDir, ".cache/terra"), ".cache/terra")
}

// -- Category: Gradle ---------------------------------------------------------

func cleanupGradle(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64
	runTool(cfg, "gradle --stop", "gradle", "--stop")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".gradle/caches"), ".gradle/caches")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".gradle/daemon"), ".gradle/daemon")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".gradle/wrapper"), ".gradle/wrapper")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".gradle/native"), ".gradle/native")
	return reclaimed
}

// -- Category: SDKMAN ---------------------------------------------------------

func cleanupSDKMAN(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".sdkman/tmp"), ".sdkman/tmp")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".sdkman/archives"), ".sdkman/archives")
	return reclaimed
}

// -- Category: Node / JS ------------------------------------------------------

func cleanupNode(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64
	if cfg.hasBinary("npm") {
		cachePath := filepath.Join(homeDir, ".npm/_cacache")
		before := pathSize(cfg, cachePath)
		runTool(cfg, "npm cache clean --force", "npm", "cache", "clean", "--force")
		if cfg.DryRun {
			reclaimed += before
		} else {
			reclaimed += before - pathSize(cfg, cachePath)
		}
	} else {
		reclaimed += removePath(cfg, filepath.Join(homeDir, ".npm/_cacache"), ".npm/_cacache")
	}
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".npm/_npx"), ".npm/_npx")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".npm/_logs"), ".npm/_logs")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".yarn/berry/cache"), ".yarn/berry/cache")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/node-gyp"), ".cache/node-gyp")
	return reclaimed
}

// -- Category: Python ---------------------------------------------------------

func cleanupPython(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64
	if cfg.hasBinary("pip") {
		pipCachePath := filepath.Join(homeDir, ".cache/pip")
		before := pathSize(cfg, pipCachePath)
		runTool(cfg, "pip cache purge", "pip", "cache", "purge")
		if cfg.DryRun {
			reclaimed += before
		} else {
			reclaimed += before - pathSize(cfg, pipCachePath)
		}
	} else {
		reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/pip"), ".cache/pip")
	}
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/pdm"), ".cache/pdm")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/pysigma"), ".cache/pysigma")
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".cache/black"), ".cache/black")
	return reclaimed
}

// -- Category: CLI agent old versions ----------------------------------------

func cleanupAgentVersions(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64
	reclaimed += keepLatestVersion(cfg, filepath.Join(homeDir, ".local/share/claude/versions"),
		"claude agent versions")
	reclaimed += keepLatestVersion(cfg, filepath.Join(homeDir, ".local/share/cursor-agent/versions"),
		"cursor-agent versions")
	return reclaimed
}

// -- Category: Misc small caches ---------------------------------------------

func cleanupMisc(cfg CleanupConfig, homeDir string) int64 {
	miscPaths := []struct{ rel, label string }{
		{".cache/trivy", ".cache/trivy"},
		{".cache/helm", ".cache/helm"},
		{".cache/github-copilot", ".cache/github-copilot"},
		{".cache/claude-cli-nodejs", ".cache/claude-cli-nodejs"},
		{".cache/gh", ".cache/gh"},
		{".cache/fontconfig", ".cache/fontconfig"},
		{".cache/JNA", ".cache/JNA"},
		{".cache/zinit", ".cache/zinit"},
	}
	var reclaimed int64
	for _, p := range miscPaths {
		reclaimed += removePath(cfg, filepath.Join(homeDir, p.rel), p.label)
	}
	return reclaimed
}

// -- Category: Claude Code transient state -----------------------------------

func cleanupClaudeState(cfg CleanupConfig, homeDir string) int64 {
	// Preserved: settings*.json, rules/, agents/, commands/, memory/,
	// projects/ (conversation history), plans/, tasks/, todos/, plugins/,
	// backups/, history.jsonl.
	transient := []struct{ rel, label string }{
		{".claude/file-history", ".claude/file-history"},
		{".claude/shell-snapshots", ".claude/shell-snapshots"},
		{".claude/paste-cache", ".claude/paste-cache"},
		{".claude/session-env", ".claude/session-env"},
		{".claude/telemetry", ".claude/telemetry"},
		{".claude/debug", ".claude/debug"},
		{".claude/cache", ".claude/cache"},
	}
	var reclaimed int64
	for _, p := range transient {
		reclaimed += removePath(cfg, filepath.Join(homeDir, p.rel), p.label)
	}
	return reclaimed
}

// -- Category: Misc stale files ----------------------------------------------

func cleanupMiscStale(cfg CleanupConfig, homeDir string) int64 {
	var reclaimed int64
	reclaimed += removePath(cfg, filepath.Join(homeDir, ".wget-hsts"), ".wget-hsts")
	reclaimed += removeGlob(cfg, filepath.Join(homeDir, ".zcompdump*"), ".zcompdump*")
	return reclaimed
}

// -- Helpers -----------------------------------------------------------------

// removePath removes a file or directory recursively. In dry-run mode it
// only reports what would be removed. Returns the number of bytes that were
// (or would have been) freed.
func removePath(cfg CleanupConfig, path, label string) int64 {
	size := pathSize(cfg, path)
	if size == 0 {
		logf(cfg.Output, "[skip] %s (absent or empty)", label)
		return 0
	}
	if cfg.DryRun {
		logf(cfg.Output, "[plan] %s (%s)", label, formatBytes(size))
		return size
	}
	logf(cfg.Output, "[plan] %s (%s)", label, formatBytes(size))
	if err := cfg.FS.RemoveAll(path); err != nil {
		logf(cfg.Output, "warning: %v", err)
		return 0
	}
	logf(cfg.Output, "[done] %s (%s reclaimed)", label, formatBytes(size))
	return size
}

// removeGlob expands the pattern and removes every match.
func removeGlob(cfg CleanupConfig, pattern, label string) int64 {
	matches, err := cfg.FS.Glob(pattern)
	if err != nil {
		logf(cfg.Output, "warning: glob %s: %v", pattern, err)
		return 0
	}
	if len(matches) == 0 {
		logf(cfg.Output, "[skip] %s (no matches)", label)
		return 0
	}
	var reclaimed int64
	for _, match := range matches {
		reclaimed += removePath(cfg, match, match)
	}
	return reclaimed
}

// keepLatestVersion removes every child of dir except the one with the
// highest natural-order version. Children may be directories (cursor-agent)
// or regular files (claude, whose "versions" are named single-file binaries).
func keepLatestVersion(cfg CleanupConfig, dir, label string) int64 {
	entries, err := cfg.FS.ReadDir(dir)
	if err != nil {
		logf(cfg.Output, "[skip] %s (absent)", label)
		return 0
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) <= 1 {
		logf(cfg.Output, "[skip] %s (only %d version present)", label, len(names))
		return 0
	}
	sort.Slice(names, func(i, j int) bool {
		return compareVersions(names[i], names[j]) < 0
	})
	latest := names[len(names)-1]
	var reclaimed int64
	for _, name := range names {
		if name == latest {
			continue
		}
		target := filepath.Join(dir, name)
		reclaimed += removePath(cfg, target, label+": "+name)
	}
	if !cfg.DryRun {
		logf(cfg.Output, "[done] %s (kept %s)", label, latest)
	}
	return reclaimed
}

// runTool invokes a tool-native cleaner via Runner. Skips with a log line if
// the binary is missing from PATH. Respects dry-run mode.
func runTool(cfg CleanupConfig, label, bin string, args ...string) {
	if !cfg.hasBinary(bin) {
		logf(cfg.Output, "[skip] %s (%s not on PATH)", label, bin)
		return
	}
	if cfg.DryRun {
		logf(cfg.Output, "[plan] %s", label)
		return
	}
	logf(cfg.Output, "[plan] %s", label)
	if err := cfg.Runner.Run(bin, args...); err != nil {
		logf(cfg.Output, "warning: %s: %v", label, err)
		return
	}
	logf(cfg.Output, "[done] %s", label)
}

// pathSize returns the total size of a path in bytes via `du -sb`. Returns 0
// if the path does not exist or du fails.
func pathSize(cfg CleanupConfig, path string) int64 {
	raw, err := cfg.Runner.Output("du", "-sb", path)
	if err != nil {
		return 0
	}
	parts := strings.SplitN(strings.TrimSpace(raw), "\t", duFieldCount)
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// compareVersions performs natural-order comparison of version-like strings.
// Parts separated by '.' or '-' are compared numerically when both parts are
// numeric, and lexicographically otherwise. Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	partsA := splitVersion(a)
	partsB := splitVersion(b)
	n := min(len(partsB), len(partsA))
	for i := range n {
		ai, aerr := strconv.Atoi(partsA[i])
		bi, berr := strconv.Atoi(partsB[i])
		if aerr == nil && berr == nil {
			if ai != bi {
				if ai < bi {
					return -1
				}
				return 1
			}
			continue
		}
		if partsA[i] != partsB[i] {
			if partsA[i] < partsB[i] {
				return -1
			}
			return 1
		}
	}
	if len(partsA) != len(partsB) {
		if len(partsA) < len(partsB) {
			return -1
		}
		return 1
	}
	return 0
}

func splitVersion(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == '-'
	})
	return fields
}
