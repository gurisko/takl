package kanban

import (
	"bytes"
	"context"
	"fmt"

	"github.com/takl/takl/internal/charts"
	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/paradigm"
	"gopkg.in/yaml.v3"
)

// Register the Kanban paradigm
func init() {
	paradigm.Register("kanban", func() paradigm.Paradigm {
		return &Kanban{}
	})
}

// Options represents Kanban-specific configuration
type Options struct {
	WIPLimits             map[string]int `yaml:"wip_limits"`
	BlockOnDownstreamFull bool           `yaml:"block_on_downstream_full"`
}

// defaultOptions returns default Kanban options
func defaultOptions() Options {
	return Options{
		WIPLimits: map[string]int{
			"doing":  3,
			"review": 2,
		},
		BlockOnDownstreamFull: true,
	}
}

// Kanban implements the paradigm.Paradigm interface for Kanban workflow
type Kanban struct {
	deps paradigm.Deps
	opts Options
}

// ID returns the paradigm identifier
func (k *Kanban) ID() string {
	return "kanban"
}

// Name returns the human-readable name
func (k *Kanban) Name() string {
	return "Kanban"
}

// Category returns the paradigm category
func (k *Kanban) Category() paradigm.Category {
	return paradigm.CategoryLean
}

// Init initializes the Kanban paradigm with typed options
func (k *Kanban) Init(ctx context.Context, deps paradigm.Deps, rawOptions map[string]any) error {
	k.deps = deps

	// Start with defaults
	k.opts = defaultOptions()

	// If no raw options provided, use defaults
	if len(rawOptions) == 0 {
		return nil
	}

	// Re-marshal raw options into YAML and decode strictly into typed Options
	yamlData, err := yaml.Marshal(rawOptions)
	if err != nil {
		return fmt.Errorf("failed to marshal kanban options: %w", err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(yamlData))
	dec.KnownFields(true) // Strict validation - unknown fields fail

	if err := dec.Decode(&k.opts); err != nil {
		return fmt.Errorf("invalid kanban options: %w", err)
	}

	// Validate options
	return k.validateOptions()
}

// validateOptions validates the Kanban options
func (k *Kanban) validateOptions() error {
	for col, limit := range k.opts.WIPLimits {
		if limit < 0 {
			return fmt.Errorf("wip limit for %q cannot be negative", col)
		}
		if limit > 1000 {
			return fmt.Errorf("wip limit for %q cannot exceed 1000", col)
		}
	}
	return nil
}

// GetTimeModel returns the time model for Kanban
func (k *Kanban) GetTimeModel() paradigm.TimeModel {
	return paradigm.TimeModel{
		Kind:   "continuous",
		Params: map[string]any{},
	}
}

// GetWorkUnit returns the work unit definition for Kanban
func (k *Kanban) GetWorkUnit() paradigm.WorkUnit {
	return paradigm.WorkUnit{
		Kind:   "task",
		Fields: []string{"title", "description"},
	}
}

// GetWorkflowStates returns the workflow states for Kanban
func (k *Kanban) GetWorkflowStates() []paradigm.WorkflowState {
	return []paradigm.WorkflowState{
		{
			Key:         "backlog",
			DisplayName: "Backlog",
			Guards:      []paradigm.Guard{},
		},
		{
			Key:         "doing",
			DisplayName: "Doing",
			Guards:      []paradigm.Guard{k.guardWIP},
		},
		{
			Key:         "review",
			DisplayName: "Review",
			Guards:      []paradigm.Guard{k.guardWIP},
		},
		{
			Key:         "done",
			DisplayName: "Done",
			Guards:      []paradigm.Guard{},
		},
	}
}

// GetValidTransitions returns valid transitions from the current state
func (k *Kanban) GetValidTransitions(currentState string) []string {
	switch currentState {
	case "backlog":
		return []string{"doing"}
	case "doing":
		return []string{"review", "backlog"}
	case "review":
		return []string{"done", "doing"}
	case "done":
		return []string{"backlog"} // Allow reopening
	default:
		return []string{}
	}
}

// ValidateTransition validates a state transition
func (k *Kanban) ValidateTransition(ctx context.Context, issue *domain.Issue, from, to string) error {
	if from == to {
		return nil // Self-transition is always valid
	}

	validTransitions := k.GetValidTransitions(from)
	for _, valid := range validTransitions {
		if valid == to {
			// Run guards for the target state
			for _, state := range k.GetWorkflowStates() {
				if state.Key == to {
					for _, guard := range state.Guards {
						if err := guard(ctx, issue, from, to); err != nil {
							return err
						}
					}
					return nil
				}
			}
			return nil
		}
	}

	return fmt.Errorf("%w: cannot transition from %s to %s", paradigm.ErrInvalidTransition, from, to)
}

// guardWIP implements WIP limit checking
func (k *Kanban) guardWIP(ctx context.Context, issue *domain.Issue, from, to string) error {
	if to == "" || to == "backlog" || to == "done" {
		return nil // No WIP limits for backlog or done
	}

	limit, ok := k.opts.WIPLimits[to]
	if !ok || limit <= 0 {
		return nil // No limit defined or unlimited
	}

	// Count current issues in the target state
	current, err := k.countIssuesInState(ctx, to)
	if err != nil {
		return fmt.Errorf("failed to count issues in state %s: %w", to, err)
	}

	if current >= limit {
		return fmt.Errorf("%w: %s at capacity (%d/%d)", paradigm.ErrWIPLimitExceeded, to, current, limit)
	}

	// Optional: Check downstream capacity
	if k.opts.BlockOnDownstreamFull && !k.hasDownstreamCapacity(ctx, to) {
		return fmt.Errorf("%w: downstream bottleneck", paradigm.ErrInvalidTransition)
	}

	return nil
}

// countIssuesInState counts issues currently in the given state
func (k *Kanban) countIssuesInState(ctx context.Context, state string) (int, error) {
	issues, err := k.deps.Store.ListIssues(ctx, map[string]interface{}{
		"status": state,
	})
	if err != nil {
		return 0, err
	}
	return len(issues), nil
}

// hasDownstreamCapacity checks if downstream states have capacity
func (k *Kanban) hasDownstreamCapacity(ctx context.Context, currentState string) bool {
	// Simple implementation: check if the next state in the flow has capacity
	nextStates := k.getDownstreamStates(currentState)
	for _, nextState := range nextStates {
		if limit, ok := k.opts.WIPLimits[nextState]; ok && limit > 0 {
			if current, err := k.countIssuesInState(ctx, nextState); err == nil {
				if current >= limit {
					return false // Downstream is full
				}
			}
		}
	}
	return true
}

// getDownstreamStates returns the downstream states from current state
func (k *Kanban) getDownstreamStates(currentState string) []string {
	switch currentState {
	case "doing":
		return []string{"review"}
	case "review":
		return []string{"done"}
	default:
		return []string{}
	}
}

// CreateIssue creates a new issue with Kanban-specific handling
func (k *Kanban) CreateIssue(ctx context.Context, req paradigm.CreateIssueRequest) (*domain.Issue, error) {
	now := k.deps.Clock.Now()

	issue := &domain.Issue{
		Type:     req.Type,
		Title:    req.Title,
		Content:  req.Description,
		Status:   "backlog", // Always start in backlog for Kanban
		Priority: req.Priority,
		Assignee: req.Assignee,
		Labels:   req.Labels,
		Created:  now,
		Updated:  now,
	}

	// Apply paradigm-specific extensions
	// For Kanban, we might track cycle time start, etc.

	return issue, nil
}

// GetPlanningOperations returns planning operations for Kanban
func (k *Kanban) GetPlanningOperations() []paradigm.PlanningOperation {
	return []paradigm.PlanningOperation{
		{
			Name: "replenish",
			Run:  k.runReplenishment,
		},
	}
}

// GetExecutionOperations returns execution operations for Kanban
func (k *Kanban) GetExecutionOperations() []paradigm.ExecutionOperation {
	return []paradigm.ExecutionOperation{
		{
			Name: "pull",
			Run:  k.runPull,
		},
	}
}

// runReplenishment handles backlog replenishment
func (k *Kanban) runReplenishment(ctx context.Context, args map[string]any) (*paradigm.PlanningResult, error) {
	// Analyze current WIP utilization
	wipAnalysis := make(map[string]WIPAnalysis)

	for state, limit := range k.opts.WIPLimits {
		if limit <= 0 {
			continue // Skip unlimited states
		}

		current, err := k.countIssuesInState(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("failed to count issues in %s: %w", state, err)
		}

		utilization := float64(current) / float64(limit)
		wipAnalysis[state] = WIPAnalysis{
			State:       state,
			Current:     current,
			Limit:       limit,
			Utilization: utilization,
			Available:   limit - current,
		}
	}

	// Count issues in backlog
	backlogCount, err := k.countIssuesInState(ctx, "backlog")
	if err != nil {
		return nil, fmt.Errorf("failed to count backlog issues: %w", err)
	}

	// Generate replenishment recommendations
	recommendations := k.generateReplenishmentRecommendations(wipAnalysis, backlogCount)

	result := &paradigm.PlanningResult{
		Success: true,
		Message: fmt.Sprintf("Analyzed %d backlog items with %d recommendations",
			backlogCount, len(recommendations)),
		Data: map[string]any{
			"wip_analysis":    wipAnalysis,
			"backlog_count":   backlogCount,
			"recommendations": recommendations,
		},
	}

	return result, nil
}

// WIPAnalysis represents Work-In-Progress analysis for a state
type WIPAnalysis struct {
	State       string  `json:"state"`
	Current     int     `json:"current"`
	Limit       int     `json:"limit"`
	Utilization float64 `json:"utilization"`
	Available   int     `json:"available"`
}

// ReplenishmentRecommendation suggests actions for backlog management
type ReplenishmentRecommendation struct {
	Type        string `json:"type"` // "pull_ready", "capacity_warning", "bottleneck"
	State       string `json:"state"`
	Priority    string `json:"priority"` // "high", "medium", "low"
	Message     string `json:"message"`
	ActionItems int    `json:"action_items,omitempty"`
}

// generateReplenishmentRecommendations creates actionable recommendations
func (k *Kanban) generateReplenishmentRecommendations(analysis map[string]WIPAnalysis, backlogCount int) []ReplenishmentRecommendation {
	var recommendations []ReplenishmentRecommendation

	// Check if we can pull from backlog
	doingAnalysis, hasDoing := analysis["doing"]
	if hasDoing && doingAnalysis.Available > 0 {
		pullCount := doingAnalysis.Available
		if pullCount > backlogCount {
			pullCount = backlogCount
		}

		if pullCount > 0 {
			recommendations = append(recommendations, ReplenishmentRecommendation{
				Type:        "pull_ready",
				State:       "doing",
				Priority:    "high",
				Message:     fmt.Sprintf("Ready to pull %d item(s) from backlog to doing", pullCount),
				ActionItems: pullCount,
			})
		}
	}

	// Check for capacity warnings (>80% utilization)
	for state, analysis := range analysis {
		if analysis.Utilization > 0.8 {
			priority := "medium"
			if analysis.Utilization >= 1.0 {
				priority = "high"
			}

			recommendations = append(recommendations, ReplenishmentRecommendation{
				Type:     "capacity_warning",
				State:    state,
				Priority: priority,
				Message:  fmt.Sprintf("%s is at %.0f%% capacity (%d/%d)", state, analysis.Utilization*100, analysis.Current, analysis.Limit),
			})
		}
	}

	// Check for bottlenecks (downstream at capacity while upstream has available)
	if k.opts.BlockOnDownstreamFull {
		downstreamStates := map[string]string{
			"doing":  "review",
			"review": "done",
		}

		for upstream, downstream := range downstreamStates {
			upAnalysis, hasUp := analysis[upstream]
			downAnalysis, hasDown := analysis[downstream]

			if hasUp && hasDown && upAnalysis.Available > 0 && downAnalysis.Available == 0 {
				recommendations = append(recommendations, ReplenishmentRecommendation{
					Type:     "bottleneck",
					State:    downstream,
					Priority: "high",
					Message:  fmt.Sprintf("Bottleneck detected: %s is blocked by full %s", upstream, downstream),
				})
			}
		}
	}

	// Low backlog warning
	if backlogCount < 5 {
		recommendations = append(recommendations, ReplenishmentRecommendation{
			Type:     "low_backlog",
			State:    "backlog",
			Priority: "medium",
			Message:  fmt.Sprintf("Low backlog count (%d items) - consider adding more work", backlogCount),
		})
	}

	return recommendations
}

// runPull handles pulling work items
func (k *Kanban) runPull(ctx context.Context, args map[string]any) (*paradigm.ExecutionResult, error) {
	// Parse arguments
	targetState, _ := args["state"].(string)
	if targetState == "" {
		targetState = "doing" // Default to pulling into doing
	}

	maxPull, _ := args["max"].(int)
	if maxPull == 0 {
		maxPull = 10 // Default max pull
	}

	// Check if target state has WIP limits
	limit, hasLimit := k.opts.WIPLimits[targetState]
	if !hasLimit || limit <= 0 {
		return &paradigm.ExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Cannot pull to %s: no WIP limit configured", targetState),
		}, nil
	}

	// Count current items in target state
	currentCount, err := k.countIssuesInState(ctx, targetState)
	if err != nil {
		return nil, fmt.Errorf("failed to count issues in %s: %w", targetState, err)
	}

	// Calculate available capacity
	availableCapacity := limit - currentCount
	if availableCapacity <= 0 {
		return &paradigm.ExecutionResult{
			Success: false,
			Message: fmt.Sprintf("%s is at capacity (%d/%d)", targetState, currentCount, limit),
		}, nil
	}

	// Find source state (typically backlog)
	sourceState := k.getSourceState(targetState)

	// Get candidate issues for pulling
	candidates, err := k.getCandidatesForPull(ctx, sourceState, targetState)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull candidates: %w", err)
	}

	if len(candidates) == 0 {
		return &paradigm.ExecutionResult{
			Success: false,
			Message: fmt.Sprintf("No items available to pull from %s", sourceState),
		}, nil
	}

	// Determine how many to pull
	pullCount := availableCapacity
	if pullCount > len(candidates) {
		pullCount = len(candidates)
	}
	if pullCount > maxPull {
		pullCount = maxPull
	}

	// Select items to pull (prioritize by priority and age)
	itemsToPull := k.selectItemsToPull(candidates, pullCount)

	// Execute the pull (actual transitions would need to be done via the update API)
	pullActions := make([]PullAction, len(itemsToPull))
	for i, issue := range itemsToPull {
		pullActions[i] = PullAction{
			IssueID:   issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			FromState: sourceState,
			ToState:   targetState,
			Reason:    fmt.Sprintf("Pulled due to available capacity (%d/%d)", currentCount+i+1, limit),
		}
	}

	result := &paradigm.ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Ready to pull %d item(s) from %s to %s", len(pullActions), sourceState, targetState),
		Data: map[string]any{
			"source_state":       sourceState,
			"target_state":       targetState,
			"available_capacity": availableCapacity,
			"total_candidates":   len(candidates),
			"pull_actions":       pullActions,
		},
	}

	return result, nil
}

