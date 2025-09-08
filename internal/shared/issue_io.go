package shared

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/takl/takl/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadIssueFromFile loads an issue from a markdown file with YAML frontmatter
func LoadIssueFromFile(filePath string) (*domain.Issue, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read until we find the frontmatter
	var inFrontmatter bool
	var frontmatterLines []string
	var contentLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				// End of frontmatter
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if !inFrontmatter && len(frontmatterLines) > 0 {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse frontmatter with strict validation
	var issue domain.Issue
	if len(frontmatterLines) > 0 {
		frontmatterYAML := strings.Join(frontmatterLines, "\n")
		dec := yaml.NewDecoder(bytes.NewReader([]byte(frontmatterYAML)))
		dec.KnownFields(true) // Strict validation - catch typos in field names
		if err := dec.Decode(&issue); err != nil {
			return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
		}
	}

	// Set file path and content
	issue.FilePath = filePath
	issue.Content = strings.Join(contentLines, "\n")

	// Remove the title from content if it's duplicated
	if len(contentLines) > 0 && strings.HasPrefix(contentLines[0], "# "+issue.Title) {
		if len(contentLines) > 2 {
			issue.Content = strings.Join(contentLines[2:], "\n")
		} else {
			issue.Content = ""
		}
	}

	// Normalize ID to ensure consistent formatting
	issue.Normalize()

	return &issue, nil
}

// SaveIssueToFile writes the issue to its file path with YAML frontmatter
func SaveIssueToFile(issue *domain.Issue) error {
	issue.Normalize()
	// NO VALIDATION - validation happens at API layer

	// Marshal frontmatter to YAML
	frontmatterData, err := yaml.Marshal(issue)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Create file content with frontmatter and content
	fileContent := fmt.Sprintf("---\n%s---\n\n# %s\n\n%s", string(frontmatterData), issue.Title, issue.Content)

	// Write to file
	err = os.WriteFile(issue.FilePath, []byte(fileContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
