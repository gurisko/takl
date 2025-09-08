package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/takl/takl/internal/domain"
)

// LegacyIndexAdapter adapts the existing indexer to implement the new Index interface
// This allows gradual migration from the domain.Indexer interface to the indexer.Index interface
type LegacyIndexAdapter struct {
	legacyIndexer domain.Indexer
}

// NewLegacyIndexAdapter creates a new adapter that wraps a legacy indexer
func NewLegacyIndexAdapter(legacyIndexer domain.Indexer) *LegacyIndexAdapter {
	return &LegacyIndexAdapter{
		legacyIndexer: legacyIndexer,
	}
}

// Upsert implements Index.Upsert by calling legacy Upsert method
func (a *LegacyIndexAdapter) Upsert(ctx context.Context, projectID string, issue *domain.Issue) error {
	return a.legacyIndexer.Upsert(ctx, projectID, issue)
}

// DeleteByID implements Index.DeleteByID by calling legacy Delete method
func (a *LegacyIndexAdapter) DeleteByID(ctx context.Context, projectID, issueID string) error {
	return a.legacyIndexer.Delete(ctx, projectID, issueID)
}

// DeleteByPath implements Index.DeleteByPath
// The legacy indexer doesn't have this method, so we need to extract issue ID from path
func (a *LegacyIndexAdapter) DeleteByPath(ctx context.Context, path string) error {
	// Extract issue ID from file path
	issueID := extractIssueIDFromFilePath(path)
	if issueID == "" {
		return fmt.Errorf("could not extract issue ID from path %s", path)
	}

	// Extract project ID from path (this is a best-effort approach)
	projectID := extractProjectIDFromPath(path)
	if projectID == "" {
		return fmt.Errorf("could not extract project ID from path %s", path)
	}

	return a.legacyIndexer.Delete(ctx, projectID, issueID)
}

// Search implements Index.Search by calling legacy Search and converting results
func (a *LegacyIndexAdapter) Search(ctx context.Context, projectID, query string, f Filters) ([]Hit, error) {
	issues, err := a.legacyIndexer.Search(ctx, projectID, query)
	if err != nil {
		return nil, err
	}

	// Apply filters to results (legacy indexer may not support all filters)
	filteredIssues := a.applyFilters(issues, f)

	// Apply pagination
	filteredIssues = a.applyPagination(filteredIssues, f)

	// Convert to hits
	return ToHits(filteredIssues), nil
}

// SearchGlobal implements Index.SearchGlobal
// Legacy indexer doesn't have this method, so this is not implemented
func (a *LegacyIndexAdapter) SearchGlobal(ctx context.Context, query string, f Filters) ([]Hit, error) {
	return nil, fmt.Errorf("SearchGlobal not supported by legacy indexer adapter")
}

// List implements Index.List by performing an empty search and converting results
func (a *LegacyIndexAdapter) List(ctx context.Context, projectID string, f Filters) ([]Row, error) {
	// Use empty query to get all results
	issues, err := a.legacyIndexer.Search(ctx, projectID, "")
	if err != nil {
		return nil, err
	}

	// Apply filters
	filteredIssues := a.applyFilters(issues, f)

	// Apply pagination
	filteredIssues = a.applyPagination(filteredIssues, f)

	// Convert to rows
	return ToRows(filteredIssues), nil
}

// ListGlobal implements Index.ListGlobal
// Legacy indexer doesn't have this method, so this is not implemented
func (a *LegacyIndexAdapter) ListGlobal(ctx context.Context, f Filters) ([]Row, error) {
	return nil, fmt.Errorf("ListGlobal not supported by legacy indexer adapter")
}

// Status implements Index.Status
func (a *LegacyIndexAdapter) Status(ctx context.Context) Status {
	// Legacy indexer doesn't have a status method, so we return a basic status
	now := time.Now()
	return Status{
		Healthy:        true, // Assume healthy if no errors
		TotalDocuments: 0,    // Unknown
		LastIndexed:    &now,
		IndexSize:      0, // Unknown
		Projects:       make(map[string]interface{}),
		Errors:         []string{},
		Version:        "legacy-adapter",
	}
}

// Rebuild implements Index.Rebuild
// Legacy indexer doesn't have this method
func (a *LegacyIndexAdapter) Rebuild(ctx context.Context, projectID string) error {
	return fmt.Errorf("Rebuild not supported by legacy indexer adapter")
}

// Optimize implements Index.Optimize
// Legacy indexer doesn't have this method
func (a *LegacyIndexAdapter) Optimize(ctx context.Context) error {
	return fmt.Errorf("Optimize not supported by legacy indexer adapter")
}

// Health implements Index.Health by calling legacy GetStats if available
func (a *LegacyIndexAdapter) Health(ctx context.Context) (map[string]interface{}, error) {
	// Try to get stats from legacy indexer
	stats, err := a.legacyIndexer.GetStats(ctx, "")
	if err != nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
			"adapter": "legacy_indexer_adapter",
		}, err
	}

	// Add adapter info
	if stats == nil {
		stats = make(map[string]interface{})
	}
	stats["adapter"] = "legacy_indexer_adapter"
	stats["healthy"] = true

	return stats, nil
}

// Helper methods

// extractIssueIDFromFilePath extracts issue ID from a file path like "bug/iss-abc123-title.md"
func extractIssueIDFromFilePath(path string) string {
	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, ".md") {
		return ""
	}

	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Issue files have format: {id}-{slug}.md
	// Find first hyphen to separate ID from slug
	if idx := strings.Index(name, "-"); idx > 0 {
		return name[:idx]
	}

	return ""
}

// extractProjectIDFromPath attempts to extract project ID from file path
// This is a best-effort approach and may not work in all cases
func extractProjectIDFromPath(path string) string {
	// This is a simplified approach - in a real implementation,
	// you'd need to track project paths more carefully
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == ".takl" && i > 0 {
			// Assume project ID is the directory containing .takl
			return parts[i-1]
		}
	}

	// Fallback: use "default" as project ID
	return "default"
}

// applyFilters applies the given filters to a slice of issues
func (a *LegacyIndexAdapter) applyFilters(issues []*domain.Issue, f Filters) []*domain.Issue {
	var filtered []*domain.Issue

	for _, issue := range issues {
		if a.matchesFilters(issue, f) {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

// applyPagination applies limit and offset to a slice of issues
func (a *LegacyIndexAdapter) applyPagination(issues []*domain.Issue, f Filters) []*domain.Issue {
	// Apply offset
	if f.Offset > 0 {
		if f.Offset >= len(issues) {
			return []*domain.Issue{}
		}
		issues = issues[f.Offset:]
	}

	// Apply limit
	if f.Limit > 0 && f.Limit < len(issues) {
		issues = issues[:f.Limit]
	}

	return issues
}

// matchesFilters checks if an issue matches the given filters
func (a *LegacyIndexAdapter) matchesFilters(issue *domain.Issue, f Filters) bool {
	if f.Status != "" && issue.Status != f.Status {
		return false
	}
	if f.Type != "" && issue.Type != f.Type {
		return false
	}
	if f.Priority != "" && issue.Priority != f.Priority {
		return false
	}
	if f.Assignee != "" && issue.Assignee != f.Assignee {
		return false
	}

	// Check labels if filter has labels
	if len(f.Labels) > 0 {
		hasAllLabels := true
		for _, filterLabel := range f.Labels {
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

	// Check date filters if present
	if f.Since != nil && issue.Created.Before(*f.Since) {
		return false
	}
	if f.Before != nil && issue.Created.After(*f.Before) {
		return false
	}

	return true
}
