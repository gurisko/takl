package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/takl/takl/internal/domain"
)

func TestNewWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowType Type
		options      map[string]interface{}
		wantErr      bool
	}{
		{
			name:         "simple workflow",
			workflowType: TypeSimple,
			options:      nil,
			wantErr:      false,
		},
		{
			name:         "scrum workflow",
			workflowType: TypeScrum,
			options:      map[string]interface{}{"sprint_duration": 7},
			wantErr:      false,
		},
		{
			name:         "kanban workflow",
			workflowType: TypeKanban,
			options: map[string]interface{}{
				"wip_limits": map[string]interface{}{
					"doing": 5,
				},
			},
			wantErr: false,
		},
		{
			name:         "unknown workflow",
			workflowType: Type("unknown"),
			options:      nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewWorkflow(tt.workflowType, tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w == nil {
				t.Error("NewWorkflow() returned nil workflow")
			}
		})
	}
}

func TestWorkflow_ValidateTransition(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		workflow *Workflow
		issue    *domain.Issue
		from     string
		to       string
		wantErr  bool
	}{
		{
			name:     "simple valid transition",
			workflow: newSimpleWorkflow(),
			issue:    &domain.Issue{ID: "1", Status: "open"},
			from:     "open",
			to:       "in_progress",
			wantErr:  false,
		},
		{
			name:     "simple invalid transition",
			workflow: newSimpleWorkflow(),
			issue:    &domain.Issue{ID: "1", Status: "open"},
			from:     "open",
			to:       "done",
			wantErr:  true,
		},
		{
			name:     "scrum to sprint backlog with estimate",
			workflow: newScrumWorkflow(nil),
			issue: &domain.Issue{
				ID:     "1",
				Status: "backlog",
				Labels: []string{"points:5"},
			},
			from:    "backlog",
			to:      "sprint_backlog",
			wantErr: true, // No active sprint
		},
		{
			name:     "scrum to sprint backlog without estimate",
			workflow: newScrumWorkflow(nil),
			issue: &domain.Issue{
				ID:     "1",
				Status: "backlog",
				Labels: []string{},
			},
			from:    "backlog",
			to:      "sprint_backlog",
			wantErr: true, // No estimate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.ValidateTransition(ctx, tt.issue, tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_Sprint(t *testing.T) {
	w := newScrumWorkflow(nil)

	// Test starting a sprint
	start := time.Now()
	err := w.StartSprint("sprint-1", start)
	if err != nil {
		t.Errorf("StartSprint() error = %v", err)
	}

	if w.CurrentSprint != "sprint-1" {
		t.Errorf("CurrentSprint = %v, want sprint-1", w.CurrentSprint)
	}

	// Test ending a sprint
	err = w.EndSprint()
	if err != nil {
		t.Errorf("EndSprint() error = %v", err)
	}

	if w.CurrentSprint != "" {
		t.Errorf("CurrentSprint = %v, want empty", w.CurrentSprint)
	}
}

func TestWorkflow_WIPLimits(t *testing.T) {
	w := newKanbanWorkflow(map[string]interface{}{
		"wip_limits": map[string]interface{}{
			"doing": 5,
		},
	})

	// Check WIP limit was set
	limit, exists := w.GetWIPLimit("doing")
	if !exists {
		t.Error("GetWIPLimit() returned false for 'doing' state")
	}
	if limit != 5 {
		t.Errorf("GetWIPLimit() = %v, want 5", limit)
	}

	// Set a new limit
	err := w.SetWIPLimit("review", 3)
	if err != nil {
		t.Errorf("SetWIPLimit() error = %v", err)
	}

	limit, exists = w.GetWIPLimit("review")
	if !exists {
		t.Error("GetWIPLimit() returned false for 'review' state after setting")
	}
	if limit != 3 {
		t.Errorf("GetWIPLimit() after set = %v, want 3", limit)
	}
}

func TestWorkflow_GetValidTransitions(t *testing.T) {
	w := newSimpleWorkflow()

	transitions := w.GetValidTransitions("open")
	if len(transitions) != 1 || transitions[0] != "in_progress" {
		t.Errorf("GetValidTransitions(open) = %v, want [in_progress]", transitions)
	}

	transitions = w.GetValidTransitions("in_progress")
	if len(transitions) != 2 {
		t.Errorf("GetValidTransitions(in_progress) = %v, want 2 transitions", transitions)
	}
}

func TestWorkflow_IsTerminalState(t *testing.T) {
	w := newScrumWorkflow(nil)

	if !w.IsTerminalState("done") {
		t.Error("IsTerminalState(done) = false, want true")
	}

	if w.IsTerminalState("backlog") {
		t.Error("IsTerminalState(backlog) = true, want false")
	}
}
