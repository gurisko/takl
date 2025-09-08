package indexer

import (
	"context"
	"time"

	"github.com/takl/takl/internal/domain"
)

// Filters represents filtering criteria for index queries
type Filters struct {
	ProjectIDs []string               `json:"project_ids,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Priority   string                 `json:"priority,omitempty"`
	Assignee   string                 `json:"assignee,omitempty"`
	Labels     []string               `json:"labels,omitempty"`
	Since      *time.Time             `json:"since,omitempty"`
	Before     *time.Time             `json:"before,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"` // Paradigm-specific filters
}

// Hit represents a search result hit from the index
type Hit struct {
	Issue       *domain.Issue `json:"issue"`
	Score       float64       `json:"score"`
	Highlights  []string      `json:"highlights,omitempty"`
	Explanation string        `json:"explanation,omitempty"`
}

// Row represents a structured list result from the index
type Row struct {
	Issue    *domain.Issue          `json:"issue"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Status represents the current status of the index
type Status struct {
	Healthy        bool                   `json:"healthy"`
	TotalDocuments int64                  `json:"total_documents"`
	LastIndexed    *time.Time             `json:"last_indexed,omitempty"`
	IndexSize      int64                  `json:"index_size_bytes"`
	Projects       map[string]interface{} `json:"projects"`
	Errors         []string               `json:"errors,omitempty"`
	Version        string                 `json:"version"`
}

// Index defines the core indexing interface for fast search and retrieval
// This interface provides mockable search operations without exposing
// implementation details to handlers or services
type Index interface {
	// Core index operations
	Upsert(ctx context.Context, projectID string, issue *domain.Issue) error
	DeleteByID(ctx context.Context, projectID, issueID string) error
	DeleteByPath(ctx context.Context, path string) error

	// Search operations
	Search(ctx context.Context, projectID, query string, f Filters) ([]Hit, error)
	SearchGlobal(ctx context.Context, query string, f Filters) ([]Hit, error)

	// List operations (structured queries without text search)
	List(ctx context.Context, projectID string, f Filters) ([]Row, error)
	ListGlobal(ctx context.Context, f Filters) ([]Row, error)

	// Index management
	Status(ctx context.Context) Status
	Rebuild(ctx context.Context, projectID string) error
	Optimize(ctx context.Context) error

	// Health and diagnostics
	Health(ctx context.Context) (map[string]interface{}, error)
}

// MultiProjectIndex extends Index for multi-project operations
type MultiProjectIndex interface {
	Index

	// Project lifecycle
	AddProject(ctx context.Context, projectID string, config map[string]interface{}) error
	RemoveProject(ctx context.Context, projectID string) error
	GetProjectStatus(ctx context.Context, projectID string) (map[string]interface{}, error)
}

// FromDomainFilter converts domain filter to indexer filter
func FromDomainFilter(df domain.IssueFilter) Filters {
	return Filters{
		Status:   df.Status,
		Type:     df.Type,
		Priority: df.Priority,
		Assignee: df.Assignee,
		Labels:   df.Labels,
		Since:    df.Since,
		Before:   df.Before,
		Limit:    df.Limit,
		Offset:   df.Offset,
	}
}

// ToHits converts a slice of domain issues to search hits
func ToHits(issues []*domain.Issue) []Hit {
	hits := make([]Hit, len(issues))
	for i, issue := range issues {
		hits[i] = Hit{
			Issue: issue,
			Score: 1.0, // Default score
		}
	}
	return hits
}

// ToRows converts a slice of domain issues to list rows
func ToRows(issues []*domain.Issue) []Row {
	rows := make([]Row, len(issues))
	for i, issue := range issues {
		rows[i] = Row{
			Issue: issue,
		}
	}
	return rows
}
