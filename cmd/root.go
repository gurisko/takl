package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/takl/takl/internal/context"
)

var (
	cfgFile    string
	verbose    bool
	repoPath   string
	projectCtx *context.ProjectContext
)

var rootCmd = &cobra.Command{
	Use:   "takl",
	Short: "TAKL - Git-native issue tracker",
	Long: `TAKL is a git-native issue tracker that stores issues as markdown files.

Works in two modes:
- Standalone: Dedicated repository for issues  
- Embedded: .takl/ directory in existing projects

Context Detection:
TAKL automatically detects which registered project you're working in based on your current directory.`,
	Version:       "0.1.0",
	SilenceUsage:  true, // Don't show usage on validation errors
	SilenceErrors: true, // Let Execute() handle error printing
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check if command requires project context
		if !requiresProjectContext(cmd) {
			return nil
		}

		ctx, err := context.DetectContext(repoPath)
		if err != nil {
			return fmt.Errorf("failed to detect project context: %w", err)
		}

		projectCtx = ctx

		// Update last access if we're in a project
		if ctx.IsInProject {
			if err := ctx.UpdateLastAccess(); err != nil && verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to update last access: %v\n", err)
			}
		}

		return nil
	},
}

func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default: .takl/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"verbose output")
	rootCmd.PersistentFlags().StringVar(&repoPath, "repo", ".",
		"repository path")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		// honor --repo first if provided
		if repoPath != "" && repoPath != "." {
			viper.AddConfigPath(filepath.Join(repoPath, ".takl"))
			viper.AddConfigPath(repoPath)
		}
		viper.AddConfigPath(".takl")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("TAKL")

	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// GetProjectContext returns the current project context
func GetProjectContext() *context.ProjectContext {
	return projectCtx
}

// RequireProjectContext returns an error if not in a project context
func RequireProjectContext() (*context.ProjectContext, error) {
	if projectCtx == nil {
		return nil, fmt.Errorf("project context not available")
	}
	if err := projectCtx.RequiresProject(); err != nil {
		return nil, err
	}
	return projectCtx, nil
}

// RequiresProjectContext interface for commands that need project context
type RequiresProjectContextCmd interface {
	RequiresProject() bool
}

// requiresProjectContext determines if a command needs project context
func requiresProjectContext(cmd *cobra.Command) bool {
	// Check if command has annotation indicating it doesn't need context
	if cmd.Annotations != nil {
		if skipContext, exists := cmd.Annotations["skipProjectContext"]; exists && skipContext == "true" {
			return false
		}
	}

	// Default logic based on command names
	cmdName := cmd.Name()
	parentName := ""
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	// Commands that don't need project context
	switch cmdName {
	case "help", "version", "completion":
		return false
	case "register", "daemon":
		return false
	case "init":
		return false // init creates project context
	}

	// Subcommands of register and daemon don't need context
	switch parentName {
	case "register", "daemon":
		return false
	}

	// All other commands need project context
	return true
}
