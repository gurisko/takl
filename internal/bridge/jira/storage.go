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
	projectPath    string
	issuesDir      string
	attachmentsDir string
}

// NewStorage creates a new storage instance and ensures directories exist (for write operations)
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
		projectPath:    projectPath,
		issuesDir:      issuesDir,
		attachmentsDir: attachmentsDir,
	}, nil
}

// OpenStorage opens existing storage for read-only operations (does not create directories)
func OpenStorage(projectPath string) (*Storage, error) {
	issuesDir := filepath.Join(projectPath, ".takl", "issues")
	attachmentsDir := filepath.Join(projectPath, ".takl", "attachments")

	// Verify issues directory exists
	if st, err := os.Stat(issuesDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("issues directory not found at %s (have you run 'takl jira pull'?)", issuesDir)
	}

	return &Storage{
		projectPath:    projectPath,
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

// DeleteIssue deletes an issue file from local storage
func (s *Storage) DeleteIssue(key string) error {
	filePath := filepath.Join(s.issuesDir, key+".md")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, not an error
		}
		return fmt.Errorf("failed to delete issue %s: %w", key, err)
	}
	return nil
}

// ReadIssue reads a single issue from disk
func (s *Storage) ReadIssue(key string) (*Issue, error) {
	filePath := filepath.Join(s.issuesDir, key+".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("issue not found: %s", key)
		}
		return nil, fmt.Errorf("failed to read issue file: %w", err)
	}

	return s.parseMarkdown(string(data))
}

// IssueFilter defines filtering criteria for issues
type IssueFilter struct {
	Status   string   // Filter by status (empty = all)
	Assignee string   // Filter by assignee (empty = all)
	Labels   []string // Filter by labels (empty = all, must match ALL provided labels)
	Search   string   // Search in title and description (empty = no search)
}

// ListAllIssues returns all issues with their metadata
func (s *Storage) ListAllIssues() ([]*Issue, error) {
	return s.ListFilteredIssues(IssueFilter{})
}

// ListFilteredIssues returns issues matching the provided filter
func (s *Storage) ListFilteredIssues(filter IssueFilter) ([]*Issue, error) {
	keys, err := s.ListIssues()
	if err != nil {
		return nil, err
	}

	issues := make([]*Issue, 0, len(keys))
	for _, key := range keys {
		issue, err := s.ReadIssue(key)
		if err != nil {
			// Log but don't fail the entire list
			fmt.Fprintf(os.Stderr, "Warning: failed to read issue %s: %v\n", key, err)
			continue
		}

		// Apply filters
		if !s.matchesFilter(issue, filter) {
			continue
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// matchesFilter checks if an issue matches the filter criteria
func (s *Storage) matchesFilter(issue *Issue, filter IssueFilter) bool {
	// Status filter
	if filter.Status != "" && !strings.EqualFold(issue.Status, filter.Status) {
		return false
	}

	// Assignee filter - case-insensitive substring match (works for name or email)
	if a := strings.TrimSpace(filter.Assignee); a != "" {
		al, il := strings.ToLower(a), strings.ToLower(issue.Assignee)
		if il == "" || !(il == al || strings.Contains(il, al)) {
			return false
		}
	}

	// Labels filter (must have ALL specified labels)
	if len(filter.Labels) > 0 {
		issueLabels := make(map[string]bool)
		for _, label := range issue.Labels {
			issueLabels[strings.ToLower(label)] = true
		}
		for _, filterLabel := range filter.Labels {
			if !issueLabels[strings.ToLower(filterLabel)] {
				return false
			}
		}
	}

	// Search filter (case-insensitive search in title and description)
	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		titleLower := strings.ToLower(issue.Title)
		descLower := strings.ToLower(issue.Description)
		if !strings.Contains(titleLower, searchLower) && !strings.Contains(descLower, searchLower) {
			return false
		}
	}

	return true
}

// parseMarkdown parses a markdown file into an Issue struct
func (s *Storage) parseMarkdown(content string) (*Issue, error) {
	// Normalize newlines (handle CRLF)
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Expect frontmatter between --- markers
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("invalid issue file: missing frontmatter")
	}

	// Find end of frontmatter
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, fmt.Errorf("invalid issue file: malformed frontmatter")
	}

	frontmatter := content[4 : 4+endIdx]
	body := content[4+endIdx+5:] // Skip "\n---\n"

	// Parse frontmatter
	var issue Issue
	if err := yaml.Unmarshal([]byte(frontmatter), &issue); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse body sections
	s.parseBody(&issue, body)

	return &issue, nil
}

