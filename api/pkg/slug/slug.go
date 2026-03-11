// Package slug provides URL-safe slug generation from names.
package slug

import (
	"regexp"
	"strings"
)

var (
	// nonAlphanumeric matches anything that is not a letter, digit, or hyphen.
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]+`)
	// multipleHyphens matches consecutive hyphens.
	multipleHyphens = regexp.MustCompile(`-{2,}`)
)

// Generate creates a URL-safe slug from the given name.
// It lowercases the input, replaces non-alphanumeric characters with hyphens,
// collapses multiple hyphens, and trims leading/trailing hyphens.
func Generate(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = multipleHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
