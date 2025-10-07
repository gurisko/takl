package registry

import "time"

// Project represents a registered project in the TAKL registry
type Project struct {
	ID           string    `yaml:"id" json:"id"`                       // UUID v4
	Name         string    `yaml:"name" json:"name"`                   // Human-readable project name
	Path         string    `yaml:"path" json:"path"`                   // Absolute path to project directory
	RegisteredAt time.Time `yaml:"registered_at" json:"registered_at"` // When project was registered
}

// RegistryData holds all registered projects
type RegistryData struct {
	Projects map[string]*Project `yaml:"projects" json:"projects"` // Map of project ID to Project
}
