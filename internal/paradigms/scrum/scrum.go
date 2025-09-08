package scrum

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/takl/takl/internal/charts"
	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/paradigm"
)

// Sprint represents a simple sprint state machine
type Sprint struct {
	ID       string
	Start    time.Time
	End      time.Time
	Active   bool
	Capacity int // Total story points for sprint
}

// Toggle activates or deactivates the sprint
func (s *Sprint) Toggle() {
	s.Active = !s.Active
}

// IsActive returns whether the sprint is currently active
func (s *Sprint) IsActive() bool {
	return s.Active
}

// Scrum paradigm implementation
type Scrum struct {
	deps   paradigm.Deps
	sprint *Sprint // Simplified sprint management
}

// Register the Scrum paradigm
func init() {
	paradigm.Register("scrum", func() paradigm.Paradigm {
		return &Scrum{}
	})
}

// Paradigm interface implementation
func (s *Scrum) ID() string {
	return "scrum"
}

func (s *Scrum) Name() string {
	return "Scrum"
}

func (s *Scrum) Category() paradigm.Category {
	return paradigm.CategoryAgile
}

func (s *Scrum) Init(ctx context.Context, deps paradigm.Deps, rawOptions map[string]any) error {
	s.deps = deps

	// Initialize with a default inactive sprint
	s.sprint = &Sprint{
		ID:       "",
		Capacity: 20, // Default capacity
		Active:   false,
	}

	// Initialize current sprint if any exists
	if err := s.loadCurrentSprint(ctx); err != nil {
		// Log but don't fail initialization
		s.deps.Log.Warn("failed to load current sprint", "error", err)
	}

	return nil
}

func (s *Scrum) GetTimeModel() paradigm.TimeModel {
	return paradigm.TimeModel{
		Kind: "timeboxed",
		Params: map[string]any{
			"sprint_length": "2w",
			"ceremonies":    []string{"planning", "daily", "review", "retrospective"},
		},
	}
}

func (s *Scrum) GetWorkUnit() paradigm.WorkUnit {
	return paradigm.WorkUnit{
		Kind:   "story",
		Fields: []string{"story_points", "sprint_id", "definition_of_done"},
	}
}

// Workflow states with guards
func (s *Scrum) GetWorkflowStates() []paradigm.WorkflowState {
	return []paradigm.WorkflowState{
		{
			Key:         "backlog",
			DisplayName: "Product Backlog",
			Guards:      nil, // No guards for backlog
		},
		{
			Key:         "sprint_backlog",
			DisplayName: "Sprint Backlog",
			Guards: []paradigm.Guard{
				s.guardEstimated,
				s.guardWithinCapacity,
			},
		},
		{
			Key:         "in_progress",
			DisplayName: "In Progress",
			Guards: []paradigm.Guard{
				s.guardSprintActive,
			},
		},
		{
			Key:         "review",
			DisplayName: "In Review",
			Guards: []paradigm.Guard{
				s.guardSprintActive,
			},
		},
		{
			Key:         "done",
			DisplayName: "Done",
			Guards: []paradigm.Guard{
				s.guardDefinitionOfDone,
			},
		},
	}
}

func (s *Scrum) GetValidTransitions(currentState string) []string {
	transitions := map[string][]string{
		"backlog":        {"sprint_backlog"},
		"sprint_backlog": {"in_progress", "backlog"},
		"in_progress":    {"review", "sprint_backlog"},
		"review":         {"done", "in_progress"},
		"done":           {}, // Final state
	}
	return transitions[currentState]
}

