package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/clock"
	"github.com/takl/takl/sdk"
)

var registerCmd = &cobra.Command{
	Use:   "register [path] [name]",
	Short: "Register a TAKL project",
	Long: `Register a TAKL project in the global registry.

The project will be monitored for health and can be accessed from anywhere.

Examples:
  takl register                          # Register current directory
  takl register . "My Project"           # Register with custom name
  takl register /path/to/project         # Register specific path
  takl register --list                   # List all registered projects
  takl register --cleanup                # Remove stale projects`,
	Args: cobra.MaximumNArgs(2),
	RunE: handleRegister,
}

var (
	listProjects    bool
	cleanupProjects bool
	forceRegister   bool
	projectDesc     string
	removeProject   string
)

func init() {
	rootCmd.AddCommand(registerCmd)

	registerCmd.Flags().BoolVar(&listProjects, "list", false,
		"list all registered projects")
	registerCmd.Flags().BoolVar(&cleanupProjects, "cleanup", false,
		"remove stale/inactive projects")
	registerCmd.Flags().BoolVar(&forceRegister, "force", false,
		"force registration even if project already exists")
	registerCmd.Flags().StringVar(&projectDesc, "description", "",
		"project description")
	registerCmd.Flags().StringVar(&removeProject, "remove", "",
		"remove project by ID")
}

func handleRegister(cmd *cobra.Command, args []string) error {
	client := sdkClient()

	if listProjects {
		return listRegisteredProjects(client)
	}

	if cleanupProjects {
		return cleanupStaleProjects(client)
	}

	if removeProject != "" {
		return removeRegisteredProject(client, removeProject)
	}

	return registerProject(client, args)
}

func registerProject(client *sdk.Client, args []string) error {
	var projectPath, projectName string

	// Determine project path
	if len(args) >= 1 {
		projectPath = args[0]
	} else {
		projectPath = "."
	}

	// Always resolve to absolute path on the client side
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Determine project name
	if len(args) >= 2 {
		projectName = args[1]
	} else {
		projectName = filepath.Base(absPath)
	}

	// Register the project with absolute path
	req := sdk.RegisterProjectRequest{
		Path:        absPath, // Send absolute path to daemon
		Name:        projectName,
		Description: projectDesc,
		Force:       forceRegister,
	}

	project, err := client.RegisterProject(req)
	if err != nil {
		return err // The SDK already provides a good error message
	}

	fmt.Printf("✅ Project registered successfully!\n\n")
	fmt.Printf("  ID: %s\n", project.ID)
	fmt.Printf("  Name: %s\n", project.Name)
	fmt.Printf("  Path: %s\n", project.Path)
	fmt.Printf("  Mode: %s\n", project.Mode)
	fmt.Printf("  Registered: %s\n", project.Registered.Format("2006-01-02 15:04"))
	if project.Description != "" {
		fmt.Printf("  Description: %s\n", project.Description)
	}

	return nil
}

func listRegisteredProjects(client *sdk.Client) error {
	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects registered")
		fmt.Println("\nTo register a project, run:")
		fmt.Println("  takl register [path] [name]")
		return nil
	}

	// Sort by last access (most recent first)
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastAccess.After(projects[j].LastAccess)
	})

	// Display in table format using tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tPath\tMode\tLast Access\tStatus")
	fmt.Fprintln(w, strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 30)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 10))

	for _, project := range projects {
		status := "🟢 Active"
		if !project.Active {
			if time.Since(project.LastAccess) > 7*24*time.Hour {
				status = "🔴 Stale"
			} else {
				status = "🟡 Inactive"
			}
		}

		lastAccess := project.LastAccess.Format("2006-01-02")
		// Only show "today" format when not in testing to avoid flaky goldens
		if os.Getenv("TAKL_TESTING") == "" && clock.Since(project.LastAccess) < 24*time.Hour {
			lastAccess = project.LastAccess.Format("15:04 today")
		}

		path := project.Path
		if len(path) > 30 {
			path = "..." + path[len(path)-27:]
		}

		name := project.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			project.ID,
			name,
			path,
			project.Mode,
			lastAccess,
			status,
		)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d projects\n", len(projects))

	// Show cleanup suggestion if there are stale projects
	staleCount := 0
	for _, project := range projects {
		if !project.Active && time.Since(project.LastAccess) > 7*24*time.Hour {
			staleCount++
		}
	}

	if staleCount > 0 {
		fmt.Printf("\n💡 %d stale projects found. Run 'takl register --cleanup' to remove them.\n", staleCount)
	}

	return nil
}

func removeRegisteredProject(client *sdk.Client, projectID string) error {
	project, err := client.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	fmt.Printf("Removing project:\n")
	fmt.Printf("  ID: %s\n", project.ID)
	fmt.Printf("  Name: %s\n", project.Name)
	fmt.Printf("  Path: %s\n", project.Path)

	if err := client.UnregisterProject(projectID); err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	fmt.Println("✅ Project removed successfully")
	return nil
}

func cleanupStaleProjects(client *sdk.Client) error {
	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// Count stale projects first
	staleProjects := make([]*sdk.Project, 0)
	for _, project := range projects {
		if !project.Active && time.Since(project.LastAccess) > 7*24*time.Hour {
			staleProjects = append(staleProjects, project)
		}
	}

	if len(staleProjects) == 0 {
		fmt.Println("No stale projects found (older than 7 days)")
		return nil
	}

	fmt.Printf("Found %d stale projects to cleanup:\n\n", len(staleProjects))
	for _, project := range staleProjects {
		lastAccess := project.LastAccess.Format("2006-01-02")
		fmt.Printf("  %s - %s (last accessed: %s)\n", project.ID, project.Name, lastAccess)
	}

	fmt.Printf("\nCleaning up stale projects...\n")
	cleanupCount, err := client.CleanupProjects()
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Printf("✅ Successfully cleaned up %d stale projects\n", cleanupCount)
	return nil
}
