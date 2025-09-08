package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestHandleShowError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		issueID        string
		expectContains []string
	}{
		{
			name:    "not_found_error",
			err:     errors.New("issue not found"),
			issueID: "ISS-123",
			expectContains: []string{
				`No issue found for "ISS-123"`,
				"takl list --type=bug --status=open",
				"takl list",
				"takl status",
			},
		},
		{
			name:    "case_insensitive_not_found",
			err:     errors.New("Issue Not Found"),
			issueID: "ISS-456",
			expectContains: []string{
				`No issue found for "ISS-456"`,
				"Try:",
			},
		},
		{
			name:    "404_error",
			err:     errors.New("HTTP 404 error"),
			issueID: "ISS-789",
			expectContains: []string{
				`No issue found for "ISS-789"`,
			},
		},
		{
			name:    "project_context_error",
			err:     errors.New("project context not found"),
			issueID: "ISS-ABC",
			expectContains: []string{
				`Failed to get issue "ISS-ABC"`,
				"takl status",
				"takl init",
			},
		},
		{
			name:    "daemon_connection_error",
			err:     errors.New("failed to connect to daemon socket"),
			issueID: "ISS-XYZ",
			expectContains: []string{
				`Failed to get issue "ISS-XYZ"`,
				"takl daemon start",
				"takl status",
			},
		},
		{
			name:    "generic_error",
			err:     errors.New("some other error"),
			issueID: "ISS-999",
			expectContains: []string{
				`Failed to get issue "ISS-999"`,
				"takl list",
				"takl status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleShowError(tt.err, tt.issueID)

			if result == nil {
				t.Error("Expected error, got nil")
				return
			}

			resultStr := result.Error()

			for _, expected := range tt.expectContains {
				if !strings.Contains(resultStr, expected) {
					t.Errorf("Expected error message to contain %q, got: %s", expected, resultStr)
				}
			}

			t.Logf("✅ Error message: %s", resultStr)
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		errStr   string
		expected bool
	}{
		{"issue not found", true},
		{"Issue Not Found", true},
		{"file does not exist", true},
		{"HTTP 404 Not Found", true},
		{"no such issue exists", true},
		{"connection refused", false},
		{"invalid argument", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.errStr, func(t *testing.T) {
			result := isNotFoundError(tt.errStr)
			if result != tt.expected {
				t.Errorf("isNotFoundError(%q) = %v, expected %v", tt.errStr, result, tt.expected)
			}
		})
	}
}
