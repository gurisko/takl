package app

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/indexer"
	"github.com/takl/takl/internal/shared"
	"github.com/takl/takl/internal/store"
)

// TestIssueServiceWriteThrough demonstrates the write-through semantics
// using the new formalized interfaces
func TestIssueServiceWriteThrough(t *testing.T) {
	// Setup mocks
	mockRepo := store.NewMockRepo()
	mockIndex := indexer.NewMockIndex()

	// Create service with new interfaces
	service := NewIssueService(
		mockRepo,                    // Repo interface
		nil,                         // ProjectRepo interface (not needed for this test)
		mockIndex,                   // Index interface
		nil,                         // ParadigmRegistry (not needed for this test)
		shared.DefaultClock{},       // Clock
		shared.DefaultIDGenerator{}, // IDGenerator
		nil,                         // Validator (not needed for this test)
		slog.Default(),              // Logger
	)

	ctx := context.Background()
	projectID := "test-project"

	// Test Create with write-through
	createReq := domain.CreateIssueRequest{
		Type:        "bug",
		Title:       "Test Issue",
		Description: "This is a test issue",
		Priority:    "high",
		Labels:      []string{"test", "bug"},
	}

	// Create issue
	issue, err := service.CreateIssue(ctx, projectID, createReq)
	if err != nil {
		t.Fatalf("Failed to create issue: %v", err)
	}

	// Verify write-through: issue should be in both repo and index
	if len(mockRepo.SaveIssueCalls) != 1 {
		t.Errorf("Expected 1 SaveIssue call, got %d", len(mockRepo.SaveIssueCalls))
	}

	if len(mockIndex.UpsertCalls) != 1 {
		t.Errorf("Expected 1 Upsert call, got %d", len(mockIndex.UpsertCalls))
	}

	// Verify issue properties
	if issue.Type != "bug" {
		t.Errorf("Expected type 'bug', got '%s'", issue.Type)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Expected title 'Test Issue', got '%s'", issue.Title)
	}
	if issue.Priority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", issue.Priority)
	}

	// Test Update with write-through
	updateReq := domain.UpdateIssueRequest{
		Title:  stringPtr("Updated Test Issue"),
		Status: stringPtr("in_progress"),
	}

	// Add the issue to mock repo so update can find it
	mockRepo.AddIssue(projectID, issue)

	updatedIssue, err := service.UpdateIssue(ctx, projectID, issue.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update issue: %v", err)
	}

	// Verify write-through: should have additional calls
	if len(mockRepo.SaveIssueCalls) != 2 {
		t.Errorf("Expected 2 SaveIssue calls after update, got %d", len(mockRepo.SaveIssueCalls))
	}

	if len(mockIndex.UpsertCalls) != 2 {
		t.Errorf("Expected 2 Upsert calls after update, got %d", len(mockIndex.UpsertCalls))
	}

	// Verify updated properties
	if updatedIssue.Title != "Updated Test Issue" {
		t.Errorf("Expected title 'Updated Test Issue', got '%s'", updatedIssue.Title)
	}
	if updatedIssue.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updatedIssue.Status)
	}
	if updatedIssue.Version != 2 {
		t.Errorf("Expected version 2, got %d", updatedIssue.Version)
	}

	// Test Delete with write-through
	err = service.DeleteIssue(ctx, projectID, issue.ID)
	if err != nil {
		t.Fatalf("Failed to delete issue: %v", err)
	}

	// Verify write-through: should have delete calls
	if len(mockRepo.DeleteIssueCalls) != 1 {
		t.Errorf("Expected 1 DeleteIssue call, got %d", len(mockRepo.DeleteIssueCalls))
	}

	if len(mockIndex.DeleteByIDCalls) != 1 {
		t.Errorf("Expected 1 DeleteByID call, got %d", len(mockIndex.DeleteByIDCalls))
	}
}

// TestIssueServiceSearch demonstrates search functionality with the new Index interface
func TestIssueServiceSearch(t *testing.T) {
	// Setup mocks
	mockRepo := store.NewMockRepo()
	mockIndex := indexer.NewMockIndex()

	// Add test issues to index
	projectID := "test-project"
	testIssue1 := &domain.Issue{
		ID:      "ISSUE-001",
		Type:    "bug",
		Title:   "Login not working",
		Content: "Users cannot log in",
		Status:  "open",
		Created: time.Now(),
		Updated: time.Now(),
		Version: 1,
	}
	testIssue2 := &domain.Issue{
		ID:      "ISSUE-002",
		Type:    "feature",
		Title:   "Add dark mode",
		Content: "Users want dark mode",
		Status:  "open",
		Created: time.Now(),
		Updated: time.Now(),
		Version: 1,
	}

	mockIndex.AddIssue(projectID, testIssue1)
	mockIndex.AddIssue(projectID, testIssue2)

	// Create service
	service := NewIssueService(
		mockRepo,
		nil,
		mockIndex,
		nil,
		shared.DefaultClock{},
		shared.DefaultIDGenerator{},
		nil, // Validator (not needed for this test)
		slog.Default(),
	)

	ctx := context.Background()

	// Test search
	results, err := service.SearchIssues(ctx, projectID, "login")
	if err != nil {
		t.Fatalf("Failed to search issues: %v", err)
	}

	// Should find one result
	if len(results) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(results))
	}

	if len(results) > 0 && results[0].ID != "ISSUE-001" {
		t.Errorf("Expected to find ISSUE-001, got %s", results[0].ID)
	}

	// Verify search was called on index
	if len(mockIndex.SearchCalls) != 1 {
		t.Errorf("Expected 1 Search call, got %d", len(mockIndex.SearchCalls))
	}
}

// TestIssueServiceList demonstrates list functionality with filtering
func TestIssueServiceList(t *testing.T) {
	// Setup mocks
	mockRepo := store.NewMockRepo()

	// Add test issues to repo
	projectID := "test-project"
	testIssue1 := &domain.Issue{
		ID:       "ISSUE-001",
		Type:     "bug",
		Title:    "Bug issue",
		Status:   "open",
		Priority: "high",
		Created:  time.Now(),
		Updated:  time.Now(),
		Version:  1,
	}
	testIssue2 := &domain.Issue{
		ID:       "ISSUE-002",
		Type:     "feature",
		Title:    "Feature request",
		Status:   "closed",
		Priority: "low",
		Created:  time.Now(),
		Updated:  time.Now(),
		Version:  1,
	}

	mockRepo.AddIssue(projectID, testIssue1)
	mockRepo.AddIssue(projectID, testIssue2)

	// Create service
	service := NewIssueService(
		mockRepo,
		nil,
		nil, // No index needed for list
		nil,
		shared.DefaultClock{},
		shared.DefaultIDGenerator{},
		nil, // Validator (not needed for this test)
		slog.Default(),
	)

	ctx := context.Background()

	// Test list all
	filter := domain.IssueFilter{}
	results, err := service.ListIssues(ctx, projectID, filter)
	if err != nil {
		t.Fatalf("Failed to list issues: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(results))
	}

	// Test filtered list
	filter = domain.IssueFilter{Status: "open"}
	results, err = service.ListIssues(ctx, projectID, filter)
	if err != nil {
		t.Fatalf("Failed to list filtered issues: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 open issue, got %d", len(results))
	}

	if len(results) > 0 && results[0].Status != "open" {
		t.Errorf("Expected open issue, got status %s", results[0].Status)
	}

	// Verify list was called on repo with correct filters
	if len(mockRepo.ListIssuesCalls) != 2 {
		t.Errorf("Expected 2 ListIssues calls, got %d", len(mockRepo.ListIssuesCalls))
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
