package database

import (
	"time"

	"github.com/takl/takl/internal/domain"
)

// Database provides a queryable interface for issues
type Database interface {
	// Initialize creates tables and indexes
	Initialize() error

	// Close closes the database connection
	Close() error

	// Issue operations
	SaveIssue(issue *domain.Issue) error
	GetIssue(id string) (*domain.Issue, error)
	ListIssues(filters ListFilters) ([]*domain.Issue, error)
	UpdateIssue(issue *domain.Issue) error
	DeleteIssue(id string) error

	// Search and filtering
	SearchIssues(query string, filters ListFilters) ([]*domain.Issue, error)

	// Statistics
	GetIssueStats() (*IssueStats, error)
}

// ListFilters provides filtering options for listing issues
type ListFilters struct {
	Type     string // bug, feature, task, epic
	Status   string // open, in-progress, done, archived
	Priority string // low, medium, high, critical
	Assignee string
	Labels   []string
	Since    *time.Time // Issues created/updated since
	Before   *time.Time // Issues created/updated before
	Limit    int
	Offset   int
}

// IssueStats provides summary statistics
type IssueStats struct {
	Total           int
	ByType          map[string]int
	ByStatus        map[string]int
	ByPriority      map[string]int
	OpenCount       int
	InProgressCount int
	DoneCount       int
	ArchivedCount   int
}
