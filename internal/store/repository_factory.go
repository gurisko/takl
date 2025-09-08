package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/takl/takl/internal/domain"
)

// RepositoryFactory creates and manages repositories for projects
type RepositoryFactory struct {
	projectRepo   domain.ProjectRepository
	issueRepos    map[string]domain.IssueRepository
	issueReposMux sync.RWMutex
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(configDir string) *RepositoryFactory {
	return &RepositoryFactory{
		projectRepo: NewFileProjectRepository(configDir),
		issueRepos:  make(map[string]domain.IssueRepository),
	}
}

// GetProjectRepository returns the project repository
func (f *RepositoryFactory) GetProjectRepository() domain.ProjectRepository {
	return f.projectRepo
}

// GetIssueRepository returns an issue repository for the given project
// Creates a new one if it doesn't exist
func (f *RepositoryFactory) GetIssueRepository(projectID string) (domain.IssueRepository, error) {
	f.issueReposMux.RLock()
	if repo, exists := f.issueRepos[projectID]; exists {
		f.issueReposMux.RUnlock()
		return repo, nil
	}
	f.issueReposMux.RUnlock()

	f.issueReposMux.Lock()
	defer f.issueReposMux.Unlock()

	// Double-check pattern
	if repo, exists := f.issueRepos[projectID]; exists {
		return repo, nil
	}

	// Get project to determine issues directory
	project, err := f.projectRepo.GetByID(context.TODO(), projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", projectID, err)
	}

	// Create new issue repository
	issueRepo := NewFileIssueRepository(projectID, project.IssuesDir)
	f.issueRepos[projectID] = issueRepo

	return issueRepo, nil
}

// RemoveIssueRepository removes an issue repository for the given project
func (f *RepositoryFactory) RemoveIssueRepository(projectID string) {
	f.issueReposMux.Lock()
	defer f.issueReposMux.Unlock()
	delete(f.issueRepos, projectID)
}

// GetAllProjectIDs returns all project IDs that have repositories
func (f *RepositoryFactory) GetAllProjectIDs() []string {
	f.issueReposMux.RLock()
	defer f.issueReposMux.RUnlock()

	ids := make([]string, 0, len(f.issueRepos))
	for id := range f.issueRepos {
		ids = append(ids, id)
	}
	return ids
}
