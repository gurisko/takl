//go:build unix

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

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

var jiraPushCmd = &cobra.Command{
	Use:   "push [issue-key]",
	Short: "Push local changes to Jira",
	Long: `Upload modified issues to Jira.

Only issues with local changes will be pushed. If any issue has been modified
remotely since the last pull, push will fail with a conflict error.

If an issue key is provided (e.g., PROJ-123), only that issue will be pushed.
Otherwise, all changed issues will be pushed.`,
	RunE: runJiraPush,
}

var jiraMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "Fetch and cache project members",
	Long: `Fetch all assignable users from the Jira project and cache them locally.

The member cache is stored in .takl/jira-members.json and is used to resolve
display names and emails to Jira account IDs when pushing changes.`,
	RunE: runJiraMembers,
}

var jiraWorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Fetch and display project workflow statuses",
	Long: `Fetch all statuses for the Jira project and cache them locally.

The workflow cache is stored in .takl/jira-workflow.json and includes status
categories (new, indeterminate, done, undefined) for each status.`,
	RunE: runJiraWorkflow,
}

var membersJSONOutput bool
var workflowJSONOutput bool

func init() {
	rootCmd.AddCommand(jiraCmd)
	jiraCmd.AddCommand(jiraPullCmd)
	jiraCmd.AddCommand(jiraPushCmd)
	jiraCmd.AddCommand(jiraMembersCmd)
	jiraCmd.AddCommand(jiraWorkflowCmd)

	// Add flags for members command
	jiraMembersCmd.Flags().BoolVar(&membersJSONOutput, "json", false, "Output as JSON")

	// Add flags for workflow command
	jiraWorkflowCmd.Flags().BoolVar(&workflowJSONOutput, "json", false, "Output as JSON")
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
	if result.Deleted > 0 {
		fmt.Printf("  Deleted: %d archived/removed issues\n", result.Deleted)
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}

func runJiraPush(cmd *cobra.Command, args []string) error {
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

	// Add optional issue key filter
	if len(args) > 0 {
		reqBody["issue_key"] = args[0]
	}

	// Make API call to daemon
	var result jira.PushResult
	if err := client.PostJSON(cmd.Context(), "/api/jira/push", reqBody, &result); err != nil {
		// Check if it's a conflict error
		if apiErr, ok := err.(*apiclient.APIError); ok && apiErr.StatusCode == 409 {
			// Display conflict information
			fmt.Printf("Error: Cannot push - %d issue(s) have conflicts:\n", len(result.Conflicts))
			for _, conflict := range result.Conflicts {
				fmt.Printf("  - %s: Remote modified (last updated: %s)\n",
					conflict.IssueKey,
					conflict.Updated.Format("2006-01-02 15:04"))
			}
			fmt.Printf("\nRun 'takl jira pull' to fetch remote changes, then push again.\n")
			return fmt.Errorf("push failed due to conflicts")
		}
		return fmt.Errorf("push request failed: %w", err)
	}

	// Display results
	fmt.Printf("Jira Push Complete\n")
	fmt.Printf("  Scanned: %d issues\n", result.Scanned)
	fmt.Printf("  Pushed: %d issues\n", result.Pushed)
	fmt.Printf("  Skipped: %d issues (no changes)\n", result.Skipped)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}

func runJiraMembers(cmd *cobra.Command, args []string) error {
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
	var members []*jira.Member
	if err := client.PostJSON(cmd.Context(), "/api/jira/members", reqBody, &members); err != nil {
		return fmt.Errorf("members request failed: %w", err)
	}

	// Output results
	if membersJSONOutput {
		// JSON output
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(members); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		// Table output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "DISPLAY NAME\tEMAIL\tACCOUNT ID\tACTIVE\n")
		for _, member := range members {
			active := "Yes"
			if !member.Active {
				active = "No"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				member.DisplayName,
				member.EmailAddress,
				member.AccountID,
				active,
			)
		}
		w.Flush()
		fmt.Printf("\nTotal: %d members\n", len(members))
	}

	return nil
}

func runJiraWorkflow(cmd *cobra.Command, args []string) error {
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
	var statuses []*jira.StatusInfo
	if err := client.PostJSON(cmd.Context(), "/api/jira/workflow", reqBody, &statuses); err != nil {
		return fmt.Errorf("workflow request failed: %w", err)
	}

	// Output results
	if workflowJSONOutput {
		// JSON output
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(statuses); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		// Group by category for table output
		categories := map[string][]*jira.StatusInfo{
			"new":           {},
			"indeterminate": {},
			"done":          {},
			"undefined":     {},
		}

		for _, status := range statuses {
			categories[status.Category] = append(categories[status.Category], status)
		}

		// Display by category
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "STATUS\tCATEGORY\tID\n")

		// Display in a logical order: To Do, In Progress, Done, then undefined
		categoryOrder := []struct {
			key  string
			name string
		}{
			{"new", "To Do"},
			{"indeterminate", "In Progress"},
			{"done", "Done"},
			{"undefined", "Undefined"},
		}

		for _, cat := range categoryOrder {
			for _, status := range categories[cat.key] {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					status.Name,
					cat.name,
					status.ID,
				)
			}
		}

		w.Flush()
		fmt.Printf("\nTotal: %d statuses\n", len(statuses))
	}

	return nil
}
