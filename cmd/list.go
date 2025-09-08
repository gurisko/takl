package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takl/takl/sdk"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long: `List issues with optional filtering.

Examples:
  takl list                           # List all issues
  takl list --status=open             # List open issues
  takl list --type=bug                # List bug issues
  takl list --assignee=john@example.com  # List issues assigned to john`,
	RunE: listIssues,
}

var (
	listStatus   string
	listType     string
	listAssignee string
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (open, in-progress/in_progress, done)")
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by type (bug, feature, task, epic)")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "Filter by assignee")
}

func listIssues(cmd *cobra.Command, args []string) error {
	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	client := sdkClient()

	// Build filter request
	req := sdk.ListIssuesRequest{
		Status:   listStatus,
		Type:     listType,
		Assignee: listAssignee,
	}

	issues, err := client.ListIssues(ctx.GetProjectID(), req)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	// Display issues
	fmt.Printf("Found %d issue(s):\n\n", len(issues))

	for _, issue := range issues {
		status := strings.ToUpper(issue.Status)
		priority := strings.ToUpper(issue.Priority)

		fmt.Printf("%-12s [%s] [%s] %s\n",
			issue.ID,
			status,
			priority,
			issue.Title)

		if issue.Assignee != "" {
			fmt.Printf("             → %s\n", issue.Assignee)
		}

		if verbose && len(issue.Labels) > 0 {
			fmt.Printf("             Labels: %v\n", issue.Labels)
		}
		fmt.Println()
	}

	return nil
}
