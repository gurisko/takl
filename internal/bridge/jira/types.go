package jira

import (
	"fmt"
	"regexp"
	"strings"
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

// Member represents a Jira project member
type Member struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Active       bool   `json:"active"`
}

// MemberCache holds cached project members
type MemberCache struct {
	Members map[string]*Member `json:"members"` // Key is account ID
}

// NewMemberCache creates an empty member cache
func NewMemberCache() *MemberCache {
	return &MemberCache{
		Members: make(map[string]*Member),
	}
}

// Add adds a member to the cache
func (mc *MemberCache) Add(member *Member) {
	if mc.Members == nil {
		mc.Members = make(map[string]*Member)
	}
	mc.Members[member.AccountID] = member
}

// FindByAccountID looks up a member by account ID
func (mc *MemberCache) FindByAccountID(accountID string) *Member {
	return mc.Members[accountID]
}

// FindByEmail looks up a member by email address
func (mc *MemberCache) FindByEmail(email string) *Member {
	for _, member := range mc.Members {
		if member.EmailAddress == email {
			return member
		}
	}
	return nil
}

// FindByDisplayName looks up a member by display name
func (mc *MemberCache) FindByDisplayName(displayName string) *Member {
	for _, member := range mc.Members {
		if member.DisplayName == displayName {
			return member
		}
	}
	return nil
}

// FormatMember returns the canonical format: "Display Name <email>"
func (m *Member) FormatMember() string {
	if m.EmailAddress != "" {
		return fmt.Sprintf("%s <%s>", m.DisplayName, m.EmailAddress)
	}
	return m.DisplayName
}

// Regular expression to parse "Display Name <email>" format
var userFormatRegex = regexp.MustCompile(`^(.+?)\s*<([^>]+)>$`)

// ParseUser parses a user string in various formats and resolves to a Member
// Supports:
// - "Display Name <email>" (canonical format)
// - "email@example.com" (email only)
// - "Display Name" (display name only)
//
// Returns the resolved Member and the original string for error messages
func (mc *MemberCache) ParseUser(userStr string) (*Member, error) {
	userStr = strings.TrimSpace(userStr)
	if userStr == "" {
		return nil, fmt.Errorf("empty user string")
	}

	// Try parsing "Display Name <email>" format
	if matches := userFormatRegex.FindStringSubmatch(userStr); matches != nil {
		displayName := strings.TrimSpace(matches[1])
		email := strings.TrimSpace(matches[2])

		// Prefer email lookup (more stable)
		if member := mc.FindByEmail(email); member != nil {
			return member, nil
		}

		// Fallback to display name
		if member := mc.FindByDisplayName(displayName); member != nil {
			return member, nil
		}

		return nil, fmt.Errorf("user not found: %q", userStr)
	}

	// Check if it looks like an email (contains @)
	if strings.Contains(userStr, "@") {
		if member := mc.FindByEmail(userStr); member != nil {
			return member, nil
		}
		return nil, fmt.Errorf("user with email %q not found", userStr)
	}

	// Try as display name
	if member := mc.FindByDisplayName(userStr); member != nil {
		return member, nil
	}

	return nil, fmt.Errorf("user %q not found", userStr)
}
