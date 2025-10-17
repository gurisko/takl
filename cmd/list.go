//go:build unix

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/spf13/cobra"
)

type listIssuesResp struct {
	Issues []struct {
		JiraKey  string    `json:"jira_key"`
		Title    string    `json:"title"`
		Status   string    `json:"status"`
		Assignee string    `json:"assignee,omitempty"`
		Reporter string    `json:"reporter"`
		Created  time.Time `json:"created"`
		Updated  time.Time `json:"updated"`
		Labels   []string  `json:"labels,omitempty"`
	} `json:"issues"`
	Count int `json:"count"`
}

var (
	listStatus   string
	listAssignee string
	listLabels   []string
	listSearch   string
	listJSON     bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long: `List issues from the local .takl/issues/ directory with optional filtering.

Examples:
  takl list                              # List all issues
  takl list --status "In Progress"       # Filter by status
  takl list --assignee "John Doe"        # Filter by assignee display name
  takl list --labels bug,urgent          # Filter by labels (must match all)
  takl list --search "database error"    # Search in title and description
  takl list --json                       # Output JSON for piping`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&listStatus, "status", "", "filter by status")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "filter by assignee display name")
	listCmd.Flags().StringSliceVar(&listLabels, "labels", nil, "filter by labels (comma-separated)")
	listCmd.Flags().StringVar(&listSearch, "search", "", "search in title and description")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	// Get current working directory
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("project_path", projectPath)
	if listStatus != "" {
		params.Set("status", listStatus)
	}
	if listAssignee != "" {
		params.Set("assignee", listAssignee)
	}
	if len(listLabels) > 0 {
		params.Set("labels", strings.Join(listLabels, ","))
	}
	if listSearch != "" {
		params.Set("search", listSearch)
	}

	// Make API call
	client := apiclient.New()
	var resp listIssuesResp
	endpoint := "/api/issues?" + params.Encode()
	if err := client.GetJSON(cmd.Context(), endpoint, &resp); err != nil {
		// Provide helpful error message for common issues
		if strings.Contains(err.Error(), "issues directory not found") {
			return fmt.Errorf("no issues found in %s (have you run 'takl jira pull'?)", projectPath)
		}
		return err
	}

	// Output JSON if requested
	if listJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// No issues found
	if resp.Count == 0 {
		fmt.Println("No issues found")
		return nil
	}

	// Output table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tSTATUS\tASSIGNEE\tTITLE")
	for _, issue := range resp.Issues {
		assignee := issue.Assignee
		if assignee == "" {
			assignee = "-"
		}
		// Truncate title if too long
		title := issue.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			issue.JiraKey,
			issue.Status,
			assignee,
			title,
		)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// Print count
	fmt.Printf("\nTotal: %d issue(s)\n", resp.Count)
	return nil
}
