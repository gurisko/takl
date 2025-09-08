package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takl/takl/sdk"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search issues",
	Long: `Search issues by title and content.

Examples:
  takl search "login bug"             # Search for login bug
  takl search authentication          # Search for authentication
  takl search --global "API error"    # Search across all projects`,
	Args: cobra.ExactArgs(1),
	RunE: searchIssues,
}

var globalSearch bool

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolVar(&globalSearch, "global", false, "Search across all registered projects")
}

func searchIssues(cmd *cobra.Command, args []string) error {
	query := args[0]

	client := sdkClient()

	if globalSearch {
		// Global search across all projects
		results, err := client.GlobalSearchIssues(query)
		if err != nil {
			return fmt.Errorf("global search failed: %w", err)
		}

		return displayGlobalSearchResults(query, results)
	}

	// Project-specific search
	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	issues, err := client.SearchIssues(ctx.GetProjectID(), query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	return displaySearchResults(query, issues)
}

func displaySearchResults(query string, issues []*sdk.Issue) error {
	if len(issues) == 0 {
		fmt.Printf("No issues found matching '%s'\n", query)
		return nil
	}

	fmt.Printf("Found %d issue(s) matching '%s':\n\n", len(issues), query)

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

		if verbose && issue.Content != "" {
			// Show a snippet of content for search context
			content := strings.ReplaceAll(issue.Content, "\n", " ")
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			fmt.Printf("             %s\n", content)
		}
		fmt.Println()
	}

	return nil
}

func displayGlobalSearchResults(query string, results map[string]interface{}) error {
	totalResults := 0
	if total, ok := results["total_results"].(float64); ok {
		totalResults = int(total)
	}

	if totalResults == 0 {
		fmt.Printf("No issues found matching '%s' across all projects\n", query)
		return nil
	}

	fmt.Printf("Found %d issue(s) matching '%s' across all projects:\n\n", totalResults, query)

	if projectResults, ok := results["results"].(map[string]interface{}); ok {
		for projectID, projectData := range projectResults {
			if project, ok := projectData.(map[string]interface{}); ok {
				if projectName, ok := project["project"].(string); ok {
					if issues, ok := project["issues"].([]interface{}); ok {
						fmt.Printf("📁 %s (%s):\n", projectName, projectID)

						for _, issueData := range issues {
							if issue, ok := issueData.(map[string]interface{}); ok {
								id, _ := issue["id"].(string)
								title, _ := issue["title"].(string)
								status, _ := issue["status"].(string)
								priority, _ := issue["priority"].(string)

								fmt.Printf("   %-12s [%s] [%s] %s\n",
									id,
									strings.ToUpper(status),
									strings.ToUpper(priority),
									title)
							}
						}
						fmt.Println()
					}
				}
			}
		}
	}

	return nil
}
