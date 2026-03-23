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

- added `dev repo clone` command — discovers repos from Git providers, clones missing via SSH with parallel workers
- added `dev repo sync` command — syncs all repos under a directory with fetch/rebase and WIP branch preservation
- added gitforge integration for multi-provider repository discovery (GitHub, Azure DevOps, GitLab)
- added SSH alias clone URL support via gitforge
- added comprehensive test suite with 81%+ coverage for all business logic
- added test infrastructure with GitRunner stub, ForgeProvider stub, and Repository builder

### Changed

- changed project name from versainit to devforge (binary: `dev`)
- changed Go module path from `github.com/rios0rios0/versainit` to `github.com/rios0rios0/devforge`
- changed architecture to extract business logic into `internal/repo/` with dependency injection for testability
- changed `cmd/devforge/` to a thin CLI wiring layer delegating to `internal/repo/`

### Removed

- removed old versainit CLI code (actions.go, clone.go, config.go, versainit.yaml)

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
