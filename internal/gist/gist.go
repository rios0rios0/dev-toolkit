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

// Key returns the local directory key for a gist: "<owner>/<slug>".
func Key(g Gist) string {
	return g.Owner + "/" + Slug(g)
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
