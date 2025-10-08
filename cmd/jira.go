//go:build unix

package cmd

import (
	"fmt"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/gurisko/takl/internal/bridge/jira"
	"github.com/spf13/cobra"
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira bridge commands",
	Long:  "Pull issues from Jira to local markdown files",
}

var jiraPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull issues from Jira",
	Long: `Fetch issues from Jira and save them as markdown files in .takl/issues/

The Jira configuration should be in .takl/jira.json with the following format:
{
  "base_url": "https://your-domain.atlassian.net",
  "email": "your-email@example.com",
  "api_token": "your-api-token",
  "project": "PROJ"
}`,
	RunE: runJiraPull,
}

func init() {
	rootCmd.AddCommand(jiraCmd)
	jiraCmd.AddCommand(jiraPullCmd)
}

func runJiraPull(cmd *cobra.Command, args []string) error {
	config, projectPath, err := jira.LoadConfigFromCwd()
	if err != nil {
		return err
	}

	// Create API client
	client := apiclient.New()

	// Prepare request
	reqBody := map[string]interface{}{
		"project_path": projectPath,
		"config":       config,
	}

	// Make API call to daemon
	var result jira.PullResult
	if err := client.PostJSON(cmd.Context(), "/api/jira/pull", reqBody, &result); err != nil {
		return fmt.Errorf("pull request failed: %w", err)
	}

	// Display results
	fmt.Printf("Jira Pull Complete\n")
	fmt.Printf("  Fetched: %d issues\n", result.Fetched)
	fmt.Printf("  Created: %d new issues\n", result.Created)
	fmt.Printf("  Updated: %d existing issues\n", result.Updated)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}
