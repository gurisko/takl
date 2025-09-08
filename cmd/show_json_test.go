package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/takl/takl/sdk"
)

func TestJSONOutput(t *testing.T) {
	// Mock issue data for testing
	testIssue := &sdk.Issue{
		ID:       "ISS-ABC123",
		Type:     "feature",
		Title:    "Test JSON Feature",
		Status:   "open",
		Priority: "medium",
		Assignee: "test@example.com",
		Labels:   []string{"enhancement", "json"},
		Created:  time.Date(2023, 9, 2, 12, 0, 0, 0, time.UTC),
		Updated:  time.Date(2023, 9, 2, 12, 30, 0, 0, time.UTC),
		FilePath: "/path/to/issue.md",
		Content:  "This is test content for the JSON feature",
	}

	// Test basic JSON output (without verbose)
	t.Run("basic_json_output", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		verbose = false // Ensure verbose is false
		err := outputIssueJSON(testIssue)
		if err != nil {
			t.Fatalf("outputIssueJSON failed: %v", err)
		}

		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		// Verify JSON structure
		if !strings.Contains(output, `"id": "ISS-ABC123"`) {
			t.Error("JSON output missing ID field")
		}
		if !strings.Contains(output, `"type": "feature"`) {
			t.Error("JSON output missing type field")
		}
		if !strings.Contains(output, `"title": "Test JSON Feature"`) {
			t.Error("JSON output missing title field")
		}
		if !strings.Contains(output, `"created": "2023-09-02T12:00:00Z"`) {
			t.Error("JSON output missing or incorrect created field")
		}
		if !strings.Contains(output, `"updated": "2023-09-02T12:30:00Z"`) {
			t.Error("JSON output missing or incorrect updated field")
		}

		// Verify content is NOT included when not verbose
		if strings.Contains(output, `"content"`) {
			t.Error("JSON output should not include content when not verbose")
		}

		t.Logf("✅ Basic JSON output: %s", strings.TrimSpace(output))
	})

	// Test verbose JSON output (with content)
	t.Run("verbose_json_output", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		verbose = true // Enable verbose
		err := outputIssueJSON(testIssue)
		if err != nil {
			t.Fatalf("outputIssueJSON failed: %v", err)
		}

		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		// Verify content IS included when verbose
		if !strings.Contains(output, `"content": "This is test content for the JSON feature"`) {
			t.Error("JSON output should include content when verbose")
		}

		t.Logf("✅ Verbose JSON output includes content")
	})

	// Test stable contract compliance
	t.Run("stable_contract", func(t *testing.T) {
		// Test with minimal issue data
		minimalIssue := &sdk.Issue{
			ID:       "ISS-123",
			Type:     "bug",
			Title:    "Login fails",
			Status:   "in_progress",
			Priority: "high",
			Created:  time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC),
			FilePath: ".takl/issues/bug/ISS-123.md",
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		verbose = false
		err := outputIssueJSON(minimalIssue)
		if err != nil {
			t.Fatalf("outputIssueJSON failed: %v", err)
		}

		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		// Verify all required contract fields are present
		contractFields := []string{
			`"id": "ISS-123"`,
			`"type": "bug"`,
			`"title": "Login fails"`,
			`"status": "in_progress"`,
			`"priority": "high"`,
			`"assignee": ""`, // Should be empty string, not omitted
			`"labels": []`,   // Should be empty array, not omitted
			`"created": "2025-09-01T12:00:00Z"`,
			`"updated": ""`, // Should be empty string for unset dates
			`"file": ".takl/issues/bug/ISS-123.md"`,
		}

		for _, field := range contractFields {
			if !strings.Contains(output, field) {
				t.Errorf("JSON output missing required contract field: %s", field)
			}
		}

		t.Logf("✅ Stable contract compliance verified")
		t.Logf("Contract output: %s", strings.TrimSpace(output))
	})
}
