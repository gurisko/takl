package jira

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Storage handles reading and writing issue markdown files
type Storage struct {
	issuesDir      string
	attachmentsDir string
}

// NewStorage creates a new storage instance
func NewStorage(projectPath string) (*Storage, error) {
	issuesDir := filepath.Join(projectPath, ".takl", "issues")
	attachmentsDir := filepath.Join(projectPath, ".takl", "attachments")

	// Ensure directories exist
	if err := os.MkdirAll(issuesDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create issues directory: %w", err)
	}
	if err := os.MkdirAll(attachmentsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create attachments directory: %w", err)
	}

	return &Storage{
		issuesDir:      issuesDir,
		attachmentsDir: attachmentsDir,
	}, nil
}

// SaveIssue writes an issue to a markdown file
func (s *Storage) SaveIssue(issue *Issue) error {
	// Compute hash before saving
	issue.Hash = s.ComputeHash(issue)

	filePath := filepath.Join(s.issuesDir, issue.JiraKey+".md")

	// Create markdown content
	content := s.issueToMarkdown(issue)

	// Write atomically via temp file in the same directory (for atomic rename)
	tmpFile, err := os.CreateTemp(s.issuesDir, "."+issue.JiraKey+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on any error
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Write content and set permissions
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomically rename temp file to final location
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Success - prevent cleanup from removing the file
	tmpFile = nil
	return nil
}

// ListIssues returns all issue keys in storage
func (s *Storage) ListIssues() ([]string, error) {
	entries, err := os.ReadDir(s.issuesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read issues directory: %w", err)
	}

	var keys []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			key := strings.TrimSuffix(entry.Name(), ".md")
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// ReadExistingHash reads the hash field from an existing issue file.
// Returns the hash and true if found, or empty string and false if not found or on error.
func (s *Storage) ReadExistingHash(key string) (string, bool) {
	filePath := filepath.Join(s.issuesDir, key+".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	// Quick parse: look for "hash: " in the frontmatter
	content := string(data)
	// Find the frontmatter section (between --- markers)
	if !strings.HasPrefix(content, "---\n") {
		return "", false
	}

	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return "", false
	}

	frontmatter := content[4 : 4+endIdx]

	// Parse YAML frontmatter for hash field
	var fm struct {
		Hash string `yaml:"hash"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return "", false
	}

	return fm.Hash, fm.Hash != ""
}

// issueToMarkdown converts an Issue to markdown format
func (s *Storage) issueToMarkdown(issue *Issue) string {
	var buf strings.Builder

	// Write frontmatter
	buf.WriteString("---\n")

	// Marshal frontmatter (excluding Description and Comments)
	frontmatter := map[string]interface{}{
		"jira_key": issue.JiraKey,
		"jira_id":  issue.JiraID,
		"title":    issue.Title,
		"status":   issue.Status,
		"reporter": issue.Reporter,
		"created":  issue.Created.Format(time.RFC3339),
		"updated":  issue.Updated.Format(time.RFC3339),
		"hash":     issue.Hash,
	}

	if issue.Assignee != "" {
		frontmatter["assignee"] = issue.Assignee
	}

	if len(issue.Labels) > 0 {
		frontmatter["labels"] = issue.Labels
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		// This should never happen with our simple data types, but handle it gracefully
		return fmt.Sprintf("ERROR: failed to marshal frontmatter: %v\n", err)
	}
	buf.Write(yamlData)
	buf.WriteString("---\n\n")

	// Write description
	buf.WriteString("# Description\n\n")
	if issue.Description != "" {
		buf.WriteString(issue.Description)
		buf.WriteString("\n\n")
	}

	// Write comments
	if len(issue.Comments) > 0 {
		buf.WriteString("# Comments\n\n")
		for _, comment := range issue.Comments {
			buf.WriteString(fmt.Sprintf("## Comment by %s at %s\n\n",
				comment.Author,
				comment.Created.Format(time.RFC3339)))
			buf.WriteString(comment.Body)
			buf.WriteString("\n\n")
		}
	}

	// Write attachments
	if len(issue.Attachments) > 0 {
		buf.WriteString("# Attachments\n\n")
		for _, att := range issue.Attachments {
			buf.WriteString(fmt.Sprintf("- [%s](%s) (%d bytes, %s)\n",
				att.Filename,
				att.URL,
				att.Size,
				att.Created.Format(time.RFC3339)))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// ComputeHash calculates SHA256 hash of issue content for conflict detection.
//
// Included fields: JiraKey, Title, Description, Status, Labels, Comments
// Excluded fields: Assignee (can change without user action), Attachments (metadata only),
//
//	Created/Updated timestamps, Hash itself
//
// This hash is used to detect when both local and remote copies have been modified
// since the last sync, allowing three-way merge conflict detection.
func (s *Storage) ComputeHash(issue *Issue) string {
	// Create a canonical representation for hashing
	var buf strings.Builder

	buf.WriteString(issue.JiraKey)
	buf.WriteString("|")
	buf.WriteString(issue.Title)
	buf.WriteString("|")
	buf.WriteString(issue.Description)
	buf.WriteString("|")
	buf.WriteString(issue.Status)
	buf.WriteString("|")

	// Sort labels for consistent hashing
	labels := make([]string, len(issue.Labels))
	copy(labels, issue.Labels)
	sort.Strings(labels)
	buf.WriteString(strings.Join(labels, ","))
	buf.WriteString("|")

	// Include comments
	for _, comment := range issue.Comments {
		buf.WriteString(comment.Author)
		buf.WriteString(":")
		buf.WriteString(comment.Body)
		buf.WriteString(";")
	}

	hash := sha256.Sum256([]byte(buf.String()))
	return fmt.Sprintf("%x", hash)
}
