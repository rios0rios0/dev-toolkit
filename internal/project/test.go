package project

import "fmt"

// testCommandMap maps language identifiers to their default test commands.
var testCommandMap = map[string][]string{
	"go":         {"go test -tags unit ./..."},
	"node":       {"npm test"},
	"python":     {"pdm run pytest"},
	"java":       {"./gradlew test"},
	"csharp":     {"dotnet test"},
	"terraform":  {"terraform plan"},
}

// TestCommandsForLanguage returns the default test commands for a given language.
func TestCommandsForLanguage(language string) []string {
	cmds, ok := testCommandMap[language]
	if !ok {
		return nil
	}
	result := make([]string, len(cmds))
	copy(result, cmds)
	return result
}

// RunTest detects the project language and runs its test commands in order.
func RunTest(cfg Config) error {
	repoPath, err := resolveRepoPath(cfg.RepoPath)
	if err != nil {
		return err
	}

	info, err := cfg.Detector.Detect(repoPath)
	if err != nil {
		return err
	}

	cmds := info.TestCommands
	if len(cmds) == 0 {
		cmds = TestCommandsForLanguage(info.Language)
	}
	if len(cmds) == 0 {
		return fmt.Errorf("no test commands available for %s", info.Language)
	}

	logf(cfg.Output, "detected %s project", info.SDKName)
	for _, cmd := range cmds {
		logf(cfg.Output, "running: %s", cmd)
		if cmdErr := cfg.Runner.RunInteractive(repoPath, cmd); cmdErr != nil {
			return fmt.Errorf("test command %q failed: %w", cmd, cmdErr)
		}
	}
	logf(cfg.Output, "tests completed successfully")
	return nil
}
