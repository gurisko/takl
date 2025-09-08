package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/shared"
	"github.com/takl/takl/internal/store"
)

// IssueRepositoryAdapter adapts the existing store.LegacyManager to domain.IssueRepository
type IssueRepositoryAdapter struct {
	manager *store.LegacyManager
}

// NewIssueRepositoryAdapter creates a new adapter for store.LegacyManager
func NewIssueRepositoryAdapter(manager *store.LegacyManager) *IssueRepositoryAdapter {
	return &IssueRepositoryAdapter{
		manager: manager,
	}
}

// Create implements domain.IssueRepository.Create
func (a *IssueRepositoryAdapter) Create(ctx context.Context, req domain.CreateIssueRequest) (*domain.Issue, error) {
	opts := store.CreateOptions{
		Assignee: req.Assignee,
		Priority: req.Priority,
		Labels:   req.Labels,
		Content:  req.Description,
	}

	typesIssue, err := a.manager.Create(req.Type, req.Title, opts)
	if err != nil {
		return nil, err
	}

	return a.convertTypesToDomain(typesIssue), nil
}

// GetByID implements domain.IssueRepository.GetByID
func (a *IssueRepositoryAdapter) GetByID(ctx context.Context, projectID, issueID string) (*domain.Issue, error) {
	typesIssue, err := a.manager.LoadIssue(issueID)
	if err != nil {
		return nil, err
	}

	return a.convertTypesToDomain(typesIssue), nil
}

// Update implements domain.IssueRepository.Update
// Note: The existing issues.Manager doesn't have an update method, so we'll need to implement this
func (a *IssueRepositoryAdapter) Update(ctx context.Context, projectID, issueID string, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	// Load current issue
	currentIssue, err := a.manager.LoadIssue(issueID)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %w", err)
	}

	// Apply updates
	if req.Title != nil {
		currentIssue.Title = *req.Title
	}
	if req.Content != nil {
		currentIssue.Content = *req.Content
	}
	if req.Status != nil {
		currentIssue.Status = *req.Status
	}
	if req.Priority != nil {
		currentIssue.Priority = *req.Priority
	}
	if req.Assignee != nil {
		currentIssue.Assignee = *req.Assignee
	}
	if req.Labels != nil {
		currentIssue.Labels = req.Labels
	}

	// Update version and timestamp
	currentIssue.Version++
	currentIssue.Updated = time.Now()

	// NO VALIDATION - validation happens at API layer via centralized validator

	// Save the updated issue
	if err := shared.SaveIssueToFile(currentIssue); err != nil {
		return nil, fmt.Errorf("failed to save updated issue: %w", err)
	}

	// Update database if available
	if db := a.manager.GetDatabase(); db != nil {
		if err := db.SaveIssue(currentIssue); err != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to update database: %v\n", err)
		}
	}

	return a.convertTypesToDomain(currentIssue), nil
}

// Delete implements domain.IssueRepository.Delete
// Note: The existing issues.Manager doesn't have a delete method, so we'll need to implement this
func (a *IssueRepositoryAdapter) Delete(ctx context.Context, projectID, issueID string) error {
	// Load issue to get file path
	issue, err := a.manager.LoadIssue(issueID)
	if err != nil {
		return fmt.Errorf("issue not found: %w", err)
	}

	// Delete from database if available
	if db := a.manager.GetDatabase(); db != nil {
		if err := db.DeleteIssue(issueID); err != nil {
			// Log warning but continue with file deletion
			fmt.Printf("Warning: failed to delete from database: %v\n", err)
		}
	}

	// Delete file
	return os.Remove(issue.FilePath)
}

