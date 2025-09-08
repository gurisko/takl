package indexer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/takl/takl/internal/domain"
)

// MockIndex is a mock implementation of the Index interface for testing
type MockIndex struct {
	mu     sync.RWMutex
	issues map[string]map[string]*domain.Issue // projectID -> issueID -> issue

	// Test helpers
	UpsertFunc       func(ctx context.Context, projectID string, issue *domain.Issue) error
	DeleteByIDFunc   func(ctx context.Context, projectID, issueID string) error
	DeleteByPathFunc func(ctx context.Context, path string) error
	SearchFunc       func(ctx context.Context, projectID, query string, f Filters) ([]Hit, error)
	SearchGlobalFunc func(ctx context.Context, query string, f Filters) ([]Hit, error)
	ListFunc         func(ctx context.Context, projectID string, f Filters) ([]Row, error)
	ListGlobalFunc   func(ctx context.Context, f Filters) ([]Row, error)
	StatusFunc       func(ctx context.Context) Status
	RebuildFunc      func(ctx context.Context, projectID string) error
	OptimizeFunc     func(ctx context.Context) error
	HealthFunc       func(ctx context.Context) (map[string]interface{}, error)

	// Call tracking
	UpsertCalls       []UpsertCall
	DeleteByIDCalls   []DeleteByIDCall
	DeleteByPathCalls []DeleteByPathCall
	SearchCalls       []SearchCall
	SearchGlobalCalls []SearchGlobalCall
	ListCalls         []ListCall
	ListGlobalCalls   []ListGlobalCall
	StatusCalls       []StatusCall
	RebuildCalls      []RebuildCall
	OptimizeCalls     []OptimizeCall
	HealthCalls       []HealthCall

	// Mock state
	healthy bool
	errors  []string
}

// Call tracking structs
type UpsertCall struct {
	ProjectID string
	Issue     *domain.Issue
	Result    error
}

type DeleteByIDCall struct {
	ProjectID string
	IssueID   string
	Result    error
}

type DeleteByPathCall struct {
	Path   string
	Result error
}

type SearchCall struct {
	ProjectID string
	Query     string
	Filters   Filters
	Result    []Hit
	Error     error
}

type SearchGlobalCall struct {
	Query   string
	Filters Filters
	Result  []Hit
	Error   error
}

type ListCall struct {
	ProjectID string
	Filters   Filters
	Result    []Row
	Error     error
}

type ListGlobalCall struct {
	Filters Filters
	Result  []Row
	Error   error
}

type StatusCall struct {
	Result Status
}

type RebuildCall struct {
	ProjectID string
	Result    error
}

type OptimizeCall struct {
	Result error
}

type HealthCall struct {
	Result map[string]interface{}
	Error  error
}

// NewMockIndex creates a new mock index
func NewMockIndex() *MockIndex {
	return &MockIndex{
		issues:  make(map[string]map[string]*domain.Issue),
		healthy: true,
	}
}

// Upsert implements Index.Upsert
func (m *MockIndex) Upsert(ctx context.Context, projectID string, issue *domain.Issue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	if m.UpsertFunc != nil {
		err = m.UpsertFunc(ctx, projectID, issue)
	} else {
		if m.issues[projectID] == nil {
			m.issues[projectID] = make(map[string]*domain.Issue)
		}

		// Store a copy to prevent external modification
		issueCopy := *issue
		m.issues[projectID][issue.ID] = &issueCopy
	}

	// Track the call
	call := UpsertCall{
		ProjectID: projectID,
		Issue:     issue,
		Result:    err,
	}
	m.UpsertCalls = append(m.UpsertCalls, call)

	return err
}

// DeleteByID implements Index.DeleteByID
func (m *MockIndex) DeleteByID(ctx context.Context, projectID, issueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	if m.DeleteByIDFunc != nil {
		err = m.DeleteByIDFunc(ctx, projectID, issueID)
	} else {
		projectIssues, exists := m.issues[projectID]
		if !exists {
			err = fmt.Errorf("project not found")
		} else {
			_, exists := projectIssues[issueID]
			if !exists {
				err = fmt.Errorf("issue not found")
			} else {
				delete(projectIssues, issueID)
			}
		}
	}

	// Track the call
	call := DeleteByIDCall{
		ProjectID: projectID,
		IssueID:   issueID,
		Result:    err,
	}
	m.DeleteByIDCalls = append(m.DeleteByIDCalls, call)

	return err
}

