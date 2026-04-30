package system_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/system"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

// testHome is the synthetic home directory used by every cleanup harness.
const testHome = "/home/testuser"

// cleanupHarness bundles the test doubles and sizing map used by RunCleanup
// tests. Each test constructs its own harness so sub-tests can run in parallel.
type cleanupHarness struct {
	fs      *doubles.FileSystemStub
	runner  *doubles.SystemRunnerStub
	sizes   map[string]int64
	buf     *bytes.Buffer
	present map[string]bool
}

func newCleanupHarness() *cleanupHarness {
	harness := &cleanupHarness{
		fs:      doubles.NewFileSystemStub().WithHomeDir(testHome),
		runner:  doubles.NewSystemRunnerStub(),
		sizes:   map[string]int64{},
		buf:     &bytes.Buffer{},
		present: map[string]bool{},
	}
	// `du -sb <path>` is the sole size source used by cleanup.go. Route it
	// through the sizes map so tests can declare which paths "exist".
	harness.runner.OutputFunc = func(name string, args ...string) (string, error) {
		if name != "du" || len(args) < 2 || args[0] != "-sb" {
			return "", nil
		}
		path := args[1]
		size, ok := harness.sizes[path]
		if !ok {
			return "", errors.New("no such file or directory")
		}
		return fmt.Sprintf("%d\t%s", size, path), nil
	}
	return harness
}

// withPath marks a path as present with the given size. The presence is
// wired into the FS stub's Lstat so the cleanup code treats the path as
// existing; the size is served back through the `du` stub.
func (h *cleanupHarness) withPath(path string, size int64) *cleanupHarness {
	h.sizes[path] = size
	h.present[path] = true
	h.fs = h.fs.WithPresentPath(path)
	return h
}

// markPresent registers a path as existing without configuring a size.
// Used to verify that zero-byte files (or paths where `du` is unavailable)
// are still removed.
func (h *cleanupHarness) markPresent(path string) *cleanupHarness {
	h.present[path] = true
	h.fs = h.fs.WithPresentPath(path)
	return h
}

func (h *cleanupHarness) config(dryRun bool, lookPath func(string) bool) system.CleanupConfig {
	return system.CleanupConfig{
		Runner:   h.runner,
		FS:       h.fs,
		DryRun:   dryRun,
		Output:   h.buf,
		LookPath: lookPath,
	}
}

// noBinaries reports that no tool-native cleaners are available. Tests use
// this to force the fallback direct-removal path in each category.
func noBinaries(_ string) bool { return false }

