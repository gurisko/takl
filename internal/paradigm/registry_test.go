package paradigm

import (
	"context"
	"testing"

	"github.com/takl/takl/internal/domain"
)

// MockParadigm for testing
type MockParadigm struct {
	id       string
	name     string
	category Category
}

func (m *MockParadigm) ID() string                                       { return m.id }
func (m *MockParadigm) Name() string                                     { return m.name }
func (m *MockParadigm) Category() Category                               { return m.category }
func (m *MockParadigm) Init(context.Context, Deps, map[string]any) error { return nil }

func (m *MockParadigm) GetTimeModel() TimeModel {
	return TimeModel{Kind: "test"}
}

func (m *MockParadigm) GetWorkUnit() WorkUnit {
	return WorkUnit{Kind: "task", Fields: []string{}}
}

func (m *MockParadigm) GetWorkflowStates() []WorkflowState {
	return []WorkflowState{
		{Key: "open", DisplayName: "Open"},
		{Key: "done", DisplayName: "Done"},
	}
}

func (m *MockParadigm) GetValidTransitions(currentState string) []string {
	if currentState == "open" {
		return []string{"done"}
	}
	return []string{}
}

func (m *MockParadigm) ValidateTransition(ctx context.Context, issue *domain.Issue, from, to string) error {
	return nil
}

func (m *MockParadigm) CreateIssue(ctx context.Context, req CreateIssueRequest) (*domain.Issue, error) {
	return nil, nil
}

func (m *MockParadigm) GetPlanningOperations() []PlanningOperation {
	return []PlanningOperation{}
}

func (m *MockParadigm) GetExecutionOperations() []ExecutionOperation {
	return []ExecutionOperation{}
}

func (m *MockParadigm) CalculateProgress(ctx context.Context, issues []*domain.Issue, pc ProgressContext) (*Progress, error) {
	return &Progress{}, nil
}

func TestRegister(t *testing.T) {
	// Clear registry for test
	registry = make(map[string]func() Paradigm)

	mock := &MockParadigm{
		id:       "test",
		name:     "Test Paradigm",
		category: CategoryAgile,
	}

	Register("test", func() Paradigm {
		return mock
	})

	// Check that paradigm was registered
	ids := List()
	if len(ids) != 1 || ids[0] != "test" {
		t.Errorf("Expected [test], got %v", ids)
	}

	// Check that we can create the paradigm
	p, err := Create("test")
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	if p.ID() != "test" {
		t.Errorf("Expected ID test, got %s", p.ID())
	}
}

func TestCreateUnknownParadigm(t *testing.T) {
	_, err := Create("unknown")
	if err == nil {
		t.Error("Expected error for unknown paradigm")
	}
}

func TestDefaultResolver(t *testing.T) {
	// Clear registry for test
	registry = make(map[string]func() Paradigm)

	mock := &MockParadigm{
		id:       "test",
		name:     "Test Paradigm",
		category: CategoryAgile,
	}

	Register("test", func() Paradigm {
		return mock
	})

	deps := TestDeps(nil)
	resolver := NewDefaultResolver(deps)

	p, err := resolver.Resolve("test")
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}

	if p.ID() != "test" {
		t.Errorf("Expected ID test, got %s", p.ID())
	}
}
