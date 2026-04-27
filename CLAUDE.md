# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevForge is a Go CLI tool (binary: `dev`) that manages Git repositories across multiple providers, provides language-aware project commands, and includes system housekeeping utilities. Built with Cobra (CLI), logrus (logging), gitforge (multi-provider Git operations), langforge (language detection), and cliforge (self-update).

## Build and Development Commands

```bash
make build          # Build binary to bin/dev (~1 second), always run after changes
make run            # go run ./cmd/devforge (shows help)
make debug          # Build with debug symbols (-N -l)
make build-musl     # Fully static binary via musl-gcc (requires musl toolchain)
make install        # Build and copy to ~/.local/bin/dev
make lint           # Lint via external pipelines repo
make test           # Test via external pipelines repo
make sast           # SAST security suite via external pipelines repo
```

The Makefile includes shared targets from `$(HOME)/Development/github.com/rios0rios0/pipelines`. Never call tool binaries directly -- always use `make` targets.

## Usage

```bash
dev repo clone mine ~/Development/github.com/rios0rios0        # clone missing repos
dev repo clone mine --dry-run                                   # preview without cloning
dev repo sync ~/Development/github.com/rios0rios0               # sync all repos
dev repo fork-sync ~/Development/github.com/rios0rios0          # sync forks with upstream
dev repo fork-sync ~/Development/github.com/rios0rios0 --dry-run # preview fork sync
dev repo prune ~/Development/github.com/rios0rios0              # delete merged branches
dev repo prune ~/Development/github.com/rios0rios0 --dry-run    # preview without deleting
dev repo mirror mine ~/Development/github.com/rios0rios0        # create Codeberg pull mirrors
dev repo failover ~/Development/github.com/rios0rios0           # switch repos to Codeberg primary
dev repo restore ~/Development/github.com/rios0rios0            # restore GitHub as primary remote
dev project info .                                              # detect language and show info
dev project use .                                               # print version switch commands (eval it)
dev project start .                                             # run project start command (with .dev.yaml deps)
dev project build .                                             # run project build commands
dev project lint .                                              # run lint commands via langforge
dev project test .                                              # run test commands via langforge
dev project sast .                                              # run SAST suite (CodeQL, Semgrep, Trivy, Hadolint, Gitleaks)
dev project stop .                                              # run project stop command (with .dev.yaml deps)
dev docker ips                                                  # list container IP addresses
dev docker reset                                                # stop all, prune everything
dev docker reset --dry-run                                      # preview without executing
dev system cleanup                                              # reclaim disk space (caches, transient state)
dev system clear-history                                        # remove shell history and leftover dotfiles
dev system clear-logs                                           # remove log files older than 5 days (Linux only)
dev system top5size ~/some/dir                                  # show top 5 largest items in a directory
dev self-update                                                 # download and install latest release
dev version                                                     # print current version to stdout
```

## Architecture

