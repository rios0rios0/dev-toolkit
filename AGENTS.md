# AGENTS.md

This file provides guidance to AI coding agents (Claude Code, Codex, Cursor, GitHub Copilot) when working with code in this repository.

## Project Overview

Dev-Toolkit is a Go CLI tool (binary: `dev`) that manages Git repositories across multiple providers, provides language-aware project commands, and includes system housekeeping utilities. Built with Cobra (CLI), logrus (logging), gitforge (multi-provider Git operations), langforge (language detection), and cliforge (self-update).

## Build and Development Commands

```bash
make build          # Build binary to bin/dev (~1 second), always run after changes
make run            # go run ./cmd/dev-toolkit (shows help)
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
dev repo worktree list ~/Development/github.com/rios0rios0      # list linked worktrees + classification
dev repo worktree prune ~/Development/github.com/rios0rios0     # remove disposable worktrees (prompts)
dev repo worktree prune ~/Development/github.com/rios0rios0 --dry-run  # preview without removing
dev repo worktree prune ~/Development/github.com/rios0rios0 --yes     # remove without prompting
dev repo clone mine --prune-worktrees                           # clone, then clean up worktrees
dev repo mirror mine ~/Development/github.com/rios0rios0        # create Codeberg pull mirrors
dev repo failover ~/Development/github.com/rios0rios0           # switch repos to Codeberg primary
dev repo restore ~/Development/github.com/rios0rios0            # restore GitHub as primary remote
dev gist clone mine ~/Development/gist.github.com/rios0rios0    # clone missing GitHub gists (slug from description)
dev gist clone mine ~/Development/gist.github.com/rios0rios0 --dry-run # preview without cloning
dev gist sync ~/Development/gist.github.com/rios0rios0          # sync all gists
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

`repo sync`, `repo failover`, `repo restore` and `gist sync` have no `--dry-run`: they rebase, make WIP commits, push, or rename remotes in every repository under the root. Never run them as smoke tests against a real workspace.

## Architecture

```
cmd/dev-toolkit/
  main.go                    -- all CLI wiring (Cobra commands, dependency construction, update check)
