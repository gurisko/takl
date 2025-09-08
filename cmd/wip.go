package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/config"
	"github.com/takl/takl/sdk"
)

var wipCmd = &cobra.Command{
	Use:   "wip",
	Short: "Show WIP status and limits",
	Long: `Display Work-In-Progress status and limits for Kanban projects.

Shows the current count of issues in each column versus configured WIP limits.

Examples:
  takl wip status          # Show current WIP status
  takl wip                 # Same as 'wip status'`,
	Args: cobra.MaximumNArgs(1),
	RunE: handleWIP,
}

var wipStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show WIP limits and current counts",
	Long:  `Display current WIP status showing issue counts vs limits for each column.`,
	RunE:  showWIPStatus,
}

func init() {
	rootCmd.AddCommand(wipCmd)
	wipCmd.AddCommand(wipStatusCmd)

	// Default to status subcommand if no args
	wipCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return showWIPStatus(cmd, args)
		}
		return cmd.Help()
	}
}

func handleWIP(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] == "status" {
		return showWIPStatus(cmd, args)
	}
	return cmd.Help()
}

func showWIPStatus(cmd *cobra.Command, args []string) error {
	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	// Load project configuration
	cfg, err := config.Load(ctx.GetProjectPath())
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Check if project uses Kanban paradigm
	if cfg.Paradigm.ID != "kanban" {
		return fmt.Errorf("WIP limits are only available for Kanban projects (current paradigm: %s)", cfg.Paradigm.ID)
	}

	// Extract WIP limits from paradigm options
	wipLimits := make(map[string]int)
	if limits, ok := cfg.Paradigm.Options["wip_limits"].(map[string]interface{}); ok {
		for column, limit := range limits {
			if limitInt, ok := limit.(int); ok {
				wipLimits[column] = limitInt
			}
		}
	}

	if len(wipLimits) == 0 {
		fmt.Println("No WIP limits configured for this project")
		fmt.Println("To configure WIP limits, add them to .takl/config.yaml:")
		fmt.Println(`
paradigm:
  id: kanban
  options:
    wip_limits:
      doing: 3
      review: 2`)
		return nil
	}

	// Get current issue counts by status
	client := sdkClient()

	issues, err := client.ListIssues(ctx.GetProjectID(), sdk.ListIssuesRequest{})
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Count issues by status
	statusCounts := make(map[string]int)
	for _, issue := range issues {
		statusCounts[issue.Status]++
	}

	// Display WIP status table
	fmt.Printf("WIP Status for %s:\n\n", cfg.Project.Name)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Column\tCurrent\tLimit\tStatus\tUsage")
	fmt.Fprintln(w, strings.Repeat("-", 6)+"\t"+strings.Repeat("-", 7)+"\t"+strings.Repeat("-", 5)+"\t"+strings.Repeat("-", 6)+"\t"+strings.Repeat("-", 5))

	for column, limit := range wipLimits {
		current := statusCounts[column]
		status := "🟢 OK"
		usage := fmt.Sprintf("%d%%", (current*100)/limit)

		if current >= limit {
			status = "🔴 FULL"
			usage = "100%"
		} else if current >= (limit*80)/100 {
			status = "🟡 HIGH"
		}

		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\n",
			column, current, limit, status, usage)
	}

	w.Flush()

	// Summary
	totalAtCapacity := 0
	totalNearCapacity := 0
	for column, limit := range wipLimits {
		current := statusCounts[column]
		if current >= limit {
			totalAtCapacity++
		} else if current >= (limit*80)/100 {
			totalNearCapacity++
		}
	}

	fmt.Printf("\nSummary:\n")
	if totalAtCapacity > 0 {
		fmt.Printf("🔴 %d column(s) at capacity - consider moving work to review/done\n", totalAtCapacity)
	}
	if totalNearCapacity > 0 {
		fmt.Printf("🟡 %d column(s) nearing capacity\n", totalNearCapacity)
	}
	if totalAtCapacity == 0 && totalNearCapacity == 0 {
		fmt.Printf("🟢 All columns within WIP limits\n")
	}

	return nil
}
