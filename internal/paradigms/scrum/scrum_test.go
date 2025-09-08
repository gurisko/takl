package scrum

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/paradigm"
)

func TestScrum_GuardEstimated(t *testing.T) {
	t.Parallel()

	deps := paradigm.TestDeps(paradigm.NewFixedClock(time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)))
	s := &Scrum{}
	err := s.Init(context.Background(), deps, map[string]any{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tests := []struct {
		name      string
		issue     *domain.Issue
		wantError bool
	}{
		{
			name: "story without points should fail",
			issue: &domain.Issue{
				ID:    "iss-001",
				Title: "Test story",
				Type:  "story",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := s.guardEstimated(context.Background(), tt.issue, "backlog", "sprint_backlog")

			if (err != nil) != tt.wantError {
				t.Errorf("guardEstimated() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err != nil {
				if !errors.Is(err, paradigm.ErrMissingEstimate) {
					t.Errorf("Expected ErrMissingEstimate, got %v", err)
				}
			}
		})
	}
}

func TestScrum_GuardWithinCapacity(t *testing.T) {
	t.Parallel()

	deps := paradigm.TestDeps(nil)
	s := &Scrum{}
	err := s.Init(context.Background(), deps, map[string]any{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Set up sprint with 10 capacity (simplified - no consumed capacity tracking)
	s.SetSprintCapacity(10)

	tests := []struct {
		name        string
		storyPoints int
		wantError   bool
	}{
		{"within capacity", 5, false},
		{"exactly at capacity", 10, false},
		{"exceeds capacity", 15, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issue := &domain.Issue{
				ID:     "test",
				Title:  "test",
				Labels: []string{fmt.Sprintf("points:%d", tt.storyPoints)},
			}

			err := s.guardWithinCapacity(context.Background(), issue, "backlog", "sprint_backlog")

			if (err != nil) != tt.wantError {
				t.Errorf("guardWithinCapacity() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestScrum_GuardSprintActive(t *testing.T) {
	t.Parallel()

	deps := paradigm.TestDeps(nil)
	s := &Scrum{}
	err := s.Init(context.Background(), deps, map[string]any{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	issue := &domain.Issue{ID: "test", Title: "test"}

	// Test with inactive sprint
	err = s.guardSprintActive(context.Background(), issue, "sprint_backlog", "in_progress")
	if err == nil {
		t.Error("Expected error when sprint is not active")
	}

	// Test with active sprint
	s.SetSprintActive(true)
	err = s.guardSprintActive(context.Background(), issue, "sprint_backlog", "in_progress")
	if err != nil {
		t.Errorf("Unexpected error when sprint is active: %v", err)
	}
}

func TestScrum_ValidateTransition(t *testing.T) {
	t.Parallel()

	deps := paradigm.TestDeps(nil)
	s := &Scrum{}
	err := s.Init(context.Background(), deps, map[string]any{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	issue := &domain.Issue{ID: "test", Title: "test"}

	tests := []struct {
		name      string
		from      string
		to        string
		wantError bool
	}{
		{"valid transition", "backlog", "sprint_backlog", true}, // Will fail due to estimation guard
		{"invalid transition", "backlog", "done", true},
		{"self transition", "backlog", "backlog", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := s.ValidateTransition(context.Background(), issue, tt.from, tt.to)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTransition() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestScrum_PlanningOperations(t *testing.T) {
	t.Parallel()

	deps := paradigm.TestDeps(nil)
	s := &Scrum{}
	err := s.Init(context.Background(), deps, map[string]any{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test sprint planning
	result, err := s.planSprint(context.Background(), map[string]any{
		"capacity": 30,
	})

	if err != nil {
		t.Errorf("planSprint failed: %v", err)
	}

	if !result.Success {
		t.Error("planSprint should succeed")
	}

	if s.sprint.Capacity != 30 {
		t.Errorf("Expected capacity 30, got %d", s.sprint.Capacity)
	}
}

func TestScrum_ExecutionOperations(t *testing.T) {
	t.Parallel()

	deps := paradigm.TestDeps(nil)
	s := &Scrum{}
	err := s.Init(context.Background(), deps, map[string]any{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test start sprint
	result, err := s.startSprint(context.Background(), map[string]any{
		"sprint_id": "TEST-001",
	})

	if err != nil {
		t.Errorf("startSprint failed: %v", err)
	}

	if !result.Success {
		t.Error("startSprint should succeed")
	}

	if !s.sprint.IsActive() {
		t.Error("Sprint should be active after start")
	}

	if s.sprint.ID != "TEST-001" {
		t.Errorf("Expected sprint ID TEST-001, got %s", s.sprint.ID)
	}

	// Test end sprint
	result, err = s.endSprint(context.Background(), nil)

	if err != nil {
		t.Errorf("endSprint failed: %v", err)
	}

	if !result.Success {
		t.Error("endSprint should succeed")
	}

	if s.sprint.IsActive() {
		t.Error("Sprint should not be active after end")
	}
}
