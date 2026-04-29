package gist

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	gistHost      = "gist.github.com"
	maxSlugLength = 60
)

// Gist represents a single GitHub gist used for clone/sync orchestration.
type Gist struct {
	ID          string
	Owner       string
	Description string
}

// Key returns the natural on-disk key for a gist (just the slug). The owner
// is implied by the root directory the command is operating on, so it is not
// repeated in the key. When two gists share the same slug, callers should use
// AssignKeys to disambiguate them.
func Key(g Gist) string {
	return Slug(g)
}

// Slug derives a URL/path-safe slug from the gist description, falling back
// to the gist ID when no description is present or the description sanitizes
// to an empty string.
func Slug(g Gist) string {
	if s := slugify(g.Description); s != "" {
		return s
	}
	return g.ID
}

// AssignKeys returns a deterministic mapping from gist ID to a unique on-disk
// key. Most gists keep their natural slug. When two or more gists share the
// same slug (e.g., identical first-line descriptions), each colliding entry
// gets a "<slug>-<short-id>" suffix to keep the path unique.
func AssignKeys(gists []Gist) map[string]string {
	const shortIDLen = 7
	counts := make(map[string]int, len(gists))
	for _, g := range gists {
		counts[Slug(g)]++
	}
	keys := make(map[string]string, len(gists))
	for _, g := range gists {
		slug := Slug(g)
		if counts[slug] > 1 {
			id := g.ID
			if len(id) > shortIDLen {
				id = id[:shortIDLen]
			}
			keys[g.ID] = slug + "-" + id
		} else {
			keys[g.ID] = slug
		}
	}
	return keys
}

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a free-form description into a kebab-case slug.
// It uses the first non-empty line as the summary, lowercases it, replaces
// any run of non-alphanumeric characters with a single hyphen, trims hyphens
// from both ends, and truncates to maxSlugLength characters.
func slugify(description string) string {
	summary := firstNonEmptyLine(description)
	if summary == "" {
		return ""
	}
	lowered := strings.ToLower(summary)
	replaced := slugSeparators.ReplaceAllString(lowered, "-")
	trimmed := strings.Trim(replaced, "-")
	if len(trimmed) > maxSlugLength {
		trimmed = strings.TrimRight(trimmed[:maxSlugLength], "-")
	}
	return trimmed
}

func firstNonEmptyLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// SSHCloneURL builds the SSH clone URL for a gist, optionally appending an
// SSH config alias suffix to the host (matching the repo command convention).
func SSHCloneURL(g Gist, alias string) string {
	host := gistHost
	if alias != "" {
		host = fmt.Sprintf("%s-%s", host, alias)
	}
	return fmt.Sprintf("git@%s:%s.git", host, g.ID)
}

// Host returns the canonical gist host name.
func Host() string {
	return gistHost
}

// DetectOwner extracts the gist owner from a path containing
// ".../gist.github.com/<owner>" (with or without a trailing slug).
func DetectOwner(rootDir string) (string, error) {
	const segment = "/" + gistHost + "/"
	_, after, found := strings.Cut(rootDir, segment)
	if !found {
		return "", fmt.Errorf("could not detect gist owner from path: %s", rootDir)
	}
	owner, _, _ := strings.Cut(after, "/")
	if owner == "" {
		return "", fmt.Errorf("could not extract owner from path: %s", rootDir)
	}
	return owner, nil
}
