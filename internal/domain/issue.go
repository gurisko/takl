package domain

import (
	"fmt"
	"strings"
	"time"
)

// Valid issue types and priorities - single source of truth
var (
	ValidIssueTypes  = []string{"bug", "feature", "task", "epic"}
	ValidPriorities  = []string{"low", "medium", "high", "critical"}
	DefaultPriority  = "medium"
	DefaultIssueType = "bug"
)

// Issue represents a work item in the issue tracking system
type Issue struct {
	ID       string    `yaml:"id"`
	Type     string    `yaml:"type"`
	Title    string    `yaml:"title"`
	Status   string    `yaml:"status"`
	Priority string    `yaml:"priority"`
	Assignee string    `yaml:"assignee,omitempty"`
	Labels   []string  `yaml:"labels,omitempty"`
	Created  time.Time `yaml:"created"`
	Updated  time.Time `yaml:"updated"`
	Version  int       `yaml:"version"` // For optimistic concurrency control
	FilePath string    `yaml:"-"`
	Content  string    `yaml:"-"`
}

// ETag generates an ETag header value for HTTP concurrency control
func (i *Issue) ETag() string {
	return fmt.Sprintf("%s@%d", i.ID, i.Version)
}

// Normalize ensures consistent formatting of issue fields
func (i *Issue) Normalize() {
	i.ID = strings.ToUpper(i.ID)
	i.Type = strings.ToLower(i.Type)
}

// GenerateFileName creates a safe filename for the issue
func (i *Issue) GenerateFileName() string {
	// Generate safe filename from ID and title
	safeName := strings.ToLower(i.ID)
	return fmt.Sprintf("%s-%s.md", safeName, slugify(i.Title))
}

// slugify creates a URL-safe string from text
func slugify(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	slug := result.String()
	if len(slug) > 30 {
		slug = slug[:30]
	}
	return strings.Trim(slug, "-")
}

// IssueFilter represents filtering criteria for issues
type IssueFilter struct {
	Status   string
	Type     string
	Priority string
	Assignee string
	Labels   []string
	Since    *time.Time
	Before   *time.Time
	Limit    int
	Offset   int
}

// Project represents a registered project in the system
type Project struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Mode         string    `json:"mode"` // embedded | standalone
	Registered   time.Time `json:"registered"`
	LastSeen     time.Time `json:"last_seen"`
	LastAccess   time.Time `json:"last_access"`
	Active       bool      `json:"active"`
	IssuesDir    string    `json:"issues_dir"`
	DatabasePath string    `json:"database_path"`
	Description  string    `json:"description,omitempty"`
}

// UpdateIssueRequest represents the parameters for updating an issue
type UpdateIssueRequest struct {
	Title    *string
	Content  *string
	Status   *string
	Priority *string
	Assignee *string
	Labels   []string
}
