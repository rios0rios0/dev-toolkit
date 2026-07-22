package repo

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// Removal reasons: why a linked worktree is considered disposable.
const (
	WorktreeReasonStale    = "stale registration"
	WorktreeReasonOrphaned = "outside root"
	WorktreeReasonMerged   = "branch merged"
	WorktreeReasonGone     = "upstream gone"
)

// Keep reasons: why a linked worktree is never removed automatically.
const (
	WorktreeKeepMain     = "main worktree"
	WorktreeKeepLocked   = "locked"
	WorktreeKeepDetached = "detached HEAD"
	WorktreeKeepDirty    = "uncommitted changes"
	WorktreeKeepUnpushed = "unpushed commits"
	WorktreeKeepActive   = "active"
)

const (
	logFieldWorktree = "worktree"
	logFieldReason   = "reason"
	logFieldBranch   = "branch"
)

// Field keys emitted by `git worktree list --porcelain`.
const (
	worktreeFieldPath   = "worktree"
	worktreeFieldHead   = "HEAD"
	worktreeFieldBranch = "branch"
)

// Worktree describes a single entry of `git worktree list --porcelain`.
type Worktree struct {
	Path     string
	Branch   string
	Head     string
	IsMain   bool
	Bare     bool
	Detached bool
	Locked   bool
	Prunable bool
}

// WorktreeCandidate is a classified worktree along with the repository it belongs to.
type WorktreeCandidate struct {
	Repo     string
	RepoPath string
	Worktree Worktree
	Keep     bool
	Reason   string
}

// WorktreePruneConfig holds all dependencies for a worktree cleanup operation.
type WorktreePruneConfig struct {
	RootDir   string
	DryRun    bool
	AssumeYes bool
	Runner    GitRunner
	Output    io.Writer
	Input     io.Reader
	Logger    logger.FieldLogger
}

func (c *WorktreePruneConfig) log() logger.FieldLogger {
	if c.Logger == nil {
		c.Logger = NewLogger(c.Output)
	}
	return c.Logger
}

//nolint:gochecknoglobals // read-only configuration lookup table
var worktreeFieldSetters = map[string]func(w *Worktree, value string){
	worktreeFieldHead:   func(w *Worktree, v string) { w.Head = v },
	worktreeFieldBranch: func(w *Worktree, v string) { w.Branch = strings.TrimPrefix(v, "refs/heads/") },
	"detached":          func(w *Worktree, _ string) { w.Detached = true },
	"bare":              func(w *Worktree, _ string) { w.Bare = true },
	"locked":            func(w *Worktree, _ string) { w.Locked = true },
	"prunable":          func(w *Worktree, _ string) { w.Prunable = true },
}

// ParseWorktreeList parses the output of `git worktree list --porcelain`.
// The first entry is always the main worktree.
func ParseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree

	flush := func() {
		if current != nil {
			worktrees = append(worktrees, *current)
			current = nil
		}
	}

	for rawLine := range strings.SplitSeq(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			flush()
			continue
		}
		key, value, _ := strings.Cut(line, " ")
		if key == worktreeFieldPath {
			flush()
			current = &Worktree{Path: value, IsMain: len(worktrees) == 0}
			continue
		}
		if setter, ok := worktreeFieldSetters[key]; ok && current != nil {
			setter(current, value)
		}
	}
	flush()

	return worktrees
}

// worktreeEnv carries the per-repository facts that classification rules consult.
type worktreeEnv struct {
	rootDir string
	merged  map[string]struct{}
	gone    map[string]struct{}
	runner  GitRunner
}

