package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Source represents a configuration source with its path and data
type Source struct {
	Path string
	Data []byte
}

// readIfExists reads a file if it exists, returning nil if it doesn't exist
func readIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// decodeStrict decodes YAML data with strict field checking
func decodeStrict(data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // Reject unknown fields
	return dec.Decode(out)
}

// Merge combines two configs with overlay taking precedence
func Merge(base, overlay Config) Config {
	out := base

	// Project fields
	if overlay.Project.Name != "" {
		out.Project.Name = overlay.Project.Name
	}
	if overlay.Project.Path != "" {
		out.Project.Path = overlay.Project.Path
	}

	// Paradigm fields
	if overlay.Paradigm.ID != "" {
		out.Paradigm.ID = overlay.Paradigm.ID
	}
	if overlay.Paradigm.Options != nil {
		out.Paradigm.Options = overlay.Paradigm.Options
	}

	// Notifications - merge selectively
	out.Notifications = mergeNotifications(out.Notifications, overlay.Notifications)

	// UI fields
	if overlay.UI.DateFormat != "" {
		out.UI.DateFormat = overlay.UI.DateFormat
	}

	// Legacy fields
	if overlay.Mode != "" {
		out.Mode = overlay.Mode
	}
	if overlay.IssuesDir != "" {
		out.IssuesDir = overlay.IssuesDir
	}
	if overlay.WebPort != 0 {
		out.WebPort = overlay.WebPort
	}
	out.Git = mergeGitConfig(out.Git, overlay.Git)

	return out
}

// mergeNotifications merges notification configurations
func mergeNotifications(base, overlay Notifications) Notifications {
	result := base

	// If overlay explicitly sets enabled, use it
	if overlay.Enabled != base.Enabled {
		result.Enabled = overlay.Enabled
	}

	// GitHub Actions merge
	if overlay.GitHubActions.Workflow != "" {
		result.GitHubActions.Workflow = overlay.GitHubActions.Workflow
	}
	if overlay.GitHubActions.OnCreate != base.GitHubActions.OnCreate {
		result.GitHubActions.OnCreate = overlay.GitHubActions.OnCreate
	}
	if overlay.GitHubActions.OnTransition != nil {
		result.GitHubActions.OnTransition = overlay.GitHubActions.OnTransition
	}

	return result
}

// mergeGitConfig merges git configurations
func mergeGitConfig(base, overlay GitConfig) GitConfig {
	result := base

	if overlay.AutoCommit != base.AutoCommit {
		result.AutoCommit = overlay.AutoCommit
	}
	if overlay.CommitMessage != "" {
		result.CommitMessage = overlay.CommitMessage
	}
	if overlay.AuthorName != "" {
		result.AuthorName = overlay.AuthorName
	}
	if overlay.AuthorEmail != "" {
		result.AuthorEmail = overlay.AuthorEmail
	}

	return result
}

// Load loads configuration from all sources with proper precedence
// Precedence: defaults < paradigm defaults < project < user < env < flags
func Load(projectRoot string) (Config, error) {
	var cfg Config

	// 1) Built-in defaults
	cfg = Defaults()

	// 2) Paradigm defaults (will be filled later when paradigm resolves)

	// 3) Project file - check both embedded (.takl/config.yaml) and standalone (config.yaml) modes
	if projectRoot != "" {
		var projectPath string
		var data []byte
		var err error

		// Try embedded mode first (.takl/config.yaml)
		embeddedPath := filepath.Join(projectRoot, ".takl", "config.yaml")
		if data, err = readIfExists(embeddedPath); err != nil {
			return Config{}, err
		}

		// If not found, try standalone mode (config.yaml)
		if len(data) == 0 {
			standalonePath := filepath.Join(projectRoot, "config.yaml")
			if data, err = readIfExists(standalonePath); err != nil {
				return Config{}, err
			}
			projectPath = standalonePath
		} else {
			projectPath = embeddedPath
		}

		if len(data) > 0 {
			var projectConfig Config
			if err := decodeStrict(data, &projectConfig); err != nil {
				return Config{}, &ConfigError{
					Source: "project",
					Path:   projectPath,
					Err:    err,
				}
			}
			cfg = Merge(cfg, projectConfig)
		}
	}

	// 4) User file (~/.takl/config.yaml)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		userPath := filepath.Join(home, ".takl", "config.yaml")
		if data, err := readIfExists(userPath); err != nil {
			return Config{}, err
		} else if len(data) > 0 {
			var userConfig Config
			if err := decodeStrict(data, &userConfig); err != nil {
				return Config{}, &ConfigError{
					Source: "user",
					Path:   userPath,
					Err:    err,
				}
			}
			cfg = Merge(cfg, userConfig)
		}
	}

	// 5) Environment variables (TAKL_*)
	cfg = applyEnvOverrides(cfg)

	// 6) CLI flags will be applied later in command handlers

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(cfg Config) Config {
	if v := os.Getenv("TAKL_PARADIGM"); v != "" {
		cfg.Paradigm.ID = v
	}
	if v := os.Getenv("TAKL_DATE_FORMAT"); v != "" {
		cfg.UI.DateFormat = v
	}
	if v := os.Getenv("TAKL_PROJECT_NAME"); v != "" {
		cfg.Project.Name = v
	}
	// TAKL_WEB_PORT parsing will be added when needed

	return cfg
}

