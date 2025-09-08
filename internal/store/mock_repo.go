package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/takl/takl/internal/domain"
)

// MockRepo is a mock implementation of the Repo interface for testing
type MockRepo struct {
	mu     sync.RWMutex
	issues map[string]map[string]*domain.Issue // projectID -> issueID -> issue

	// Test helpers
	SaveIssueFunc     func(ctx context.Context, projectID string, issue *domain.Issue) error
	LoadIssueFunc     func(ctx context.Context, projectID, issueID string) (*domain.Issue, error)
	ListIssuesFunc    func(ctx context.Context, projectID string, f Filters) ([]*domain.Issue, error)
	DeleteIssueFunc   func(ctx context.Context, projectID, issueID string) error
	ListAllIssuesFunc func(ctx context.Context, f Filters) ([]*domain.Issue, error)
	HealthFunc        func(ctx context.Context, projectID string) (map[string]interface{}, error)

	// Call tracking
	SaveIssueCalls     []SaveIssueCall
	LoadIssueCalls     []LoadIssueCall
	ListIssuesCalls    []ListIssuesCall
	DeleteIssueCalls   []DeleteIssueCall
	ListAllIssuesCalls []ListAllIssuesCall
	HealthCalls        []HealthCall
}

// Call tracking structs
type SaveIssueCall struct {
	ProjectID string
	Issue     *domain.Issue
	Result    error
}

type LoadIssueCall struct {
	ProjectID string
	IssueID   string
	Result    *domain.Issue
	Error     error
}

type ListIssuesCall struct {
	ProjectID string
	Filters   Filters
	Result    []*domain.Issue
	Error     error
}

type DeleteIssueCall struct {
	ProjectID string
	IssueID   string
	Result    error
}

type ListAllIssuesCall struct {
	Filters Filters
	Result  []*domain.Issue
	Error   error
}

type HealthCall struct {
	ProjectID string
	Result    map[string]interface{}
	Error     error
}

// NewMockRepo creates a new mock repository
func NewMockRepo() *MockRepo {
	return &MockRepo{
		issues: make(map[string]map[string]*domain.Issue),
	}
}

// LoadIssue implements Repo.LoadIssue
func (m *MockRepo) LoadIssue(ctx context.Context, projectID, issueID string) (*domain.Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result *domain.Issue
	var err error

	if m.LoadIssueFunc != nil {
		result, err = m.LoadIssueFunc(ctx, projectID, issueID)
	} else {
		projectIssues, exists := m.issues[projectID]
		if !exists {
			err = fmt.Errorf("project not found")
		} else {
			issue, exists := projectIssues[issueID]
			if !exists {
				err = fmt.Errorf("issue not found")
			} else {
				// Return a copy to prevent external modification
				issueCopy := *issue
				result = &issueCopy
			}
		}
	}

	// Track the call
	call := LoadIssueCall{
		ProjectID: projectID,
		IssueID:   issueID,
		Result:    result,
		Error:     err,
	}
	m.LoadIssueCalls = append(m.LoadIssueCalls, call)

	return result, err
}

// SaveIssue implements Repo.SaveIssue
func (m *MockRepo) SaveIssue(ctx context.Context, projectID string, issue *domain.Issue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	if m.SaveIssueFunc != nil {
		err = m.SaveIssueFunc(ctx, projectID, issue)
	} else {
		if m.issues[projectID] == nil {
			m.issues[projectID] = make(map[string]*domain.Issue)
		}

		// Store a copy to prevent external modification
		issueCopy := *issue
		issueCopy.Updated = time.Now() // Simulate repository behavior
		m.issues[projectID][issue.ID] = &issueCopy
	}

	// Track the call
	call := SaveIssueCall{
		ProjectID: projectID,
		Issue:     issue,
		Result:    err,
	}
	m.SaveIssueCalls = append(m.SaveIssueCalls, call)

	return err
}