// PullAction represents an action to pull an item between states
type PullAction struct {
	IssueID   string `json:"issue_id"`
	Title     string `json:"title"`
	Priority  string `json:"priority"`
	FromState string `json:"from_state"`
	ToState   string `json:"to_state"`
	Reason    string `json:"reason"`
}

// getSourceState returns the typical source state for pulling into the target
func (k *Kanban) getSourceState(targetState string) string {
	switch targetState {
	case "doing":
		return "backlog"
	case "review":
		return "doing"
	default:
		return "backlog"
	}
}

// getCandidatesForPull retrieves issues that can be pulled from source to target
func (k *Kanban) getCandidatesForPull(ctx context.Context, sourceState, targetState string) ([]*domain.Issue, error) {
	// Get all issues in the source state
	issues, err := k.deps.Store.ListIssues(ctx, map[string]interface{}{
		"status": sourceState,
	})
	if err != nil {
		return nil, err
	}

	// Filter issues that can transition to target state
	var candidates []*domain.Issue
	validTransitions := k.GetValidTransitions(sourceState)

	canTransition := false
	for _, validTarget := range validTransitions {
		if validTarget == targetState {
			canTransition = true
			break
		}
	}

	if canTransition {
		candidates = issues
	}

	return candidates, nil
}

// selectItemsToPull selects the best items to pull based on priority and age
func (k *Kanban) selectItemsToPull(candidates []*domain.Issue, count int) []*domain.Issue {
	if count >= len(candidates) {
		return candidates
	}

	// Sort by priority (high first) then by created date (oldest first)
	sortedCandidates := make([]*domain.Issue, len(candidates))
	copy(sortedCandidates, candidates)

	// Simple priority-based sorting
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	// Sort candidates
	for i := 0; i < len(sortedCandidates)-1; i++ {
		for j := 0; j < len(sortedCandidates)-i-1; j++ {
			a, b := sortedCandidates[j], sortedCandidates[j+1]

			aPriority := priorityOrder[a.Priority]
			bPriority := priorityOrder[b.Priority]

			// Sort by priority first, then by age (older first)
			if aPriority > bPriority || (aPriority == bPriority && a.Created.After(b.Created)) {
				sortedCandidates[j], sortedCandidates[j+1] = sortedCandidates[j+1], sortedCandidates[j]
			}
		}
	}

	return sortedCandidates[:count]
}