func TestRunCleanup(t *testing.T) {
	t.Parallel()

	t.Run("should return error when home directory cannot be determined", func(t *testing.T) {
		t.Parallel()
		// given
		fs := doubles.NewFileSystemStub().WithHomeDirError(errors.New("no home"))
		runner := doubles.NewSystemRunnerStub()
		var buf bytes.Buffer

		// when
		err := system.RunCleanup(system.CleanupConfig{
			Runner:   runner,
			FS:       fs,
			Output:   &buf,
			LookPath: noBinaries,
		})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting home directory")
	})

	t.Run("should report plan lines and not remove anything when dry-run enabled", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		h := newCleanupHarness().
			withPath(filepath.Join(home, ".cache/JetBrains/RemoteDev/dist"), 9_600_000_000).
			withPath(filepath.Join(home, ".cache/golangci-lint"), 5_100_000)

		// when
		err := system.RunCleanup(h.config(true, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(), "(dry-run mode)")
		assert.Contains(t, h.buf.String(), "[plan] .cache/JetBrains/RemoteDev/dist")
		assert.Contains(t, h.buf.String(), "[plan] .cache/golangci-lint")
		assert.Contains(t, h.buf.String(), "dry-run total (would reclaim)")
		assert.Empty(t, h.fs.RemovedAll, "dry-run must not call RemoveAll")
	})

	t.Run("should call RemoveAll for each present path when not dry-run", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		jbPath := filepath.Join(home, ".cache/JetBrains/RemoteDev/dist")
		lintPath := filepath.Join(home, ".cache/golangci-lint")
		h := newCleanupHarness().
			withPath(jbPath, 9_600_000_000).
			withPath(lintPath, 5_100_000)

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.fs.RemovedAll, jbPath)
		assert.Contains(t, h.fs.RemovedAll, lintPath)
		assert.Contains(t, h.buf.String(), "[done] .cache/JetBrains/RemoteDev/dist")
		assert.Contains(t, h.buf.String(), "reclaimed: ")
	})

	t.Run("should log [skip] and not remove absent paths", func(t *testing.T) {
		t.Parallel()
		// given
		h := newCleanupHarness() // no withPath calls -- every path is absent

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(), "[skip] .cache/JetBrains/RemoteDev/dist")
		assert.Contains(t, h.buf.String(), "[skip] .gradle/caches")
		assert.Empty(t, h.fs.RemovedAll)
	})

	t.Run("should invoke go clean when go binary is available", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		goCachePath := filepath.Join(home, ".cache/go-build")
		goModPath := filepath.Join(home, "go/pkg/mod")
		h := newCleanupHarness().
			withPath(goCachePath, 568_000_000).
			withPath(goModPath, 12_000_000_000)
		hasGo := func(bin string) bool { return bin == "go" }
		var goCalls []string
		h.runner.RunFunc = func(name string, args ...string) error {
			if name == "go" {
				goCalls = append(goCalls, strings.Join(args, " "))
			}
			return nil
		}

		// when
		err := system.RunCleanup(h.config(false, hasGo))

		// then
		require.NoError(t, err)
		assert.Contains(t, goCalls, "clean -cache")
		assert.Contains(t, goCalls, "clean -modcache")
		assert.Contains(t, goCalls, "clean -testcache")
		// The direct rm fallback on go/pkg/mod should NOT fire when go is available.
		assert.NotContains(t, h.fs.RemovedAll, goModPath)
	})

	t.Run("should fall back to direct removal when go binary is missing", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		goCachePath := filepath.Join(home, ".cache/go-build")
		goModPath := filepath.Join(home, "go/pkg/mod")
		h := newCleanupHarness().
			withPath(goCachePath, 568_000_000).
			withPath(goModPath, 12_000_000_000)

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.fs.RemovedAll, goCachePath)
		assert.Contains(t, h.fs.RemovedAll, goModPath)
		// The "gradle --stop" entry exercises the runTool skip path while the
		// Go category uses the silent fallback branch.
		assert.Contains(t, h.buf.String(), "[skip] gradle --stop (gradle not on PATH)")
	})

	t.Run("should invoke terra clear --global when terra binary is available", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		terraPath := filepath.Join(home, ".cache/terra")
		h := newCleanupHarness().withPath(terraPath, 3_900_000_000)
		hasTerra := func(bin string) bool { return bin == "terra" }
		var terraCalls [][]string
		h.runner.RunFunc = func(name string, args ...string) error {
			if name == "terra" {
				terraCalls = append(terraCalls, args)
			}
			return nil
		}

		// when
		err := system.RunCleanup(h.config(false, hasTerra))

		// then
		require.NoError(t, err)
		require.Len(t, terraCalls, 1)
		assert.Equal(t, []string{"clear", "--global"}, terraCalls[0])
	})

	t.Run("should continue cleaning remaining categories after a RemoveAll failure", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		failingPath := filepath.Join(home, ".cache/JetBrains/RemoteDev/dist")
		goodPath := filepath.Join(home, ".cache/golangci-lint")
		h := newCleanupHarness().
			withPath(failingPath, 9_600_000_000).
			withPath(goodPath, 5_100_000)
		h.fs = h.fs.WithRemoveAllError(failingPath, errors.New("permission denied"))

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(), "warning: permission denied")
		assert.Contains(t, h.fs.RemovedAll, goodPath,
			"later categories must still run after an earlier failure")
	})

	t.Run("should keep only the latest claude agent version", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		versionsDir := filepath.Join(home, ".local/share/claude/versions")
		h := newCleanupHarness()
		// Mixed-width semver — ensures natural-order, not lexicographic, sort.
		// Claude stores each "version" as a regular binary file (EntryIsDir:false),
		// not a directory, so this case is the file-based child scenario.
		versions := []string{"2.1.90", "2.1.104", "2.1.107", "2.1.108"}
		entries := make([]os.DirEntry, 0, len(versions))
		for _, v := range versions {
			entries = append(entries, &doubles.FakeDirEntry{EntryName: v, EntryIsDir: false})
			h.withPath(filepath.Join(versionsDir, v), 200_000_000)
		}
		h.fs = h.fs.WithReadDir(versionsDir, entries)

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		// The oldest three should be removed; 2.1.108 (latest) kept.
		assert.Contains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2.1.90"))
		assert.Contains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2.1.104"))
		assert.Contains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2.1.107"))
		assert.NotContains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2.1.108"))
		assert.Contains(t, h.buf.String(), "kept 2.1.108")
	})

	t.Run("should keep only the latest cursor-agent date-style version", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		versionsDir := filepath.Join(home, ".local/share/cursor-agent/versions")
		h := newCleanupHarness()
		// Date-style identifiers with a git SHA suffix — the natural-order
		// comparator must fall back to lex comparison when the pieces are
		// non-numeric but maintain numeric ordering on the date segments.
		versions := []string{
			"2026.01.28-fd13201",
			"2026.02.13-41ac335",
			"2026.03.11-6dfa30c",
			"2026.03.25-933d5a6",
		}
		entries := make([]os.DirEntry, 0, len(versions))
		for _, v := range versions {
			entries = append(entries, &doubles.FakeDirEntry{EntryName: v, EntryIsDir: true})
			h.withPath(filepath.Join(versionsDir, v), 160_000_000)
		}
		h.fs = h.fs.WithReadDir(versionsDir, entries)

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(), "kept 2026.03.25-933d5a6")
		assert.Contains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2026.01.28-fd13201"))
		assert.NotContains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2026.03.25-933d5a6"))
	})

	t.Run("should skip agent version cleanup when only one version is present", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		versionsDir := filepath.Join(home, ".local/share/claude/versions")
		h := newCleanupHarness()
		entries := []os.DirEntry{&doubles.FakeDirEntry{EntryName: "2.1.108", EntryIsDir: true}}
		h.fs = h.fs.WithReadDir(versionsDir, entries).
			WithReadDir(filepath.Join(home, ".local/share/cursor-agent/versions"), nil)

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(), "[skip] claude agent versions (only 1 version present)")
		assert.NotContains(t, h.fs.RemovedAll, filepath.Join(versionsDir, "2.1.108"))
	})

	t.Run("should delete .wget-hsts and all matching zcompdump glob entries", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		hstsPath := filepath.Join(home, ".wget-hsts")
		zcdA := filepath.Join(home, ".zcompdump")
		zcdB := filepath.Join(home, ".zcompdump-host-5.9")
		h := newCleanupHarness().
			withPath(hstsPath, 266).
			withPath(zcdA, 52_000).
			withPath(zcdB, 62_000)
		h.fs = h.fs.WithGlob(filepath.Join(home, ".zcompdump*"), []string{zcdA, zcdB})

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.fs.RemovedAll, hstsPath)
		assert.Contains(t, h.fs.RemovedAll, zcdA)
		assert.Contains(t, h.fs.RemovedAll, zcdB)
	})

	t.Run("should remove zero-byte files that exist on disk", func(t *testing.T) {
		t.Parallel()
		// given
		home := testHome
		// `.wget-hsts` is often created empty; the cleanup code must still
		// remove it rather than treating zero size as "absent".
		hstsPath := filepath.Join(home, ".wget-hsts")
		h := newCleanupHarness().markPresent(hstsPath)

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.fs.RemovedAll, hstsPath,
			"zero-byte files must still be removed (du size == 0 is not absence)")
	})

	t.Run("should log a warning when version directory is unreadable", func(t *testing.T) {
		t.Parallel()
		// given — a permission error from ReadDir must not be reported as "absent".
		home := testHome
		versionsDir := filepath.Join(home, ".local/share/claude/versions")
		h := newCleanupHarness()
		h.fs = h.fs.WithReadDirError(versionsDir, errors.New("permission denied"))

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(),
			"warning: reading claude agent versions: permission denied",
			"unreadable directories should surface the real error, not a misleading skip")
		assert.NotContains(t, h.buf.String(), "[skip] claude agent versions (absent)")
	})

	t.Run("should log a warning when du fails on a present path", func(t *testing.T) {
		t.Parallel()
		// given — the path exists but `du` cannot be executed (e.g. missing binary).
		home := testHome
		hstsPath := filepath.Join(home, ".wget-hsts")
		h := newCleanupHarness().markPresent(hstsPath)
		h.runner.OutputFunc = func(name string, args ...string) (string, error) {
			if name == "du" && len(args) >= 2 && args[1] == hstsPath {
				return "", errors.New("du: command not found")
			}
			return "", nil
		}

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		assert.Contains(t, h.buf.String(), "warning: du "+hstsPath+": du: command not found")
		assert.Contains(t, h.fs.RemovedAll, hstsPath,
			"the file should still be removed even when size reporting fails")
	})

	t.Run("should skip JetBrains per-product loop when readdir returns no entries", func(t *testing.T) {
		t.Parallel()
		// given
		h := newCleanupHarness() // no JetBrains children

		// when
		err := system.RunCleanup(h.config(false, noBinaries))

		// then
		require.NoError(t, err)
		// The single RemoteDev/dist plan line is the only JetBrains log we expect.
		assert.Contains(t, h.buf.String(), "[skip] .cache/JetBrains/RemoteDev/dist")
		// No per-product sub-directories should appear.
		assert.NotContains(t, h.buf.String(), "GoLand")
	})
}
