# DevForge

DevForge is a Go-based CLI tool (binary: `dev`) that manages Git repositories across multiple providers and bootstraps projects by detecting their language. It consolidates gitforge (Git hosting abstractions) and langforge (language detection) into a single workspace toolkit.

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

## Validation

### ALWAYS Test These Scenarios After Changes
1. **Build validation**: `make build` should complete in ~1 second without errors.
2. **Help commands**: Test `./bin/dev --help`, `./bin/dev repo clone --help`, `./bin/dev repo sync --help`.
3. **Clone dry-run**: `./bin/dev repo clone mine --dry-run ~/Development/github.com/rios0rios0`
4. **Sync**: `./bin/dev repo sync ~/Development/github.com/rios0rios0`

## Project Structure

### Key Files and Directories
- `cmd/devforge/` -- Main application code
  - `main.go` -- CLI command setup with `repo` command group
  - `clone.go` -- Repository cloning with gitforge integration and parallel SSH workers
  - `sync.go` -- Repository syncing with fetch/rebase and WIP branch preservation
- `install.sh` -- Generic installer for GitHub releases
- `CONTRIBUTING.md` -- Development workflow and prerequisites
- `Makefile` -- Build targets and development commands
- `.github/workflows/default.yaml` -- CI pipeline (uses external rios0rios0/pipelines)

### Key Design Patterns
- **Mapper pattern**: All provider detection uses maps (no switch/case)
- **Parallel execution**: Goroutines with semaphore channel for controlled concurrency
- **SSH preflight**: Verifies SSH connectivity before batch cloning
- **WIP branches**: Preserves dirty state during sync via temporary commits

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
```
