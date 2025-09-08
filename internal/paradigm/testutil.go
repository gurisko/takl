package paradigm

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/takl/takl/internal/domain"
)

// Test doubles for dependency interfaces

// FixedClock always returns the same time
type FixedClock struct {
	Time time.Time
}

func NewFixedClock(t time.Time) *FixedClock {
	return &FixedClock{Time: t}
}

func (c *FixedClock) Now() time.Time {
	return c.Time
}

func (c *FixedClock) Since(t time.Time) time.Duration {
	return c.Time.Sub(t)
}

// MemStore is an in-memory storage implementation for testing
type MemStore struct {
	issues map[string]*domain.Issue
	mu     sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		issues: make(map[string]*domain.Issue),
	}
}

func (m *MemStore) ListIssues(ctx context.Context, filters map[string]interface{}) ([]*domain.Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Issue
	for _, issue := range m.issues {
		// Simple filtering - could be enhanced
		matches := true
		for key, value := range filters {
			switch key {
			case "state":
				if issue.Status != value.(string) {
					matches = false
				}
			case "type":
				if issue.Type != value.(string) {
					matches = false
				}
			}
		}
		if matches {
			// Return a copy to avoid race conditions
			issueCopy := *issue
			result = append(result, &issueCopy)
		}
	}
	return result, nil
}

func (m *MemStore) SaveIssue(ctx context.Context, iss *domain.Issue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy to avoid mutation issues
	issueCopy := *iss
	m.issues[iss.ID] = &issueCopy
	return nil
}

func (m *MemStore) LoadIssue(ctx context.Context, id string) (*domain.Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if issue, exists := m.issues[id]; exists {
		// Return a copy
		issueCopy := *issue
		return &issueCopy, nil
	}
	return nil, ErrIssueNotFound
}

// Add a helper method for tests
func (m *MemStore) AddIssue(issue *domain.Issue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	issueCopy := *issue
	m.issues[issue.ID] = &issueCopy
}

// FakeRepo is a test repository implementation
type FakeRepo struct {
	RootPath string
	Clean    bool
}

func NewFakeRepo(rootPath string) *FakeRepo {
	return &FakeRepo{
		RootPath: rootPath,
		Clean:    true,
	}
}

func (r *FakeRepo) Root() string {
	return r.RootPath
}

func (r *FakeRepo) IsClean(ctx context.Context) (bool, error) {
	return r.Clean, nil
}

// TestDeps creates a complete set of test dependencies
func TestDeps(clock Clock) Deps {
	if clock == nil {
		clock = NewFixedClock(time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC))
	}

	return Deps{
		Clock: clock,
		Repo:  NewFakeRepo("/tmp/test"),
		Store: NewMemStore(),
		Log:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}
