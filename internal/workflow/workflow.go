package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/takl/takl/internal/domain"
)

// Type represents the workflow type
type Type string

const (
	TypeScrum  Type = "scrum"
	TypeKanban Type = "kanban"
	TypeSimple Type = "simple"
)

// State represents a workflow state
type State struct {
	Key         string
	DisplayName string
	WIPLimit    int // For Kanban, 0 means no limit
}

// Workflow manages issue state transitions and validations
type Workflow struct {
	Type        Type
	States      []State
	Transitions map[string][]string // from -> []to

	// Scrum-specific fields
	SprintDuration int    // days
	CurrentSprint  string // sprint ID
	SprintStart    time.Time
	SprintEnd      time.Time

	// Kanban-specific fields
	WIPLimits      map[string]int
	CycleTimeGoals map[string]int // state -> hours
}

// NewWorkflow creates a workflow based on type
func NewWorkflow(workflowType Type, options map[string]interface{}) (*Workflow, error) {
	switch workflowType {
	case TypeScrum:
		return newScrumWorkflow(options), nil
	case TypeKanban:
		return newKanbanWorkflow(options), nil
	case TypeSimple:
		return newSimpleWorkflow(), nil
	default:
		return nil, fmt.Errorf("unknown workflow type: %s", workflowType)
	}
}

func newSimpleWorkflow() *Workflow {
	return &Workflow{
		Type: TypeSimple,
		States: []State{
			{Key: "open", DisplayName: "Open"},
			{Key: "in_progress", DisplayName: "In Progress"},
			{Key: "done", DisplayName: "Done"},
		},
		Transitions: map[string][]string{
			"open":        {"in_progress"},
			"in_progress": {"done", "open"},
			"done":        {"open"}, // Allow reopening
		},
	}
}

func newScrumWorkflow(options map[string]interface{}) *Workflow {
	// Default sprint duration
	sprintDuration := 14
	if duration, ok := options["sprint_duration"].(int); ok {
		sprintDuration = duration
	}

	return &Workflow{
		Type: TypeScrum,
		States: []State{
			{Key: "backlog", DisplayName: "Product Backlog"},
			{Key: "sprint_backlog", DisplayName: "Sprint Backlog"},
			{Key: "in_progress", DisplayName: "In Progress"},
			{Key: "review", DisplayName: "In Review"},
			{Key: "done", DisplayName: "Done"},
		},
		Transitions: map[string][]string{
			"backlog":        {"sprint_backlog"},
			"sprint_backlog": {"in_progress", "backlog"},
			"in_progress":    {"review", "sprint_backlog"},
			"review":         {"done", "in_progress"},
			"done":           {}, // Terminal state in sprint
		},
		SprintDuration: sprintDuration,
	}
}

func newKanbanWorkflow(options map[string]interface{}) *Workflow {
	w := &Workflow{
		Type: TypeKanban,
		States: []State{
			{Key: "backlog", DisplayName: "Backlog", WIPLimit: 0},
			{Key: "ready", DisplayName: "Ready", WIPLimit: 10},
			{Key: "doing", DisplayName: "Doing", WIPLimit: 3},
			{Key: "review", DisplayName: "Review", WIPLimit: 2},
			{Key: "done", DisplayName: "Done", WIPLimit: 0},
		},
		Transitions: map[string][]string{
			"backlog": {"ready"},
			"ready":   {"doing"},
			"doing":   {"review"},
			"review":  {"done", "doing"},
			"done":    {}, // Terminal state
		},
		WIPLimits:      make(map[string]int),
		CycleTimeGoals: make(map[string]int),
	}

	// Set WIP limits from options
	if limits, ok := options["wip_limits"].(map[string]interface{}); ok {
		for state, limit := range limits {
			if l, ok := limit.(int); ok {
				w.WIPLimits[state] = l
				// Update state WIP limits
				for i, s := range w.States {
					if s.Key == state {
						w.States[i].WIPLimit = l
						break
					}
				}
			}
		}
	} else {
		// Use defaults from states
		for _, state := range w.States {
			w.WIPLimits[state.Key] = state.WIPLimit
		}
	}

	return w
}