// CalculateProgress calculates progress metrics for Kanban
func (k *Kanban) CalculateProgress(ctx context.Context, issues []*domain.Issue, pc paradigm.ProgressContext) (*paradigm.Progress, error) {
	// Calculate basic counts
	completed := 0
	inProgress := 0
	total := len(issues)

	// Track cycle times for velocity calculation
	var cycleTimes []float64
	timeRange := paradigm.TimeRange(pc)

	for _, issue := range issues {
		switch issue.Status {
		case "done":
			completed++
			// Calculate cycle time if we can determine when work started
			cycleTime := k.calculateCycleTime(issue)
			if cycleTime > 0 {
				cycleTimes = append(cycleTimes, cycleTime)
			}

		case "doing", "review":
			inProgress++
		}
	}

	// Calculate completion percentage
	completion := 0.0
	if total > 0 {
		completion = float64(completed) / float64(total)
	}

	// Calculate average cycle time (velocity proxy)
	velocity := 0.0
	if len(cycleTimes) > 0 {
		sum := 0.0
		for _, ct := range cycleTimes {
			sum += ct
		}
		velocity = sum / float64(len(cycleTimes))
	}

	// Generate flow charts
	charts := k.generateProgressCharts(issues, timeRange)

	return &paradigm.Progress{
		Completion: completion,
		Velocity:   velocity, // Average cycle time in days
		Charts:     charts,
	}, nil
}

