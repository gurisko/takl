package jira

import (
	"fmt"
	"time"
)

// Issue represents a JIRA issue with all its metadata
type Issue struct {
	JiraKey  string    `yaml:"jira_key" json:"jira_key"`
	JiraID   string    `yaml:"jira_id" json:"jira_id"`
	Title    string    `yaml:"title" json:"title"`
	Status   string    `yaml:"status" json:"status"`
	Assignee string    `yaml:"assignee,omitempty" json:"assignee,omitempty"`
	Reporter string    `yaml:"reporter" json:"reporter"`
	Created  time.Time `yaml:"created" json:"created"`
	Updated  time.Time `yaml:"updated" json:"updated"`
	Labels   []string  `yaml:"labels,omitempty" json:"labels,omitempty"`
	Hash     string    `yaml:"hash" json:"hash"` // SHA256 of content (excluding hash field)

	Description string       `yaml:"-" json:"description,omitempty"` // Not in frontmatter
	Comments    []Comment    `yaml:"-" json:"comments,omitempty"`    // Not in frontmatter
	Attachments []Attachment `yaml:"-" json:"attachments,omitempty"` // Not in frontmatter
}

// Comment represents a comment on an issue
type Comment struct {
	ID      string    `yaml:"-" json:"id,omitempty"`
	Author  string    `yaml:"-" json:"author"`
	Body    string    `yaml:"-" json:"body"`
	Created time.Time `yaml:"-" json:"created"`
	Updated time.Time `yaml:"-" json:"updated,omitempty"`
}

// Attachment represents a file attachment on an issue
type Attachment struct {
	ID       string    `yaml:"-" json:"id,omitempty"`
	Filename string    `yaml:"-" json:"filename"`
	URL      string    `yaml:"-" json:"url"`
	MimeType string    `yaml:"-" json:"mime_type,omitempty"`
	Size     int64     `yaml:"-" json:"size,omitempty"`
	Created  time.Time `yaml:"-" json:"created,omitempty"`
}

// JiraConfig holds Jira connection configuration
type JiraConfig struct {
	BaseURL  string `yaml:"base_url" json:"base_url"`
	Email    string `yaml:"email" json:"email"`
	APIToken string `yaml:"api_token" json:"api_token"`
	Project  string `yaml:"project" json:"project"`
}

// String returns a sanitized string representation (hides API token)
func (c JiraConfig) String() string {
	token := "***REDACTED***"
	if len(c.APIToken) > 4 {
		token = c.APIToken[:4] + "***"
	}
	return fmt.Sprintf("JiraConfig{BaseURL: %s, Email: %s, Token: %s, Project: %s}",
		c.BaseURL, c.Email, token, c.Project)
}
