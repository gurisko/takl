package shared

import (
	"regexp"
	"strings"
)

// Slugify converts a title to a URL-safe slug
func Slugify(title string) string {
	// Convert to lowercase and replace spaces with hyphens
	slug := strings.ToLower(title)

	// Replace non-alphanumeric characters with spaces first
	reg := regexp.MustCompile(`[^a-z0-9\s]+`)
	slug = reg.ReplaceAllString(slug, " ")

	// Replace multiple spaces with single space
	reg = regexp.MustCompile(`\s+`)
	slug = reg.ReplaceAllString(slug, " ")

	// Trim and replace spaces with hyphens
	slug = strings.TrimSpace(slug)
	slug = strings.ReplaceAll(slug, " ", "-")

	return slug
}