// ConfigError represents a configuration loading error with context
type ConfigError struct {
	Source string // "project", "user", etc.
	Path   string
	Err    error
}

func (e *ConfigError) Error() string {
	// Enhanced error message with suggestions
	msg := fmt.Sprintf("config error in %s file (%s): %s", e.Source, e.Path, e.Err.Error())

	// Add helpful suggestions for common errors
	errStr := e.Err.Error()
	if strings.Contains(errStr, "field") && strings.Contains(errStr, "not found") {
		msg += "\n\nValid configuration fields are documented at: https://takl.dev/config"
	}
	if strings.Contains(errStr, "cannot unmarshal") {
		msg += "\n\nCheck the YAML syntax and field types in your configuration file."
	}

	return msg
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ValidationError represents validation errors with detailed context
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 1 {
		return "config validation failed: " + e.Errors[0]
	}
	return fmt.Sprintf("config validation failed with %d errors:\n  - %s",
		len(e.Errors), strings.Join(e.Errors, "\n  - "))
}

// ValidateConfig validates a configuration for correctness
func ValidateConfig(cfg Config) error {
	var errors []string

	// Validate project configuration
	if cfg.Project.Name != "" && len(cfg.Project.Name) > 100 {
		errors = append(errors, "project.name exceeds maximum length of 100 characters")
	}

	// Validate paradigm configuration
	if cfg.Paradigm.ID != "" {
		validParadigms := []string{"scrum", "kanban", "support"}
		valid := false
		for _, p := range validParadigms {
			if cfg.Paradigm.ID == p {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("paradigm.id '%s' is not valid (supported: %v)", cfg.Paradigm.ID, validParadigms))
		}
	}

	// Validate date format by attempting to parse with it
	if cfg.UI.DateFormat != "" {
		// Test with a known time to ensure the format produces expected results
		testTime := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
		formatted := testTime.Format(cfg.UI.DateFormat)
		parsed, err := time.Parse(cfg.UI.DateFormat, formatted)
		if err != nil {
			errors = append(errors, fmt.Sprintf("ui.date_format '%s' is invalid: %v", cfg.UI.DateFormat, err))
		} else {
			// Additional validation: ensure the format actually represents date/time meaningfully
			if !containsDateComponent(cfg.UI.DateFormat) {
				errors = append(errors, fmt.Sprintf("ui.date_format '%s' does not contain standard date components (year, month, day)", cfg.UI.DateFormat))
			} else if parsed.Year() != testTime.Year() || parsed.Month() != testTime.Month() || parsed.Day() != testTime.Day() {
				errors = append(errors, fmt.Sprintf("ui.date_format '%s' produces unexpected parsing results", cfg.UI.DateFormat))
			}
		}
	} else {
		// Apply default if empty
		cfg.UI.DateFormat = "2006-01-02 15:04"
	}

	// Validate legacy fields with warnings
	if cfg.Mode != "" {
		errors = append(errors, "field 'mode' is deprecated and will be ignored")
	}
	if cfg.IssuesDir != "" {
		errors = append(errors, "field 'issues_dir' is deprecated and will be ignored")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Errors: errors,
		}
	}

	return nil
}

// containsDateComponent checks if the format string contains meaningful date components
func containsDateComponent(format string) bool {
	// Check for presence of year, month, and day components
	hasYear := strings.Contains(format, "2006") || strings.Contains(format, "06")
	hasMonth := strings.Contains(format, "01") || strings.Contains(format, "1") || strings.Contains(format, "Jan") || strings.Contains(format, "January")
	hasDay := strings.Contains(format, "02") || strings.Contains(format, "2") || strings.Contains(format, "_2")
	return hasYear && hasMonth && hasDay
}
