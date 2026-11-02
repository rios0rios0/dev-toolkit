package repo

// Provider name identifiers used across path detection, the provider registry, scan
// depth lookups, token env mappings, and host alias resolution.
const (
	ProviderGitHub      = "github"
	ProviderAzureDevOps = "azuredevops"
	ProviderGitLab      = "gitlab"
	ProviderCodeberg    = "codeberg"
)

// Remote names the sync workflows read default branches from.
const (
	originRemote   = "origin"
	upstreamRemote = "upstream"
)

// localBranchRefPrefix qualifies a branch name as a local branch ref.
const localBranchRefPrefix = "refs/heads/"

// Log field keys reused across repo workflows (clone, sync, prune, mirror, failover, restore).
const (
	logFieldRepo   = "repo"
	logFieldStatus = "status"
	logFieldOwner  = "owner"
	logFieldTarget = "target"
)

// Status categories surfaced in summary log fields and result-classification maps.
const (
	statusSkipped  = "skipped"
	statusSwitched = "switched"
	statusSynced   = "synced"
	statusRestored = "restored"
	statusMirrored = "mirrored"
)
