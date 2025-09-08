package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Project struct {
	ID           string    `yaml:"id"`                    // Unique project identifier
	Name         string    `yaml:"name"`                  // Human-readable name
	Path         string    `yaml:"path"`                  // Absolute path to project
	Mode         string    `yaml:"mode"`                  // embedded | standalone
	Registered   time.Time `yaml:"registered"`            // When project was registered
	LastSeen     time.Time `yaml:"last_seen"`             // Last successful health check
	LastAccess   time.Time `yaml:"last_access"`           // Last CLI access
	Active       bool      `yaml:"active"`                // Currently loaded in daemon
	IssuesDir    string    `yaml:"issues_dir"`            // Path to issues directory
	DatabasePath string    `yaml:"database_path"`         // Path to project database
	Description  string    `yaml:"description,omitempty"` // Optional description
}

type Registry struct {
	Projects map[string]*Project `yaml:"projects"`
	mu       sync.RWMutex
	path     string
}

func New() (*Registry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	taklDir := filepath.Join(homeDir, ".takl")
	if err := os.MkdirAll(taklDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create takl directory: %w", err)
	}

	// Create projects directory
	projectsDir := filepath.Join(taklDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create projects directory: %w", err)
	}

	registryPath := filepath.Join(taklDir, "projects.yaml")

	registry := &Registry{
		Projects: make(map[string]*Project),
		path:     registryPath,
	}

	if err := registry.load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return registry, nil
}

func (r *Registry) RegisterProject(path, name, description string) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize and validate path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if already registered
	for _, project := range r.Projects {
		if project.Path == absPath {
			return project, fmt.Errorf("project already registered at %s", absPath)
		}
	}

	// Detect project structure
	mode, issuesDir, err := r.detectProjectStructure(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project structure: %w", err)
	}

	// Generate unique ID
	projectID := r.generateProjectID(absPath, name)

	// Create project entry
	project := &Project{
		ID:           projectID,
		Name:         name,
		Path:         absPath,
		Mode:         mode,
		Registered:   time.Now(),
		LastSeen:     time.Now(),
		LastAccess:   time.Now(),
		Active:       false,
		IssuesDir:    issuesDir,
		DatabasePath: filepath.Join(os.Getenv("HOME"), ".takl", "projects", projectID+".db"),
		Description:  description,
	}

	r.Projects[projectID] = project

	if err := r.save(); err != nil {
		delete(r.Projects, projectID)
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return project, nil
}

func (r *Registry) UnregisterProject(projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, exists := r.Projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Remove database file
	if err := os.Remove(project.DatabasePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove database: %w", err)
	}

	delete(r.Projects, projectID)

	return r.save()
}

func (r *Registry) GetProject(projectID string) (*Project, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, exists := r.Projects[projectID]
	return project, exists
}

func (r *Registry) FindProjectByPath(path string) (*Project, bool) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, project := range r.Projects {
		if project.Path == absPath {
			return project, true
		}
	}

	return nil, false
}

func (r *Registry) ListProjects() []*Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]*Project, 0, len(r.Projects))
	for _, project := range r.Projects {
		projects = append(projects, project)
	}

	return projects
}

func (r *Registry) UpdateLastAccess(projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if project, exists := r.Projects[projectID]; exists {
		project.LastAccess = time.Now()
		return r.save()
	}

	return fmt.Errorf("project not found: %s", projectID)
}

func (r *Registry) HealthCheck() ([]*Project, []*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var healthy, unhealthy []*Project

	for _, project := range r.Projects {
		if r.isProjectHealthy(project) {
			project.LastSeen = time.Now()
			healthy = append(healthy, project)
		} else {
			unhealthy = append(unhealthy, project)
		}
	}

	if err := r.save(); err != nil {
		return healthy, unhealthy, err
	}

	return healthy, unhealthy, nil
}

func (r *Registry) CleanupStaleProjects(olderThan time.Duration) ([]*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	var removed []*Project

	for id, project := range r.Projects {
		if !project.Active && project.LastAccess.Before(cutoff) && project.LastSeen.Before(cutoff) {
			// Remove database file
			os.Remove(project.DatabasePath)

			removed = append(removed, project)
			delete(r.Projects, id)
		}
	}

	if len(removed) > 0 {
		if err := r.save(); err != nil {
			return removed, err
		}
	}

	return removed, nil
}

func (r *Registry) detectProjectStructure(path string) (mode, issuesDir string, err error) {
	// Check for embedded mode (.takl directory)
	taklDir := filepath.Join(path, ".takl")
	if stat, err := os.Stat(taklDir); err == nil && stat.IsDir() {
		issuesPath := filepath.Join(taklDir, "issues")
		if _, err := os.Stat(issuesPath); err == nil {
			return "embedded", issuesPath, nil
		}
	}

	// Check for standalone mode (.issues directory)
	issuesPath := filepath.Join(path, ".issues")
	if stat, err := os.Stat(issuesPath); err == nil && stat.IsDir() {
		return "standalone", issuesPath, nil
	}

	return "", "", fmt.Errorf("no TAKL structure found (no .takl/issues or .issues directory)")
}

func (r *Registry) isProjectHealthy(project *Project) bool {
	// Check if project directory exists
	if _, err := os.Stat(project.Path); os.IsNotExist(err) {
		return false
	}

	// Check if issues directory exists
	if _, err := os.Stat(project.IssuesDir); os.IsNotExist(err) {
		return false
	}

	// Check if it's still a git repository (for embedded mode)
	if project.Mode == "embedded" {
		gitDir := filepath.Join(project.Path, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

func (r *Registry) generateProjectID(path, name string) string {
	// Generate ID from path hash + timestamp for uniqueness
	hash := 0
	for _, c := range path {
		hash = hash*31 + int(c)
	}

	timestamp := time.Now().Unix()
	return fmt.Sprintf("proj-%x-%x", hash&0x7fffffff, timestamp&0xffff)
}

func (r *Registry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // New registry, no file yet
		}
		return err
	}

	// Use strict decoding to catch any structural issues
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(r)
}

func (r *Registry) save() error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	return os.WriteFile(r.path, data, 0644)
}
