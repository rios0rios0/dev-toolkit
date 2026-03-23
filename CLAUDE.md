# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevForge is a Go CLI tool (binary: `dev`) that manages Git repositories across multiple providers and bootstraps projects by detecting their language. Built with Cobra (CLI), logrus (logging), gitforge (multi-provider Git operations), and langforge (language detection).

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
```

## Architecture

All application code lives in `cmd/devforge/` (3 files):

- **main.go** -- Cobra CLI setup with `repo` command group containing `clone` and `sync`
- **clone.go** -- Repository cloning: provider detection from path, gitforge discovery, parallel SSH cloning with goroutines, interactive extra repo deletion
- **sync.go** -- Repository syncing: parallel fetch/rebase with WIP branch preservation for dirty trees

### Key Design Decisions

- **Provider detection**: Mapper pattern from directory path segments (`github.com` → `"github"`, `dev.azure.com` → `"azuredevops"`)
- **Parallel operations**: Goroutines with semaphore channel (`runtime.NumCPU()` workers)
- **Git operations**: Uses `exec.Command("git", ...)` for clone/fetch/rebase (go-git lacks rebase support)
- **SSH cloning**: Sets `GIT_SSH_COMMAND` with `StrictHostKeyChecking=accept-new` and `BatchMode=yes`
- **Private repos**: gitforge uses authenticated `/user/repos?affiliation=owner` endpoint when the token owner matches
- **No switch/case**: All dispatch uses mapper pattern (maps of string → value/function)

### Dependencies

- **gitforge** -- Multi-provider Git hosting abstractions (GitHub, Azure DevOps, GitLab)
- **langforge** -- Language detection and ecosystem abstractions (planned for Phase 2)

### Local Development with Replace Directives

During development, `go.mod` uses `replace` directives to point to local gitforge/langforge checkouts. Remove these before releasing.
