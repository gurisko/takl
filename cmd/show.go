//go:build unix

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/spf13/cobra"
)

type showIssueResp struct {
	Issue struct {
		JiraKey     string    `json:"jira_key"`
		JiraID      string    `json:"jira_id"`
		Title       string    `json:"title"`
		Status      string    `json:"status"`
		Assignee    string    `json:"assignee,omitempty"`
		Reporter    string    `json:"reporter"`
		Created     time.Time `json:"created"`
		Updated     time.Time `json:"updated"`
		Labels      []string  `json:"labels,omitempty"`
		Description string    `json:"description"`
		Comments    []struct {
			Author  string    `json:"author"`
			Body    string    `json:"body"`
			Created time.Time `json:"created"`
		} `json:"comments"`
		Attachments []struct {
			Filename string `json:"filename"`
			URL      string `json:"url"`
		} `json:"attachments"`
	} `json:"issue"`
}

var showJSON bool

var showCmd = &cobra.Command{
	Use:   "show <issue-key>",
	Short: "Show issue details",
	Long: `Display full details of an issue including description, comments, and attachments.

Examples:
  takl show PROJ-123
  takl show TEAM-456`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showJSON, "json", false, "output JSON")
}

func runShow(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	// Get current working directory
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("project_path", projectPath)

	// Make API call
	client := apiclient.New()
	var resp showIssueResp
	endpoint := fmt.Sprintf("/api/issues/%s?%s", url.PathEscape(issueKey), params.Encode())
	if err := client.GetJSON(cmd.Context(), endpoint, &resp); err != nil {
		if apiclient.IsNotFound(err) {
			return fmt.Errorf("issue %q not found in %s", issueKey, projectPath)
		}
		return err
	}

	// Output JSON if requested
	if showJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	issue := resp.Issue

	// Print header
	fmt.Printf("# %s: %s\n\n", issue.JiraKey, issue.Title)

	// Print metadata
	fmt.Printf("Status:   %s\n", issue.Status)
	if issue.Assignee != "" {
		fmt.Printf("Assignee: %s\n", issue.Assignee)
	}
	fmt.Printf("Reporter: %s\n", issue.Reporter)
	fmt.Printf("Created:  %s\n", issue.Created.Format(time.RFC3339))
	fmt.Printf("Updated:  %s\n", issue.Updated.Format(time.RFC3339))

	if len(issue.Labels) > 0 {
		fmt.Printf("Labels:   %s\n", strings.Join(issue.Labels, ", "))
	}

	// Print description
	if issue.Description != "" {
		fmt.Printf("\n## Description\n\n%s\n", issue.Description)
	}

	// Print comments
	if len(issue.Comments) > 0 {
		fmt.Printf("\n## Comments (%d)\n\n", len(issue.Comments))
		for i, comment := range issue.Comments {
			if i > 0 {
				fmt.Println("\n---")
			}
			fmt.Printf("\n**%s** at %s:\n\n%s\n",
				comment.Author,
				comment.Created.Format(time.RFC3339),
				comment.Body)
		}
	}

	// Print attachments
	if len(issue.Attachments) > 0 {
		fmt.Printf("\n## Attachments (%d)\n\n", len(issue.Attachments))
		for _, att := range issue.Attachments {
			fmt.Printf("- [%s](%s)\n", att.Filename, att.URL)
		}
	}

	return nil
}
