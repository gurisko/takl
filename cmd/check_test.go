package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/validation"
)

func TestCheckValidation(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		fileContent string
		expectError bool
		errorCount  int
		warnCount   int
	}{
		{
			name:     "valid_issue",
			fileName: "iss-001-login-button.md",
			fileContent: `---
id: ISS-001
type: bug
title: Login button not working
status: open
priority: high
assignee: dev@example.com
labels: ["ui", "critical"]
created: 2025-09-01T15:30:00Z
updated: 2025-09-01T15:30:00Z
---

# Login button not working

The login button doesn't respond to clicks.`,
			expectError: false,
			errorCount:  0,
			warnCount:   0,
		},
		{
			name:     "missing_required_fields",
			fileName: "test-missing.md",
			fileContent: `---
type: bug
priority: high
---

# Some issue without ID`,
			expectError: true,
			errorCount:  1, // centralized validator stops at first error
			warnCount:   0,
		},
		{
			name:     "invalid_type",
			fileName: "iss-002-test.md",
			fileContent: `---
id: ISS-002
type: unknown
title: Test issue
status: open
---

# Test`,
			expectError: true,
			errorCount:  1, // invalid type
			warnCount:   0,
		},
		{
			name:     "invalid_priority",
			fileName: "iss-003-test.md",
			fileContent: `---
id: ISS-003
type: bug
title: Test issue
status: open
priority: super-urgent
---

# Test`,
			expectError: true,
			errorCount:  1, // centralized validator treats invalid priority as error
			warnCount:   0,
		},
		{
			name:     "invalid_id_format",
			fileName: "bad-name-test.md", // This won't match WRONG-123-
			fileContent: `---
id: WRONG-123
type: bug
title: Test issue
status: open
---

# Test`,
			expectError: true,
			errorCount:  1, // invalid ID format
			warnCount:   1, // file name mismatch
		},
		{
			name:     "invalid_assignee_email",
			fileName: "iss-004-test.md",
			fileContent: `---
id: ISS-004
type: bug
title: Test issue
status: open
priority: medium
assignee: not-an-email
---

# Test`,
			expectError: false,
			errorCount:  0,
			warnCount:   1, // invalid email
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			issuesDir := filepath.Join(tmpDir, ".takl", "issues", "bug")
			if err := os.MkdirAll(issuesDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create test file with specified name
			testFile := filepath.Join(issuesDir, tt.fileName)
			if err := os.WriteFile(testFile, []byte(tt.fileContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Validate the project
			result := validateProject(tmpDir, "test-proj", "Test Project")

			// Check error count
			if len(result.Errors) != tt.errorCount {
				t.Errorf("Expected %d errors, got %d", tt.errorCount, len(result.Errors))
				for _, err := range result.Errors {
					t.Logf("  Error: %s", err.Message)
				}
			}

			// Check warning count
			if len(result.Warnings) != tt.warnCount {
				t.Errorf("Expected %d warnings, got %d", tt.warnCount, len(result.Warnings))
				for _, warn := range result.Warnings {
					t.Logf("  Warning: %s", warn.Message)
				}
			}

			// Check overall validation result
			hasErrors := len(result.Errors) > 0
			if hasErrors != tt.expectError {
				t.Errorf("Expected error=%v, got %v", tt.expectError, hasErrors)
			}
		})
	}
}

func TestCentralizedValidation(t *testing.T) {
	validator := validation.NewValidator(nil)

	tests := []struct {
		name          string
		issue         *domain.Issue
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_issue",
			issue: &domain.Issue{
				ID:       "ISS-001",
				Type:     "bug",
				Title:    "Valid issue title",
				Status:   "open",
				Priority: "medium",
			},
			expectError: false,
		},
		{
			name: "invalid_type",
			issue: &domain.Issue{
				ID:     "ISS-002",
				Type:   "unknown",
				Title:  "Test issue",
				Status: "open",
			},
			expectError:   true,
			errorContains: "invalid issue type",
		},
		{
			name: "invalid_priority",
			issue: &domain.Issue{
				ID:       "ISS-003",
				Type:     "bug",
				Title:    "Test issue",
				Status:   "open",
				Priority: "super-urgent",
			},
			expectError:   true,
			errorContains: "invalid priority",
		},
		{
			name: "missing_title",
			issue: &domain.Issue{
				ID:     "ISS-004",
				Type:   "bug",
				Status: "open",
			},
			expectError:   true,
			errorContains: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIssue(tt.issue)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Login Button Not Working", "login-button-not-working"},
		{"Add dark mode!", "add-dark-mode"},
		{"Fix bug #123", "fix-bug-123"},
		{"Update API (v2.0)", "update-api-v20"},
		{"Very Long Title That Should Be Truncated After Fifty Characters Total", "very-long-title-that-should-be-truncated-after-fif"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