// ListIssues implements Repo.ListIssues
func (m *MockRepo) ListIssues(ctx context.Context, projectID string, f Filters) ([]*domain.Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Issue
	var err error

	if m.ListIssuesFunc != nil {
		result, err = m.ListIssuesFunc(ctx, projectID, f)
	} else {
		projectIssues, exists := m.issues[projectID]
		if !exists {
			result = []*domain.Issue{}
		} else {
			for _, issue := range projectIssues {
				if m.matchesFilter(issue, f) {
					// Return copies to prevent external modification
					issueCopy := *issue
					result = append(result, &issueCopy)
				}
			}

			// Apply limit and offset
			if f.Offset > 0 {
				if f.Offset >= len(result) {
					result = []*domain.Issue{}
				} else {
					result = result[f.Offset:]
				}
			}

			if f.Limit > 0 && f.Limit < len(result) {
				result = result[:f.Limit]
			}
		}
	}

	// Track the call
	call := ListIssuesCall{
		ProjectID: projectID,
		Filters:   f,
		Result:    result,
		Error:     err,
	}
	m.ListIssuesCalls = append(m.ListIssuesCalls, call)

	return result, err
}

// DeleteIssue implements Repo.DeleteIssue
func (m *MockRepo) DeleteIssue(ctx context.Context, projectID, issueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	if m.DeleteIssueFunc != nil {
		err = m.DeleteIssueFunc(ctx, projectID, issueID)
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
	call := DeleteIssueCall{
		ProjectID: projectID,
		IssueID:   issueID,
		Result:    err,
	}
	m.DeleteIssueCalls = append(m.DeleteIssueCalls, call)

	return err
}

// ListAllIssues implements Repo.ListAllIssues
func (m *MockRepo) ListAllIssues(ctx context.Context, f Filters) ([]*domain.Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Issue
	var err error

	if m.ListAllIssuesFunc != nil {
		result, err = m.ListAllIssuesFunc(ctx, f)
	} else {
		for _, projectIssues := range m.issues {
			for _, issue := range projectIssues {
				if m.matchesFilter(issue, f) {
					// Return copies to prevent external modification
					issueCopy := *issue
					result = append(result, &issueCopy)
				}
			}
		}

		// Apply limit and offset
		if f.Offset > 0 {
			if f.Offset >= len(result) {
				result = []*domain.Issue{}
			} else {
				result = result[f.Offset:]
			}
		}

		if f.Limit > 0 && f.Limit < len(result) {
			result = result[:f.Limit]
		}
	}

	// Track the call
	call := ListAllIssuesCall{
		Filters: f,
		Result:  result,
		Error:   err,
	}
	m.ListAllIssuesCalls = append(m.ListAllIssuesCalls, call)

	return result, err
}

// Health implements Repo.Health
func (m *MockRepo) Health(ctx context.Context, projectID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result map[string]interface{}
	var err error

	if m.HealthFunc != nil {
		result, err = m.HealthFunc(ctx, projectID)
	} else {
		projectIssues := m.issues[projectID]
		issueCount := 0
		if projectIssues != nil {
			issueCount = len(projectIssues)
		}

		result = map[string]interface{}{
			"healthy":     true,
			"issue_count": issueCount,
			"project_id":  projectID,
		}
	}

	// Track the call
	call := HealthCall{
		ProjectID: projectID,
		Result:    result,
		Error:     err,
	}
	m.HealthCalls = append(m.HealthCalls, call)

	return result, err
}

// Test helpers

// Reset clears all data and call tracking
func (m *MockRepo) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.issues = make(map[string]map[string]*domain.Issue)
	m.SaveIssueCalls = nil
	m.LoadIssueCalls = nil
	m.ListIssuesCalls = nil
	m.DeleteIssueCalls = nil
	m.ListAllIssuesCalls = nil
	m.HealthCalls = nil
}

// AddIssue adds an issue directly to the mock store (for test setup)
func (m *MockRepo) AddIssue(projectID string, issue *domain.Issue) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.issues[projectID] == nil {
		m.issues[projectID] = make(map[string]*domain.Issue)
	}

	issueCopy := *issue
	m.issues[projectID][issue.ID] = &issueCopy
}

// GetIssueCount returns the number of issues for a project
func (m *MockRepo) GetIssueCount(projectID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if projectIssues := m.issues[projectID]; projectIssues != nil {
		return len(projectIssues)
	}
	return 0
}

// matchesFilter checks if an issue matches the given filters
func (m *MockRepo) matchesFilter(issue *domain.Issue, f Filters) bool {
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

	// Note: Since and Before filtering would require parsing the string dates
	// For now, we'll skip those filters in the mock

	return true
}