// List implements domain.IssueRepository.List
func (a *IssueRepositoryAdapter) List(ctx context.Context, projectID string, filter domain.IssueFilter) ([]*domain.Issue, error) {
	// Convert domain filter to manager filter format
	managerFilters := map[string]interface{}{}

	if filter.Status != "" {
		managerFilters["status"] = filter.Status
	}
	if filter.Type != "" {
		managerFilters["type"] = filter.Type
	}
	if filter.Priority != "" {
		managerFilters["priority"] = filter.Priority
	}
	if filter.Assignee != "" {
		managerFilters["assignee"] = filter.Assignee
	}
	if len(filter.Labels) > 0 {
		managerFilters["labels"] = filter.Labels
	}
	if filter.Limit > 0 {
		managerFilters["limit"] = filter.Limit
	}
	if filter.Offset > 0 {
		managerFilters["offset"] = filter.Offset
	}

	typesIssues, err := a.manager.ListIssues(managerFilters)
	if err != nil {
		return nil, err
	}

	// Convert to domain issues
	domainIssues := make([]*domain.Issue, len(typesIssues))
	for i, issue := range typesIssues {
		domainIssues[i] = a.convertTypesToDomain(issue)
	}

	// Apply additional filters that manager doesn't support
	domainIssues = a.applyAdditionalFilters(domainIssues, filter)

	return domainIssues, nil
}

// Search implements domain.IssueRepository.Search
func (a *IssueRepositoryAdapter) Search(ctx context.Context, projectID, query string) ([]*domain.Issue, error) {
	typesIssues, err := a.manager.SearchIssues(query)
	if err != nil {
		return nil, err
	}

	// Convert to domain issues
	domainIssues := make([]*domain.Issue, len(typesIssues))
	for i, issue := range typesIssues {
		domainIssues[i] = a.convertTypesToDomain(issue)
	}

	return domainIssues, nil
}

// ListAll implements domain.IssueRepository.ListAll
// Note: This would require scanning all projects, not supported by current manager
func (a *IssueRepositoryAdapter) ListAll(ctx context.Context, filter domain.IssueFilter) ([]*domain.Issue, error) {
	return nil, fmt.Errorf("ListAll not supported by adapter - requires multi-project scanning")
}

// SearchAll implements domain.IssueRepository.SearchAll
// Note: This would require scanning all projects, not supported by current manager
func (a *IssueRepositoryAdapter) SearchAll(ctx context.Context, query string) ([]*domain.Issue, error) {
	return nil, fmt.Errorf("SearchAll not supported by adapter - requires multi-project scanning")
}

// Helper methods

// convertTypesToDomain converts domain.Issue to domain.Issue
func (a *IssueRepositoryAdapter) convertTypesToDomain(issue *domain.Issue) *domain.Issue {
	return &domain.Issue{
		ID:       issue.ID,
		Type:     issue.Type,
		Title:    issue.Title,
		Status:   issue.Status,
		Priority: issue.Priority,
		Assignee: issue.Assignee,
		Labels:   issue.Labels,
		Created:  issue.Created,
		Updated:  issue.Updated,
		Version:  issue.Version,
		FilePath: issue.FilePath,
		Content:  issue.Content,
	}
}

// applyAdditionalFilters applies filters that the manager doesn't support
func (a *IssueRepositoryAdapter) applyAdditionalFilters(issues []*domain.Issue, filter domain.IssueFilter) []*domain.Issue {
	filtered := make([]*domain.Issue, 0, len(issues))

	for _, issue := range issues {
		// Date filters
		if filter.Since != nil && issue.Created.Before(*filter.Since) {
			continue
		}
		if filter.Before != nil && issue.Created.After(*filter.Before) {
			continue
		}

		// Label matching (if manager doesn't handle it properly)
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
				continue
			}
		}

		filtered = append(filtered, issue)
	}

	// Apply pagination if not handled by manager
	if filter.Offset > 0 {
		if filter.Offset >= len(filtered) {
			return []*domain.Issue{}
		}
		filtered = filtered[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(filtered) {
		filtered = filtered[:filter.Limit]
	}

	return filtered
}
