# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevForge is a Go CLI tool (binary: `dev`) that manages Git repositories across multiple providers and provides language-aware project commands. Built with Cobra (CLI), logrus (logging), gitforge (multi-provider Git operations), and langforge (language detection).

## Build and Development Commands

```bash
make build          # Build binary to bin/dev (~1 second), always run after changes
make run            # go run ./cmd/devforge (shows help)
make debug          # Build with debug symbols (-N -l)
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
dev repo prune ~/Development/github.com/rios0rios0              # delete merged branches
dev repo prune ~/Development/github.com/rios0rios0 --dry-run    # preview without deleting
dev project info .                                              # detect language and show info
dev project use .                                               # print version switch commands (eval it)
dev project start .                                             # run project start command (with .dev.yaml deps)
dev project build .                                             # run project build commands
dev project stop .                                              # run project stop command (with .dev.yaml deps)
dev docker ips                                                  # list container IP addresses
dev docker reset                                                # stop all, prune everything
dev docker reset --dry-run                                      # preview without executing
```

## Architecture

```
cmd/devforge/
  main.go                    -- thin CLI wiring (Cobra commands, dependency construction)
internal/
  repo/
    git.go                   -- GitRunner interface + DefaultGitRunner (exec.Command wrapper)
    provider.go              -- provider detection, maps, registry, Logf helper
    scanner.go               -- local repo scanning (flat/nested/recursive)
    clone.go                 -- clone orchestration with dependency injection
    sync.go                  -- sync orchestration with dependency injection
    prune.go                 -- prune merged branches with dry-run support
    *_test.go                -- BDD tests (81%+ coverage)
  project/
    runner.go                -- CommandRunner interface + DefaultCommandRunner (passthrough I/O)
    detect.go                -- LanguageDetector interface + DefaultLanguageDetector (wraps langforge)
    devconfig.go             -- ConfigReader interface + FileConfigReader (.dev.yaml) + dependency graph resolver
    orchestrate.go           -- RunStartWithDeps/RunStopWithDeps: recursive dependency start/stop
    use.go                   -- RunUse: detect language, print version switch commands to stdout
    start.go                 -- RunStart: detect language, run start command
    build.go                 -- RunBuild: detect language, run build commands
    stop.go                  -- RunStop: detect language, run stop command
    info.go                  -- RunInfo: detect language, display metadata + dependencies
    *_test.go                -- BDD tests
  docker/
    runner.go                -- Runner interface + DefaultRunner (exec.Command wrapper for docker)
    ips.go                   -- RunIPs: list container IP addresses
    reset.go                 -- RunReset: stop all containers, prune resources with dry-run support
    *_test.go                -- BDD tests
  testutil/
    doubles/                 -- GitRunnerStub, ForgeProviderStub, CommandRunnerStub, LanguageDetectorStub, LanguageDetectorMultiStub, ConfigReaderStub, DockerRunnerStub
    builders/                -- RepositoryBuilder
```

### Key Design Decisions

- **Provider detection**: Mapper pattern from directory path segments (`github.com` -> `"github"`, `dev.azure.com` -> `"azuredevops"`)
- **Parallel operations**: Goroutines with semaphore channel (`runtime.NumCPU()` workers)
- **Git operations**: Uses `exec.Command("git", ...)` behind `GitRunner` interface for testability
- **SSH cloning**: Sets `GIT_SSH_COMMAND` with `StrictHostKeyChecking=accept-new` and `BatchMode=yes`
- **Language detection**: Uses langforge's `LanguageRegistry` behind `LanguageDetector` interface for testability
- **Docker operations**: Uses `exec.Command("docker", ...)` behind `docker.Runner` interface for testability
- **Dependency injection**: Business logic accepts interfaces (`GitRunner`, `ForgeProvider`, `LanguageDetector`, `CommandRunner`, `ConfigReader`, `docker.Runner`, `io.Writer`) for testability
- **Project dependencies**: `.dev.yaml` declares relative paths to dependent projects; resolved via DFS topological sort with cycle detection
- **No switch/case**: All dispatch uses mapper pattern (maps of string -> value/function)

### Dependencies

- **gitforge** -- Multi-provider Git hosting abstractions (GitHub, Azure DevOps, GitLab)
- **langforge** -- Language detection, version management, and runtime information (Go, Node, Python, Java, C#, Terraform)
