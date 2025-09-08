# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

```bash
# Build and test
make build           # Build the takl binary
make test           # Run unit tests (fast, with -short flag)
make test-race      # Run tests with race detector
make test-cover     # Run tests with coverage report (generates coverage.html)
make check          # Run all quality checks (fmt, vet, lint, test-race)

# Single test patterns
go test ./internal/paradigm -v                                    # Test specific package  
go test ./internal/paradigms/scrum -v -run TestScrum_GuardEstimated  # Test specific function

# Development workflow
make dev            # Full development workflow: clean, fmt, vet, lint, test, build
make fmt            # Format code
make lint           # Run golangci-lint (installs if missing)

# CI pipeline
make ci-test        # CI test pipeline with 80% coverage gate
```

## Architecture Overview

TAKL is a **git-native issue tracker** with a **paradigm-driven architecture** that supports different workflow methodologies (Scrum, Kanban, etc.) through a clean plugin system.

### Core Architecture Components

**1. Multi-Mode Operation**
- **Embedded Mode**: `.takl/` directory in existing projects (default for git repos)  
- **Standalone Mode**: `.issues/` directory for dedicated issue repositories

**2. Issue Storage System** (`internal/issues/`, `internal/types/`)
- Issues stored as markdown files with YAML frontmatter
- Shared `types.Issue` struct to prevent circular imports
- Git integration with auto-commit functionality
- File structure: `{type}/{id}-{slug}.md` (e.g., `bug/iss-001-login-broken.md`)

**3. Paradigm System** (`internal/paradigm/`, `internal/paradigms/`)
- **Interface-driven architecture** with dependency injection
- **Guard functions** for workflow state validation  
- **Registry pattern** for paradigm management
- **Event system** for loose coupling
- **Test framework** with doubles (FixedClock, MemStore, FakeRepo)

### Key Paradigm Architecture Patterns

```go
// Core interfaces (internal/paradigm/base.go)
type Paradigm interface {
    // Lifecycle
    Init(ctx context.Context, deps Deps) error
    
    // Workflow  
    GetWorkflowStates() []WorkflowState
    ValidateTransition(ctx context.Context, issue *types.Issue, from, to string) error
    
    // Operations
    GetPlanningOperations() []PlanningOperation
    GetExecutionOperations() []ExecutionOperation
    
    // Analytics
    CalculateProgress(ctx context.Context, issues []*types.Issue, pc ProgressContext) (*Progress, error)
}

// Guard functions for state validation
type Guard func(ctx context.Context, issue *types.Issue, from, to string) error

// Dependency injection for testing
type Deps struct {
    Clock   Clock      // Time operations
    Repo    Repository // Git operations  
    Store   Storage    // Issue persistence
    Metrics MetricsSink // Analytics
    Log     *slog.Logger
}
```

**4. CLI System** (`cmd/`)
- Cobra-based command structure
- Multiple creation modes: Git-style (`-m`), Interactive (prompts), Traditional
- Context detection (embedded vs standalone mode)
- Viper-based configuration management

### Story Points Implementation

Story points are tracked via issue labels using `points:N` format:
```go
// In Scrum paradigm
issue := &types.Issue{
    Labels: []string{"points:5", "backend", "critical"},
}
// Guards validate: unestimated stories (0 points) fail transition to sprint_backlog
```

### Testing Patterns

**Paradigm Testing** uses dependency injection with test doubles:
```go
deps := paradigm.TestDeps(paradigm.NewFixedClock(testTime))
s := &Scrum{}
err := s.Init(context.Background(), deps)

// Test guards with proper story point setup
issue := &types.Issue{
    Labels: []string{fmt.Sprintf("points:%d", storyPoints)},
}
```

**Table-driven tests** with parallel execution are used throughout.

## Issue Types and Workflow

- **bug** 🐛 - Something broken
- **feature** ✨ - New functionality  
- **task** ✅ - Work to be done
- **epic** 🎯 - Large initiatives

**Scrum Workflow States**: `backlog → sprint_backlog → in_progress → review → done`

## Key Dependencies

- **Cobra**: CLI framework
- **Viper**: Configuration management  
- **go-git**: Git operations
- **frontmatter**: YAML frontmatter parsing
- **golangci-lint**: Code quality (auto-installs via Makefile)

## Package Structure Notes

- `internal/types/issue.go` - Shared Issue type (breaks circular imports)
- `internal/paradigm/` - Core paradigm interfaces and registry
- `internal/paradigms/scrum/` - Complete Scrum implementation with guards
- `internal/paradigm/testutil.go` - Test doubles and utilities  
- `cmd/` - CLI commands with embedded/standalone mode detection

## Configuration

TAKL auto-detects mode and creates `config.yaml`:
```yaml
mode: embedded  # or standalone
git:
  auto_commit: true
  commit_message: "Update issue: %s"
```

## Important Implementation Notes

- **Circular Import Prevention**: Issue type moved to `internal/types/` 
- **Guard Function Pattern**: First-class workflow validation without reflection
- **Registry Pattern**: Thread-safe paradigm management
- **Test Seams**: All dependencies are interfaces for easy mocking
- **Race Detection**: All tests run with `-race` flag in CI
- **Coverage Gate**: CI requires 80% test coverage to pass
- Always run `make check` to validate code changes.