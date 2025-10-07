package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	// ErrProjectNotFound indicates the project ID doesn't exist
	ErrProjectNotFound = errors.New("project not found")
	// ErrProjectAlreadyExists indicates the path is already registered
	ErrProjectAlreadyExists = errors.New("project already exists")
	// ErrInvalidPath indicates the path doesn't exist or is not accessible
	ErrInvalidPath = errors.New("invalid path")
)

// Registry manages the collection of registered projects
type Registry struct {
	filePath string
	data     *RegistryData
	mu       sync.RWMutex
}

// New creates a new Registry instance
func New(filePath string) (*Registry, error) {
	r := &Registry{
		filePath: filePath,
		data: &RegistryData{
			Projects: make(map[string]*Project),
		},
	}

	// Try to load existing registry
	if err := r.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return r, nil
}

// Load reads the registry from disk
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.data = &RegistryData{Projects: make(map[string]*Project)}
			return nil
		}
		return err
	}

	var registryData RegistryData
	if err := yaml.Unmarshal(data, &registryData); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	// Initialize map if nil
	if registryData.Projects == nil {
		registryData.Projects = make(map[string]*Project)
	}

	r.data = &registryData
	return nil
}

// saveNoLock persists registry without locking (caller must hold lock)
func (r *Registry) saveNoLock() error {
	// Ensure parent directory exists
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(r.data)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to temp file for atomic replacement
	f, err := os.CreateTemp(dir, ".projects-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmp := f.Name()
	// Best-effort cleanup if we fail
	defer func() { _ = os.Remove(tmp) }()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write registry: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to fsync registry: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close registry file: %w", err)
	}

	// Atomic replace
	if err := os.Rename(tmp, r.filePath); err != nil {
		return fmt.Errorf("failed to replace registry: %w", err)
	}

	// Ensure directory metadata is persisted
	if dirf, err := os.Open(dir); err == nil {
		_, _ = dirf.Readdirnames(1) // no-op read
		_ = dirf.Sync()
		_ = dirf.Close()
	}

	return nil
}

// RegisterAndSave atomically registers and persists.
// On save failure, the in-memory change is rolled back.
func (r *Registry) RegisterAndSave(project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate path exists and canonicalize it
	absPath, err := filepath.Abs(project.Path)
	if err != nil {
		return fmt.Errorf("%w: path=%s: %v", ErrInvalidPath, project.Path, err)
	}

	// Follow symlinks to get canonical path
	realPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = realPath
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("%w: path=%s: %v", ErrInvalidPath, absPath, err)
	}

	// Check if path already registered
	for _, p := range r.data.Projects {
		if p.Path == absPath {
			return ErrProjectAlreadyExists
		}
	}

	// Set absolute path
	project.Path = absPath

	// Set timestamps if not already set
	if project.RegisteredAt.IsZero() {
		project.RegisteredAt = time.Now().UTC()
	}

	// Generate ID if not set
	if project.ID == "" {
		project.ID = GenerateProjectID()
	}

	// Apply in-memory
	r.data.Projects[project.ID] = project

	// Persist
	if err := r.saveNoLock(); err != nil {
		// Rollback in-memory change
		delete(r.data.Projects, project.ID)
		return fmt.Errorf("persist failed: %w", err)
	}
	return nil
}

// UnregisterAndSave atomically removes and persists.
// On save failure, the removal is rolled back.
func (r *Registry) UnregisterAndSave(projectID string) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	proj, ok := r.data.Projects[projectID]
	if !ok {
		return nil, ErrProjectNotFound
	}

	// Apply in-memory
	delete(r.data.Projects, projectID)

	// Persist
	if err := r.saveNoLock(); err != nil {
		// Rollback removal
		r.data.Projects[projectID] = proj
		return nil, fmt.Errorf("persist failed: %w", err)
	}
	return proj, nil
}

// List returns all registered projects sorted by name then ID
func (r *Registry) List() []*Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]*Project, 0, len(r.data.Projects))
	for _, p := range r.data.Projects {
		projects = append(projects, p)
	}

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Name == projects[j].Name {
			return projects[i].ID < projects[j].ID
		}
		return projects[i].Name < projects[j].Name
	})

	return projects
}
