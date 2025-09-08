package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/git"
)

// repoProvider allows injecting a mock repository for testing
type repoProvider func(path string) (git.Repository, error)

var openRepo repoProvider = git.OpenRepository

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository status",
	Long:  "Show the status of the TAKL repository and git working directory.",
	RunE:  showStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func showStatus(cmd *cobra.Command, args []string) error {
	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	client := sdkClient()

	// Get project details from SDK
	project, err := client.GetProject(ctx.GetProjectID())
	if err != nil {
		return fmt.Errorf("failed to get project details: %w", err)
	}

	// Display project information
	fmt.Printf("Project: %s (%s)\n", project.Name, project.ID)
	fmt.Printf("Path: %s\n", project.Path)
	fmt.Printf("Mode: %s\n", project.Mode)
	fmt.Printf("Registered: %s\n", project.Registered.Format("2006-01-02 15:04"))
	fmt.Printf("Last access: %s\n", project.LastAccess.Format("2006-01-02 15:04"))
	if project.Description != "" {
		fmt.Printf("Description: %s\n", project.Description)
	}

	// Show daemon status
	daemonStatus, err := client.GetDaemonStatus()
	if err != nil {
		fmt.Printf("Daemon status: Error - %v\n", err)
	} else {
		if daemonStatus.Running {
			fmt.Printf("Daemon status: Running")
			if daemonStatus.Uptime != "" {
				fmt.Printf(" (uptime: %s)", daemonStatus.Uptime)
			}
			fmt.Println()
			if daemonStatus.RequestCount > 0 {
				fmt.Printf("API requests: %d\n", daemonStatus.RequestCount)
			}
		} else {
			fmt.Println("Daemon status: Not running")
		}
	}

	// Try to open git repository
	repo, err := openRepo(project.Path)
	if err != nil {
		fmt.Printf("Git status: Not a git repository or git error: %v\n", err)
		return nil
	}

	clean, err := repo.IsClean()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if clean {
		fmt.Println("Git status: Clean working directory")
	} else {
		fmt.Println("Git status: Uncommitted changes present")
	}

	return nil
}
