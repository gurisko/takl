package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takl/takl/sdk"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show issue details",
	Long: `Display the details of an issue by ID.

Examples:
  takl show ISS-001                                   # Show by issue ID
  takl show iss-001                                   # Show by case-insensitive ID
  takl show ISS-001 --json                            # Machine-readable output
  takl show ISS-001 --json -v                         # JSON with content included`,
	Args: cobra.ExactArgs(1),
	RunE: showIssue,
}

var jsonOutput bool

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format for automation")
}

func showIssue(cmd *cobra.Command, args []string) error {
	arg := args[0]

	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	client := sdkClient()

	// Check if argument looks like an issue ID (ISS-xxx format)
	issueIDPattern := regexp.MustCompile(`^ISS-[A-Za-z0-9]+$`)
	var issueID string

	if issueIDPattern.MatchString(strings.ToUpper(arg)) {
		issueID = strings.ToUpper(arg)
	} else {
		// If not an ID pattern, return error - file path support removed
		return fmt.Errorf("please provide an issue ID (format: ISS-XXXXXX)")
	}

	issue, err := client.GetIssue(ctx.GetProjectID(), issueID)
	if err != nil {
		return handleShowError(err, issueID)
	}

	if jsonOutput {
		return outputIssueJSON(issue)
	}

	fmt.Printf("ID: %s\n", issue.ID)
	fmt.Printf("Type: %s\n", issue.Type)
	fmt.Printf("Title: %s\n", issue.Title)
	fmt.Printf("Status: %s\n", issue.Status)
	fmt.Printf("Priority: %s\n", issue.Priority)
	if issue.Assignee != "" {
		fmt.Printf("Assignee: %s\n", issue.Assignee)
	}
	if len(issue.Labels) > 0 {
		fmt.Printf("Labels: %v\n", issue.Labels)
	}
	fmt.Printf("Created: %s\n", issue.Created.Format("2006-01-02 15:04:05"))
	if !issue.Updated.IsZero() {
		fmt.Printf("Updated: %s\n", issue.Updated.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("File: %s\n", issue.FilePath)

	if verbose && issue.Content != "" {
		fmt.Println("\nContent:")
		fmt.Println(issue.Content)
	}

	return nil
}

// JSONIssue represents an issue in JSON format for machine consumption
// This struct defines the stable contract for JSON output used by scripts and automation
type JSONIssue struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Priority string   `json:"priority"`
	Assignee string   `json:"assignee"`
	Labels   []string `json:"labels"`
	Created  string   `json:"created"`
	Updated  string   `json:"updated"`
	File     string   `json:"file"`
	Content  string   `json:"content,omitempty"`
}

func outputIssueJSON(issue *sdk.Issue) error {
	jsonIssue := JSONIssue{
		ID:       issue.ID,
		Type:     issue.Type,
		Title:    issue.Title,
		Status:   issue.Status,
		Priority: issue.Priority,
		Assignee: issue.Assignee,
		Labels:   issue.Labels,
		Created:  issue.Created.Format("2006-01-02T15:04:05Z"),
		File:     issue.FilePath,
	}

	// Ensure labels is never nil for stable contract
	if jsonIssue.Labels == nil {
		jsonIssue.Labels = []string{}
	}

	// Add updated time if set, otherwise use empty string for stable contract
	if !issue.Updated.IsZero() {
		jsonIssue.Updated = issue.Updated.Format("2006-01-02T15:04:05Z")
	} else {
		jsonIssue.Updated = ""
	}

	// Ensure assignee is never empty for stable contract
	if jsonIssue.Assignee == "" {
		jsonIssue.Assignee = ""
	}

	// Include content only if verbose flag is set
	if verbose && issue.Content != "" {
		jsonIssue.Content = issue.Content
	}

	jsonData, err := json.MarshalIndent(jsonIssue, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// handleShowError provides helpful error messages and suggestions when show fails
func handleShowError(err error, issueID string) error {
	errStr := err.Error()

	// Check more specific errors first before generic "not found"

	// For project/context errors
	if strings.Contains(errStr, "project") || strings.Contains(errStr, "context") {
		return fmt.Errorf(`Failed to get issue "%s": %v

Try:
  - takl status                            # Check your project context
  - takl init                              # Initialize TAKL if needed`, issueID, err)
	}

	// For connection or daemon errors
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "daemon") || strings.Contains(errStr, "socket") {
		return fmt.Errorf(`Failed to get issue "%s": %v

Try:
  - takl daemon start                      # Start the TAKL daemon
  - takl status                            # Check daemon status`, issueID, err)
	}

	// Check if this is a general "not found" error (after more specific checks)
	if isNotFoundError(errStr) {
		return fmt.Errorf(`No issue found for "%s".

Try:
  - takl list --type=bug --status=open     # List issues by type and status
  - takl list                              # Show all issues  
  - takl status                            # Check your project context`, issueID)
	}

	// Generic fallback with basic suggestions
	return fmt.Errorf(`Failed to get issue "%s": %v

Try:
  - takl list                              # Show all issues
  - takl status                            # Check your project status`, issueID, err)
}

// isNotFoundError checks if an error indicates a resource was not found
func isNotFoundError(errStr string) bool {
	notFoundPhrases := []string{
		"not found",
		"does not exist",
		"no such issue",
		"issue not found",
		"404",
	}

	lowerErr := strings.ToLower(errStr)
	for _, phrase := range notFoundPhrases {
		if strings.Contains(lowerErr, phrase) {
			return true
		}
	}
	return false
}
