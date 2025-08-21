# VersaInit

VersaInit is a Go-based CLI tool that automatically bootstraps projects by detecting the project language and executing predefined commands. The tool works by analyzing special files (like go.mod, pyproject.toml, build.gradle) and file extensions to determine the project type, then runs configured commands for that language.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Prerequisites
- **Go 1.20+**: Required for building. Check with `go version`.

### Bootstrap and Build
- `make build` -- builds the `vinit` binary in `bin/` directory. Takes ~1 second. NEVER CANCEL.
- `make run` -- builds and runs the tool showing help output.
- `make debug` -- builds without optimizations for debugging.
- `make install` -- builds and copies binary to `/usr/local/bin/vinit` (requires sudo).

### Testing
- No unit tests exist currently. Test manually by creating sample projects.
- `go fmt ./...` -- format all Go code. Always run before committing.
- `go vet ./...` -- static analysis. Always run before committing.
- Manual validation is required - create sample projects and test functionality.

### Running the Application
- Build first: `make build`
- Basic usage: `./bin/vinit --help`
- Test with project: `./bin/vinit -c configs/versainit.yaml start -p /path/to/project`
- Configuration file: `configs/versainit.yaml` defines language commands and detection patterns.

## Validation

### ALWAYS Test These Scenarios After Changes
1. **Build validation**: `make build` should complete in ~1 second without errors.
2. **Help commands**: Test `./bin/vinit --help`, `./bin/vinit start --help`, `./bin/vinit build --help`.
3. **Language detection**: Create sample projects and verify detection works:
   - Go project: Create directory with `go.mod` file, should detect as "go"
   - Python project: Create directory with `pyproject.toml`, should detect as "python"  
   - Java project: Create directory with `build.gradle` or `pom.xml`, should detect as "java"
   - Extension detection: Create directory with `.go`, `.py`, or `.java` files, should detect correctly
4. **Command execution**: Test `start` and `build` commands on sample projects.
5. **Configuration validation**: Ensure `configs/versainit.yaml` is valid YAML and loads without errors.

### Language Detection Priority
- **Special patterns checked first** (go.mod, pyproject.toml, build.gradle, etc.)
- **File extensions checked second** (.go, .py, .java files)
- If both exist, special patterns take priority

### Manual Testing Process
```bash
# Test Go project
cd /tmp && mkdir test-go && cd test-go
echo 'module test' > go.mod
echo 'package main
import "fmt"
func main() { fmt.Println("Hello!") }' > main.go
/path/to/versainit/bin/vinit -c /path/to/versainit/configs/versainit.yaml start -p .
# Should output: "INFO[...] Detected project language: go" and "Hello!"

# Test with extension-based detection
cd /tmp && mkdir test-ext && cd test-ext  
echo 'print("Hello!")' > hello.py
/path/to/versainit/bin/vinit -c /path/to/versainit/configs/versainit.yaml start -p .
# Should output: "INFO[...] Detected project language: python" (may fail due to missing pdm)
```

## Project Structure

### Key Files and Directories
- `cmd/versainit/` -- Main application code
  - `main.go` -- CLI command setup and entry point  
  - `actions.go` -- Language detection and command execution
  - `config.go` -- Configuration file parsing and management
  - `clone.go` -- Dependency management and repository cloning
- `configs/versainit.yaml` -- Default language configuration file
- `Makefile` -- Build targets and development commands
- `.github/workflows/default.yaml` -- CI pipeline (uses external rios0rios0/pipelines)

### Configuration System
- Languages defined in YAML with `start`, `stop`, `build` commands
- Language detection via `special_patterns` (priority) or `extensions` (fallback)
- Global config merged with local `vinit.yaml` files
- Dependencies can be cloned and managed automatically

## Common Tasks

### Adding Support for New Language
1. Edit `configs/versainit.yaml`
2. Add language entry with commands and detection patterns:
   ```yaml
   languages:
     newlang:
       start: "command to start"
       build: "command to build"  
       extensions:
         - "ext1"
         - "ext2"
       special_patterns:
         - "special-file.conf"
   ```
3. Test detection: Create project with special file/extensions
4. Test commands: Verify start/build commands work correctly

### Debugging Language Detection Issues
- Check `configs/versainit.yaml` syntax with `./bin/vinit -c configs/versainit.yaml --help`
- Verify special_patterns files exist in test project
- Check file extensions match configured patterns
- Language detection logic in `cmd/versainit/actions.go:detectLanguage()`

### Repository Structure Output
```
ls -la /
.
..
.editorconfig
.git/
.github/
.gitignore
CHANGELOG.md
LICENSE
Makefile
README.md
cmd/
configs/
go.mod
go.sum
horusec.json
```

### Key Commands Reference
```bash
# Build (fast, ~1 second)
make build

# Test basic functionality  
make run

# Format and validate code
go fmt ./...
go vet ./...

# Test with sample project
mkdir /tmp/test && cd /tmp/test
echo 'module test' > go.mod
/path/to/versainit/bin/vinit -c /path/to/versainit/configs/versainit.yaml start -p .
```

### Configuration File Content
```yaml
languages:
  docker-compose:
    start: "docker-compose -f docker-compose.yaml up -d"
    stop: "docker-compose down"
    special_patterns:
      - "docker-compose.yaml"
  go:
    start: "go run ."
    build: "go build ."
    extensions:
      - "go"
    special_patterns:
      - "go.mod"
      - "go.sum"
  python:
    start: "pdm install && pdm start"
    build: "pdm build"
    extensions:
      - "py"
    special_patterns:
      - "setup.cfg"
      - "setup.py" 
      - "pyproject.toml"
  java:
    start: gradle bootRun
    build: gradle build -x check -x test
    extensions:
      - "java"
    special_patterns:
      - "build.gradle"
      - "pom.xml"
```