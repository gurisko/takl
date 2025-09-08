package cmd

import (
	"regexp"
	"testing"
)

func TestIssueIDPattern(t *testing.T) {
	issueIDPattern := regexp.MustCompile(`^ISS-[A-Za-z0-9]+$`)

	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{"numeric ID", "ISS-123", true},
		{"hex ID", "ISS-ABC123", true},
		{"lowercase", "ISS-abc123", true},
		{"mixed case", "ISS-AbC123", true},
		{"just letters", "ISS-HELLO", true},
		{"just numbers", "ISS-999", true},
		{"with dash in suffix", "ISS-ABC-123", false}, // Should not match
		{"empty suffix", "ISS-", false},
		{"no prefix", "123", false},
		{"wrong prefix", "BUG-123", false},
		{"special chars", "ISS-@#$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := issueIDPattern.MatchString(tt.id)
			if result != tt.expected {
				t.Errorf("Pattern match for %s: expected %v, got %v", tt.id, tt.expected, result)
			}
		})
	}
}
