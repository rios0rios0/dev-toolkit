package repo

import (
	"errors"
	"fmt"
	"strings"
)

// ErrDefaultBranchNotFound reports that no detection rule could tell which branch a remote
// uses as its default. Callers report the repository as failed instead of assuming a
// conventional name such as "main".
var ErrDefaultBranchNotFound = errors.New("default branch not found")

// advertisedBranchPrefix starts the line `git ls-remote --symref` prints for a symbolic ref
// pointing at a branch.
const advertisedBranchPrefix = "ref: " + localBranchRefPrefix

// conventionalDefaultBranches are tried, in order, only when neither the recorded remote
// HEAD nor the remote itself can say which branch is the default.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var conventionalDefaultBranches = []string{"main", "master"}

// defaultBranchRule reports the default branch of remote, or false when it cannot tell.
// A rule never answers with a branch it has not seen exist.
type defaultBranchRule func(repoPath, remote string, runner GitRunner) (string, bool)

// remoteDefaultBranchRules detect the default branch of a remote, most trusted first. The
// network is only consulted when the pointer recorded locally cannot be trusted.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var remoteDefaultBranchRules = []defaultBranchRule{
	fromRecordedRemoteHEAD,
	fromAdvertisedRemoteHEAD,
	fromConventionalRemoteBranch,
}

// repositoryDefaultBranchRules detect the default branch of the repository itself: the
// rules for its origin remote, then the conventional names among its local branches,
// which are all a repository without a usable origin has to go on.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var repositoryDefaultBranchRules = []defaultBranchRule{
	fromRecordedRemoteHEAD,
	fromAdvertisedRemoteHEAD,
	fromConventionalRemoteBranch,
	fromConventionalLocalBranch,
}

// DetectDefaultBranch returns the default branch of the repository, which is the default
// branch of its origin remote. It returns [ErrDefaultBranchNotFound] rather than guessing
// when nothing reveals it, so a repository whose default is "develop" or "master" is never
// synced or pruned as if it used "main".
func DetectDefaultBranch(repoPath string, runner GitRunner) (string, error) {
	return detectDefaultBranch(repoPath, originRemote, repositoryDefaultBranchRules, runner)
}

// DetectRemoteDefaultBranch returns the default branch of the named remote. Unlike
// [DetectDefaultBranch] it never falls back to local branches, which say nothing about a
// remote such as a fork's upstream.
func DetectRemoteDefaultBranch(repoPath, remote string, runner GitRunner) (string, error) {
	return detectDefaultBranch(repoPath, remote, remoteDefaultBranchRules, runner)
}

func detectDefaultBranch(repoPath, remote string, rules []defaultBranchRule, runner GitRunner) (string, error) {
	for _, rule := range rules {
		if branch, ok := rule(repoPath, remote, runner); ok {
			return branch, nil
		}
	}
	return "", fmt.Errorf("%w for remote %s", ErrDefaultBranchNotFound, remote)
}

// refreshRemoteHEAD asks the remote for its default branch again and records the answer in
// refs/remotes/<remote>/HEAD. git writes that pointer when cloning and, by default, no
// fetch updates it afterwards, so a remote that renamed or switched its default branch
// leaves it stale. Best effort: it needs the remote to be reachable and the branch to be
// fetched already, and when it fails the detection rules still apply.
func refreshRemoteHEAD(repoPath, remote string, runner GitRunner) {
	_ = runner.Run(repoPath, "remote", "set-head", remote, "--auto")
}

// fromRecordedRemoteHEAD reads refs/remotes/<remote>/HEAD, the pointer recorded by git clone
// or [refreshRemoteHEAD]. A pointer to a branch that no longer exists is ignored: after the
// remote renames its default and a fetch prunes the old name, git keeps reporting the
// deleted branch.
func fromRecordedRemoteHEAD(repoPath, remote string, runner GitRunner) (string, bool) {
	prefix := remoteRefPrefix(remote)
	branch, found := strings.CutPrefix(runner.Output(repoPath, "symbolic-ref", prefix+"HEAD"), prefix)
	if !found || branch == "" || !refExists(repoPath, prefix+branch, runner) {
		return "", false
	}
	return branch, true
}

// fromAdvertisedRemoteHEAD asks the remote which branch its HEAD points to, the question git
// clone asks. `git ls-remote --symref <remote> HEAD` prints:
//
//	ref: refs/heads/main	HEAD
//	<object id>	HEAD
func fromAdvertisedRemoteHEAD(repoPath, remote string, runner GitRunner) (string, bool) {
	output := runner.Output(repoPath, "ls-remote", "--symref", remote, "HEAD")
	for line := range strings.SplitSeq(output, "\n") {
		target, name, found := strings.Cut(strings.TrimSpace(line), "\t")
		if !found || name != "HEAD" {
			continue
		}
		if branch, isBranch := strings.CutPrefix(target, advertisedBranchPrefix); isBranch && branch != "" {
			return branch, true
		}
	}
	return "", false
}

// fromConventionalRemoteBranch picks the first conventional name the remote is known to
// have, for when the remote cannot be reached and no HEAD was recorded for it.
func fromConventionalRemoteBranch(repoPath, remote string, runner GitRunner) (string, bool) {
	return firstExistingBranch(repoPath, remoteRefPrefix(remote), runner)
}

// fromConventionalLocalBranch picks the first conventional name among the local branches,
// for a repository without a usable origin.
func fromConventionalLocalBranch(repoPath, _ string, runner GitRunner) (string, bool) {
	return firstExistingBranch(repoPath, localBranchRefPrefix, runner)
}

func firstExistingBranch(repoPath, refPrefix string, runner GitRunner) (string, bool) {
	for _, branch := range conventionalDefaultBranches {
		if refExists(repoPath, refPrefix+branch, runner) {
			return branch, true
		}
	}
	return "", false
}

// refExists reports whether a fully qualified ref resolves in the repository.
func refExists(repoPath, ref string, runner GitRunner) bool {
	return runner.Output(repoPath, "rev-parse", "--verify", ref) != ""
}

func remoteRefPrefix(remote string) string {
	return "refs/remotes/" + remote + "/"
}