// parseBody parses the markdown body into description, comments, and attachments
// Recognizes setext-style headings (underlined with = or -)
func (s *Storage) parseBody(issue *Issue, body string) {
	var sec string
	var buf strings.Builder

	flush := func() {
		t := strings.TrimSpace(buf.String())
		switch sec {
		case "Description":
			issue.Description = t
		case "Comments":
			s.parseComments(issue, t)
		case "Attachments":
			s.parseAttachments(issue, t)
		}
		buf.Reset()
	}

	lines := strings.Split(body, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check if this is a setext-style heading
		// (line followed by a line of = or - characters)
		if i+1 < len(lines) {
			nextLine := lines[i+1]
			// Check if next line is all = or all -
			if len(nextLine) > 0 && (isAllChars(nextLine, '=') || isAllChars(nextLine, '-')) {
				// This is a section heading
				flush()
				sec = strings.TrimSpace(line)
				i += 2 // Skip both the heading line and the underline
				continue
			}
		}

		// Regular content line
		buf.WriteString(line)
		buf.WriteByte('\n')
		i++
	}
	flush()
}

// isAllChars checks if a string consists entirely of a specific character (and optional whitespace)
func isAllChars(s string, ch rune) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r != ch {
			return false
		}
	}
	return true
}

// parseComments parses the comments section
func (s *Storage) parseComments(issue *Issue, content string) {
	// Split by ## Comment markers
	commentSections := strings.Split(content, "\n## Comment by ")

	for _, cs := range commentSections {
		cs = strings.TrimSpace(cs)
		if cs == "" {
			continue
		}

		// Parse "author at timestamp"
		lines := strings.SplitN(cs, "\n", 2)
		if len(lines) < 2 {
			continue
		}

		header := lines[0]
		body := strings.TrimSpace(lines[1])

		// Extract author and timestamp
		parts := strings.SplitN(header, " at ", 2)
		if len(parts) != 2 {
			continue
		}

		author := parts[0]
		timestamp, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			// Try without parsing if it fails
			timestamp = time.Time{}
		}

		issue.Comments = append(issue.Comments, Comment{
			Author:  author,
			Body:    body,
			Created: timestamp,
		})
	}
}

// parseAttachments parses the attachments section
// Format: - [filename](url) (size bytes, timestamp)
// Handles URLs with parentheses by counting balanced parentheses
func (s *Storage) parseAttachments(issue *Issue, content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [") && !strings.HasPrefix(line, "-    [") {
			continue
		}

		// Extract filename: find [ and ]
		filenameStart := strings.Index(line, "[")
		if filenameStart == -1 {
			continue
		}
		filenameEnd := strings.Index(line[filenameStart:], "]")
		if filenameEnd == -1 {
			continue
		}
		filenameEnd += filenameStart

		filename := line[filenameStart+1 : filenameEnd]
		if filename == "" {
			// Empty filename - skip unless there's a URL
			// (allow for edge case testing)
		}

		// Check for ]( after filename
		if filenameEnd+2 >= len(line) || line[filenameEnd:filenameEnd+2] != "](" {
			continue
		}

		// Extract URL by finding matching closing parenthesis
		urlStart := filenameEnd + 2
		urlEnd := findMatchingParen(line, urlStart)
		if urlEnd == -1 {
			continue
		}

		url := line[urlStart:urlEnd]
		if url == "" {
			continue
		}

		// Skip if both filename and URL are empty
		if filename == "" && url == "" {
			continue
		}

		issue.Attachments = append(issue.Attachments, Attachment{
			Filename: filename,
			URL:      url,
		})
	}
}

// findMatchingParen finds the position of the closing parenthesis that matches
// the opening parenthesis at position start-1, handling nested parentheses.
// Returns -1 if no matching parenthesis is found.
func findMatchingParen(s string, start int) int {
	depth := 1 // We're already inside one level of parens
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
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
		// Sort labels for stable output and reduced diff noise
		frontmatter["labels"] = normalizeLabels(issue.Labels)
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		// This should never happen with our simple data types, but handle it gracefully
		return fmt.Sprintf("ERROR: failed to marshal frontmatter: %v\n", err)
	}
	buf.Write(yamlData)
	buf.WriteString("---\n\n")

	// Write description (setext-style heading)
	buf.WriteString("Description\n")
	buf.WriteString("===========\n\n")
	if issue.Description != "" {
		buf.WriteString(issue.Description)
		buf.WriteString("\n\n")
	}

	// Write comments (setext-style heading)
	if len(issue.Comments) > 0 {
		buf.WriteString("Comments\n")
		buf.WriteString("========\n\n")
		for _, comment := range issue.Comments {
			buf.WriteString(fmt.Sprintf("## Comment by %s at %s\n\n",
				comment.Author,
				comment.Created.Format(time.RFC3339)))
			buf.WriteString(comment.Body)
			buf.WriteString("\n\n")
		}
	}

	// Write attachments (setext-style heading)
	if len(issue.Attachments) > 0 {
		buf.WriteString("Attachments\n")
		buf.WriteString("===========\n\n")
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

// normalizeLabels returns a sorted copy of the labels slice.
// This ensures consistent ordering for storage, API updates, and hashing.
func normalizeLabels(labels []string) []string {
	if len(labels) == 0 {
		return labels
	}
	normalized := make([]string, len(labels))
	copy(normalized, labels)
	sort.Strings(normalized)
	return normalized
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
	buf.WriteString(strings.Join(normalizeLabels(issue.Labels), ","))
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