// DeleteByPath implements Index.DeleteByPath
func (m *MockIndex) DeleteByPath(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	if m.DeleteByPathFunc != nil {
		err = m.DeleteByPathFunc(ctx, path)
	} else {
		// Simple mock: assume path contains issue ID
		// In real implementation, this would parse the file path
		err = fmt.Errorf("delete by path not implemented in mock")
	}

	// Track the call
	call := DeleteByPathCall{
		Path:   path,
		Result: err,
	}
	m.DeleteByPathCalls = append(m.DeleteByPathCalls, call)

	return err
}

// Search implements Index.Search
func (m *MockIndex) Search(ctx context.Context, projectID, query string, f Filters) ([]Hit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Hit
	var err error

	if m.SearchFunc != nil {
		result, err = m.SearchFunc(ctx, projectID, query, f)
	} else {
		projectIssues, exists := m.issues[projectID]
		if exists {
			queryLower := strings.ToLower(query)

			for _, issue := range projectIssues {
				if m.matchesSearchQuery(issue, queryLower) && m.matchesFilters(issue, f) {
					// Create hit with copy of issue
					issueCopy := *issue
					hit := Hit{
						Issue: &issueCopy,
						Score: 1.0, // Simple mock score
					}
					result = append(result, hit)
				}
			}

			// Apply limit and offset
			result = m.applyPaginationToHits(result, f)
		}
	}

	// Track the call
	call := SearchCall{
		ProjectID: projectID,
		Query:     query,
		Filters:   f,
		Result:    result,
		Error:     err,
	}
	m.SearchCalls = append(m.SearchCalls, call)

	return result, err
}

// SearchGlobal implements Index.SearchGlobal
func (m *MockIndex) SearchGlobal(ctx context.Context, query string, f Filters) ([]Hit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Hit
	var err error

	if m.SearchGlobalFunc != nil {
		result, err = m.SearchGlobalFunc(ctx, query, f)
	} else {
		queryLower := strings.ToLower(query)

		for _, projectIssues := range m.issues {
			for _, issue := range projectIssues {
				if m.matchesSearchQuery(issue, queryLower) && m.matchesFilters(issue, f) {
					// Create hit with copy of issue
					issueCopy := *issue
					hit := Hit{
						Issue: &issueCopy,
						Score: 1.0, // Simple mock score
					}
					result = append(result, hit)
				}
			}
		}

		// Apply limit and offset
		result = m.applyPaginationToHits(result, f)
	}

	// Track the call
	call := SearchGlobalCall{
		Query:   query,
		Filters: f,
		Result:  result,
		Error:   err,
	}
	m.SearchGlobalCalls = append(m.SearchGlobalCalls, call)

	return result, err
}

// List implements Index.List
func (m *MockIndex) List(ctx context.Context, projectID string, f Filters) ([]Row, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Row
	var err error

	if m.ListFunc != nil {
		result, err = m.ListFunc(ctx, projectID, f)
	} else {
		projectIssues, exists := m.issues[projectID]
		if exists {
			for _, issue := range projectIssues {
				if m.matchesFilters(issue, f) {
					// Create row with copy of issue
					issueCopy := *issue
					row := Row{
						Issue: &issueCopy,
					}
					result = append(result, row)
				}
			}

			// Apply limit and offset
			result = m.applyPaginationToRows(result, f)
		}
	}

	// Track the call
	call := ListCall{
		ProjectID: projectID,
		Filters:   f,
		Result:    result,
		Error:     err,
	}
	m.ListCalls = append(m.ListCalls, call)

	return result, err
}

// ListGlobal implements Index.ListGlobal
func (m *MockIndex) ListGlobal(ctx context.Context, f Filters) ([]Row, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Row
	var err error

	if m.ListGlobalFunc != nil {
		result, err = m.ListGlobalFunc(ctx, f)
	} else {
		for _, projectIssues := range m.issues {
			for _, issue := range projectIssues {
				if m.matchesFilters(issue, f) {
					// Create row with copy of issue
					issueCopy := *issue
					row := Row{
						Issue: &issueCopy,
					}
					result = append(result, row)
				}
			}
		}

		// Apply limit and offset
		result = m.applyPaginationToRows(result, f)
	}

	// Track the call
	call := ListGlobalCall{
		Filters: f,
		Result:  result,
		Error:   err,
	}
	m.ListGlobalCalls = append(m.ListGlobalCalls, call)

	return result, err
}

