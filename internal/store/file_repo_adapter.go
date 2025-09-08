package store

import (
	"context"
	"time"

	"github.com/takl/takl/internal/domain"
)

// FileRepoAdapter adapts the existing FileIssueRepository to implement the new Repo interface
// This allows gradual migration from the domain.IssueRepository interface to the store.Repo interface
type FileRepoAdapter struct {
	legacyRepo domain.IssueRepository
}

// NewFileRepoAdapter creates a new adapter that wraps a legacy repository
func NewFileRepoAdapter(legacyRepo domain.IssueRepository) *FileRepoAdapter {
	return &FileRepoAdapter{
		legacyRepo: legacyRepo,
	}
}

// LoadIssue implements Repo.LoadIssue by wrapping the legacy GetByID method
func (a *FileRepoAdapter) LoadIssue(ctx context.Context, projectID, issueID string) (*domain.Issue, error) {
	return a.legacyRepo.GetByID(ctx, projectID, issueID)
}

// SaveIssue implements Repo.SaveIssue
// This requires creating the issue using the legacy Create method or updating with Update method
func (a *FileRepoAdapter) SaveIssue(ctx context.Context, projectID string, issue *domain.Issue) error {
	// Check if issue already exists by trying to load it
	existingIssue, err := a.legacyRepo.GetByID(ctx, projectID, issue.ID)

	if err != nil {
		// Issue doesn't exist, create it
		req := domain.CreateIssueRequest{
			Type:        issue.Type,
			Title:       issue.Title,
			Description: issue.Content,
			Assignee:    issue.Assignee,
			Labels:      issue.Labels,
			Priority:    issue.Priority,
		}

		_, createErr := a.legacyRepo.Create(ctx, req)
		return createErr
	} else {
		// Issue exists, update it
		req := domain.UpdateIssueRequest{}

		// Only update fields that have changed
		if existingIssue.Title != issue.Title {
			req.Title = &issue.Title
		}
		if existingIssue.Content != issue.Content {
			req.Content = &issue.Content
		}
		if existingIssue.Status != issue.Status {
			req.Status = &issue.Status
		}
		if existingIssue.Priority != issue.Priority {
			req.Priority = &issue.Priority
		}
		if existingIssue.Assignee != issue.Assignee {
			req.Assignee = &issue.Assignee
		}

		// Always update labels to handle additions/removals
		req.Labels = issue.Labels

		_, updateErr := a.legacyRepo.Update(ctx, projectID, issue.ID, req)
		return updateErr
	}
}

// ListIssues implements Repo.ListIssues by converting filters and calling legacy List method
func (a *FileRepoAdapter) ListIssues(ctx context.Context, projectID string, f Filters) ([]*domain.Issue, error) {
	// Convert store.Filters to domain.IssueFilter
	domainFilter := a.toDomainFilter(f)
	return a.legacyRepo.List(ctx, projectID, domainFilter)
}

// DeleteIssue implements Repo.DeleteIssue by calling legacy Delete method
func (a *FileRepoAdapter) DeleteIssue(ctx context.Context, projectID, issueID string) error {
	return a.legacyRepo.Delete(ctx, projectID, issueID)
}

// ListAllIssues implements Repo.ListAllIssues by calling legacy ListAll method
func (a *FileRepoAdapter) ListAllIssues(ctx context.Context, f Filters) ([]*domain.Issue, error) {
	domainFilter := a.toDomainFilter(f)
	return a.legacyRepo.ListAll(ctx, domainFilter)
}

// Health implements Repo.Health by returning basic health info
func (a *FileRepoAdapter) Health(ctx context.Context, projectID string) (map[string]interface{}, error) {
	// Try to list issues to check if repository is healthy
	_, err := a.legacyRepo.List(ctx, projectID, domain.IssueFilter{Limit: 1})
	if err != nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}, err
	}

	// Get total count
	allIssues, countErr := a.legacyRepo.List(ctx, projectID, domain.IssueFilter{})
	issueCount := 0
	if countErr == nil {
		issueCount = len(allIssues)
	}

	return map[string]interface{}{
		"healthy":     true,
		"project_id":  projectID,
		"issue_count": issueCount,
		"adapter":     "file_repo_adapter",
	}, nil
}

// Helper methods

// toDomainFilter converts store.Filters to domain.IssueFilter
func (a *FileRepoAdapter) toDomainFilter(f Filters) domain.IssueFilter {
	domainFilter := domain.IssueFilter{
		Status:   f.Status,
		Type:     f.Type,
		Priority: f.Priority,
		Assignee: f.Assignee,
		Labels:   f.Labels,
		Limit:    f.Limit,
		Offset:   f.Offset,
	}

	// Convert string dates back to time.Time if present
	if f.Since != nil {
		if sinceTime, err := time.Parse("2006-01-02T15:04:05Z", *f.Since); err == nil {
			domainFilter.Since = &sinceTime
		}
	}

	if f.Before != nil {
		if beforeTime, err := time.Parse("2006-01-02T15:04:05Z", *f.Before); err == nil {
			domainFilter.Before = &beforeTime
		}
	}

	return domainFilter
}
