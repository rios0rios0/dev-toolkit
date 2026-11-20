<h1 align="center">Dev-Toolkit</h1>
<p align="center">
    <a href="https://github.com/rios0rios0/dev-toolkit/releases/latest">
        <img src="https://img.shields.io/github/release/rios0rios0/dev-toolkit.svg?style=for-the-badge&logo=github" alt="Latest Release"/></a>
    <a href="https://github.com/rios0rios0/dev-toolkit/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/rios0rios0/dev-toolkit.svg?style=for-the-badge&logo=github" alt="License"/></a>
    <a href="https://github.com/rios0rios0/dev-toolkit/actions/workflows/default.yaml">
        <img src="https://img.shields.io/github/actions/workflow/status/rios0rios0/dev-toolkit/default.yaml?branch=main&style=for-the-badge&logo=github" alt="Build Status"/></a>
    <a href="https://www.bestpractices.dev/projects/12033">
        <img src="https://img.shields.io/cii/level/12033?style=for-the-badge&logo=opensourceinitiative" alt="OpenSSF Best Practices"/></a>
</p>

Dev-Toolkit is a developer workspace toolkit that manages Git repositories across multiple providers and bootstraps projects by detecting their language. It consolidates [gitforge](https://github.com/rios0rios0/gitforge) and [langforge](https://github.com/rios0rios0/langforge) into a single CLI.

## Features

- **Repository Cloning**: discovers repos from GitHub, Azure DevOps, or GitLab and clones missing ones via SSH in parallel
- **Repository Syncing**: fetches all repos under a directory and rebases each one's default branch -- detected per repository, never assumed to be `main` -- preserving uncommitted work via WIP branches
- **Gist Cloning & Syncing**: discovers GitHub gists for a user, clones missing ones via SSH using a description-derived slug, and syncs them with the same WIP-aware workflow as repos
- **Fork Syncing**: detects forked repos via provider API, syncs with upstream parent, and handles conflicts by creating reference branches
- **Branch Pruning**: deletes local branches merged into each repo's detected default branch
- **Worktree Cleanup**: finds linked Git worktrees that outlived their purpose -- stale registrations, worktrees outside the root, merged branches, deleted upstreams -- and clears them through Git itself (`git worktree prune` for stale registrations, `git worktree remove` for the rest), protecting anything locked, detached, dirty, or unpushed
- **Docker Management**: lists container IPs and resets the Docker environment (stop, prune)
- **System Cleanup**: reclaims disk space by clearing caches across Go, Node, Python, Gradle, JetBrains, Terra, and SDKMAN, pruning obsolete CLI-agent binary versions, and wiping transient state -- credentials, SDKs, and shell history are preserved
- **Multi-Provider Support**: automatic provider detection from directory path, authenticating through an exported token or, failing that, an already authenticated `gh` or `az` CLI

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/rios0rios0/dev-toolkit/main/install.sh | sh
```

Or build from source:

```bash
go install github.com/rios0rios0/dev-toolkit/cmd/dev-toolkit@latest
```

Download pre-built binaries from the [releases page](https://github.com/rios0rios0/dev-toolkit/releases).

## Usage

```bash
# Clone all repos for a GitHub user/org
dev repo clone <ssh-alias> [root-dir]
dev repo clone mine ~/Development/github.com/rios0rios0
dev repo clone my-org ~/Development/dev.azure.com/my-org
dev repo clone mine --dry-run  # preview without cloning
dev repo clone mine --prune-worktrees  # also clean up disposable linked worktrees

# Sync all repos under a directory
dev repo sync [root-dir]
dev repo sync ~/Development/github.com/rios0rios0

# Sync forked repos with their upstream parent
dev repo fork-sync [root-dir]
dev repo fork-sync ~/Development/github.com/rios0rios0
dev repo fork-sync --dry-run     # preview without syncing

# Delete local merged branches
dev repo prune [root-dir]
dev repo prune ~/Development/github.com/rios0rios0 --dry-run

# Inspect and clean up linked worktrees (run before "repo prune": a branch checked
# out in a worktree cannot be deleted by "git branch -d")
dev repo worktree list [root-dir]
dev repo worktree prune [root-dir]           # confirm each removal interactively
dev repo worktree prune ~/Development/github.com/rios0rios0 --dry-run
dev repo worktree prune ~/Development/github.com/rios0rios0 --yes  # no prompts

# Clone all GitHub gists for a user (slug derived from description; falls back to gist ID)
dev gist clone <ssh-alias> [root-dir]
dev gist clone mine ~/Development/gist.github.com/rios0rios0
dev gist clone mine ~/Development/gist.github.com/rios0rios0 --dry-run

# Sync all gists under a directory
dev gist sync [root-dir]
dev gist sync ~/Development/gist.github.com/rios0rios0

# Docker environment management
dev docker ips                  # list container IP addresses
dev docker reset                # stop all containers, prune everything
dev docker reset --dry-run      # preview without executing
```

### Default Branch Detection

`repo sync`, `repo prune`, `repo fork-sync`, `repo worktree`, and `gist sync` act on each
repository's own default branch -- `main`, `master`, `develop`, or any other name -- and never
assume one. The first source that answers wins:

1. `origin/HEAD` as recorded by `git clone`, but only while the branch it names still exists:
   after the remote renames its default (say `master` to `main`), git keeps pointing
   `origin/HEAD` at the deleted branch.
2. The remote itself (`git ls-remote --symref origin HEAD`), when the recorded pointer is missing
   or names a deleted branch.
3. `main`, then `master`, among the remote-tracking branches, then among the local branches --
   for an unreachable remote or a repository without one.

When none answers, the repository is reported as `FAIL (default branch not found for remote
origin)` and left untouched. Make `origin` reachable, or record the branch yourself with
`git remote set-head origin <branch>`.

`repo sync` fetches before touching anything, so a network failure leaves the repository as it
was, then runs `git remote set-head origin --auto`: git records `origin/HEAD` only when cloning,
and no fetch updates it. A renamed or switched default is therefore followed -- created from
`origin` if it was never checked out locally -- and `repo prune` and `repo worktree`, which do
not fetch, see it from then on. `repo fork-sync` reads the upstream parent's default branch the
same way, without the local-branch fallback.

### Authentication

Provider credentials come from [gitforge](https://github.com/rios0rios0/gitforge)'s shared
credential chain, in order: a token environment variable if you exported one, otherwise the
provider's own CLI if it is installed and authenticated. If you have already run `gh auth login`,
cloning your repositories needs nothing else -- there is no second token to export.

| Provider | Environment Variables | CLI Fallback | Authenticate With |
|----------|----------------------|--------------|-------------------|
| GitHub | `GITHUB_TOKEN` or `GH_TOKEN` | `gh auth token --hostname github.com` | `gh auth login --hostname github.com` |
| Azure DevOps | `AZURE_DEVOPS_EXT_PAT` or `SYSTEM_ACCESSTOKEN` | `az account get-access-token` | `az login` |
| GitLab | `GITLAB_TOKEN` or `GL_TOKEN` (`gitlab.com` only) | -- | -- |
| Codeberg | `CODEBERG_TOKEN` | -- | -- |

The variables are read in the order listed, so an exported `GITHUB_TOKEN` wins over `GH_TOKEN` when
both are set, and any of them wins over the CLI, so an explicitly exported token still overrides
whatever the CLI happens to be logged in as. `gh` is asked for its `github.com` token even when
`GH_HOST` names a GitHub Enterprise server, because every GitHub API call goes to `api.github.com`.
The CLI is found on `PATH` and run by its absolute path, with your home directory as its working
directory rather than the repository you are working in. GitLab has no CLI fallback: `glab` has no
`auth token` command, so export `GITLAB_TOKEN` or `GL_TOKEN`. Every GitLab API call goes to
`gitlab.com`, so either is declined while a variable that names a GitLab server names another one:
glab's `GITLAB_HOST`, `GITLAB_URI`, `GL_HOST`, `GITLAB_API_HOST` and `GITLAB_REPO`, semantic-release's
`GL_URL` and `GITLAB_URL`, Lerna's `GL_API_URL`, Terraform's `GITLAB_BASE_URL`, and a GitLab CI job's
`CI_SERVER_URL`, `CI_SERVER_HOST`, `CI_SERVER_FQDN` and `CI_API_V4_URL`. A token exported for a
self-managed GitLab beside one of them is not sent to `gitlab.com`. The check is best effort: a tool
can also name its server in its own configuration file, which no variable shows. Codeberg has no widely installed official CLI,
so it stays on the environment variable alone.

For Azure DevOps the `az` fallback requests a Microsoft Entra access token scoped to the Azure
DevOps resource (`499b84ac-1321-427f-aa17-267ca6975798`). Repository cloning discovers the
organization's tenant from public Azure DevOps metadata and passes `--tenant` to `az`, so the
default Azure subscription can belong to a different tenant. No token export or default-account
change is required, and the Azure DevOps CLI extension is not required for token retrieval.

```bash
az login --tenant <organization-tenant-id> --allow-no-subscriptions
dev repo clone work ~/Development/dev.azure.com/example-org --dry-run
dev repo clone work ~/Development/dev.azure.com/example-org
dev repo sync ~/Development/dev.azure.com/example-org
```

An existing `az login` session in that tenant is sufficient. If it expires, the error suggests
the tenant-specific login command. `az devops login` stores a PAT separately and does not create
an `az login` session; use `az login` for this fallback, or export `AZURE_DEVOPS_EXT_PAT` explicitly.
Tenant discovery is skipped when `AZURE_DEVOPS_EXT_PAT` or `SYSTEM_ACCESSTOKEN` is exported.
Organizations without an Entra tenant keep using the default Azure CLI account. See Microsoft's [Azure CLI token documentation](https://learn.microsoft.com/en-us/azure/devops/cli/entra-tokens)
for details about Entra authentication.

When nothing can authenticate, the error names every option rather than only the first one that
failed:

```
no GitHub credential available: GITHUB_TOKEN and GH_TOKEN are not set; the gh CLI is not installed
```

These credentials are only used for the provider APIs -- discovering repositories, gists, and fork
parents. Cloning and syncing go over SSH, so SSH keys are still required (see below).

`repo fork-sync` looks fork parents up on GitHub only. In another provider's directory it stops with
`fork resolution not supported` before asking for any credential, so an Azure DevOps directory never
runs `az` just to be told that; on GitHub, one credential serves both the discovery and the parent
lookups.

### SSH Aliases

The clone command uses SSH config aliases (e.g., `github.com-mine`). Configure these in `~/.ssh/config`:

```
Host github.com-mine
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_personal
```

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

See [LICENSE](LICENSE) file for details.
