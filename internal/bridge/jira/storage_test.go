package jira

import (
	"strings"
	"testing"
)

// TestParseAttachments_Simple tests basic attachment parsing
func TestParseAttachments_Simple(t *testing.T) {
	s := &Storage{}
	issue := &Issue{}

	content := `- [document.pdf](https://example.com/file.pdf) (1024 bytes, 2025-01-01 12:00:00)
- [image.png](https://example.com/image.png)`

	s.parseAttachments(issue, content)

	if len(issue.Attachments) != 2 {
		t.Fatalf("Expected 2 attachments, got %d", len(issue.Attachments))
	}

	if issue.Attachments[0].Filename != "document.pdf" {
		t.Errorf("Expected filename 'document.pdf', got %q", issue.Attachments[0].Filename)
	}
	if issue.Attachments[0].URL != "https://example.com/file.pdf" {
		t.Errorf("Expected URL 'https://example.com/file.pdf', got %q", issue.Attachments[0].URL)
	}

	if issue.Attachments[1].Filename != "image.png" {
		t.Errorf("Expected filename 'image.png', got %q", issue.Attachments[1].Filename)
	}
	if issue.Attachments[1].URL != "https://example.com/image.png" {
		t.Errorf("Expected URL 'https://example.com/image.png', got %q", issue.Attachments[1].URL)
	}
}

// TestParseAttachments_URLWithParentheses tests attachment URLs containing parentheses
func TestParseAttachments_URLWithParentheses(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantFile string
		wantURL  string
	}{
		{
			name:     "single parentheses in path",
			content:  "- [file.pdf](https://example.com/path(with)parens/file.pdf)",
			wantFile: "file.pdf",
			wantURL:  "https://example.com/path(with)parens/file.pdf",
		},
		{
			name:     "multiple parentheses",
			content:  "- [doc.pdf](https://example.com/foo(bar)baz(qux)/doc.pdf)",
			wantFile: "doc.pdf",
			wantURL:  "https://example.com/foo(bar)baz(qux)/doc.pdf",
		},
		{
			name:     "nested parentheses",
			content:  "- [test.txt](https://example.com/path(nested(deep))/test.txt)",
			wantFile: "test.txt",
			wantURL:  "https://example.com/path(nested(deep))/test.txt",
		},
		{
			name:     "parentheses with metadata",
			content:  "- [data.csv](https://example.com/path(version2)/data.csv) (2048 bytes, 2025-01-15)",
			wantFile: "data.csv",
			wantURL:  "https://example.com/path(version2)/data.csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{}
			issue := &Issue{}

			s.parseAttachments(issue, tt.content)

			if len(issue.Attachments) != 1 {
				t.Fatalf("Expected 1 attachment, got %d", len(issue.Attachments))
			}

			if issue.Attachments[0].Filename != tt.wantFile {
				t.Errorf("Expected filename %q, got %q", tt.wantFile, issue.Attachments[0].Filename)
			}
			if issue.Attachments[0].URL != tt.wantURL {
				t.Errorf("Expected URL %q, got %q", tt.wantURL, issue.Attachments[0].URL)
			}
		})
	}
}

// TestParseAttachments_MultipleAttachments tests parsing multiple attachments
func TestParseAttachments_MultipleAttachments(t *testing.T) {
	s := &Storage{}
	issue := &Issue{}

	content := `Some description text

Attachments:

- [normal-file.txt](https://example.com/normal.txt)
- [file(with)parens.pdf](https://example.com/path(special)/file.pdf)
- [another.doc](https://example.com/another.doc) (512 bytes)

Some other content`

	s.parseAttachments(issue, content)

	if len(issue.Attachments) != 3 {
		t.Fatalf("Expected 3 attachments, got %d", len(issue.Attachments))
	}

	expected := []struct {
		filename string
		url      string
	}{
		{"normal-file.txt", "https://example.com/normal.txt"},
		{"file(with)parens.pdf", "https://example.com/path(special)/file.pdf"},
		{"another.doc", "https://example.com/another.doc"},
	}

	for i, exp := range expected {
		if issue.Attachments[i].Filename != exp.filename {
			t.Errorf("Attachment %d: expected filename %q, got %q", i, exp.filename, issue.Attachments[i].Filename)
		}
		if issue.Attachments[i].URL != exp.url {
			t.Errorf("Attachment %d: expected URL %q, got %q", i, exp.url, issue.Attachments[i].URL)
		}
	}
}

// TestParseAttachments_NoAttachments tests content without attachments
func TestParseAttachments_NoAttachments(t *testing.T) {
	s := &Storage{}
	issue := &Issue{}

	content := `This is just regular content
with no attachments
and some [inline links](https://example.com) that are not attachments`

	s.parseAttachments(issue, content)

	if len(issue.Attachments) != 0 {
		t.Errorf("Expected 0 attachments, got %d", len(issue.Attachments))
	}
}

// TestParseAttachments_FilenameWithSpecialChars tests filenames with special characters
func TestParseAttachments_FilenameWithSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantFile string
		wantURL  string
	}{
		{
			name:     "spaces in filename",
			content:  "- [my document.pdf](https://example.com/my%20document.pdf)",
			wantFile: "my document.pdf",
			wantURL:  "https://example.com/my%20document.pdf",
		},
		{
			name:     "dashes and underscores",
			content:  "- [test-file_v2.txt](https://example.com/test-file_v2.txt)",
			wantFile: "test-file_v2.txt",
			wantURL:  "https://example.com/test-file_v2.txt",
		},
		{
			name:     "unicode filename",
			content:  "- [文档.pdf](https://example.com/%E6%96%87%E6%A1%A3.pdf)",
			wantFile: "文档.pdf",
			wantURL:  "https://example.com/%E6%96%87%E6%A1%A3.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{}
			issue := &Issue{}

			s.parseAttachments(issue, tt.content)

			if len(issue.Attachments) != 1 {
				t.Fatalf("Expected 1 attachment, got %d", len(issue.Attachments))
			}

			if issue.Attachments[0].Filename != tt.wantFile {
				t.Errorf("Expected filename %q, got %q", tt.wantFile, issue.Attachments[0].Filename)
			}
			if issue.Attachments[0].URL != tt.wantURL {
				t.Errorf("Expected URL %q, got %q", tt.wantURL, issue.Attachments[0].URL)
			}
		})
	}
}

