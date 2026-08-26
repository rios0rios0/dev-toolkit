package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// azureDevOpsResource is the fixed Microsoft Entra application ID of the Azure DevOps
// service. An access token issued for it is accepted anywhere a PAT is, which is what
// lets "az account get-access-token" stand in for AZURE_DEVOPS_EXT_PAT.
const azureDevOpsResource = "499b84ac-1321-427f-aa17-267ca6975798"

// cliTimeout bounds how long a provider CLI may take to hand over a token. Minting one
// can involve a network round-trip (az contacts Microsoft Entra), so an unreachable
// endpoint would otherwise hang the whole command instead of falling through to the
// next credential source.
const cliTimeout = 30 * time.Second

// Credential is an authentication token for a Git hosting provider together with a
// human-readable description of where it came from. The description exists so failures
// can name the source that produced the rejected token without ever printing the token.
type Credential struct {
	Token  string
	Source string
}

// CredentialResolver obtains a Credential for a provider. Implementations return a
// descriptive error when they cannot supply one, so a chain can explain every avenue
// the user has rather than only the last one it tried.
type CredentialResolver interface {
	Resolve(providerName string) (Credential, error)
}

// ProviderCLI describes the CLI that can mint a credential for a provider without the
// user exporting a token: the binary to probe on PATH, the arguments that print a
// usable token to stdout, and the command that authenticates it when it is not.
type ProviderCLI struct {
	Binary    string
	TokenArgs []string
	LoginHint string
}

// providerCLIMap maps a provider to the CLI that already holds the user's credentials.
// Codeberg is deliberately absent: it has no widely installed official CLI, so it stays
// on the environment variable alone.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var providerCLIMap = map[string]ProviderCLI{
	ProviderGitHub: {
		Binary:    "gh",
		TokenArgs: []string{"auth", "token"},
		LoginHint: "gh auth login",
	},
	ProviderAzureDevOps: {
		Binary: "az",
		TokenArgs: []string{
			"account", "get-access-token",
			"--resource", azureDevOpsResource,
			"--query", "accessToken",
			"--output", "tsv",
		},
		LoginHint: "az login",
	},
	ProviderGitLab: {
		Binary:    "glab",
		TokenArgs: []string{"auth", "token"},
		LoginHint: "glab auth login",
	},
}

// ProviderCLIFor returns the CLI integration registered for a provider, and whether one
// exists at all.
func ProviderCLIFor(providerName string) (ProviderCLI, bool) {
	cli, ok := providerCLIMap[providerName]
	return cli, ok
}

// DefaultCredentialResolver returns the standard chain: a token exported in the
// environment wins, and an authenticated provider CLI covers the far more common case
// where none was exported. Keeping the environment first means an explicitly configured
// token still overrides whatever the CLI happens to be logged in as.
func DefaultCredentialResolver() CredentialResolver {
	return NewChainCredentialResolver(NewEnvCredentialResolver(), NewCLICredentialResolver())
}

// CLIRunner abstracts probing for and executing provider CLIs, for testability.
type CLIRunner interface {
	LookPath(binary string) (string, error)
	Output(binary string, args ...string) (string, error)
}

// DefaultCLIRunner probes the real PATH via [exec.LookPath] and executes real commands
// via [exec.CommandContext].
type DefaultCLIRunner struct{}

func (r *DefaultCLIRunner) LookPath(binary string) (string, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("%s not found in PATH: %w", binary, err)
	}
	return path, nil
}

func (r *DefaultCLIRunner) Output(binary string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...) // #nosec G204
	// A nil stdin makes an unexpectedly interactive CLI fail on EOF rather than block
	// forever waiting for input that is never coming.
	cmd.Stdin = nil
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Only stderr is surfaced: the token the CLI would have printed goes to stdout,
	// so an error message built from stderr cannot leak it.
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return "", fmt.Errorf("%s %s: %s",
			binary, strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
	}
	return "", fmt.Errorf("%s %s: %w", binary, strings.Join(args, " "), err)
}

