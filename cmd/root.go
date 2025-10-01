package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "takl",
	Short: "TAKL - Git-native issue tracker",
	Long:  `TAKL (pronounced "tackle") is a git-native issue tracker with daemon-first architecture.`,
}

func Execute() error {
	// Silence usage and errors to avoid cluttering output with Cobra defaults
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	return rootCmd.Execute()
}