func (s *Scrum) ValidateTransition(ctx context.Context, issue *domain.Issue, from, to string) error {
	// Check if transition is valid
	validTransitions := s.GetValidTransitions(from)
	isValid := false
	for _, valid := range validTransitions {
		if valid == to {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("%w: cannot transition from %s to %s", paradigm.ErrInvalidTransition, from, to)
	}

	// Run guards for the target state
	states := s.GetWorkflowStates()
	for _, state := range states {
		if state.Key == to {
			for _, guard := range state.Guards {
				if err := guard(ctx, issue, from, to); err != nil {
					return err
				}
			}
			break
		}
	}

	return nil
}

// Guard functions
func (s *Scrum) guardEstimated(ctx context.Context, issue *domain.Issue, from, to string) error {
	// Check if issue has story points using shared utility
	issues := []*domain.Issue{issue}
	storyPoints, _ := charts.CalculateStoryPoints(issues)
	if storyPoints == 0 {
		return fmt.Errorf("%w: story points required for sprint planning", paradigm.ErrMissingEstimate)
	}
	return nil
}

func (s *Scrum) guardWithinCapacity(ctx context.Context, issue *domain.Issue, from, to string) error {
	if s.sprint == nil {
		return nil // No sprint capacity to check
	}

	issues := []*domain.Issue{issue}
	storyPoints, _ := charts.CalculateStoryPoints(issues)

	// Simple capacity check - in practice you'd track consumed capacity
	if storyPoints > s.sprint.Capacity {
		return fmt.Errorf("%w: issue exceeds sprint capacity", paradigm.ErrCapacityExceeded)
	}
	return nil
}

func (s *Scrum) guardSprintActive(ctx context.Context, issue *domain.Issue, from, to string) error {
	if s.sprint == nil || !s.sprint.IsActive() {
		return fmt.Errorf("%w: no active sprint", paradigm.ErrSprintNotActive)
	}
	return nil
}

func (s *Scrum) guardDefinitionOfDone(ctx context.Context, issue *domain.Issue, from, to string) error {
	// Simplified DoD check - in practice this would be more sophisticated
	if issue.Title == "" {
		return fmt.Errorf("%w: title is required", paradigm.ErrDefinitionOfDone)
	}

	// Check if all subtasks are done (simplified)
	if s.hasIncompleteSubtasks(ctx, issue) {
		return fmt.Errorf("%w: all subtasks must be completed", paradigm.ErrDefinitionOfDone)
	}

	return nil
}

func (s *Scrum) CreateIssue(ctx context.Context, req paradigm.CreateIssueRequest) (*domain.Issue, error) {
	issue := &domain.Issue{
		Type:     req.Type,
		Title:    req.Title,
		Status:   "backlog", // All new issues start in backlog
		Priority: req.Priority,
		Assignee: req.Assignee,
		Labels:   req.Labels,
		Created:  s.deps.Clock.Now(),
		Updated:  s.deps.Clock.Now(),
		Content:  req.Description,
	}

	// Set paradigm-specific extensions
	if extensions, ok := req.Extensions["scrum"].(map[string]any); ok {
		if sp, exists := extensions["story_points"]; exists {
			// Story points would be stored in a paradigm extension field
			_ = sp // For now, just acknowledge it exists
		}
	}

	// Generate ID using the existing function
	issue.ID = s.generateIssueID()

	if err := s.deps.Store.SaveIssue(ctx, issue); err != nil {
		return nil, fmt.Errorf("failed to save issue: %w", err)
	}

	return issue, nil
}

func (s *Scrum) GetPlanningOperations() []paradigm.PlanningOperation {
	return []paradigm.PlanningOperation{
		{
			Name: "sprint_plan",
			Run:  s.planSprint,
		},
		{
			Name: "estimate",
			Run:  s.estimateIssues,
		},
	}
}

func (s *Scrum) GetExecutionOperations() []paradigm.ExecutionOperation {
	return []paradigm.ExecutionOperation{
		{
			Name: "start_sprint",
			Run:  s.startSprint,
		},
		{
			Name: "end_sprint",
			Run:  s.endSprint,
		},
	}
}

func (s *Scrum) CalculateProgress(ctx context.Context, issues []*domain.Issue, pc paradigm.ProgressContext) (*paradigm.Progress, error) {
	sprintIssues := s.filterSprintIssues(issues)

	totalPoints, donePoints := charts.CalculateStoryPoints(sprintIssues)
	completion := float64(donePoints) / math.Max(1, float64(totalPoints))

	velocity := s.calculateVelocity(ctx)

	// Create burndown chart
	burndown := s.createBurndownChart(sprintIssues, pc.Start, pc.End)

	return &paradigm.Progress{
		Completion: completion,
		Velocity:   velocity,
		Charts:     []paradigm.Chart{burndown},
		Prediction: s.predictCompletion(burndown),
	}, nil
}

// Helper methods
// Removed - using shared charts.CalculateStoryPoints utility instead

func (s *Scrum) generateIssueID() string {
	// Use the existing ID generation from issues package
	// In a real implementation, we might want scrum-specific IDs
	return fmt.Sprintf("iss-%06d", time.Now().UnixNano()%1000000)
}

func (s *Scrum) loadCurrentSprint(ctx context.Context) error {
	// Implementation would load current sprint from storage
	return nil
}

func (s *Scrum) hasIncompleteSubtasks(ctx context.Context, issue *domain.Issue) bool {
	// Simplified check - would query for subtasks in real implementation
	return false
}

func (s *Scrum) filterSprintIssues(issues []*domain.Issue) []*domain.Issue {
	// Filter issues by current sprint
	var sprintIssues []*domain.Issue
	for _, issue := range issues {
		if issue.Status != "backlog" { // Simple approximation
			sprintIssues = append(sprintIssues, issue)
		}
	}
	return sprintIssues
}

// Removed - using shared charts.CalculateStoryPoints utility instead

func (s *Scrum) calculateVelocity(ctx context.Context) float64 {
	// Simplified velocity calculation
	// In practice, this would look at completed story points over past sprints
	if s.sprint == nil {
		return 0
	}
	return float64(s.sprint.Capacity) * 0.8 // Assume 80% completion rate
}

func (s *Scrum) createBurndownChart(issues []*domain.Issue, start, end time.Time) paradigm.Chart {
	// Create a simplified burndown chart using shared utilities
	totalPoints, _ := charts.CalculateStoryPoints(issues)

	return charts.NewChart("burndown").
		WithData("total_points", totalPoints).
		WithTimeRange(start, end).
		WithCurrentTime(s.deps.Clock.Now()).
		Build()
}

func (s *Scrum) predictCompletion(burndown paradigm.Chart) *time.Time {
	// Simple prediction based on current progress
	prediction := s.deps.Clock.Now().Add(7 * 24 * time.Hour) // One week from now
	return &prediction
}

// Planning operations implementation
func (s *Scrum) planSprint(ctx context.Context, args map[string]any) (*paradigm.PlanningResult, error) {
	capacity, _ := args["capacity"].(int)
	sprintID, _ := args["sprint_id"].(string)

	// Initialize sprint if it doesn't exist
	if s.sprint == nil {
		s.sprint = &Sprint{
			ID:       sprintID,
			Capacity: capacity,
			Active:   false,
		}
	} else if capacity > 0 {
		s.sprint.Capacity = capacity
	}

	return &paradigm.PlanningResult{
		Success: true,
		Message: fmt.Sprintf("Sprint planned with capacity of %d story points", s.sprint.Capacity),
		Data: map[string]any{
			"capacity":  s.sprint.Capacity,
			"sprint_id": s.sprint.ID,
		},
	}, nil
}

func (s *Scrum) estimateIssues(ctx context.Context, args map[string]any) (*paradigm.PlanningResult, error) {
	issueIDs, _ := args["issues"].([]string)
	estimates, _ := args["estimates"].([]int)

	if len(issueIDs) != len(estimates) {
		return &paradigm.PlanningResult{
			Success: false,
			Message: "Issue IDs and estimates must have the same length",
		}, nil
	}

	// In real implementation, would update issue estimates
	return &paradigm.PlanningResult{
		Success: true,
		Message: fmt.Sprintf("Estimated %d issues", len(issueIDs)),
		Data: map[string]any{
			"estimated_count": len(issueIDs),
		},
	}, nil
}

// Execution operations implementation
func (s *Scrum) startSprint(ctx context.Context, args map[string]any) (*paradigm.ExecutionResult, error) {
	sprintID, _ := args["sprint_id"].(string)
	if sprintID == "" {
		sprintID = fmt.Sprintf("SPR-%s", s.deps.Clock.Now().Format("2006-01-02"))
	}

	// Initialize sprint if it doesn't exist
	if s.sprint == nil {
		s.sprint = &Sprint{ID: sprintID, Capacity: 40} // Default capacity
	}

	// Start the sprint using simplified state machine
	s.sprint.ID = sprintID
	s.sprint.Start = s.deps.Clock.Now()
	s.sprint.End = s.sprint.Start.Add(14 * 24 * time.Hour) // 2 week sprint
	s.sprint.Active = true

	return &paradigm.ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Started sprint %s", sprintID),
		Data: map[string]any{
			"sprint_id":  s.sprint.ID,
			"start_date": s.sprint.Start,
			"end_date":   s.sprint.End,
		},
	}, nil
}

func (s *Scrum) endSprint(ctx context.Context, args map[string]any) (*paradigm.ExecutionResult, error) {
	if s.sprint == nil || !s.sprint.IsActive() {
		return &paradigm.ExecutionResult{
			Success: false,
			Message: "No active sprint to end",
		}, nil
	}

	oldSprintID := s.sprint.ID
	// End the sprint using simple state toggle
	s.sprint.Toggle()

	return &paradigm.ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Ended sprint %s", oldSprintID),
		Data: map[string]any{
			"completed_sprint": oldSprintID,
			"end_date":         s.deps.Clock.Now(),
		},
	}, nil
}

// Test helpers - simplified using Sprint struct
func (s *Scrum) SetSprintCapacity(capacity int) {
	if s.sprint == nil {
		s.sprint = &Sprint{}
	}
	s.sprint.Capacity = capacity
}

func (s *Scrum) SetSprintActive(active bool) {
	if s.sprint == nil {
		s.sprint = &Sprint{}
	}
	s.sprint.Active = active
}
