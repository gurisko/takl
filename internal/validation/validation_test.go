package validation

import (
	"testing"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/workflow"
)

func TestValidator_ValidateCreate(t *testing.T) {
	// Create a simple workflow for testing
	w, err := workflow.NewWorkflow(workflow.TypeSimple, nil)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	validator := NewValidator(w)

	tests := []struct {
		name    string
		issue   *domain.Issue
		wantErr bool
	}{
		{
			name: "valid issue",
			issue: &domain.Issue{
				ID:       "ISS-001",
				Type:     "bug",
				Title:    "Test issue",
				Priority: "medium",
				Status:   "open",
			},
			wantErr: false,
		},
		{
			name: "empty title",
			issue: &domain.Issue{
				ID:       "ISS-002",
				Type:     "bug",
				Title:    "",
				Priority: "medium",
			},
			wantErr: true,
		},
		{
			name: "short title",
			issue: &domain.Issue{
				ID:       "ISS-003",
				Type:     "bug",
				Title:    "Hi",
				Priority: "medium",
			},
			wantErr: true,
		},
		{
			name: "long title",
			issue: &domain.Issue{
				ID:       "ISS-004",
				Type:     "bug",
				Title:    "This is a very long title that exceeds the maximum allowed length for issue titles and should be rejected by the validation system because it is way too long and would definitely cause serious display issues in the UI",
				Priority: "medium",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			issue: &domain.Issue{
				ID:       "ISS-005",
				Type:     "invalid-type",
				Title:    "Test issue",
				Priority: "medium",
			},
			wantErr: true,
		},
		{
			name: "invalid priority",
			issue: &domain.Issue{
				ID:       "ISS-006",
				Type:     "bug",
				Title:    "Test issue",
				Priority: "invalid-priority",
			},
			wantErr: true,
		},
		{
			name: "invalid assignee",
			issue: &domain.Issue{
				ID:       "ISS-007",
				Type:     "bug",
				Title:    "Test issue",
				Priority: "medium",
				Assignee: "invalid@assignee@email.com",
			},
			wantErr: true,
		},
		{
			name: "valid assignee email",
			issue: &domain.Issue{
				ID:       "ISS-008",
				Type:     "bug",
				Title:    "Test issue",
				Priority: "medium",
				Assignee: "user@example.com",
			},
			wantErr: false,
		},
		{
			name: "valid assignee username",
			issue: &domain.Issue{
				ID:       "ISS-009",
				Type:     "bug",
				Title:    "Test issue",
				Priority: "medium",
				Assignee: "username123",
			},
			wantErr: false,
		},
		{
			name: "invalid labels with whitespace",
			issue: &domain.Issue{
				ID:       "ISS-010",
				Type:     "bug",
				Title:    "Test issue",
				Priority: "medium",
				Labels:   []string{"valid-label", "invalid label with spaces"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCreate(tt.issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_NormalizeIssue(t *testing.T) {
	validator := NewValidator(nil)

	issue := &domain.Issue{
		ID:       "test-001",
		Type:     "  BUG  ",
		Status:   "IN-PROGRESS",
		Priority: "  HIGH  ",
		Title:    "  Test Issue  ",
		Assignee: "  user@example.com  ",
		Labels:   []string{"  frontend  ", "  urgent  "},
	}

	validator.NormalizeIssue(issue)

	if issue.ID != "ISS-TEST-001" {
		t.Errorf("Expected ID to be normalized to ISS-TEST-001, got %s", issue.ID)
	}
	if issue.Type != "bug" {
		t.Errorf("Expected type to be normalized to bug, got %s", issue.Type)
	}
	if issue.Status != "in_progress" {
		t.Errorf("Expected status to be normalized to in_progress, got %s", issue.Status)
	}
	if issue.Priority != "high" {
		t.Errorf("Expected priority to be normalized to high, got %s", issue.Priority)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Expected title to be trimmed, got '%s'", issue.Title)
	}
	if issue.Assignee != "user@example.com" {
		t.Errorf("Expected assignee to be trimmed, got '%s'", issue.Assignee)
	}
	if len(issue.Labels) != 2 || issue.Labels[0] != "frontend" || issue.Labels[1] != "urgent" {
		t.Errorf("Expected labels to be trimmed, got %v", issue.Labels)
	}
}

func TestValidator_ValidateTransition(t *testing.T) {
	// Create a simple workflow for testing
	w, err := workflow.NewWorkflow(workflow.TypeSimple, nil)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	validator := NewValidator(w)

	issue := &domain.Issue{
		ID:     "ISS-001",
		Type:   "bug",
		Title:  "Test issue",
		Status: "open",
	}

	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{
			name:    "valid transition open to in_progress",
			from:    "open",
			to:      "in_progress",
			wantErr: false,
		},
		{
			name:    "invalid transition open to done",
			from:    "open",
			to:      "done",
			wantErr: true,
		},
		{
			name:    "valid transition in_progress to done",
			from:    "in_progress",
			to:      "done",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTransition(issue, tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateUpdate(t *testing.T) {
	validator := NewValidator(nil)

	oldIssue := &domain.Issue{
		ID:      "ISS-001",
		Type:    "bug",
		Title:   "Original title",
		Status:  "open",
		Updated: time.Now().Add(-time.Hour),
	}

	newIssue := &domain.Issue{
		ID:       "ISS-001",
		Type:     "bug",
		Title:    "Updated title",
		Status:   "open",
		Priority: "medium",
		Updated:  time.Now().Add(-time.Hour), // Old timestamp
	}

	err := validator.ValidateUpdate(oldIssue, newIssue)
	if err != nil {
		t.Errorf("ValidateUpdate() error = %v", err)
	}

	// Check that Updated timestamp was set
	if !newIssue.Updated.After(oldIssue.Updated) {
		t.Error("Expected Updated timestamp to be refreshed")
	}
}
