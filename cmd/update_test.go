package cmd

import (
	"os"
	"testing"

	"github.com/takl/takl/internal/store"
)

func TestUpdateCommand(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo (required for some operations)
	if err := initRealGitRepo(tempDir); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Save and restore original repoPath
	oldRepoPath := repoPath
	defer func() { repoPath = oldRepoPath }()
	repoPath = tempDir

	// Initialize TAKL project
	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("Failed to init TAKL project: %v", err)
	}

	// Create an issue using the store manager
	manager, err := store.NewLegacyManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create issue manager: %v", err)
	}

	opts := store.CreateOptions{
		Priority: "medium",
		Content:  "Test content",
	}

	testIssue, err := manager.Create("bug", "Test issue", opts)
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	t.Run("update_status", func(t *testing.T) {
		// Test updating status
		updateStatus = "in_progress"
		updatePriority = ""
		updateAssignee = ""
		updateLabels = nil
		updateTitle = ""
		updateContent = ""
		updateForce = false

		err := updateCmd.RunE(updateCmd, []string{testIssue.ID})
		if err != nil {
			t.Fatalf("Update command failed: %v", err)
		}

		// Verify the issue was updated
		updated, err := manager.LoadIssue(testIssue.ID)
		if err != nil {
			t.Fatalf("Failed to load updated issue: %v", err)
		}

		if updated.Status != "in_progress" {
			t.Errorf("Expected status 'in_progress', got '%s'", updated.Status)
		}
	})

	t.Run("update_multiple_fields", func(t *testing.T) {
		// Test updating multiple fields
		updateStatus = "done"
		updatePriority = "high"
		updateAssignee = "alice@example.com"
		updateLabels = []string{"urgent", "frontend"}
		updateTitle = "Updated test issue"
		updateContent = "Updated content"
		updateForce = false

		err := updateCmd.RunE(updateCmd, []string{testIssue.ID})
		if err != nil {
			t.Fatalf("Update command failed: %v", err)
		}

		// Verify all fields were updated
		updated, err := manager.LoadIssue(testIssue.ID)
		if err != nil {
			t.Fatalf("Failed to load updated issue: %v", err)
		}

		if updated.Status != "done" {
			t.Errorf("Expected status 'done', got '%s'", updated.Status)
		}
		if updated.Priority != "high" {
			t.Errorf("Expected priority 'high', got '%s'", updated.Priority)
		}
		if updated.Assignee != "alice@example.com" {
			t.Errorf("Expected assignee 'alice@example.com', got '%s'", updated.Assignee)
		}
		if len(updated.Labels) != 2 || updated.Labels[0] != "urgent" || updated.Labels[1] != "frontend" {
			t.Errorf("Expected labels ['urgent', 'frontend'], got %v", updated.Labels)
		}
		if updated.Title != "Updated test issue" {
			t.Errorf("Expected title 'Updated test issue', got '%s'", updated.Title)
		}
		if updated.Content != "Updated content" {
			t.Errorf("Expected content 'Updated content', got '%s'", updated.Content)
		}
	})

	t.Run("no_changes", func(t *testing.T) {
		// Test with no changes specified
		updateStatus = ""
		updatePriority = ""
		updateAssignee = ""
		updateLabels = nil
		updateTitle = ""
		updateContent = ""
		updateForce = false

		err := updateCmd.RunE(updateCmd, []string{testIssue.ID})
		if err != nil {
			t.Fatalf("Update command should succeed with no changes: %v", err)
		}
	})
}
