package daemon

import (
	"strings"
)

// normalizeStatus converts status strings to their canonical form
// Handles case insensitivity and normalizes all statuses to lowercase
func normalizeStatus(status string) string {
	if status == "" {
		return ""
	}

	// Simply convert to lowercase - no aliases
	return strings.ToLower(status)
}
