package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takl/takl/sdk"
)

var (
	updateStatus   string
	updatePriority string
	updateAssignee string
	updateLabels   []string
	updateTitle    string
	updateContent  string
	updateForce    bool
)

var updateCmd = &cobra.Command{
	Use:   "update <issue-id>",
	Short: "Update an existing issue",
	Long: `Update an existing issue with new status, priority, assignee, labels, title, or content.

Status transitions are validated by the project's paradigm (Scrum, Kanban, etc.) unless --force is used.

Examples:
  takl update ISS-001 --status doing
  takl update ISS-002 --priority high --assignee alice
  takl update ISS-003 --status done --force
  takl update ISS-004 --labels "backend,critical" --title "New title"`,
	Args: cobra.ExactArgs(1),
	RunE: updateIssue,
}

func updateIssue(cmd *cobra.Command, args []string) error {
	issueID := args[0]

	// Validate issue ID format
	issueIDPattern := regexp.MustCompile(`^ISS-[A-Za-z0-9]+$`)
	if !issueIDPattern.MatchString(issueID) {
		return fmt.Errorf("invalid issue ID format: %s (expected ISS-XXXXXX)", issueID)
	}

	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	client := sdkClient()

	// Get the current issue to show what we're updating
	issue, err := client.GetIssue(ctx.GetProjectID(), issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue %s: %w", issueID, err)
	}

	fmt.Printf("Updating issue %s: %s\n", issue.ID, issue.Title)

	// Build update request
	updateReq := sdk.UpdateIssueRequest{}
	updated := false

	if updateStatus != "" && updateStatus != issue.Status {
		if !updateForce {
			// TODO: Add paradigm validation - for now just warn
			fmt.Printf("⚠️  Status transition validation not yet implemented - use --force to skip\n")
			if !updateForce {
				return fmt.Errorf("status transition validation required - use --force to skip")
			}
		}
		updateReq.Status = &updateStatus
		updated = true
		fmt.Printf("  Status: %s → %s\n", issue.Status, updateStatus)
	}

	if updatePriority != "" && updatePriority != issue.Priority {
		updateReq.Priority = &updatePriority
		updated = true
		fmt.Printf("  Priority: %s → %s\n", issue.Priority, updatePriority)
	}

	if updateAssignee != "" && updateAssignee != issue.Assignee {
		updateReq.Assignee = &updateAssignee
		updated = true
		fmt.Printf("  Assignee: %s → %s\n", issue.Assignee, updateAssignee)
	}

	if len(updateLabels) > 0 {
		updateReq.Labels = updateLabels
		updated = true
		fmt.Printf("  Labels: [%s] → [%s]\n", strings.Join(issue.Labels, ", "), strings.Join(updateLabels, ", "))
	}

	if updateTitle != "" && updateTitle != issue.Title {
		updateReq.Title = &updateTitle
		updated = true
		fmt.Printf("  Title: %s → %s\n", issue.Title, updateTitle)
	}

	if updateContent != "" && updateContent != issue.Content {
		updateReq.Content = &updateContent
		updated = true
		fmt.Printf("  Content updated\n")
	}

	if !updated {
		fmt.Println("No changes specified.")
		return nil
	}

	// Apply the update via daemon
	updatedIssue, err := client.UpdateIssue(ctx.GetProjectID(), issueID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	fmt.Printf("\n✅ Issue %s updated successfully\n", updatedIssue.ID)
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&updateStatus, "status", "", "New status (open, in-progress, done, etc.)")
	updateCmd.Flags().StringVar(&updatePriority, "priority", "", "New priority (low, medium, high, critical)")
	updateCmd.Flags().StringVar(&updateAssignee, "assignee", "", "New assignee")
	updateCmd.Flags().StringSliceVar(&updateLabels, "labels", nil, "New labels (comma-separated)")
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "New title")
	updateCmd.Flags().StringVar(&updateContent, "content", "", "New content/description")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Skip paradigm validation for status transitions")
}
