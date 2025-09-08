package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takl/takl/internal/context"
)

// testResolveIssueID is a version of resolveIssueID that takes a project context for testing
func testResolveIssueID(issueID string, ctx *context.ProjectContext) (string, error) {
	issuesDir := ctx.GetIssuesDir()
	if issuesDir == "" {
		return "", fmt.Errorf("no issues directory found - not in a TAKL project")
	}

	target := strings.ToLower(issueID)
	var match string

	err := filepath.WalkDir(issuesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Check if filename contains the issue ID
		if strings.Contains(strings.ToLower(filepath.Base(path)), target) {
			match = path
			return io.EOF // Stop walking on first match
		}
		return nil
	})

	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to search for issue files: %w", err)
	}

	if match == "" {
		return "", fmt.Errorf("no issue file found for ID %s (try a full path or run 'takl register --list' to check your context)", issueID)
	}

	return match, nil
}

func TestResolveIssueID(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := os.MkdirTemp("", "takl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test directory structure
	issuesDir := filepath.Join(tempDir, ".takl", "issues")
	bugDir := filepath.Join(issuesDir, "bug")
	featureDir := filepath.Join(issuesDir, "feature")

	if err := os.MkdirAll(bugDir, 0755); err != nil {
		t.Fatalf("Failed to create bug dir: %v", err)
	}
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature dir: %v", err)
	}

	// Create test files
	testFiles := []string{
		filepath.Join(bugDir, "iss-123-test-bug.md"),
		filepath.Join(featureDir, "iss-456-new-feature.md"),
		filepath.Join(bugDir, "iss-abc-another-bug.md"),
	}

	for _, file := range testFiles {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create a mock project context
	ctx, err := context.DetectContext(tempDir)
	if err != nil {
		t.Fatalf("Failed to detect context: %v", err)
	}

	tests := []struct {
		name     string
		issueID  string
		expected string
		wantErr  bool
	}{
		{
			name:     "find exact ID match",
			issueID:  "ISS-123",
			expected: testFiles[0],
			wantErr:  false,
		},
		{
			name:     "find case insensitive match",
			issueID:  "iss-456",
			expected: testFiles[1],
			wantErr:  false,
		},
		{
			name:     "find alpha ID",
			issueID:  "ISS-ABC",
			expected: testFiles[2],
			wantErr:  false,
		},
		{
			name:     "ID not found",
			issueID:  "ISS-999",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := testResolveIssueID(tt.issueID, ctx)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
