# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

When a new release is proposed:

1. Create a new branch `bump/x.x.x` (this isn't a long-lived branch!!!);
2. The Unreleased section on `CHANGELOG.md` gets a version number and date;
3. Open a Pull Request with the bump version changes targeting the `main` branch;
4. When the Pull Request is merged, a new Git tag must be created using [GitHub environment](https://github.com/rios0rios0/versainit/tags).

Releases to productive environments should run from a tagged version.
Exceptions are acceptable depending on the circumstances (critical bug fixes that can be cherry-picked, etc.).

## [Unreleased]

### Added

- added `dev project lint` command -- detects language and runs lint commands via `langforge`
- added `dev project test` command -- detects language and runs test commands via `langforge`
- added `dev project sast` command -- runs the full SAST suite (CodeQL, Semgrep, Trivy, Hadolint, Gitleaks) with per-tool failure isolation and embedded default configs
- added `dev repo fork-sync` command -- syncs forked repositories with their upstream parent, auto-detects forks via provider API, configures `upstream` remote automatically, and creates a `fork-sync/upstream` branch on conflict for manual resolution
- added `ForkResolver` interface and GitHub implementation for resolving fork parent repository info via the GitHub API
- added `ForkResolverStub` test double and `WithFork` builder method for fork-related testing

### Changed

- changed `DefaultCommandRunner.RunInteractive` to use `sh -c` for proper shell operator support (redirection, pipes)
- changed the Go module dependencies to their latest versions

## [0.4.0] - 2026-03-31

### Added

- added `dev system clear-history` command -- removes shell history files and leftover dotfiles
- added `dev system clear-logs` command -- removes log files older than 5 days from `/var/log` (Linux only)
- added `dev system top5size` command -- shows the top 5 largest items in a directory
- added `Runner` and `FileSystem` interfaces in `internal/system/` with test doubles for testability
- added platform detection (`IsAndroid`, `IsLinux`) via `runtime.GOOS` for conditional command registration

### Changed

- changed the Go module dependencies to their latest versions

## [0.3.0] - 2026-03-30

### Added

- added `dev repo failover` command — switches all repos from GitHub to Codeberg as primary remote
- added `dev repo mirror` command — creates Codeberg pull mirrors for all repositories via the Forgejo migration API
- added `dev repo restore` command — restores GitHub as primary remote after a failover
- added `NewLogger` factory in `internal/repo/logger.go` for creating isolated `logrus` instances
- added Codeberg provider support (`codeberg.org` path detection, `CODEBERG_TOKEN`, SSH host mapping)
- added structured `logrus` logging to the `repo` package (`clone`, `sync`, `prune`) with per-repo real-time visibility during parallel operations

### Changed

- changed `DiscoverRepos`, `ParallelClone`, `HandleExtraRepos`, and `PromptDeleteExtra` to use structured `logrus` logging
- changed `gitforge` dependency to latest main branch commit with Codeberg provider support
- changed `main.go` `logrus` import alias from `log` to `logger`
- changed `PreflightFunc` signature to accept `logger.FieldLogger` instead of `io.Writer`
- changed clone workflow to log each repository's URL and target directory in real-time during parallel clone
- changed the Go module dependencies to their latest versions

### Fixed

- fixed SSH preflight to detect successful authentication from stderr output instead of relying on exit codes, resolving false failures with Azure DevOps (which returns exit code 255 on success)

### Removed

- removed `Logf` helper function from `provider.go` in favor of structured `logrus` logging
- removed unused `SSHFailCode` constant

## [0.2.0] - 2026-03-25

### Added

- added `.dev.yaml` dependency orchestration — `dev project start` and `dev project stop` recursively resolve and start/stop project dependencies in topological order
- added `dev docker ips` command — lists IP addresses of all running Docker containers
- added `dev docker reset` command — stops all containers and prunes containers, volumes, networks, and build cache
- added `dev project build` command — detects language and runs build commands via `langforge`
- added `dev project info` command — detects language and displays SDK, version, and available commands
- added `dev project info` dependency display — shows declared dependencies when `.dev.yaml` exists
- added `dev project start` command — detects language and runs start command via `langforge`
- added `dev project stop` command — detects language and runs stop command via `langforge`
- added `dev project use` command — detects required SDK version and prints shell commands to install/switch versions
- added `dev repo clone` command — discovers repos from Git providers, clones missing via SSH with parallel workers
- added `dev repo prune` command — deletes local branches merged into the default branch across repos
- added `dev repo sync` command — syncs all repos under a directory with fetch/rebase and WIP branch preservation
- added `gitforge` integration for multi-provider repository discovery (GitHub, Azure DevOps, GitLab)
- added `langforge` integration for automatic language detection (Go, Node, Python, Java, C#, Terraform)
- added comprehensive test suite with 81%+ coverage for all business logic
- added SSH alias clone URL support via gitforge
- added test infrastructure with `GitRunner` stub, `ForgeProvider` stub, `DockerRunner` stub, and `Repository` builder

### Changed

- changed `cmd/devforge/` to a thin CLI wiring layer delegating to `internal/repo/`
- changed architecture to extract business logic into `internal/repo/` with dependency injection for testability
- changed Go module path from `github.com/rios0rios0/versainit` to `github.com/rios0rios0/devforge`
- changed project name from `versainit` to `devforge` (binary: `dev`)

### Removed

- removed old `versainit` CLI code (`actions.go`, `clone.go`, `config.go`, `versainit.yaml`)

## [0.1.2] - 2026-03-19

### Changed

- changed the Go module dependencies to their latest versions
- changed version injection to use `ldflags` at build time instead of a hardcoded constant

## [0.1.1] - 2026-03-13

### Changed

- created missing boilerplate and documentation with `CLAUDE.md` file

## [0.1.0] - 2026-03-12

### Changed

- changed the Go version to `1.26.1` and updated all module dependencies
