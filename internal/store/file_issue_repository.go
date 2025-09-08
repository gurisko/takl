package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/shared"
)

// FileIssueRepository implements domain.IssueRepository using filesystem storage
// This repository is project-scoped - one instance per project
type FileIssueRepository struct {
	clock       domain.Clock
	idGenerator domain.IDGenerator
	projectID   string // The project this repository handles
	issuesDir   string // Project-specific issues directory
}

// NewFileIssueRepository creates a new file-based issue repository for a specific project
func NewFileIssueRepository(projectID, issuesDir string) *FileIssueRepository {
	return &FileIssueRepository{
		clock:       shared.DefaultClock{},
		idGenerator: shared.DefaultIDGenerator{},
		projectID:   projectID,
		issuesDir:   issuesDir,
	}
}

// Create implements domain.IssueRepository.Create
// NO VALIDATION - Store layer is "dumb", validation happens at API layer
func (r *FileIssueRepository) Create(ctx context.Context, req domain.CreateIssueRequest) (*domain.Issue, error) {
	now := r.clock.Now()

	issue := &domain.Issue{
		ID:       r.idGenerator.Generate(),
		Type:     req.Type,
		Title:    req.Title,
		Status:   "open", // Default status
		Priority: req.Priority,
		Assignee: req.Assignee,
		Labels:   req.Labels,
		Content:  req.Description,
		Created:  now,
		Updated:  now,
		Version:  1, // Start at version 1
	}

	// Set default priority if empty
	if issue.Priority == "" {
		issue.Priority = "medium"
	}

	// Determine file path
	typeDir := filepath.Join(r.issuesDir, issue.Type)

	// Create type directory if it doesn't exist
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create type directory: %w", err)
	}

	// Generate filename: {id}-{slug}.md
	slug := shared.Slugify(issue.Title)
	filename := fmt.Sprintf("%s-%s.md", issue.ID, slug)
	issue.FilePath = filepath.Join(typeDir, filename)

	// Save to file
	if err := r.saveIssueToFile(issue); err != nil {
		return nil, fmt.Errorf("failed to save issue to file: %w", err)
	}

	return issue, nil
}

// GetByID implements domain.IssueRepository.GetByID
func (r *FileIssueRepository) GetByID(ctx context.Context, projectID, issueID string) (*domain.Issue, error) {
	issue, err := r.findIssueFile(issueID)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %w", err)
	}
	return issue, nil
}

