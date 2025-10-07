//go:build unix

package cmd

import "github.com/spf13/cobra"

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects in the TAKL registry",
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
