//go:build unix

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/spf13/cobra"
)

var (
	pipeStatus   string
	pipeAssignee string
	pipeLabels   []string
	pipeSearch   string
)

var pipeCmd = &cobra.Command{
	Use:   "pipe",
	Short: "Output issues as JSON for piping",
	Long: `Output issues as JSON for Unix pipeline composition.

This command is identical to 'takl list --pipe' but provided as a separate
command for clarity and Unix-style composability.

Examples:
  takl pipe | jq '.issues[] | select(.status=="Open")'
  takl pipe --status Open | jq '.count'
  takl pipe | jq -r '.issues[] | "\(.jira_key): \(.title)"'
  takl pipe --labels bug | jq '.issues[].jira_key'`,
	RunE: runPipe,
}

func init() {
	rootCmd.AddCommand(pipeCmd)
	pipeCmd.Flags().StringVar(&pipeStatus, "status", "", "filter by status")
	pipeCmd.Flags().StringVar(&pipeAssignee, "assignee", "", "filter by assignee display name")
	pipeCmd.Flags().StringSliceVar(&pipeLabels, "labels", nil, "filter by labels (comma-separated)")
	pipeCmd.Flags().StringVar(&pipeSearch, "search", "", "search in title and description")
}

func runPipe(cmd *cobra.Command, args []string) error {
	// Get current working directory
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("project_path", projectPath)
	if pipeStatus != "" {
		params.Set("status", pipeStatus)
	}
	if pipeAssignee != "" {
		params.Set("assignee", pipeAssignee)
	}
	if len(pipeLabels) > 0 {
		params.Set("labels", strings.Join(pipeLabels, ","))
	}
	if pipeSearch != "" {
		params.Set("search", pipeSearch)
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

	// Output JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}
