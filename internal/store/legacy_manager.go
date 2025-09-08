package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/takl/takl/internal/config"
	"github.com/takl/takl/internal/database"
	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/git"
	"github.com/takl/takl/internal/shared"
)

// CreateOptions matches the legacy issues package structure for compatibility
type CreateOptions struct {
	Priority string
	Assignee string
	Labels   []string
	Content  string
}

// LegacyManager provides the same interface as issues.Manager but using the new store layer
// This is a temporary bridge during migration
type LegacyManager struct {
	config    *config.Config
	repoPath  string
	gitRepo   git.Repository
	issuesDir string
	clock     domain.Clock
	db        *database.DB
	projectID string
	repo      *FileIssueRepository
}

// NewLegacyManager creates a new legacy manager wrapper
func NewLegacyManager(repoPath string) (*LegacyManager, error) {
	config := &config.Config{} // TODO: Load actual config

	// Initialize git repo (simplified for now)
	var gitRepo git.Repository
	// TODO: Initialize git repo properly

	// Determine mode and issues directory
	var issuesDir string
	if _, err := os.Stat(filepath.Join(repoPath, ".takl")); err == nil {
		// Embedded mode
		issuesDir = filepath.Join(repoPath, ".takl", "issues")
	} else {
		// Standalone mode
		issuesDir = filepath.Join(repoPath, ".issues")
	}

	// Create issues directory if it doesn't exist
	if err := os.MkdirAll(issuesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create issues directory: %w", err)
	}

	repo := NewFileIssueRepository("default", issuesDir)

	return &LegacyManager{
		config:    config,
		repoPath:  repoPath,
		gitRepo:   gitRepo,
		issuesDir: issuesDir,
		clock:     shared.DefaultClock{},
		projectID: "default",
		repo:      repo,
	}, nil
}

// NewLegacyManagerWithDatabase creates a manager with database support
func NewLegacyManagerWithDatabase(repoPath, projectID string, db *database.DB) (*LegacyManager, error) {
	manager, err := NewLegacyManager(repoPath)
	if err != nil {
		return nil, err
	}

	manager.db = db
	manager.projectID = projectID
	return manager, nil
}

// Create creates a new issue using the store layer
func (m *LegacyManager) Create(issueType, title string, opts CreateOptions) (*domain.Issue, error) {
	req := domain.CreateIssueRequest{
		Type:        issueType,
		Title:       title,
		Description: opts.Content,
		Priority:    opts.Priority,
		Assignee:    opts.Assignee,
		Labels:      opts.Labels,
	}

	return m.repo.Create(context.Background(), req)
}

// LoadIssue loads an issue by ID
func (m *LegacyManager) LoadIssue(issueID string) (*domain.Issue, error) {
	return m.repo.GetByID(context.Background(), m.projectID, issueID)
}

// ListIssues lists issues with optional filters
func (m *LegacyManager) ListIssues(filters map[string]interface{}) ([]*domain.Issue, error) {
	filter := m.convertMapToFilter(filters)
	return m.repo.List(context.Background(), m.projectID, filter)
}

// SearchIssues performs full-text search
func (m *LegacyManager) SearchIssues(query string) ([]*domain.Issue, error) {
	return m.repo.Search(context.Background(), m.projectID, query)
}

// GetProjectID returns the project ID
func (m *LegacyManager) GetProjectID() string {
	return m.projectID
}

// GetDatabase returns the database connection
func (m *LegacyManager) GetDatabase() *database.DB {
	return m.db
}

// Helper function to convert legacy filters to domain.IssueFilter
func (m *LegacyManager) convertMapToFilter(filters map[string]interface{}) domain.IssueFilter {
	filter := domain.IssueFilter{}

	if status, ok := filters["status"].(string); ok {
		filter.Status = status
	}
	if issueType, ok := filters["type"].(string); ok {
		filter.Type = issueType
	}
	if priority, ok := filters["priority"].(string); ok {
		filter.Priority = priority
	}
	if assignee, ok := filters["assignee"].(string); ok {
		filter.Assignee = assignee
	}
	if labels, ok := filters["labels"].([]string); ok {
		filter.Labels = labels
	}

	return filter
}

// ValidateCreateOptions validates create options
// NOTE: Validation removed - should happen at API layer
func (m *LegacyManager) ValidateCreateOptions(opts *CreateOptions) error {
	// NO VALIDATION - Legacy layer should not validate, API layer handles this
	return nil
}

// Legacy validation functions removed - validation moved to API layer
// These functions are no longer needed as validation is centralized

// Legacy constants for backward compatibility
var (
	ValidIssueTypes  = domain.ValidIssueTypes
	ValidPriorities  = domain.ValidPriorities
	DefaultPriority  = domain.DefaultPriority
	DefaultIssueType = domain.DefaultIssueType
)