internal/
  repo/
    git.go                   -- GitRunner interface + DefaultGitRunner (exec.Command wrapper around the resolved git binary)
    provider.go              -- provider detection, maps, registry (includes Codeberg)
    credential.go            -- maps provider names to gitforge ServiceTypes and asks gitforge's credential chain for a token
    logger.go                -- NewLogger factory for isolated logrus instances
    scanner.go               -- local repo scanning (flat/nested/recursive)
    clone.go                 -- clone orchestration with dependency injection
    sync.go                  -- sync orchestration with dependency injection
    default_branch.go        -- default-branch detection: ordered rules, ErrDefaultBranchNotFound instead of a guess
    fork_resolver.go         -- ForkResolver interface + factory (mapper pattern)
    fork_resolver_github.go  -- GitHub implementation using go-github API
    fork_sync.go             -- fork-sync orchestration: detect forks, add upstream, rebase, handle conflicts
    prune.go                 -- prune merged branches with dry-run support
    worktree.go              -- linked worktree parsing, rule-based classification, and cleanup
    mirror.go                -- create Codeberg pull mirrors via Forgejo migration API; ResolveMirrorTarget asks for the Codeberg credential alone (Codeberg pulls each repo from its public address, so no source provider credential)
    failover.go              -- switch repos from GitHub to Codeberg as primary remote
    restore.go               -- restore GitHub as primary remote after failover
    *_test.go                -- BDD tests
  project/
    runner.go                -- CommandRunner interface + DefaultCommandRunner (passthrough I/O via the resolved sh -c)
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
    runner.go                -- Runner interface + DefaultRunner (exec.Command wrapper around the resolved docker binary)
    ips.go                   -- RunIPs: list container IP addresses
    reset.go                 -- RunReset: stop all containers, prune resources with dry-run support
    *_test.go                -- BDD tests
  gist/
    gist.go                  -- Gist entity, slug derivation, owner detection, SSH URL builder
    provider.go              -- Provider interface + GitHubProvider (go-github gist API)
    scanner.go               -- ScanLocalGists: walk <root>/<owner>/<slug> for .git directories
    clone.go                 -- RunClone: discover, diff, parallel clone (reuses repo.GitRunner/SSHPreflight/NotReturnedFor)
    sync.go                  -- RunSync: delegates to repo.SyncSingleRepo for each gist
    *_test.go                -- BDD tests
  system/
    runner.go                -- Runner interface (exec.Command wrapper for system commands)
    platform.go              -- platform detection (IsAndroid, IsLinux) via runtime.GOOS
    cleanup.go               -- reclaim disk space: Go/Node/Python/Gradle/JetBrains/Terra/SDKMAN caches
    clear_history.go         -- remove shell history files and leftover dotfiles
    clear_logs.go            -- remove log files older than 5 days (Linux only)
    top5size.go              -- show top 5 largest items in a directory
    *_test.go                -- BDD tests
  executable/
    executable.go            -- Resolver + Resolve: locate external tools (git, docker, ssh, sh) on PATH once per process
    *_test.go                -- BDD tests
  testutil/
    doubles/                 -- hand-written <Interface>Stub test doubles, one file per interface (credential doubles come from gitforge's test/doubles)
    builders/                -- RepositoryBuilder
```

### Key Design Decisions

- **Provider detection**: Mapper pattern from directory path segments (`github.com` -> `"github"`, `dev.azure.com` -> `"azuredevops"`, `codeberg.org` -> `"codeberg"`)
- **Parallel operations**: Goroutines with semaphore channel (`runtime.NumCPU()` workers)
- **External tools**: `git`, `docker`, `ssh` and `sh` are deliberately taken from the user's PATH (they live somewhere different on every supported platform, so absolute paths are not an option). `executable.Resolve` looks each one up once per process in a single audited place and refuses relative results; commands then run through that absolute path, never by bare name (SonarCloud `go:S4036`)
- **Git operations**: Uses `exec.Command` on the resolved `git` binary behind `GitRunner` interface for testability
- **SSH cloning**: Sets `GIT_SSH_COMMAND` with `StrictHostKeyChecking=accept-new` and `BatchMode=yes`
- **Language detection**: Uses langforge's `LanguageRegistry` behind `LanguageDetector` interface for testability
- **Docker operations**: Uses `exec.Command` on the resolved `docker` binary behind `docker.Runner` interface for testability
- **System operations**: Uses `exec.Command(...)` behind `system.Runner` and `FileSystem` interfaces; platform-gated via `runtime.GOOS`
- **Credential resolution**: gitforge's shared chain (`credentialInfra.NewDefaultCredentialResolver()` from `gitforge/v4/pkg/credentials`), not a bare `os.Getenv` and not a dev-toolkit copy. It reads the provider's token variables first (`GH_TOKEN` then `GITHUB_TOKEN` -- declined when `GITHUB_SERVER_URL`, `GITHUB_API_URL` or a `*.ghe.com` `GH_HOST` names another GitHub server -- `AZURE_DEVOPS_EXT_PAT` then `SYSTEM_ACCESSTOKEN`, `GITLAB_TOKEN` then `GL_TOKEN` -- declined, best effort, when glab's `GITLAB_HOST`/`GITLAB_URI`/`GL_HOST`/`GITLAB_API_HOST`/`GITLAB_REPO`, semantic-release's `GL_URL`/`GITLAB_URL`, Lerna's `GL_API_URL`, Terraform's `GITLAB_BASE_URL` or GitLab CI's `CI_SERVER_*`/`CI_API_V4_URL` names a server other than gitlab.com -- and `CODEBERG_TOKEN`), so an explicit token still overrides the CLI, then asks the provider's own CLI (`gh auth token --hostname github.com`, `az account get-access-token` scoped to the Azure DevOps resource), run by absolute path from the home directory; on total failure the error matches `ErrNoCredential` and names *every* reason rather than only the last. GitLab (`glab` has no `auth token` command) and Codeberg have no CLI and stay env-only. `credential.go` keeps only what is dev-toolkit's: `providerServiceTypeMap` (the inverse of the registry's `ServiceTypeToProviderName`), `resolveCredential`, which refuses an unknown provider before any source is consulted, and `newCredentialResolver`, the factory every function without a resolver argument asks for the default chain. That factory is the package's one test seam: only `credential_internal_test.go` replaces it, which is why that test is deliberately not parallel. Every consumer (`ResolveProviderForOwner`, `ResolveForkSync`, `ResolveMirrorTarget`, `gist.ResolveProvider`) goes through the chain, and their `...With` variants take gitforge's `CredentialResolver` for tests. Each also returns the credential's `Source` ("GH_TOKEN environment variable", "gh CLI") beside the client it built, never the token: main.go passes it on as `CredentialSource` (`TargetCredentialSource` for mirror), and the workflow logs it at Info in the `credential` field of its start line, so the user can tell which identity discovery ran as. gitforge itself only logs it at debug
- **Default branch detection**: Never a hardcoded `main`. `DetectDefaultBranch` walks an ordered rule table (`repositoryDefaultBranchRules`): the recorded `refs/remotes/origin/HEAD` only while the branch it names still exists (after a remote renames its default and a pruning fetch removes the old branch, `git symbolic-ref` keeps printing the dead name), then the remote's own answer (`git ls-remote --symref origin HEAD`), then `main`/`master` among the remote-tracking and finally the local branches. A rule never answers with a branch it has not seen exist; when none answers, `ErrDefaultBranchNotFound` makes the repository fail rather than be synced or pruned against a guess. `DetectRemoteDefaultBranch` (used for a fork's `upstream`) drops the local-branch rule, since local branches say nothing about another remote. `SyncSingleRepo` fetches first, then runs `git remote set-head origin --auto` because git records `origin/HEAD` only at clone time and no fetch updates it; that refresh is what keeps `prune` and `worktree`, which do not fetch, accurate afterwards. A default branch never checked out locally is created with `git checkout --track origin/<branch>`, since a bare checkout is ambiguous when another remote has the same branch
- **Fork sync**: Uses `ForkResolver` interface to query provider APIs for parent repo info; auto-adds `upstream` remote. `ResolveForkSync` refuses a provider without fork resolution (all but GitHub) before any credential source is consulted -- in an Azure DevOps directory, asking first would run `az` in the default tenant and could fail with a sign-in error instead -- and builds the discovery provider and the fork resolver from one credential
- **Extra local repositories**: `PromptDeleteExtra` asks before `repo clone` deletes a local repository the provider did not return, and says it "was not returned for the credential from the GH_TOKEN environment variable", naming the credential's `Source` -- never "not on remote" -- because a credential that cannot see a repository (a narrower token, another account) leaves it out exactly as a deletion does, and the deletion takes unpushed work with it. Only `y` deletes; any other answer, Enter or end of input keeps it, and `--dry-run` or a non-terminal stdin never asks. That explanation is one phrase, `NotReturnedFor(credentialSource)` in `clone.go`, shared by every message about a checkout the remote did not return -- the question, the `--dry-run` and non-interactive warnings, and `gist clone`'s warning -- so none of them says "not on remote" again
- **Azure CLI tenant selection**: Repository cloning uses `ResolveProviderForOwner`, which sends the owner as the credential request's `Organization` (`credential_internal_test.go` pins it for every registered provider through the `newCredentialResolver` seam); gitforge then reads the organization's tenant from public Azure DevOps metadata and passes `--tenant` to `az`. Other providers ignore the organization. Explicit PATs skip discovery; organizations without an Entra tenant retain the default Azure CLI account. Git sync uses the existing Git/SSH credentials.
- **Worktree detection**: A linked worktree stores `.git` as a *file*, so the `.git`-directory scanners (`ScanFlatRepos`, `ScanNestedRepos`, `FindAllRepos`) never see one. This is intentional: worktrees are extra checkouts of repos that exist on the remote, so they must stay out of the clone remote-vs-local diff. `worktree.go` reads them from `git worktree list --porcelain` instead
- **Worktree classification**: Ordered rule tables (`worktreeGuardRules`, `worktreeRemovalRules`) instead of branching; guards (locked, detached, dirty, unpushed) are always evaluated before removal rules, so preserving work wins over cleaning up. Removal always goes through `git worktree remove`/`git worktree prune`, never `os.RemoveAll`, to keep the parent repo's metadata consistent
- **SAST orchestration**: Runs each tool (CodeQL, Semgrep, Trivy, Hadolint, Gitleaks) with per-tool failure isolation and embedded default configs
- **Dependency injection**: Business logic accepts interfaces (`GitRunner`, `ForgeProvider`, `ForkResolver`, gitforge's `CredentialResolver`, `gist.Provider`, `SASTTool`, `LanguageDetector`, `CommandRunner`, `ConfigReader`, `docker.Runner`, `system.Runner`, `FileSystem`, `io.Writer`) for testability
- **Project dependencies**: `.dev.yaml` declares relative paths to dependent projects; resolved via DFS topological sort with cycle detection
- **Automatic update check**: On startup (via cliforge), skipped for `version`, `self-update`, shell completion (`completion` and cobra's hidden `__complete`, which run on every shell start and TAB press), and local dev builds
- **No switch/case**: All dispatch uses mapper pattern (maps of string -> value/function)

### Dependencies

- **gitforge** -- Multi-provider Git hosting abstractions (GitHub, Azure DevOps, GitLab, Codeberg) and the provider credential chain
- **go-github** -- GitHub API client (used by `ForkResolver` to get fork parent info)
- **langforge** -- Language detection, version management, and runtime information (Go, Dart, Node, Python, Java, C#, Ruby, Terraform)
- **cliforge** -- Self-update mechanism and automatic version check on startup

<!-- chlog:start -->
## Changelog (chlog) — MANDATORY

If the repository you are working in uses chlog (a `.chlog.yaml` or `.chlog.yml`
config file, or a `.changes/` directory, exists at the project root), the
following is binding and ALWAYS applies: whenever you make ANY change, you MUST
create a changelog fragment as part of the same change — automatically, without
being asked, before committing.

- Do NOT edit CHANGELOG.md directly; it is generated from fragments.
- Create the fragment with:
  `chlog new --kind <Kind> --body '<past-tense description>'`
- Write an apostrophe inside the single-quoted body as `'\''`.
- Valid kinds: Added, Changed, Deprecated, Removed, Fixed, Security
- Choose the kind that best matches the change (e.g., new feature → Added,
  bug fix → Fixed, behavior change → Changed, removal → Removed, security fix → Security).
- If the change is backward-INCOMPATIBLE with the public API (a breaking
  change), you MUST add the `--breaking` flag:
  `chlog new --kind <Kind> --breaking --body '<past-tense description>'`.
  This is the ONLY thing that triggers a major version bump — the kind alone
  never does (per SemVer, major = incompatible change). When unsure whether a
  change breaks compatibility, ask the user instead of guessing.
- Fragments are YAML files in `.changes/unreleased/`; stage them with your commit.
- `chlog check` fails the build when a fragment is missing — never skip it.
<!-- chlog:end -->
