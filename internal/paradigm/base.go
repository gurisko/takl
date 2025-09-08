package paradigm

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/takl/takl/internal/domain"
)

// Category represents the high-level classification of a paradigm
type Category int

const (
	CategoryAgile Category = iota
	CategoryLean
	CategoryService
)

func (c Category) String() string {
	switch c {
	case CategoryAgile:
		return "agile"
	case CategoryLean:
		return "lean"
	case CategoryService:
		return "service"
	default:
		return "unknown"
	}
}

// Typed errors for user-friendly CLI/HTTP responses
var (
	ErrCapacityExceeded  = errors.New("capacity_exceeded")
	ErrInvalidTransition = errors.New("invalid_transition")
	ErrWIPLimitExceeded  = errors.New("wip_limit_exceeded")
	ErrMissingEstimate   = errors.New("missing_estimate")
	ErrSprintNotActive   = errors.New("sprint_not_active")
	ErrDefinitionOfDone  = errors.New("definition_of_done_not_met")
	ErrIssueNotFound     = errors.New("issue_not_found")
)

// Error checking helpers
func IsWIPLimitExceeded(err error) bool {
	return errors.Is(err, ErrWIPLimitExceeded)
}

func IsCapacityExceeded(err error) bool {
	return errors.Is(err, ErrCapacityExceeded)
}

func IsInvalidTransition(err error) bool {
	return errors.Is(err, ErrInvalidTransition)
}

// Core dependency interfaces for testing seams
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

// DefaultClock implements Clock using system time
type DefaultClock struct{}

func (c DefaultClock) Now() time.Time {
	return time.Now()
}

func (c DefaultClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

type Repo interface {
	Root() string
	IsClean(ctx context.Context) (bool, error)
}

type Storage interface {
	ListIssues(ctx context.Context, filters map[string]interface{}) ([]*domain.Issue, error)
	SaveIssue(ctx context.Context, iss *domain.Issue) error
	LoadIssue(ctx context.Context, id string) (*domain.Issue, error)
}

// Dependencies bag passed to paradigms
type Deps struct {
	Clock Clock
	Repo  Repo
	Store Storage
	Log   *slog.Logger
}

// Guard functions for workflow state validation
type Guard func(ctx context.Context, issue *domain.Issue, from, to string) error

// WorkflowState represents a state in the paradigm's workflow
type WorkflowState struct {
	Key         string  // e.g. "backlog","sprint_backlog","doing","done"
	DisplayName string  // Human-readable name
	Guards      []Guard // Validation functions run on entry
}

// Transition represents a valid state change
type Transition struct {
	From string
	To   string
}

// TimeModel defines how the paradigm handles time
type TimeModel struct {
	Kind   string         // "timeboxed" | "continuous"
	Params map[string]any // Sprint length, cadences, etc.
}

// WorkUnit defines the type of work items the paradigm handles
type WorkUnit struct {
	Kind   string   // "story" | "task" | "ticket"
	Fields []string // Required fields for this work unit
}

// Operation represents paradigm-specific operations
type PlanningOperation struct {
	Name string // "sprint_plan", "groom"
	Run  func(ctx context.Context, args map[string]any) (*PlanningResult, error)
}

type ExecutionOperation struct {
	Name string // "start_sprint", "pull", "escalate"
	Run  func(ctx context.Context, args map[string]any) (*ExecutionResult, error)
}

// Results from paradigm operations
type PlanningResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ExecutionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Progress calculation context
type ProgressContext struct {
	Start time.Time
	End   time.Time
}

// Progress represents calculated progress metrics
type Progress struct {
	Completion float64    `json:"completion"` // 0.0 to 1.0
	Velocity   float64    `json:"velocity"`   // Work units per time period
	Charts     []Chart    `json:"charts"`
	Prediction *time.Time `json:"prediction,omitempty"`
}

// TimeRange for metrics calculation
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Chart represents a data visualization
type Chart struct {
	Type string `json:"type"` // "burndown", "cfd", "velocity"
	Data any    `json:"data"`
}

// CreateIssueRequest for paradigm-specific issue creation
type CreateIssueRequest struct {
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Assignee    string         `json:"assignee"`
	Labels      []string       `json:"labels"`
	Priority    string         `json:"priority"`
	Extensions  map[string]any `json:"extensions"` // Paradigm-specific fields
}

// Main paradigm interface
type Paradigm interface {
	// Identity
	ID() string
	Name() string
	Category() Category

	// Lifecycle
	Init(ctx context.Context, deps Deps, rawOptions map[string]any) error

	// Work model
	GetTimeModel() TimeModel
	GetWorkUnit() WorkUnit

	// Workflow
	GetWorkflowStates() []WorkflowState
	GetValidTransitions(currentState string) []string
	ValidateTransition(ctx context.Context, issue *domain.Issue, from, to string) error

	// Issue management
	CreateIssue(ctx context.Context, req CreateIssueRequest) (*domain.Issue, error)

	// Operations
	GetPlanningOperations() []PlanningOperation
	GetExecutionOperations() []ExecutionOperation

	// Analytics
	CalculateProgress(ctx context.Context, issues []*domain.Issue, pc ProgressContext) (*Progress, error)
}