// TestParseAttachments_EdgeCases tests edge cases and malformed inputs
func TestParseAttachments_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantCount   int
		description string
	}{
		{
			name:        "empty content",
			content:     "",
			wantCount:   0,
			description: "Empty content should yield no attachments",
		},
		{
			name:        "missing URL",
			content:     "- [filename.txt]()",
			wantCount:   0,
			description: "Missing URL should be ignored",
		},
		{
			name:        "missing filename",
			content:     "- [](https://example.com/file.pdf)",
			wantCount:   1,
			description: "Empty filename should still parse URL",
		},
		{
			name:        "not a list item",
			content:     "[filename.txt](https://example.com/file.txt)",
			wantCount:   0,
			description: "Non-list items should be ignored",
		},
		{
			name:        "multiple spaces before bracket",
			content:     "-    [file.txt](https://example.com/file.txt)",
			wantCount:   1,
			description: "Multiple spaces should be handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{}
			issue := &Issue{}

			s.parseAttachments(issue, tt.content)

			if len(issue.Attachments) != tt.wantCount {
				t.Errorf("%s: expected %d attachments, got %d", tt.description, tt.wantCount, len(issue.Attachments))
			}
		})
	}
}

// TestParseAttachments_WithMetadata tests parsing attachments with metadata
func TestParseAttachments_WithMetadata(t *testing.T) {
	s := &Storage{}
	issue := &Issue{}

	content := `- [file1.pdf](https://example.com/file1.pdf) (1024 bytes, 2025-01-01 12:00:00)
- [file2.png](https://example.com/file2.png) (2048 bytes)
- [file3.txt](https://example.com/path(version)/file3.txt) (512 bytes, 2025-01-15)`

	s.parseAttachments(issue, content)

	if len(issue.Attachments) != 3 {
		t.Fatalf("Expected 3 attachments, got %d", len(issue.Attachments))
	}

	// Verify all filenames and URLs are correctly extracted despite metadata presence
	expected := []struct {
		filename string
		url      string
	}{
		{"file1.pdf", "https://example.com/file1.pdf"},
		{"file2.png", "https://example.com/file2.png"},
		{"file3.txt", "https://example.com/path(version)/file3.txt"},
	}

	for i, exp := range expected {
		if issue.Attachments[i].Filename != exp.filename {
			t.Errorf("Attachment %d: expected filename %q, got %q", i, exp.filename, issue.Attachments[i].Filename)
		}
		if issue.Attachments[i].URL != exp.url {
			t.Errorf("Attachment %d: expected URL %q, got %q", i, exp.url, issue.Attachments[i].URL)
		}
	}
}

// TestParseAttachments_RealWorldExample tests a realistic attachment section
func TestParseAttachments_RealWorldExample(t *testing.T) {
	s := &Storage{}
	issue := &Issue{}

	// Simulate a real Jira issue markdown file
	content := `Description
===========

This issue tracks the deployment of version 2.0.

## Steps

1. Review changes
2. Test locally
3. Deploy to staging

## Attachments

- [deployment-plan.pdf](https://jira.example.com/secure/attachment/12345/deployment-plan.pdf) (156789 bytes, 2025-01-10 15:30:00)
- [config(prod).yaml](https://jira.example.com/secure/attachment/12346/config(prod).yaml) (2048 bytes, 2025-01-11 09:15:00)
- [screenshot_error(v1.9).png](https://jira.example.com/secure/attachment/12347/screenshot_error(v1.9).png) (45678 bytes, 2025-01-11 10:45:00)

## Comments

Author: John Doe (2025-01-12 14:00:00)

Looks good!`

	// Extract just the attachments section
	lines := strings.Split(content, "\n")
	var attachmentLines []string
	inAttachments := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## Attachments") {
			inAttachments = true
			continue
		}
		if inAttachments && strings.HasPrefix(line, "##") {
			break
		}
		if inAttachments {
			attachmentLines = append(attachmentLines, line)
		}
	}

	s.parseAttachments(issue, strings.Join(attachmentLines, "\n"))

	if len(issue.Attachments) != 3 {
		t.Fatalf("Expected 3 attachments, got %d", len(issue.Attachments))
	}

	expected := []struct {
		filename string
		url      string
	}{
		{"deployment-plan.pdf", "https://jira.example.com/secure/attachment/12345/deployment-plan.pdf"},
		{"config(prod).yaml", "https://jira.example.com/secure/attachment/12346/config(prod).yaml"},
		{"screenshot_error(v1.9).png", "https://jira.example.com/secure/attachment/12347/screenshot_error(v1.9).png"},
	}

	for i, exp := range expected {
		if issue.Attachments[i].Filename != exp.filename {
			t.Errorf("Attachment %d: expected filename %q, got %q", i, exp.filename, issue.Attachments[i].Filename)
		}
		if issue.Attachments[i].URL != exp.url {
			t.Errorf("Attachment %d: expected URL %q, got %q", i, exp.url, issue.Attachments[i].URL)
		}
	}
}
