package gist

// Log field names used across gist clone and sync workflows.
const (
	logFieldGist   = "gist"
	logFieldTarget = "target"
	logFieldFailed = "failed"
	// logFieldCredential carries where the GitHub credential came from -- gitforge's
	// Credential.Source, such as "gh CLI" -- and never the token itself.
	logFieldCredential = "credential"
)
