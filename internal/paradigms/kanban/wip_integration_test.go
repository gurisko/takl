package kanban

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/paradigm"
	"github.com/takl/takl/internal/shared"
	"github.com/takl/takl/internal/store"
)

func TestKanban_WIPLimitsIntegration(t *testing.T) {
	// Create temporary directory for test
	testDir, err := os.MkdirTemp("", "kanban-wip-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// Initialize git repo using git command
	if err := initRealGitRepo(testDir); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create store manager
	manager, err := store.NewLegacyManager(testDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create Kanban paradigm with strict WIP limits
	kanban := &Kanban{}
	deps := paradigm.Deps{
		Clock: paradigm.DefaultClock{},
		Store: &TestStorageAdapter{manager: manager},
	}

	// Configure with very low WIP limits for testing
	options := map[string]any{
		"wip_limits": map[string]int{
			"doing":  2, // Only 2 items allowed in doing
			"review": 2, // Increase review limit to test capacity freeing
		},
		"block_on_downstream_full": false, // Disable for cleaner test
	}

	err = kanban.Init(context.Background(), deps, options)
	if err != nil {
		t.Fatalf("Failed to initialize Kanban: %v", err)
	}

	t.Logf("WIP limits configured: doing=%d, review=%d",
		kanban.opts.WIPLimits["doing"],
		kanban.opts.WIPLimits["review"])

	ctx := context.Background()
	opts := store.CreateOptions{
		Priority: "medium",
		Content:  "WIP limit test issue",
	}

	// Test 1: Fill "doing" column to capacity
	t.Run("fill_doing_to_capacity", func(t *testing.T) {
		for i := 1; i <= 2; i++ {
			issue, err := manager.Create("task", "Task "+string(rune('A'+i-1)), opts)
			if err != nil {
				t.Fatalf("Failed to create task %d: %v", i, err)
			}

			// Should be able to move to doing
			err = kanban.ValidateTransition(ctx, issue, "backlog", "doing")
			if err != nil {
				t.Fatalf("Task %d should be allowed into doing: %v", i, err)
			}

			// Actually move to doing and save
			issue.Status = "doing"
			if err := shared.SaveIssueToFile(issue); err != nil {
				t.Fatalf("Failed to save issue: %v", err)
			}

			t.Logf("✅ Task %d moved to doing", i)
		}
	})

	// Test 2: WIP limit should block third item
	t.Run("wip_limit_blocks_excess", func(t *testing.T) {
		issue, err := manager.Create("task", "Task C (should be blocked)", opts)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Should be blocked by WIP limit
		err = kanban.ValidateTransition(ctx, issue, "backlog", "doing")
		if err == nil {
			t.Fatal("Expected WIP limit to block transition to doing, but it didn't")
		}

		if !paradigm.IsWIPLimitExceeded(err) {
			t.Fatalf("Expected WIPLimitExceeded error, got: %v", err)
		}

		t.Logf("✅ WIP limit correctly blocked task: %v", err)
	})

	// Test 3: Moving item out of doing should free up capacity
	t.Run("capacity_freed_when_moved", func(t *testing.T) {
		// Find an issue in doing and move it to review
		doingIssues, err := manager.ListIssues(map[string]interface{}{"status": "doing"})
		if err != nil {
			t.Fatalf("Failed to list doing issues: %v", err)
		}

		if len(doingIssues) == 0 {
			t.Fatal("Expected issues in doing state")
		}

		// Move first issue to review
		firstIssue := doingIssues[0]
		err = kanban.ValidateTransition(ctx, firstIssue, "doing", "review")
		if err != nil {
			t.Fatalf("Should be able to move to review: %v", err)
		}

		firstIssue.Status = "review"
		if err := shared.SaveIssueToFile(firstIssue); err != nil {
			t.Fatalf("Failed to save issue: %v", err)
		}

		t.Logf("✅ Moved issue to review, freeing doing capacity")

		// Create a new issue to test the freed capacity
		newIssue, err := manager.Create("task", "Task D (should now work)", opts)
		if err != nil {
			t.Fatalf("Failed to create new task: %v", err)
		}
		err = kanban.ValidateTransition(ctx, newIssue, "backlog", "doing")
		if err != nil {
			t.Fatalf("Should now be able to move to doing: %v", err)
		}

		t.Logf("✅ Capacity freed - blocked issue can now move to doing")
	})
}

// TestStorageAdapter adapts store.LegacyManager to paradigm.Storage
type TestStorageAdapter struct {
	manager *store.LegacyManager
}

func (tsa *TestStorageAdapter) ListIssues(ctx context.Context, filters map[string]interface{}) ([]*domain.Issue, error) {
	return tsa.manager.ListIssues(filters)
}

func (tsa *TestStorageAdapter) SaveIssue(ctx context.Context, iss *domain.Issue) error {
	return shared.SaveIssueToFile(iss)
}

func (tsa *TestStorageAdapter) LoadIssue(ctx context.Context, id string) (*domain.Issue, error) {
	return tsa.manager.LoadIssue(id)
}

func initRealGitRepo(dir string) error {
	cmd := fmt.Sprintf("cd %s && git init", dir)
	if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
		return err
	}
	// Set minimal git config for tests
	configCmd := fmt.Sprintf("cd %s && git config user.name 'Test' && git config user.email 'test@example.com'", dir)
	return exec.Command("sh", "-c", configCmd).Run()
}
