package dto

import (
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/registry"
)

// RegisterProjectRequest represents the HTTP request for registering a project
type RegisterProjectRequest struct {
	ID          string `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Path        string `json:"path" binding:"required"`
	Mode        string `json:"mode" binding:"required"`
	Description string `json:"description"`
}

// ToDomain converts HTTP DTO to domain project
func (r RegisterProjectRequest) ToDomain() *domain.Project {
	return &domain.Project{
		ID:          r.ID,
		Name:        r.Name,
		Path:        r.Path,
		Mode:        r.Mode,
		Description: r.Description,
		Active:      false, // Will be set by the service
		// Other fields will be set by the service layer
	}
}

// ProjectResponse represents the HTTP response for a project
type ProjectResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Mode         string    `json:"mode"`
	Registered   time.Time `json:"registered"`
	LastSeen     time.Time `json:"last_seen"`
	LastAccess   time.Time `json:"last_access"`
	Active       bool      `json:"active"`
	IssuesDir    string    `json:"issues_dir"`
	DatabasePath string    `json:"database_path"`
	Description  string    `json:"description,omitempty"`
}

// FromRegistryProject creates HTTP response from registry project
func FromRegistryProject(project *registry.Project) *ProjectResponse {
	return &ProjectResponse{
		ID:           project.ID,
		Name:         project.Name,
		Path:         project.Path,
		Mode:         project.Mode,
		Registered:   project.Registered,
		LastSeen:     project.LastSeen,
		LastAccess:   project.LastAccess,
		Active:       project.Active,
		IssuesDir:    project.IssuesDir,
		DatabasePath: project.DatabasePath,
		Description:  project.Description,
	}
}

// ProjectResponseFromDomain creates HTTP response from domain project (legacy)
func ProjectResponseFromDomain(project *domain.Project) ProjectResponse {
	return ProjectResponse{
		ID:           project.ID,
		Name:         project.Name,
		Path:         project.Path,
		Mode:         project.Mode,
		Registered:   project.Registered,
		LastSeen:     project.LastSeen,
		LastAccess:   project.LastAccess,
		Active:       project.Active,
		IssuesDir:    project.IssuesDir,
		DatabasePath: project.DatabasePath,
		Description:  project.Description,
	}
}

// ProjectListResponse represents a list of projects
type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int               `json:"total"`
}

// HealthResponse represents project health status
type HealthResponse struct {
	ProjectID string                 `json:"project_id"`
	Status    string                 `json:"status"`
	Score     int                    `json:"score"`
	Checks    []HealthCheck          `json:"checks"`
	Issues    []string               `json:"issues,omitempty"`
	Warnings  []string               `json:"warnings,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Name        string      `json:"name"`
	Status      string      `json:"status"`
	Message     string      `json:"message"`
	Details     interface{} `json:"details,omitempty"`
	Duration    string      `json:"duration,omitempty"`
	LastChecked time.Time   `json:"last_checked"`
}