// Update implements domain.IssueRepository.Update
func (r *FileIssueRepository) Update(ctx context.Context, projectID, issueID string, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	// Load current issue
	issue, err := r.GetByID(ctx, projectID, issueID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Content != nil {
		issue.Content = *req.Content
	}
	if req.Status != nil {
		issue.Status = *req.Status
	}
	if req.Priority != nil {
		issue.Priority = *req.Priority
	}
	if req.Assignee != nil {
		issue.Assignee = *req.Assignee
	}
	if req.Labels != nil {
		issue.Labels = req.Labels
	}

	// Update metadata
	issue.Updated = r.clock.Now()
	issue.Version++ // Increment version for optimistic concurrency

	// NO VALIDATION - Store layer is 'dumb', validation happens at API layer

	// Save changes
	if err := r.saveIssueToFile(issue); err != nil {
		return nil, fmt.Errorf("failed to save updated issue: %w", err)
	}

	return issue, nil
}

// Delete implements domain.IssueRepository.Delete
func (r *FileIssueRepository) Delete(ctx context.Context, projectID, issueID string) error {
	issue, err := r.GetByID(ctx, projectID, issueID)
	if err != nil {
		return err
	}

	if err := os.Remove(issue.FilePath); err != nil {
		return fmt.Errorf("failed to delete issue file: %w", err)
	}

	return nil
}

// List implements domain.IssueRepository.List
func (r *FileIssueRepository) List(ctx context.Context, projectID string, filter domain.IssueFilter) ([]*domain.Issue, error) {
	issuesDir := r.issuesDir

	var issues []*domain.Issue

	// Walk through all issue files
	err := filepath.Walk(issuesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		issue, err := r.loadIssueFromFile(path)
		if err != nil {
			return nil // Skip invalid issues
		}

		// Apply filters
		if !r.matchesFilter(issue, filter) {
			return nil
		}

		issues = append(issues, issue)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Apply limit and offset
	if filter.Offset > 0 {
		if filter.Offset >= len(issues) {
			return []*domain.Issue{}, nil
		}
		issues = issues[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(issues) {
		issues = issues[:filter.Limit]
	}

	return issues, nil
}

// Search implements domain.IssueRepository.Search (basic text search)
func (r *FileIssueRepository) Search(ctx context.Context, projectID, query string) ([]*domain.Issue, error) {
	// For now, implement basic text search
	// The indexer layer provides full-text search capabilities
	allIssues, err := r.List(ctx, projectID, domain.IssueFilter{})
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var matches []*domain.Issue

	for _, issue := range allIssues {
		if strings.Contains(strings.ToLower(issue.Title), query) ||
			strings.Contains(strings.ToLower(issue.Content), query) {
			matches = append(matches, issue)
		}
	}

	return matches, nil
}

// ListAll implements domain.IssueRepository.ListAll
func (r *FileIssueRepository) ListAll(ctx context.Context, filter domain.IssueFilter) ([]*domain.Issue, error) {
	// For now, this would require scanning all project directories
	// This is a placeholder - in a real implementation, you'd iterate through all projects
	return nil, fmt.Errorf("ListAll not implemented for file repository")
}

// SearchAll implements domain.IssueRepository.SearchAll
func (r *FileIssueRepository) SearchAll(ctx context.Context, query string) ([]*domain.Issue, error) {
	// For now, this would require scanning all project directories
	// This is a placeholder - in a real implementation, you'd iterate through all projects
	return nil, fmt.Errorf("SearchAll not implemented for file repository")
}

// Helper methods

func (r *FileIssueRepository) findIssueFile(issueID string) (*domain.Issue, error) {

	// Search through issue type directories
	issueTypes := []string{"bug", "feature", "task", "epic"}

	for _, issueType := range issueTypes {
		typeDir := filepath.Join(r.issuesDir, issueType)
		if _, err := os.Stat(typeDir); os.IsNotExist(err) {
			continue
		}

		files, err := filepath.Glob(filepath.Join(typeDir, issueID+"-*.md"))
		if err != nil {
			continue
		}

		for _, file := range files {
			issue, err := r.loadIssueFromFile(file)
			if err != nil {
				continue
			}
			if issue.ID == issueID {
				return issue, nil
			}
		}
	}

	return nil, fmt.Errorf("issue %s not found in project %s", issueID, r.projectID)
}

func (r *FileIssueRepository) loadIssueFromFile(filePath string) (*domain.Issue, error) {
	return shared.LoadIssueFromFile(filePath)
}

func (r *FileIssueRepository) saveIssueToFile(issue *domain.Issue) error {
	return shared.SaveIssueToFile(issue)
}

func (r *FileIssueRepository) matchesFilter(issue *domain.Issue, filter domain.IssueFilter) bool {
	if filter.Status != "" && issue.Status != filter.Status {
		return false
	}
	if filter.Type != "" && issue.Type != filter.Type {
		return false
	}
	if filter.Priority != "" && issue.Priority != filter.Priority {
		return false
	}
	if filter.Assignee != "" && issue.Assignee != filter.Assignee {
		return false
	}
	if filter.Since != nil && issue.Created.Before(*filter.Since) {
		return false
	}
	if filter.Before != nil && issue.Created.After(*filter.Before) {
		return false
	}

	// Check labels if filter has labels
	if len(filter.Labels) > 0 {
		hasAllLabels := true
		for _, filterLabel := range filter.Labels {
			found := false
			for _, issueLabel := range issue.Labels {
				if issueLabel == filterLabel {
					found = true
					break
				}
			}
			if !found {
				hasAllLabels = false
				break
			}
		}
		if !hasAllLabels {
			return false
		}
	}

	return true
}