```
cmd/devforge/
  main.go                    -- all CLI wiring (Cobra commands, dependency construction, update check)
internal/
  repo/
    git.go                   -- GitRunner interface + DefaultGitRunner (exec.Command wrapper)
    provider.go              -- provider detection, maps, registry (includes Codeberg)
    logger.go                -- NewLogger factory for isolated logrus instances
    scanner.go               -- local repo scanning (flat/nested/recursive)
    clone.go                 -- clone orchestration with dependency injection
    sync.go                  -- sync orchestration with dependency injection
    fork_resolver.go         -- ForkResolver interface + factory (mapper pattern)
    fork_resolver_github.go  -- GitHub implementation using go-github API
    fork_sync.go             -- fork-sync orchestration: detect forks, add upstream, rebase, handle conflicts
    prune.go                 -- prune merged branches with dry-run support
    mirror.go                -- create Codeberg pull mirrors via Forgejo migration API
    failover.go              -- switch repos from GitHub to Codeberg as primary remote
    restore.go               -- restore GitHub as primary remote after failover
    *_test.go                -- BDD tests
  project/
    runner.go                -- CommandRunner interface + DefaultCommandRunner (passthrough I/O via sh -c)
    detect.go                -- LanguageDetector interface + DefaultLanguageDetector (wraps langforge)
    devconfig.go             -- ConfigReader interface + FileConfigReader (.dev.yaml) + dependency graph resolver
    orchestrate.go           -- RunStartWithDeps/RunStopWithDeps: recursive dependency start/stop
    use.go                   -- RunUse: detect language, print version switch commands to stdout
    start.go                 -- RunStart: detect language, run start command
    build.go                 -- RunBuild: detect language, run build commands
    lint.go                  -- RunLint: detect language, run lint commands
    test.go                  -- RunTest: detect language, run test commands
    sast.go                  -- RunSAST: orchestrate SAST tools with per-tool failure isolation
    sast_codeql.go           -- CodeQL integration
    sast_semgrep.go          -- Semgrep integration
    sast_trivy.go            -- Trivy integration
    sast_hadolint.go         -- Hadolint integration
    sast_gitleaks.go         -- Gitleaks integration
    sast_defaults/           -- embedded default configs for each SAST tool
    stop.go                  -- RunStop: detect language, run stop command
    info.go                  -- RunInfo: detect language, display metadata + dependencies
    *_test.go                -- BDD tests
  docker/
    runner.go                -- Runner interface + DefaultRunner (exec.Command wrapper for docker)
    ips.go                   -- RunIPs: list container IP addresses
    reset.go                 -- RunReset: stop all containers, prune resources with dry-run support
    *_test.go                -- BDD tests
  system/
    runner.go                -- Runner interface (exec.Command wrapper for system commands)
    platform.go              -- platform detection (IsAndroid, IsLinux) via runtime.GOOS
    cleanup.go               -- reclaim disk space: Go/Node/Python/Gradle/JetBrains/Terra/SDKMAN caches
    clear_history.go         -- remove shell history files and leftover dotfiles
    clear_logs.go            -- remove log files older than 5 days (Linux only)
    top5size.go              -- show top 5 largest items in a directory
    *_test.go                -- BDD tests
  testutil/
    doubles/                 -- GitRunnerStub, ForgeProviderStub, ForkResolverStub, CommandRunnerStub, LanguageDetectorStub, LanguageDetectorMultiStub, ConfigReaderStub, DockerRunnerStub, FileSystemStub, MirrorProviderStub, SystemRunnerStub
    builders/                -- RepositoryBuilder
```

### Key Design Decisions

- **Provider detection**: Mapper pattern from directory path segments (`github.com` -> `"github"`, `dev.azure.com` -> `"azuredevops"`, `codeberg.org` -> `"codeberg"`)
- **Parallel operations**: Goroutines with semaphore channel (`runtime.NumCPU()` workers)
- **Git operations**: Uses `exec.Command("git", ...)` behind `GitRunner` interface for testability
- **SSH cloning**: Sets `GIT_SSH_COMMAND` with `StrictHostKeyChecking=accept-new` and `BatchMode=yes`
- **Language detection**: Uses langforge's `LanguageRegistry` behind `LanguageDetector` interface for testability
- **Docker operations**: Uses `exec.Command("docker", ...)` behind `docker.Runner` interface for testability
- **System operations**: Uses `exec.Command(...)` behind `system.Runner` and `FileSystem` interfaces; platform-gated via `runtime.GOOS`
- **Fork sync**: Uses `ForkResolver` interface to query provider APIs for parent repo info; auto-adds `upstream` remote
- **SAST orchestration**: Runs each tool (CodeQL, Semgrep, Trivy, Hadolint, Gitleaks) with per-tool failure isolation and embedded default configs
- **Dependency injection**: Business logic accepts interfaces (`GitRunner`, `ForgeProvider`, `ForkResolver`, `LanguageDetector`, `CommandRunner`, `ConfigReader`, `docker.Runner`, `system.Runner`, `FileSystem`, `io.Writer`) for testability
- **Project dependencies**: `.dev.yaml` declares relative paths to dependent projects; resolved via DFS topological sort with cycle detection
- **Automatic update check**: On startup (via cliforge), skipped for `version`, `self-update`, and local dev builds
- **No switch/case**: All dispatch uses mapper pattern (maps of string -> value/function)

### Dependencies

- **gitforge** -- Multi-provider Git hosting abstractions (GitHub, Azure DevOps, GitLab, Codeberg)
- **go-github** -- GitHub API client (used by `ForkResolver` to get fork parent info)
- **langforge** -- Language detection, version management, and runtime information (Go, Node, Python, Java, C#, Terraform)
- **cliforge** -- Self-update mechanism and automatic version check on startup