func (e worktreeEnv) isInsideRoot(w Worktree) bool {
	rel, err := filepath.Rel(e.rootDir, w.Path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (e worktreeEnv) isDirty(w Worktree) bool {
	return e.runner.Output(w.Path, "status", "--porcelain") != ""
}

// isAhead reports whether the worktree branch holds commits its upstream does not.
// A branch whose upstream was deleted has nothing to compare against, so it is not
// treated as ahead: the "upstream gone" rule surfaces it for confirmation instead.
func (e worktreeEnv) isAhead(w Worktree) bool {
	if _, gone := e.gone[w.Branch]; gone {
		return false
	}
	count := e.runner.Output(w.Path, "rev-list", "--count", "@{upstream}..HEAD")
	return count != "" && count != "0"
}

func (e worktreeEnv) isMerged(w Worktree) bool {
	_, ok := e.merged[w.Branch]
	return ok
}

func (e worktreeEnv) isGone(w Worktree) bool {
	_, ok := e.gone[w.Branch]
	return ok
}

// worktreeRule pairs a classification reason with the predicate that detects it.
type worktreeRule struct {
	reason string
	match  func(w Worktree, env worktreeEnv) bool
}

// worktreeGuardRules protect a worktree from automatic removal. They are evaluated
// in order, before any removal rule, so losing work always wins over cleaning up.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var worktreeGuardRules = []worktreeRule{
	{reason: WorktreeKeepLocked, match: func(w Worktree, _ worktreeEnv) bool { return w.Locked }},
	{reason: WorktreeKeepDetached, match: func(w Worktree, _ worktreeEnv) bool { return w.Detached || w.Branch == "" }},
	{reason: WorktreeKeepDirty, match: func(w Worktree, env worktreeEnv) bool { return env.isDirty(w) }},
	{reason: WorktreeKeepUnpushed, match: func(w Worktree, env worktreeEnv) bool { return env.isAhead(w) }},
}

// worktreeRemovalRules mark a worktree as disposable, evaluated in order.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var worktreeRemovalRules = []worktreeRule{
	{reason: WorktreeReasonOrphaned, match: func(w Worktree, env worktreeEnv) bool { return !env.isInsideRoot(w) }},
	{reason: WorktreeReasonMerged, match: func(w Worktree, env worktreeEnv) bool { return env.isMerged(w) }},
	{reason: WorktreeReasonGone, match: func(w Worktree, env worktreeEnv) bool { return env.isGone(w) }},
}

// classifyWorktree decides whether a worktree is disposable and records why.
func classifyWorktree(w Worktree, env worktreeEnv) WorktreeCandidate {
	candidate := WorktreeCandidate{Worktree: w}

	if w.IsMain {
		candidate.Keep, candidate.Reason = true, WorktreeKeepMain
		return candidate
	}

	// A stale registration has no directory left on disk, so no guard can apply and
	// dropping it only touches the parent repository's metadata.
	if w.Prunable {
		candidate.Reason = WorktreeReasonStale
		return candidate
	}

	for _, guard := range worktreeGuardRules {
		if guard.match(w, env) {
			candidate.Keep, candidate.Reason = true, guard.reason
			return candidate
		}
	}

	for _, rule := range worktreeRemovalRules {
		if rule.match(w, env) {
			candidate.Reason = rule.reason
			return candidate
		}
	}

	candidate.Keep, candidate.Reason = true, WorktreeKeepActive
	return candidate
}

// ListGoneBranches returns local branches whose upstream has been deleted on the remote.
func ListGoneBranches(repoPath string, runner GitRunner) map[string]struct{} {
	gone := make(map[string]struct{})
	output := runner.Output(
		repoPath, "for-each-ref", "--format=%(refname:short) %(upstream:track)", "refs/heads",
	)
	if output == "" {
		return gone
	}
	for line := range strings.SplitSeq(output, "\n") {
		name, track, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found || !strings.Contains(track, "[gone]") {
			continue
		}
		gone[name] = struct{}{}
	}
	return gone
}

