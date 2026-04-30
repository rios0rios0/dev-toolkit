# Dev-Toolkit

Dev-Toolkit is a Go-based CLI tool (binary: `dev`) that manages Git repositories across multiple providers and bootstraps projects by detecting their language. It consolidates gitforge (Git hosting abstractions) and langforge (language detection) into a single workspace toolkit.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Prerequisites
- **Go 1.26+**: Required for building. Check with `go version`.

### Bootstrap and Build
- `make build` -- builds the `dev` binary in `bin/` directory. Takes ~1 second. NEVER CANCEL.
- `make run` -- builds and runs the tool showing help output.
- `make debug` -- builds without optimizations for debugging.
- `make build-musl` -- builds a fully static binary using musl-gcc (requires musl toolchain).
- `make install` -- builds and copies binary to `~/.local/bin/dev`.

### Testing
- `make lint` -- lint via external pipelines repo.
- `make test` -- test via external pipelines repo.
- `make sast` -- SAST security suite via external pipelines repo.
- `go fmt ./...` -- format all Go code. Always run before committing.
- `go vet ./...` -- static analysis. Always run before committing.

### Running the Application
- Build first: `make build`
- Basic usage: `./bin/dev --help`
- Clone repos: `./bin/dev repo clone mine ~/Development/github.com/rios0rios0`
- Sync repos: `./bin/dev repo sync ~/Development/github.com/rios0rios0`
- Fork sync: `./bin/dev repo fork-sync ~/Development/github.com/rios0rios0`
- Mirror to Codeberg: `./bin/dev repo mirror mine ~/Development/github.com/rios0rios0`
- Project commands: `./bin/dev project {info,use,start,build,lint,test,sast,stop} .`
- Docker: `./bin/dev docker {ips,reset}`
- System: `./bin/dev system {cleanup,clear-history,clear-logs,top5size}`
- Self-update: `./bin/dev self-update`

## Validation

### ALWAYS Test These Scenarios After Changes
1. **Build validation**: `make build` should complete in ~1 second without errors.
2. **Help commands**: Test `./bin/dev --help` and help for any modified subcommand.
3. **Clone dry-run**: `./bin/dev repo clone mine --dry-run ~/Development/github.com/rios0rios0`
4. **Sync**: `./bin/dev repo sync ~/Development/github.com/rios0rios0`
5. **Project info**: `./bin/dev project info .` (verifies language detection)

## Project Structure

### Key Files and Directories
- `cmd/dev-toolkit/main.go` -- All CLI wiring (Cobra commands, dependency construction, update check)
- `internal/repo/` -- Repository operations: clone, sync, fork-sync, prune, mirror, failover, restore
- `internal/project/` -- Language-aware commands: start, build, lint, test, sast, stop, use, info
- `internal/docker/` -- Docker management: container IPs, environment reset
- `internal/system/` -- System utilities: cleanup, clear-history, clear-logs, top5size
- `internal/testutil/` -- Test doubles (stubs) and builders for all interfaces
- `install.sh` -- Generic installer for GitHub releases
- `Makefile` -- Build targets and development commands

### Key Design Patterns
- **Mapper pattern**: All provider detection uses maps (no switch/case), including Codeberg
- **Parallel execution**: Goroutines with semaphore channel for controlled concurrency
- **SSH preflight**: Verifies SSH connectivity before batch cloning
- **WIP branches**: Preserves dirty state during sync via temporary commits
- **Dependency injection**: All business logic accepts interfaces for testability
- **SAST orchestration**: Per-tool failure isolation with embedded default configs
- **Platform gating**: System commands conditionally registered via `runtime.GOOS`
- **Automatic update check**: On startup via cliforge (skipped for `version`, `self-update`, dev builds)

### Authentication
| Provider | Environment Variable |
|----------|---------------------|
| GitHub | `GH_TOKEN` |
| Azure DevOps | `AZURE_DEVOPS_EXT_PAT` |
| GitLab | `GITLAB_TOKEN` |

### Key Commands Reference
```bash
# Build (fast, ~1 second)
make build

# Test basic functionality
make run

# Format and validate code
go fmt ./...
go vet ./...

# Test clone dry-run
./bin/dev repo clone mine --dry-run ~/Development/github.com/rios0rios0

# Test sync
./bin/dev repo sync ~/Development/github.com/rios0rios0

# Test project detection
./bin/dev project info .

# Test SAST suite
./bin/dev project sast .
```