// calculateCycleTime calculates the cycle time for an issue in days
func (k *Kanban) calculateCycleTime(issue *domain.Issue) float64 {
	if issue.Status != "done" {
		return 0
	}

	// Simple cycle time: created to updated (when marked done)
	// In a real implementation, you'd track state transition timestamps
	duration := issue.Updated.Sub(issue.Created)
	return duration.Hours() / 24.0 // Convert to days
}

// generateProgressCharts creates visual charts for progress tracking
func (k *Kanban) generateProgressCharts(issues []*domain.Issue, timeRange paradigm.TimeRange) []paradigm.Chart {
	chartList := []paradigm.Chart{}

	// Cumulative Flow Diagram data
	cfdData := k.generateCumulativeFlowData(issues, timeRange)
	if len(cfdData) > 0 {
		chart := charts.NewChart("cumulative_flow").WithData("cfd_data", cfdData).Build()
		chartList = append(chartList, chart)
	}

	// WIP visualization
	wipData := k.generateWIPChart(issues)
	if len(wipData) > 0 {
		chart := charts.NewChart("wip_limits").WithData("wip_data", wipData).Build()
		chartList = append(chartList, chart)
	}

	return chartList
}

// generateCumulativeFlowData creates data for cumulative flow diagram
func (k *Kanban) generateCumulativeFlowData(issues []*domain.Issue, timeRange paradigm.TimeRange) map[string]interface{} {
	// Simplified CFD - in practice, you'd need historical state transition data
	stateCounts := make(map[string]int)
	for _, issue := range issues {
		stateCounts[issue.Status]++
	}

	return map[string]interface{}{
		"states": []string{"backlog", "doing", "review", "done"},
		"data": map[string]int{
			"backlog": stateCounts["backlog"],
			"doing":   stateCounts["doing"],
			"review":  stateCounts["review"],
			"done":    stateCounts["done"],
		},
	}
}

// generateWIPChart creates WIP limit visualization data
func (k *Kanban) generateWIPChart(issues []*domain.Issue) map[string]interface{} {
	stateCounts := make(map[string]int)
	for _, issue := range issues {
		stateCounts[issue.Status]++
	}

	wipData := make([]map[string]interface{}, 0)
	for state, limit := range k.opts.WIPLimits {
		current := stateCounts[state]
		wipData = append(wipData, map[string]interface{}{
			"state":       state,
			"current":     current,
			"limit":       limit,
			"utilization": float64(current) / float64(limit),
		})
	}

	return map[string]interface{}{
		"wip_data": wipData,
	}
}
