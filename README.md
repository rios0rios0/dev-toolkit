<h1 align="center">DevForge</h1>
<p align="center">
    <a href="https://github.com/rios0rios0/devforge/releases/latest">
        <img src="https://img.shields.io/github/release/rios0rios0/devforge.svg?style=for-the-badge&logo=github" alt="Latest Release"/></a>
    <a href="https://github.com/rios0rios0/devforge/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/rios0rios0/devforge.svg?style=for-the-badge&logo=github" alt="License"/></a>
    <a href="https://github.com/rios0rios0/devforge/actions/workflows/default.yaml">
        <img src="https://img.shields.io/github/actions/workflow/status/rios0rios0/devforge/default.yaml?branch=main&style=for-the-badge&logo=github" alt="Build Status"/></a>
    <a href="https://www.bestpractices.dev/projects/12033">
        <img src="https://img.shields.io/cii/level/12033?style=for-the-badge&logo=opensourceinitiative" alt="OpenSSF Best Practices"/></a>
</p>

DevForge is a developer workspace toolkit that manages Git repositories across multiple providers and bootstraps projects by detecting their language. It consolidates [gitforge](https://github.com/rios0rios0/gitforge) and [langforge](https://github.com/rios0rios0/langforge) into a single CLI.

## Features

- **Repository Cloning**: discovers repos from GitHub, Azure DevOps, or GitLab and clones missing ones via SSH in parallel
- **Repository Syncing**: fetches and rebases all repos under a directory, preserving uncommitted work via WIP branches
- **Gist Cloning & Syncing**: discovers GitHub gists for a user, clones missing ones via SSH using a description-derived slug, and syncs them with the same WIP-aware workflow as repos
- **Fork Syncing**: detects forked repos via provider API, syncs with upstream parent, and handles conflicts by creating reference branches
- **Branch Pruning**: deletes local branches merged into the default branch across all repos
- **Docker Management**: lists container IPs and resets the Docker environment (stop, prune)
- **System Cleanup**: reclaims disk space by clearing caches across Go, Node, Python, Gradle, JetBrains, Terra, and SDKMAN, pruning obsolete CLI-agent binary versions, and wiping transient state -- credentials, SDKs, and shell history are preserved
- **Multi-Provider Support**: automatic provider detection from directory path with per-provider auth tokens

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/rios0rios0/devforge/main/install.sh | sh
```

Or build from source:

```bash
go install github.com/rios0rios0/devforge/cmd/devforge@latest
```

Download pre-built binaries from the [releases page](https://github.com/rios0rios0/devforge/releases).

## Usage

```bash
# Clone all repos for a GitHub user/org
dev repo clone <ssh-alias> [root-dir]
dev repo clone mine ~/Development/github.com/rios0rios0
dev repo clone my-org ~/Development/dev.azure.com/my-org
dev repo clone mine --dry-run  # preview without cloning

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

### Authentication

Set the appropriate environment variable for your provider:

| Provider | Environment Variable |
|----------|---------------------|
| GitHub | `GH_TOKEN` |
| Azure DevOps | `AZURE_DEVOPS_EXT_PAT` |
| GitLab | `GITLAB_TOKEN` |

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
