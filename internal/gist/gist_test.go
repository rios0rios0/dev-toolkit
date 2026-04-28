package gist_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/gist"
)

func TestSlug(t *testing.T) {
	t.Parallel()

	t.Run("should derive slug from a single-line description", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc123", Description: "My Cool Snippet"}

		// when
		slug := gist.Slug(g)

		// then
		assert.Equal(t, "my-cool-snippet", slug)
	})

	t.Run("should fall back to gist ID when description is empty", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc123", Description: ""}

		// when
		slug := gist.Slug(g)

		// then
		assert.Equal(t, "abc123", slug)
	})

	t.Run("should fall back to gist ID when description sanitizes to empty", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc123", Description: "***"}

		// when
		slug := gist.Slug(g)

		// then
		assert.Equal(t, "abc123", slug)
	})

	t.Run("should use only the first non-empty line as a summary", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "id", Description: "\n\nFirst summary line\nSecond line that should be ignored"}

		// when
		slug := gist.Slug(g)

		// then
		assert.Equal(t, "first-summary-line", slug)
	})

	t.Run("should collapse runs of non-alphanumeric characters into single hyphens", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "id", Description: "Hello, World!! -- 123"}

		// when
		slug := gist.Slug(g)

		// then
		assert.Equal(t, "hello-world-123", slug)
	})

	t.Run("should truncate slugs longer than the maximum length", func(t *testing.T) {
		t.Parallel()
		// given
		long := "a-very-long-description-that-easily-exceeds-the-sixty-character-cap-for-slugs"
		g := gist.Gist{ID: "id", Description: long}

		// when
		slug := gist.Slug(g)

		// then
		assert.LessOrEqual(t, len(slug), 60)
		assert.False(t, slug[len(slug)-1] == '-', "should not end with a hyphen")
	})
}

func TestKey(t *testing.T) {
	t.Parallel()

	t.Run("should join the owner and slug with a slash", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc", Owner: "alice", Description: "Snippet"}

		// when
		key := gist.Key(g)

		// then
		assert.Equal(t, "alice/snippet", key)
	})

	t.Run("should fall back to the ID when there is no description", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc", Owner: "alice"}

		// when
		key := gist.Key(g)

		// then
		assert.Equal(t, "alice/abc", key)
	})
}

func TestSSHCloneURL(t *testing.T) {
	t.Parallel()

	t.Run("should build SSH URL without an alias", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc"}

		// when
		url := gist.SSHCloneURL(g, "")

		// then
		assert.Equal(t, "git@gist.github.com:abc.git", url)
	})

	t.Run("should append the alias suffix to the host", func(t *testing.T) {
		t.Parallel()
		// given
		g := gist.Gist{ID: "abc"}

		// when
		url := gist.SSHCloneURL(g, "mine")

		// then
		assert.Equal(t, "git@gist.github.com-mine:abc.git", url)
	})
}

func TestDetectOwner(t *testing.T) {
	t.Parallel()

	t.Run("should extract owner from a path containing gist.github.com/<owner>", func(t *testing.T) {
		t.Parallel()
		// given
		path := "/home/user/Development/gist.github.com/alice"

		// when
		owner, err := gist.DetectOwner(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, "alice", owner)
	})

	t.Run("should ignore trailing path segments below the owner", func(t *testing.T) {
		t.Parallel()
		// given
		path := "/dev/gist.github.com/bob/some-gist"

		// when
		owner, err := gist.DetectOwner(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, "bob", owner)
	})

	t.Run("should return an error when the path does not contain gist.github.com", func(t *testing.T) {
		t.Parallel()
		// given
		path := "/home/user/Development/github.com/alice"

		// when
		_, err := gist.DetectOwner(path)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not detect")
	})
}