// Status implements Index.Status
func (m *MockIndex) Status(ctx context.Context) Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result Status

	if m.StatusFunc != nil {
		result = m.StatusFunc(ctx)
	} else {
		totalDocs := int64(0)
		projects := make(map[string]interface{})

		for projectID, projectIssues := range m.issues {
			count := len(projectIssues)
			totalDocs += int64(count)
			projects[projectID] = map[string]interface{}{
				"issue_count": count,
				"healthy":     true,
			}
		}

		now := time.Now()
		result = Status{
			Healthy:        m.healthy,
			TotalDocuments: totalDocs,
			LastIndexed:    &now,
			IndexSize:      totalDocs * 1024, // Mock size
			Projects:       projects,
			Errors:         m.errors,
			Version:        "mock-1.0",
		}
	}

	// Track the call
	call := StatusCall{
		Result: result,
	}
	m.StatusCalls = append(m.StatusCalls, call)

	return result
}

// Rebuild implements Index.Rebuild
func (m *MockIndex) Rebuild(ctx context.Context, projectID string) error {
	var err error

	if m.RebuildFunc != nil {
		err = m.RebuildFunc(ctx, projectID)
	} else {
		// Mock rebuild - just mark as successful
		err = nil
	}

	// Track the call
	call := RebuildCall{
		ProjectID: projectID,
		Result:    err,
	}
	m.RebuildCalls = append(m.RebuildCalls, call)

	return err
}

// Optimize implements Index.Optimize
func (m *MockIndex) Optimize(ctx context.Context) error {
	var err error

	if m.OptimizeFunc != nil {
		err = m.OptimizeFunc(ctx)
	} else {
		// Mock optimize - just mark as successful
		err = nil
	}

	// Track the call
	call := OptimizeCall{
		Result: err,
	}
	m.OptimizeCalls = append(m.OptimizeCalls, call)

	return err
}

// Health implements Index.Health
func (m *MockIndex) Health(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result map[string]interface{}
	var err error

	if m.HealthFunc != nil {
		result, err = m.HealthFunc(ctx)
	} else {
		totalDocs := 0
		for _, projectIssues := range m.issues {
			totalDocs += len(projectIssues)
		}

		result = map[string]interface{}{
			"healthy":         m.healthy,
			"total_documents": totalDocs,
			"projects":        len(m.issues),
		}
	}

	// Track the call
	call := HealthCall{
		Result: result,
		Error:  err,
	}
	m.HealthCalls = append(m.HealthCalls, call)

	return result, err
}

// Test helpers

// Reset clears all data and call tracking
func (m *MockIndex) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.issues = make(map[string]map[string]*domain.Issue)
	m.healthy = true
	m.errors = nil

	m.UpsertCalls = nil
	m.DeleteByIDCalls = nil
	m.DeleteByPathCalls = nil
	m.SearchCalls = nil
	m.SearchGlobalCalls = nil
	m.ListCalls = nil
	m.ListGlobalCalls = nil
	m.StatusCalls = nil
	m.RebuildCalls = nil
	m.OptimizeCalls = nil
	m.HealthCalls = nil
}

// AddIssue adds an issue directly to the mock index (for test setup)
func (m *MockIndex) AddIssue(projectID string, issue *domain.Issue) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.issues[projectID] == nil {
		m.issues[projectID] = make(map[string]*domain.Issue)
	}

	issueCopy := *issue
	m.issues[projectID][issue.ID] = &issueCopy
}

// SetHealthy sets the health status for testing
func (m *MockIndex) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// AddError adds an error for testing
func (m *MockIndex) AddError(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, err)
}

// GetIssueCount returns the number of issues for a project
func (m *MockIndex) GetIssueCount(projectID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if projectIssues := m.issues[projectID]; projectIssues != nil {
		return len(projectIssues)
	}
	return 0
}

// Helper methods

func (m *MockIndex) matchesSearchQuery(issue *domain.Issue, queryLower string) bool {
	if queryLower == "" {
		return true
	}

	titleLower := strings.ToLower(issue.Title)
	contentLower := strings.ToLower(issue.Content)

	return strings.Contains(titleLower, queryLower) || strings.Contains(contentLower, queryLower)
}

func (m *MockIndex) matchesFilters(issue *domain.Issue, f Filters) bool {
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

	// Note: Since and Before filtering would require time comparison
	// For now, we'll skip those filters in the mock

	return true
}

func (m *MockIndex) applyPaginationToHits(hits []Hit, f Filters) []Hit {
	if f.Offset > 0 {
		if f.Offset >= len(hits) {
			return []Hit{}
		}
		hits = hits[f.Offset:]
	}

	if f.Limit > 0 && f.Limit < len(hits) {
		hits = hits[:f.Limit]
	}

	return hits
}

func (m *MockIndex) applyPaginationToRows(rows []Row, f Filters) []Row {
	if f.Offset > 0 {
		if f.Offset >= len(rows) {
			return []Row{}
		}
		rows = rows[f.Offset:]
	}

	if f.Limit > 0 && f.Limit < len(rows) {
		rows = rows[:f.Limit]
	}

	return rows
}
