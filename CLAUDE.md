# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VersaInit is a Go CLI tool that automatically bootstraps projects by detecting the programming language (via special files like `go.mod`, `pyproject.toml`, `build.gradle` or file extensions) and executing predefined commands. Built with Cobra (CLI), logrus (logging), go-git (cloning), and gopkg.in/yaml.v3 (config).

## Build and Development Commands

```bash
make build          # Build binary to bin/vinit (~1 second), always run after changes
make run            # go run ./cmd/versainit (shows help)
make debug          # Build with debug symbols (-N -l)
make install        # Build and copy to ~/.local/bin/vinit
make lint           # Lint via external pipelines repo
make test           # Test via external pipelines repo
make sast           # SAST security suite via external pipelines repo
```

The Makefile includes shared targets from `$(HOME)/Development/github.com/rios0rios0/pipelines`. Never call tool binaries directly -- always use `make` targets.

## Usage

```bash
./bin/vinit -c configs/versainit.yaml start -p /path/to/project
./bin/vinit -c configs/versainit.yaml build -p /path/to/project
```

## Architecture

All application code lives in `cmd/versainit/` (4 files, ~400 lines total):

- **main.go** -- Cobra CLI setup, registers `start` and `build` subcommands, initializes global config
- **actions.go** -- Language detection (special files first, extensions fallback) and command execution via `/bin/sh -c`
- **config.go** -- YAML config parsing, `GlobalConfig` struct, config merging (local `vinit.yaml` overrides global)
- **clone.go** -- Dependency resolution (searches parent dirs) and shallow git cloning

Language definitions live in `configs/versainit.yaml`, not in code. Adding a new language requires only editing this YAML file.

## Key Design Decisions

- **Detection priority**: Special pattern files (e.g. `go.mod`) take precedence over file extension scanning
- **Config merging**: Local `vinit.yaml` in a project directory overrides the global config passed via `-c`
- **Global config**: Single `Globalconf` variable initialized at startup (appropriate for CLI scope)
- **No tests yet**: `internal/` and `test/` directories are reserved with `.gitkeep` for future use
- **`stop` command**: Defined in config schema but no CLI subcommand is registered for it
