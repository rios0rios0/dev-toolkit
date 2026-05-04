package repo

// Provider name identifiers used across path detection, the provider registry, scan
// depth lookups, token env mappings, and host alias resolution.
const (
	ProviderGitHub      = "github"
	ProviderAzureDevOps = "azuredevops"
	ProviderGitLab      = "gitlab"
	ProviderCodeberg    = "codeberg"
)

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
