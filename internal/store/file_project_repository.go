package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/shared"
)

// FileProjectRepository implements domain.ProjectRepository using filesystem storage
type FileProjectRepository struct {
	clock     domain.Clock
	configDir string // Directory for storing project configurations
}

// NewFileProjectRepository creates a new file-based project repository
func NewFileProjectRepository(configDir string) *FileProjectRepository {
	return &FileProjectRepository{
		clock:     shared.DefaultClock{},
		configDir: configDir,
	}
}

// Register implements domain.ProjectRepository.Register
func (r *FileProjectRepository) Register(ctx context.Context, project *domain.Project) error {
	// Set registration time if not set
	if project.Registered.IsZero() {
		project.Registered = r.clock.Now()
	}
	project.LastSeen = r.clock.Now()

	// Validate the project
	if err := r.validateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(r.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save to file
	projectFile := filepath.Join(r.configDir, fmt.Sprintf("%s.json", project.ID))
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	return nil
}

// GetByID implements domain.ProjectRepository.GetByID
func (r *FileProjectRepository) GetByID(ctx context.Context, projectID string) (*domain.Project, error) {
	projectFile := filepath.Join(r.configDir, fmt.Sprintf("%s.json", projectID))

	data, err := os.ReadFile(projectFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project %s not found", projectID)
		}
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	var project domain.Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return &project, nil
}

// List implements domain.ProjectRepository.List
func (r *FileProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	if _, err := os.Stat(r.configDir); os.IsNotExist(err) {
		return []*domain.Project{}, nil
	}

	files, err := filepath.Glob(filepath.Join(r.configDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list project files: %w", err)
	}

	var projects []*domain.Project
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files with read errors
		}

		var project domain.Project
		if err := json.Unmarshal(data, &project); err != nil {
			continue // Skip files with parse errors
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

// Update implements domain.ProjectRepository.Update
func (r *FileProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	// Validate the project
	if err := r.validateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	// Update last seen
	project.LastSeen = r.clock.Now()

	// Save to file
	projectFile := filepath.Join(r.configDir, fmt.Sprintf("%s.json", project.ID))
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	return nil
}

// Delete implements domain.ProjectRepository.Delete
func (r *FileProjectRepository) Delete(ctx context.Context, projectID string) error {
	projectFile := filepath.Join(r.configDir, fmt.Sprintf("%s.json", projectID))

	if err := os.Remove(projectFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project %s not found", projectID)
		}
		return fmt.Errorf("failed to delete project file: %w", err)
	}

	return nil
}

// Health implements domain.ProjectRepository.Health
func (r *FileProjectRepository) Health(ctx context.Context, projectID string) (map[string]interface{}, error) {
	project, err := r.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	health := map[string]interface{}{
		"project_id":    project.ID,
		"name":          project.Name,
		"active":        project.Active,
		"mode":          project.Mode,
		"last_seen":     project.LastSeen,
		"last_access":   project.LastAccess,
		"issues_dir":    project.IssuesDir,
		"database_path": project.DatabasePath,
	}

	// Check if directories exist
	health["issues_dir_exists"] = false
	if project.IssuesDir != "" {
		if stat, err := os.Stat(project.IssuesDir); err == nil && stat.IsDir() {
			health["issues_dir_exists"] = true
		}
	}

	health["database_exists"] = false
	if project.DatabasePath != "" {
		if _, err := os.Stat(project.DatabasePath); err == nil {
			health["database_exists"] = true
		}
	}

	// Calculate health score
	score := 0
	checks := []string{}

	if project.Active {
		score += 25
		checks = append(checks, "active")
	}

	if health["issues_dir_exists"].(bool) {
		score += 25
		checks = append(checks, "issues_dir_exists")
	}

	if health["database_exists"].(bool) {
		score += 25
		checks = append(checks, "database_exists")
	}

	// Check if recently accessed (within last 24 hours)
	if time.Since(project.LastAccess) < 24*time.Hour {
		score += 25
		checks = append(checks, "recently_accessed")
	}

	health["score"] = score
	health["checks"] = checks
	health["status"] = "healthy"
	if score < 50 {
		health["status"] = "warning"
	}
	if score < 25 {
		health["status"] = "unhealthy"
	}

	return health, nil
}

// Helper methods

func (r *FileProjectRepository) validateProject(project *domain.Project) error {
	if project.ID == "" {
		return fmt.Errorf("project ID is required")
	}
	if project.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if project.Path == "" {
		return fmt.Errorf("project path is required")
	}
	if project.Mode == "" {
		return fmt.Errorf("project mode is required")
	}
	if project.Mode != "embedded" && project.Mode != "standalone" {
		return fmt.Errorf("project mode must be 'embedded' or 'standalone'")
	}
	return nil
}
