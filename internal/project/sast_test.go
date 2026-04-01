package project_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/project"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

// sastToolStub is a test double for project.SASTTool.
type sastToolStub struct {
	name string
	err  error
}

func (s *sastToolStub) Name() string { return s.name }
func (s *sastToolStub) Run(_ string, _ project.CommandRunner, _ io.Writer) error {
	return s.err
}

func TestRunSAST(t *testing.T) {
	t.Parallel()

	t.Run("should run all SAST tools when language detected successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "go",
			SDKName:  "Go",
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: t.TempDir(),
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunSAST(cfg)

		// then
		// With stubbed dependencies, RunSAST should succeed and log the detected language.
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "detected Go project")
	})

	t.Run("should return error when language detection fails", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(nil).WithError(errors.New("no language"))
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunSAST(cfg)

		// then
		require.Error(t, err)
	})
}

func TestDefaultSASTTools(t *testing.T) {
	t.Parallel()

	t.Run("should include codeql for go projects", func(t *testing.T) {
		t.Parallel()
		// given / when
		tools := project.DefaultSASTTools("go")

		// then
		names := toolNames(tools)
		assert.Contains(t, names, "semgrep")
		assert.Contains(t, names, "trivy")
		assert.Contains(t, names, "hadolint")
		assert.Contains(t, names, "gitleaks")
		assert.Contains(t, names, "codeql")
	})

	t.Run("should exclude codeql for terraform projects", func(t *testing.T) {
		t.Parallel()
		// given / when
		tools := project.DefaultSASTTools("terraform")

		// then
		names := toolNames(tools)
		assert.Contains(t, names, "semgrep")
		assert.NotContains(t, names, "codeql")
	})

	t.Run("should include all non-codeql tools for unknown language", func(t *testing.T) {
		t.Parallel()
		// given / when
		tools := project.DefaultSASTTools("unknown")

		// then
		names := toolNames(tools)
		assert.Contains(t, names, "semgrep")
		assert.Contains(t, names, "trivy")
		assert.Contains(t, names, "hadolint")
		assert.Contains(t, names, "gitleaks")
		assert.NotContains(t, names, "codeql")
	})
}

func TestSemgrepLanguageMap(t *testing.T) {
	t.Parallel()

	t.Run("should map go to golang for semgrep", func(t *testing.T) {
		t.Parallel()
		// given / when
		tools := project.DefaultSASTTools("go")

		// then
		for _, tool := range tools {
			if tool.Name() == "semgrep" {
				st, ok := tool.(*project.SemgrepTool)
				require.True(t, ok)
				assert.Equal(t, "golang", st.Language)
			}
		}
	})
}

func TestCodeQLLanguageMap(t *testing.T) {
	t.Parallel()

	t.Run("should map node to javascript for codeql", func(t *testing.T) {
		t.Parallel()
		// given / when
		tools := project.DefaultSASTTools("node")

		// then
		for _, tool := range tools {
			if tool.Name() == "codeql" {
				ct, ok := tool.(*project.CodeQLTool)
				require.True(t, ok)
				assert.Equal(t, "javascript", ct.Language)
			}
		}
	})
}

func toolNames(tools []project.SASTTool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name()
	}
	return names
}
