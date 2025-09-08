package domain

import (
	"context"
	"errors"
	"time"
)

// Paradigm-specific error types for user-friendly CLI/HTTP responses
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

// IssueRepository defines the interface for issue persistence
type IssueRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, req CreateIssueRequest) (*Issue, error)
	GetByID(ctx context.Context, projectID, issueID string) (*Issue, error)
	Update(ctx context.Context, projectID, issueID string, req UpdateIssueRequest) (*Issue, error)
	Delete(ctx context.Context, projectID, issueID string) error

	// Query operations
	List(ctx context.Context, projectID string, filter IssueFilter) ([]*Issue, error)
	Search(ctx context.Context, projectID, query string) ([]*Issue, error)

	// Batch operations
	ListAll(ctx context.Context, filter IssueFilter) ([]*Issue, error)
	SearchAll(ctx context.Context, query string) ([]*Issue, error)
}

// ProjectRepository defines the interface for project management
type ProjectRepository interface {
	Register(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, projectID string) (*Project, error)
	List(ctx context.Context) ([]*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, projectID string) error
	Health(ctx context.Context, projectID string) (map[string]interface{}, error)
}

// Indexer defines the interface for search indexing
type Indexer interface {
	Upsert(ctx context.Context, projectID string, issue *Issue) error
	Delete(ctx context.Context, projectID, issueID string) error
	Search(ctx context.Context, projectID, query string) ([]*Issue, error)
	GetStats(ctx context.Context, projectID string) (map[string]interface{}, error)
}

// EventType represents the type of filesystem event
type EventType int

const (
	EventUpsert EventType = iota
	EventDelete
	EventRename
)

// FileEvent represents a filesystem change event
type FileEvent struct {
	ProjectID string
	Path      string
	Type      EventType
	Timestamp time.Time
}

// Watcher defines the interface for filesystem monitoring
type Watcher interface {
	Start(ctx context.Context) error
	Stop() error
	AddProject(projectID, issuesDir string) error
	RemoveProject(projectID string) error
	Events() <-chan FileEvent
	GetStats() map[string]interface{}
}

// ParadigmTransition represents a workflow state transition
type ParadigmTransition struct {
	From   string
	To     string
	Reason string
}

// Category represents the high-level classification of a paradigm
type ParadigmCategory int

const (
	CategoryAgile ParadigmCategory = iota
	CategoryLean
	CategoryService
)

func (c ParadigmCategory) String() string {
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

// TimeModel represents how paradigms handle time
type TimeModel int

const (
	TimeIterative   TimeModel = iota // Scrum (sprints)
	TimeContinuous                   // Kanban (flow)
	TimeEventDriven                  // Support (reactive)
)

// WorkUnit represents the unit of work
type WorkUnit int

const (
	WorkStoryPoints WorkUnit = iota // Scrum
	WorkTasks                       // Kanban
	WorkTickets                     // Support
)

// WorkflowState represents a state in the paradigm's workflow
type WorkflowState struct {
	Key         string                                                // e.g. "backlog","sprint_backlog","doing","done"
	DisplayName string                                                // Human-readable name
	Guards      []func(context.Context, *Issue, string, string) error // Validation functions run on entry
}

// Progress represents calculated progress metrics
type Progress struct {
	Completion float64    `json:"completion"` // 0.0 to 1.0
	Velocity   float64    `json:"velocity"`   // units per time period
	Charts     []Chart    `json:"charts"`
	Prediction *time.Time `json:"prediction,omitempty"` // predicted completion
}

// Chart represents a data visualization
type Chart struct {
	Type string `json:"type"` // "burndown", "cfd", "velocity"
	Data any    `json:"data"`
}

// TimeRange for progress calculations
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ProgressContext provides context for progress calculations
type ProgressContext struct {
	TimeRange TimeRange              `json:"time_range"`
	Filters   map[string]interface{} `json:"filters"`
}

// PlanningOperation represents a planning-phase operation
type PlanningOperation struct {
	Name string                                                                `json:"name"`
	Run  func(ctx context.Context, issues []*Issue, args map[string]any) error `json:"-"`
}

// ExecutionOperation represents an execution-phase operation
type ExecutionOperation struct {
	Name string                                                                `json:"name"`
	Run  func(ctx context.Context, issues []*Issue, args map[string]any) error `json:"-"`
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

// Paradigm defines the interface for workflow paradigms (Scrum, Kanban, etc.)
type Paradigm interface {
	// Identity
	ID() string
	Name() string
	Category() ParadigmCategory

	// Lifecycle
	Init(ctx context.Context, deps ParadigmDeps, rawOptions map[string]any) error

	// Work model
	GetTimeModel() TimeModel
	GetWorkUnit() WorkUnit

	// Workflow states
	GetWorkflowStates() []WorkflowState
	GetValidTransitions(currentState string) []string
	ValidateTransition(ctx context.Context, issue *Issue, from, to string) error

	// Issue management
	CreateIssue(ctx context.Context, req CreateIssueRequest) (*Issue, error)

	// Operations
	GetPlanningOperations() []PlanningOperation
	GetExecutionOperations() []ExecutionOperation

	// Analytics
	CalculateProgress(ctx context.Context, issues []*Issue, pc ProgressContext) (*Progress, error)
}

// ParadigmDeps represents dependencies injected into paradigms
type ParadigmDeps struct {
	Clock Clock
	Repo  ParadigmRepo
	Store ParadigmStorage
	Log   ParadigmLogger
}

// ParadigmRepo interface for git operations
type ParadigmRepo interface {
	Root() string
	IsClean(ctx context.Context) (bool, error)
}

// ParadigmStorage interface for issue persistence
type ParadigmStorage interface {
	ListIssues(ctx context.Context, filters map[string]interface{}) ([]*Issue, error)
	SaveIssue(ctx context.Context, iss *Issue) error
	LoadIssue(ctx context.Context, id string) (*Issue, error)
}

// ParadigmLogger interface for structured logging
type ParadigmLogger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// ParadigmRegistry manages available paradigms
type ParadigmRegistry interface {
	Register(paradigm Paradigm) error
	Get(id string) (Paradigm, error)
	List() []Paradigm
}

// Clock provides time operations (useful for testing)
type Clock interface {
	Now() time.Time
}

// IDGenerator provides unique ID generation
type IDGenerator interface {
	Generate() string
}