// EnvCredentialResolver reads a provider's token from the environment, preserving the
// original way of authenticating dev-toolkit.
type EnvCredentialResolver struct {
	Lookup func(key string) string
}

// NewEnvCredentialResolver creates a resolver backed by [os.Getenv].
func NewEnvCredentialResolver() *EnvCredentialResolver {
	return &EnvCredentialResolver{Lookup: os.Getenv}
}

func (r *EnvCredentialResolver) Resolve(providerName string) (Credential, error) {
	envVar := ProviderTokenEnv(providerName)
	if envVar == "" {
		return Credential{}, fmt.Errorf("unknown provider: %s", providerName)
	}

	lookup := r.Lookup
	if lookup == nil {
		lookup = os.Getenv
	}

	token := lookup(envVar)
	if token == "" {
		return Credential{}, fmt.Errorf("%s environment variable not set", envVar)
	}

	return Credential{Token: token, Source: envVar + " environment variable"}, nil
}

// CLICredentialResolver asks the provider's own CLI for a token. A developer who has
// already run "gh auth login" (or "az login", or "glab auth login") is authenticated
// for that provider, so requiring a separate exported token asks them to prove the same
// thing twice.
type CLICredentialResolver struct {
	Runner CLIRunner
}

// NewCLICredentialResolver creates a resolver that probes and runs the real CLIs.
func NewCLICredentialResolver() *CLICredentialResolver {
	return &CLICredentialResolver{Runner: &DefaultCLIRunner{}}
}

func (r *CLICredentialResolver) Resolve(providerName string) (Credential, error) {
	cli, ok := providerCLIMap[providerName]
	if !ok {
		return Credential{}, fmt.Errorf("no CLI integration for provider: %s", providerName)
	}

	runner := r.Runner
	if runner == nil {
		runner = &DefaultCLIRunner{}
	}

	if _, err := runner.LookPath(cli.Binary); err != nil {
		return Credential{}, fmt.Errorf("%s CLI is not installed", cli.Binary)
	}

	// The token command can fail for reasons that have nothing to do with being logged
	// out -- a timeout, a transient network error while minting the token -- so the
	// message reports only what was observed and leaves the cause to the wrapped error.
	token, err := runner.Output(cli.Binary, cli.TokenArgs...)
	if err != nil {
		return Credential{}, fmt.Errorf("%s CLI could not provide a token (authenticate with %q): %w",
			cli.Binary, cli.LoginHint, err)
	}
	if token == "" {
		return Credential{}, fmt.Errorf("%s CLI returned an empty token, run %q",
			cli.Binary, cli.LoginHint)
	}

	return Credential{Token: token, Source: cli.Binary + " CLI"}, nil
}

// ChainCredentialResolver tries each resolver in order and returns the first credential
// produced. When every resolver declines it reports all of their reasons, so the user
// sees every way of authenticating instead of only the first one that failed.
type ChainCredentialResolver struct {
	Resolvers []CredentialResolver
}

// NewChainCredentialResolver creates a chain over the given resolvers, in priority order.
func NewChainCredentialResolver(resolvers ...CredentialResolver) *ChainCredentialResolver {
	return &ChainCredentialResolver{Resolvers: resolvers}
}

func (r *ChainCredentialResolver) Resolve(providerName string) (Credential, error) {
	if len(r.Resolvers) == 0 {
		return Credential{}, fmt.Errorf("no credential resolver configured for %s", providerName)
	}

	reasons := make([]string, 0, len(r.Resolvers))
	for _, resolver := range r.Resolvers {
		cred, err := resolver.Resolve(providerName)
		if err == nil {
			return cred, nil
		}
		reasons = append(reasons, err.Error())
	}

	return Credential{}, fmt.Errorf("no %s credential available: %s",
		providerName, strings.Join(reasons, "; "))
}
