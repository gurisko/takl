package store

import (
	"context"

	"github.com/takl/takl/internal/domain"
)

// Filters represents filtering criteria for repository queries
type Filters struct {
	Status   string
	Type     string
	Priority string
	Assignee string
	Labels   []string
	Since    *string // ISO date string for easier serialization
	Before   *string // ISO date string for easier serialization
	Limit    int
	Offset   int
}

// Repo defines the core repository interface for issue persistence
// This interface provides mockable storage operations without exposing
// implementation details to handlers or services
type Repo interface {
	// Core CRUD operations
	LoadIssue(ctx context.Context, projectID, issueID string) (*domain.Issue, error)
	SaveIssue(ctx context.Context, projectID string, issue *domain.Issue) error
	ListIssues(ctx context.Context, projectID string, f Filters) ([]*domain.Issue, error)
	DeleteIssue(ctx context.Context, projectID, issueID string) error

	// Batch operations
	ListAllIssues(ctx context.Context, f Filters) ([]*domain.Issue, error)

	// Repository health and status
	Health(ctx context.Context, projectID string) (map[string]interface{}, error)
}

// ProjectRepo defines the interface for project management operations
type ProjectRepo interface {
	// Project lifecycle
	RegisterProject(ctx context.Context, project *domain.Project) error
	GetProject(ctx context.Context, projectID string) (*domain.Project, error)
	ListProjects(ctx context.Context) ([]*domain.Project, error)
	UpdateProject(ctx context.Context, project *domain.Project) error
	DeleteProject(ctx context.Context, projectID string) error

	// Project health and status
	HealthCheck(ctx context.Context, projectID string) (map[string]interface{}, error)
}

// FromDomainFilter converts domain filter to store filter
func FromDomainFilter(df domain.IssueFilter) Filters {
	var since, before *string

	if df.Since != nil {
		s := df.Since.Format("2006-01-02T15:04:05Z")
		since = &s
	}
	if df.Before != nil {
		b := df.Before.Format("2006-01-02T15:04:05Z")
		before = &b
	}

	return Filters{
		Status:   df.Status,
		Type:     df.Type,
		Priority: df.Priority,
		Assignee: df.Assignee,
		Labels:   df.Labels,
		Since:    since,
		Before:   before,
		Limit:    df.Limit,
		Offset:   df.Offset,
	}
}
