package kanban

import (
	"context"
	"testing"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/paradigm"
)

func TestKanban_Options(t *testing.T) {
	tests := []struct {
		name        string
		rawOptions  map[string]any
		wantErr     bool
		expectedWIP map[string]int
	}{
		{
			name:       "default_options",
			rawOptions: map[string]any{},
			wantErr:    false,
			expectedWIP: map[string]int{
				"doing":  3,
				"review": 2,
			},
		},
		{
			name: "custom_wip_limits",
			rawOptions: map[string]any{
				"wip_limits": map[string]any{
					"doing":  5,
					"review": 3,
				},
			},
			wantErr: false,
			expectedWIP: map[string]int{
				"doing":  5,
				"review": 3,
			},
		},
		{
			name: "invalid_negative_wip",
			rawOptions: map[string]any{
				"wip_limits": map[string]any{
					"doing": -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_excessive_wip",
			rawOptions: map[string]any{
				"wip_limits": map[string]any{
					"doing": 2000,
				},
			},
			wantErr: true,
		},
		{
			name: "unknown_field",
			rawOptions: map[string]any{
				"unknown_field": "should fail",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kanban{}

			// Create mock deps
			deps := paradigm.Deps{}

			err := k.Init(context.Background(), deps, tt.rawOptions)

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

			if tt.expectedWIP != nil {
				for col, expectedLimit := range tt.expectedWIP {
					if actualLimit, ok := k.opts.WIPLimits[col]; !ok {
						t.Errorf("Expected WIP limit for %q not found", col)
					} else if actualLimit != expectedLimit {
						t.Errorf("WIP limit for %q: expected %d, got %d", col, expectedLimit, actualLimit)
					}
				}
			}
		})
	}
}

func TestKanban_WIPGuard(t *testing.T) {
	k := &Kanban{}
	k.opts = Options{
		WIPLimits: map[string]int{
			"doing": 2,
		},
		BlockOnDownstreamFull: false,
	}

	// Mock storage that returns 2 issues (at limit)
	k.deps = paradigm.Deps{
		Store: &mockStorage{issueCount: 2},
	}

	// Test WIP guard when at capacity
	err := k.guardWIP(context.Background(), nil, "backlog", "doing")
	if err == nil {
		t.Error("Expected WIP limit error but got none")
	}

	// Test WIP guard for unlimited column
	err = k.guardWIP(context.Background(), nil, "doing", "done")
	if err != nil {
		t.Errorf("Unexpected error for unlimited column: %v", err)
	}
}

// Mock storage for testing
type mockStorage struct {
	issueCount int
}

func (m *mockStorage) ListIssues(ctx context.Context, filters map[string]interface{}) ([]*domain.Issue, error) {
	// Return mock issues based on issueCount
	issues := make([]*domain.Issue, m.issueCount)
	return issues, nil
}

func (m *mockStorage) SaveIssue(ctx context.Context, issue *domain.Issue) error {
	return nil
}

func (m *mockStorage) LoadIssue(ctx context.Context, id string) (*domain.Issue, error) {
	return nil, nil
}
