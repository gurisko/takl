package daemon

import (
	"testing"
)

func TestStatusNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Case normalization - no aliases
		{"in-progress", "in-progress"},
		{"IN-PROGRESS", "in-progress"}, // case insensitive
		{"In-Progress", "in-progress"}, // mixed case
		{"todo", "todo"},
		{"TODO", "todo"}, // case insensitive
		{"completed", "completed"},
		{"COMPLETED", "completed"}, // case insensitive

		// Canonical forms
		{"in_progress", "in_progress"},
		{"open", "open"},
		{"done", "done"},
		{"backlog", "backlog"},
		{"review", "review"},
		{"doing", "doing"},
		{"DOING", "doing"}, // case insensitive canonical

		// Unknown statuses - should be lowercased
		{"unknown", "unknown"},
		{"UNKNOWN", "unknown"},
		{"Custom_Status", "custom_status"},
		{"SOME-NEW-STATUS", "some-new-status"},

		// Edge cases
		{"", ""}, // empty string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeStatus(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}

	t.Log("✅ Status normalization working correctly")
}
