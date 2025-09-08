package store

import (
	"context"
	"log/slog"
	"sync"

	"github.com/takl/takl/internal/domain"
)

// ServiceRegistry provides a centralized registry for all store-layer services
// This implements the dependency inversion principle by exposing domain interfaces
type ServiceRegistry struct {
	// Core repositories
	repositoryFactory *RepositoryFactory

	// Infrastructure services
	watcher domain.Watcher

	// Configuration
	configDir string

	// Thread safety
	mu sync.RWMutex
}

// NewServiceRegistry creates a new service registry with the given configuration
func NewServiceRegistry(configDir string, logger *slog.Logger) (*ServiceRegistry, error) {
	// Create repository factory
	repoFactory := NewRepositoryFactory(configDir)

	// Create watcher adapter
	watcherAdapter, err := NewWatcherAdapter(logger)
	if err != nil {
		return nil, err
	}

	registry := &ServiceRegistry{
		repositoryFactory: repoFactory,
		watcher:           watcherAdapter,
		configDir:         configDir,
	}

	return registry, nil
}

// GetProjectRepository returns the project repository
func (r *ServiceRegistry) GetProjectRepository() domain.ProjectRepository {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.repositoryFactory.GetProjectRepository()
}

// GetIssueRepository returns an issue repository for the given project
func (r *ServiceRegistry) GetIssueRepository(projectID string) (domain.IssueRepository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.repositoryFactory.GetIssueRepository(projectID)
}

// GetWatcher returns the filesystem watcher
func (r *ServiceRegistry) GetWatcher() domain.Watcher {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.watcher
}

// RegisterProject registers a new project and sets up its infrastructure
func (r *ServiceRegistry) RegisterProject(project *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Register project in repository
	projectRepo := r.repositoryFactory.GetProjectRepository()
	if err := projectRepo.Register(context.TODO(), project); err != nil {
		return err
	}

	// Add project to watcher
	if err := r.watcher.AddProject(project.ID, project.IssuesDir); err != nil {
		return err
	}

	// Pre-create issue repository to warm the cache
	_, err := r.repositoryFactory.GetIssueRepository(project.ID)
	return err
}

// UnregisterProject removes a project and cleans up its infrastructure
func (r *ServiceRegistry) UnregisterProject(projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove from watcher
	if err := r.watcher.RemoveProject(projectID); err != nil {
		// Log warning but continue with cleanup - watcher removal is not critical
		_ = err // Ignore watcher errors for now
	}

	// Remove from repository factory cache
	r.repositoryFactory.RemoveIssueRepository(projectID)

	// Delete project from repository
	projectRepo := r.repositoryFactory.GetProjectRepository()
	return projectRepo.Delete(context.TODO(), projectID)
}

// GetAllProjects returns all registered projects
func (r *ServiceRegistry) GetAllProjects() ([]*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectRepo := r.repositoryFactory.GetProjectRepository()
	return projectRepo.List(context.TODO())
}

// GetProjectHealth returns health information for a project
func (r *ServiceRegistry) GetProjectHealth(projectID string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectRepo := r.repositoryFactory.GetProjectRepository()
	return projectRepo.Health(context.TODO(), projectID)
}

// GetSystemStats returns overall system statistics
func (r *ServiceRegistry) GetSystemStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"config_dir":      r.configDir,
		"active_projects": r.repositoryFactory.GetAllProjectIDs(),
		"watcher_stats":   r.watcher.GetStats(),
		"repository_stats": map[string]interface{}{
			"cached_issue_repos": len(r.repositoryFactory.GetAllProjectIDs()),
		},
	}
}

// Close shuts down all services gracefully
func (r *ServiceRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop watcher
	return r.watcher.Stop()
}