// ScanWorktrees classifies every worktree registered in a single repository.
func ScanWorktrees(repoPath, rootDir string, runner GitRunner) []WorktreeCandidate {
	worktrees := ParseWorktreeList(runner.Output(repoPath, "worktree", "list", "--porcelain"))
	if len(worktrees) <= 1 {
		return nil
	}

	name, err := filepath.Rel(rootDir, repoPath)
	if err != nil {
		name = repoPath
	}

	defaultBranch := DetectDefaultBranch(repoPath, runner)
	env := worktreeEnv{
		rootDir: rootDir,
		merged:  toBranchSet(ListMergedBranches(repoPath, defaultBranch, runner)),
		gone:    ListGoneBranches(repoPath, runner),
		runner:  runner,
	}

	candidates := make([]WorktreeCandidate, 0, len(worktrees))
	for _, w := range worktrees {
		candidate := classifyWorktree(w, env)
		candidate.Repo, candidate.RepoPath = name, repoPath
		candidates = append(candidates, candidate)
	}
	return candidates
}

func toBranchSet(branches []string) map[string]struct{} {
	set := make(map[string]struct{}, len(branches))
	for _, b := range branches {
		set[b] = struct{}{}
	}
	return set
}

// ScanAllWorktrees classifies the worktrees of every repository in parallel,
// preserving the repository order for deterministic reporting.
func ScanAllWorktrees(repos []string, rootDir string, runner GitRunner) []WorktreeCandidate {
	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([][]WorktreeCandidate, len(repos))
	var wg sync.WaitGroup

	for i, repoPath := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = ScanWorktrees(path, rootDir, runner)
		}(i, repoPath)
	}

	wg.Wait()

	var all []WorktreeCandidate
	for _, r := range results {
		all = append(all, r...)
	}
	return all
}

// FilterRemovableWorktrees returns only the candidates marked for removal.
func FilterRemovableWorktrees(candidates []WorktreeCandidate) []WorktreeCandidate {
	var removable []WorktreeCandidate
	for _, c := range candidates {
		if !c.Keep {
			removable = append(removable, c)
		}
	}
	return removable
}

// RemoveWorktree drops a worktree through git so the parent repository's metadata
// stays consistent. Stale registrations are cleared with `worktree prune`; live ones
// go through `worktree remove`, which refuses to discard uncommitted work.
func RemoveWorktree(c WorktreeCandidate, runner GitRunner) error {
	if c.Reason == WorktreeReasonStale {
		if err := runner.Run(c.RepoPath, "worktree", "prune"); err != nil {
			return fmt.Errorf("worktree prune: %w", err)
		}
		return nil
	}
	if err := runner.Run(c.RepoPath, "worktree", "remove", c.Worktree.Path); err != nil {
		return fmt.Errorf("worktree remove: %w", err)
	}
	return nil
}

// RunWorktreeList reports every worktree found under rootDir with its classification.
func RunWorktreeList(rootDir string, runner GitRunner, output io.Writer) error {
	log := NewLogger(output)

	repos := FindAllRepos(rootDir)
	if len(repos) == 0 {
		log.WithField("dir", rootDir).Warn("no git repositories found")
		return nil
	}

	candidates := ScanAllWorktrees(repos, rootDir, runner)
	linked := 0
	for _, c := range candidates {
		if c.Worktree.IsMain {
			continue
		}
		linked++
		log.WithFields(logger.Fields{
			logFieldRepo:     c.Repo,
			logFieldWorktree: c.Worktree.Path,
			logFieldBranch:   c.Worktree.Branch,
			logFieldReason:   c.Reason,
			"removable":      !c.Keep,
		}).Info("worktree")
	}

	log.WithFields(logger.Fields{
		"repos":     len(repos),
		"worktrees": linked,
		"removable": len(FilterRemovableWorktrees(candidates)),
	}).Info("summary")
	return nil
}