// ValidateTransition checks if a state transition is valid
func (w *Workflow) ValidateTransition(ctx context.Context, issue *domain.Issue, from, to string) error {
	// Check if transition is allowed
	validTransitions, exists := w.Transitions[from]
	if !exists {
		return fmt.Errorf("invalid source state: %s", from)
	}

	isValid := false
	for _, valid := range validTransitions {
		if valid == to {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("transition from %s to %s is not allowed", from, to)
	}

	// Workflow-specific validations
	switch w.Type {
	case TypeScrum:
		return w.validateScrumTransition(ctx, issue, from, to)
	case TypeKanban:
		return w.validateKanbanTransition(ctx, issue, from, to)
	}

	return nil
}

func (w *Workflow) validateScrumTransition(ctx context.Context, issue *domain.Issue, from, to string) error {
	// Moving to sprint backlog requires estimation
	if to == "sprint_backlog" {
		hasEstimate := false
		for _, label := range issue.Labels {
			if len(label) > 7 && label[:7] == "points:" {
				hasEstimate = true
				break
			}
		}
		if !hasEstimate {
			return fmt.Errorf("issue must be estimated before adding to sprint")
		}
	}

	// Check if sprint is active
	if (to == "sprint_backlog" || to == "in_progress") && w.CurrentSprint == "" {
		return fmt.Errorf("no active sprint")
	}

	return nil
}

func (w *Workflow) validateKanbanTransition(ctx context.Context, issue *domain.Issue, from, to string) error {
	// Check WIP limits
	if limit, hasLimit := w.WIPLimits[to]; hasLimit && limit > 0 {
		// In a real implementation, we'd count issues in the target state
		// For now, we'll skip this check as it requires access to the issue store
		_ = limit // Satisfy linter - will be used when WIP limit checking is implemented
	}

	return nil
}

// GetStates returns all workflow states
func (w *Workflow) GetStates() []State {
	return w.States
}

// GetValidTransitions returns valid transitions from a given state
func (w *Workflow) GetValidTransitions(from string) []string {
	return w.Transitions[from]
}

// IsTerminalState checks if a state is terminal (no outgoing transitions)
func (w *Workflow) IsTerminalState(state string) bool {
	transitions := w.Transitions[state]
	return len(transitions) == 0
}

// StartSprint starts a new sprint (Scrum only)
func (w *Workflow) StartSprint(sprintID string, start time.Time) error {
	if w.Type != TypeScrum {
		return fmt.Errorf("sprints are only supported in Scrum workflow")
	}

	w.CurrentSprint = sprintID
	w.SprintStart = start
	w.SprintEnd = start.AddDate(0, 0, w.SprintDuration)

	return nil
}

// EndSprint ends the current sprint (Scrum only)
func (w *Workflow) EndSprint() error {
	if w.Type != TypeScrum {
		return fmt.Errorf("sprints are only supported in Scrum workflow")
	}

	w.CurrentSprint = ""
	w.SprintStart = time.Time{}
	w.SprintEnd = time.Time{}

	return nil
}

// GetWIPLimit returns the WIP limit for a state (Kanban only)
func (w *Workflow) GetWIPLimit(state string) (int, bool) {
	if w.Type != TypeKanban {
		return 0, false
	}

	limit, exists := w.WIPLimits[state]
	return limit, exists
}

// SetWIPLimit sets the WIP limit for a state (Kanban only)
func (w *Workflow) SetWIPLimit(state string, limit int) error {
	if w.Type != TypeKanban {
		return fmt.Errorf("WIP limits are only supported in Kanban workflow")
	}

	w.WIPLimits[state] = limit

	// Update state as well
	for i, s := range w.States {
		if s.Key == state {
			w.States[i].WIPLimit = limit
			break
		}
	}

	return nil
}
