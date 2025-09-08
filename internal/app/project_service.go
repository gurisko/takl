package app

import (
	"context"
	"fmt"

	"github.com/takl/takl/internal/domain"
)

// ProjectService orchestrates project-related use cases
type ProjectService struct {
	projectRepo domain.ProjectRepository
	watcher     domain.Watcher
	clock       domain.Clock
}

// NewProjectService creates a new project service
func NewProjectService(
	projectRepo domain.ProjectRepository,
	watcher domain.Watcher,
	clock domain.Clock,
) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		watcher:     watcher,
		clock:       clock,
	}
}

// RegisterProject registers a new project
func (s *ProjectService) RegisterProject(ctx context.Context, project *domain.Project) error {
	// Set timestamps
	now := s.clock.Now()
	project.Registered = now
	project.LastSeen = now
	project.LastAccess = now

	// Register in repository
	if err := s.projectRepo.Register(ctx, project); err != nil {
		return fmt.Errorf("failed to register project: %w", err)
	}

	// Add to filesystem watcher
	if err := s.watcher.AddProject(project.ID, project.IssuesDir); err != nil {
		// Log warning but don't fail registration - watcher is not critical for basic operation
		_ = err // Ignore watcher errors for now
	}

	return nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Update last access time
	project.LastAccess = s.clock.Now()
	_ = s.projectRepo.Update(ctx, project) // Best effort, ignore errors

	return project, nil
}

// ListProjects retrieves all registered projects
func (s *ProjectService) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	return s.projectRepo.List(ctx)
}

// UpdateProject updates project metadata
func (s *ProjectService) UpdateProject(ctx context.Context, project *domain.Project) error {
	return s.projectRepo.Update(ctx, project)
}

// UnregisterProject removes a project
func (s *ProjectService) UnregisterProject(ctx context.Context, projectID string) error {
	// Remove from watcher
	_ = s.watcher.RemoveProject(projectID) // Best effort, ignore errors

	// Remove from repository
	return s.projectRepo.Delete(ctx, projectID)
}

// GetProjectHealth checks the health of a project
func (s *ProjectService) GetProjectHealth(ctx context.Context, projectID string) (map[string]interface{}, error) {
	return s.projectRepo.Health(ctx, projectID)
}