// RunWorktreePrune removes disposable worktrees under rootDir. Scanning runs in
// parallel; removal is sequential so each worktree can be confirmed interactively.
func RunWorktreePrune(cfg WorktreePruneConfig) error {
	log := cfg.log()

	repos := FindAllRepos(cfg.RootDir)
	if len(repos) == 0 {
		log.WithField("dir", cfg.RootDir).Warn("no git repositories found")
		return nil
	}

	log.WithField("count", len(repos)).Info("scanning repositories for worktrees")
	if cfg.DryRun {
		log.Info("dry-run mode enabled")
	}

	candidates := ScanAllWorktrees(repos, cfg.RootDir, cfg.Runner)
	LogGuardedWorktrees(candidates, log)

	removable := FilterRemovableWorktrees(candidates)
	if len(removable) == 0 {
		log.Info("no disposable worktrees found")
		return nil
	}

	removed, kept, failed := ProcessWorktreeRemovals(removable, cfg, log)
	log.WithFields(logger.Fields{
		"removed":          removed,
		"kept":             kept,
		mirrorStatusFailed: failed,
	}).Info("summary")
	return nil
}

// LogGuardedWorktrees reports worktrees that were protected for a notable reason,
// so a dirty or locked worktree never disappears silently from the report.
func LogGuardedWorktrees(candidates []WorktreeCandidate, log logger.FieldLogger) {
	for _, c := range candidates {
		if !c.Keep || c.Reason == WorktreeKeepMain || c.Reason == WorktreeKeepActive {
			continue
		}
		log.WithFields(logger.Fields{
			logFieldRepo:     c.Repo,
			logFieldWorktree: c.Worktree.Path,
			logFieldReason:   c.Reason,
		}).Info("kept worktree")
	}
}

// ProcessWorktreeRemovals walks the removal candidates and applies the configured
// policy to each one. Returns the removed, kept, and failed counts.
func ProcessWorktreeRemovals(
	removable []WorktreeCandidate, cfg WorktreePruneConfig, log logger.FieldLogger,
) (int, int, int) {
	isInteractive := isTerminal(cfg.Input)
	scanner := bufio.NewScanner(cfg.Input)
	removed, kept, failed := 0, 0, 0

	for _, c := range removable {
		entry := log.WithFields(logger.Fields{
			logFieldRepo:     c.Repo,
			logFieldWorktree: c.Worktree.Path,
			logFieldBranch:   c.Worktree.Branch,
			logFieldReason:   c.Reason,
		})

		switch {
		case cfg.DryRun:
			entry.Info("would remove worktree")
			kept++
		case cfg.AssumeYes:
			removed, failed = applyWorktreeRemoval(c, cfg.Runner, entry, removed, failed)
		case !isInteractive:
			entry.Warn("disposable worktree (kept, non-interactive)")
			kept++
		case PromptRemoveWorktree(c, scanner, cfg.Output):
			removed, failed = applyWorktreeRemoval(c, cfg.Runner, entry, removed, failed)
		default:
			entry.Info("kept worktree")
			kept++
		}
	}

	return removed, kept, failed
}

func applyWorktreeRemoval(
	c WorktreeCandidate, runner GitRunner, entry logger.FieldLogger, removed, failed int,
) (int, int) {
	if err := RemoveWorktree(c, runner); err != nil {
		entry.WithError(err).Error("could not remove worktree")
		return removed, failed + 1
	}
	entry.Info("removed worktree")
	return removed + 1, failed
}

// PromptRemoveWorktree asks the user to confirm removal of a single worktree.
// The scanner is shared across prompts so buffered input is not discarded between them.
func PromptRemoveWorktree(c WorktreeCandidate, scanner *bufio.Scanner, output io.Writer) bool {
	branch := c.Worktree.Branch
	if branch == "" {
		branch = c.Worktree.Head
	}
	fmt.Fprintf(output, "[dev] worktree \"%s\" of \"%s\" (branch %s) is %s. Remove? [y/N] ",
		c.Worktree.Path, c.Repo, branch, c.Reason)
	return scanner.Scan() && strings.EqualFold(strings.TrimSpace(scanner.Text()), "y")
}
