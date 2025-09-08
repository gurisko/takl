package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage TAKL configuration",
	Long: `Manage TAKL configuration files and settings.

Configuration is loaded from multiple sources with precedence:
1. Built-in defaults (lowest)
2. Paradigm defaults
3. Project file (.takl/config.yaml)
4. User file (~/.takl/config.yaml)
5. Environment variables (TAKL_*)
6. CLI flags (highest)`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective configuration",
	Long: `Display the current effective configuration after merging all sources.

This shows the final configuration that TAKL will use, including all
defaults and overrides from project files, user files, and environment.`,
	RunE: showConfig,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration files",
	Long: `Validate all configuration files for syntax and schema correctness.

This checks both project and user configuration files, reporting any
parsing errors or unknown fields.`,
	RunE: validateConfig,
}

var (
	outputJSON bool
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)

	configShowCmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
}

func showConfig(cmd *cobra.Command, args []string) error {
	// Get current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load effective configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Output in requested format
	if outputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
	}

	return nil
}

func validateConfig(cmd *cobra.Command, args []string) error {
	// Get current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Try to load configuration - this will validate all files
	cfg, err := config.Load(projectRoot)
	if err != nil {
		// Check if it's a config error with detailed context
		if configErr, ok := err.(*config.ConfigError); ok {
			fmt.Printf("❌ Configuration error in %s file:\n", configErr.Source)
			fmt.Printf("   File: %s\n", configErr.Path)
			fmt.Printf("   Error: %s\n", configErr.Err.Error())
			return fmt.Errorf("configuration validation failed")
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate the merged configuration
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ Configuration validation failed:\n")
		fmt.Printf("   Error: %s\n", err.Error())
		return fmt.Errorf("configuration validation failed")
	}

	fmt.Println("✅ Configuration is valid")

	// Show which files were loaded
	fmt.Println("\nConfiguration sources:")

	// Check which files exist and were loaded
	if _, err := os.Stat(fmt.Sprintf("%s/.takl/config.yaml", projectRoot)); err == nil {
		fmt.Printf("  📁 Project: %s/.takl/config.yaml\n", projectRoot)
	}

	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(fmt.Sprintf("%s/.takl/config.yaml", home)); err == nil {
			fmt.Printf("  👤 User: %s/.takl/config.yaml\n", home)
		}
	}

	// Show active environment variables
	envVars := []string{"TAKL_PARADIGM", "TAKL_DATE_FORMAT", "TAKL_PROJECT_NAME"}
	hasEnv := false
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			if !hasEnv {
				fmt.Println("  🌍 Environment:")
				hasEnv = true
			}
			fmt.Printf("    %s=%s\n", envVar, value)
		}
	}

	fmt.Printf("\nEffective paradigm: %s\n", cfg.Paradigm.ID)
	if cfg.Project.Name != "" {
		fmt.Printf("Project name: %s\n", cfg.Project.Name)
	}

	return nil
}
