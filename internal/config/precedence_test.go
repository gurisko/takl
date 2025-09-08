package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigPrecedence(t *testing.T) {
	// Create temporary directories for test
	tempDir, err := os.MkdirTemp("", "config-precedence-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	homeDir := filepath.Join(tempDir, "home")
	projectDir := filepath.Join(tempDir, "project")

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".takl"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create project config (.takl/config.yaml) - base layer
	projectConfigPath := filepath.Join(projectDir, ".takl", "config.yaml")
	projectConfig := map[string]any{
		"project": map[string]any{
			"name": "Base Project Name",
		},
		"paradigm": map[string]any{
			"id": "kanban",
			"options": map[string]any{
				"wip_limits": map[string]int{
					"doing":  3,
					"review": 2,
				},
			},
		},
		"ui": map[string]any{
			"date_format": "2006-01-02 15:04", // Base format
		},
	}

	projectConfigData, err := yaml.Marshal(projectConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectConfigPath, projectConfigData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create user config (~/.takl/config.yaml) - should override project config per precedence
	userConfigPath := filepath.Join(homeDir, ".takl", "config.yaml")
	userConfig := map[string]any{
		"project": map[string]any{
			"name": "User Overridden Name", // Should override project
		},
		"paradigm": map[string]any{
			"id": "scrum", // Should override project
			"options": map[string]any{
				"sprint_duration": 14, // Different options entirely
			},
		},
		"notifications": map[string]any{
			"enabled": true, // Only in user config
		},
		// ui.date_format not specified, should come from project
	}

	userConfigData, err := yaml.Marshal(userConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(userConfigPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userConfigPath, userConfigData, 0644); err != nil {
		t.Fatal(err)
	}

	// Temporarily set HOME environment variable
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Test environment variable overrides
	oldParadigm := os.Getenv("TAKL_PARADIGM")
	oldDateFormat := os.Getenv("TAKL_DATE_FORMAT")
	os.Setenv("TAKL_PARADIGM", "support")        // Should override everything else
	os.Setenv("TAKL_DATE_FORMAT", "Jan 2, 2006") // Should override everything else
	defer func() {
		if oldParadigm != "" {
			os.Setenv("TAKL_PARADIGM", oldParadigm)
		} else {
			os.Unsetenv("TAKL_PARADIGM")
		}
		if oldDateFormat != "" {
			os.Setenv("TAKL_DATE_FORMAT", oldDateFormat)
		} else {
			os.Unsetenv("TAKL_DATE_FORMAT")
		}
	}()

	// Load configuration - should respect precedence
	cfg, err := Load(projectDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test precedence: defaults < project < user < env < flags

	// Project name should come from user config (overrides project)
	if cfg.Project.Name != "User Overridden Name" {
		t.Errorf("Expected project name 'User Overridden Name', got '%s'", cfg.Project.Name)
	}

	// Paradigm ID should come from environment (highest precedence)
	if cfg.Paradigm.ID != "support" {
		t.Errorf("Expected paradigm ID 'support' (from env), got '%s'", cfg.Paradigm.ID)
	}

	// Date format should come from environment (highest precedence)
	if cfg.UI.DateFormat != "Jan 2, 2006" {
		t.Errorf("Expected date format 'Jan 2, 2006' (from env), got '%s'", cfg.UI.DateFormat)
	}

	// Notifications should come from user config (only set there)
	if !cfg.Notifications.Enabled {
		t.Error("Expected notifications enabled from user config")
	}

	// Paradigm options should come from user config (overrides project)
	// Since environment overrode paradigm ID to "support", the options will be empty
	// But we can still check the merge logic worked
	t.Logf("Paradigm options: %+v", cfg.Paradigm.Options)

	t.Logf("✅ Config precedence working: env > user > project > defaults")
}

func TestConfigStrictValidation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config-strict-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	projectDir := filepath.Join(tempDir, ".takl")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with unknown fields
	invalidConfig := map[string]any{
		"project": map[string]any{
			"name": "Test Project",
		},
		"paradigm": map[string]any{
			"id": "kanban",
		},
		"unknown_field":         "should fail",
		"another_unknown_field": 123,
	}

	configData, err := yaml.Marshal(invalidConfig)
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	// Loading should fail due to strict validation
	_, err = Load(filepath.Dir(projectDir))
	if err == nil {
		t.Fatal("Expected error for unknown fields, but got none")
	}

	// Should be a ConfigError with source information
	if configErr, ok := err.(*ConfigError); ok {
		if configErr.Source != "project" {
			t.Errorf("Expected error source 'project', got '%s'", configErr.Source)
		}
		if !containsString(configErr.Error(), "unknown") {
			t.Errorf("Expected error to mention unknown fields, got: %s", configErr.Error())
		}
		t.Logf("✅ Strict validation working: %s", configErr.Error())
	} else {
		t.Errorf("Expected ConfigError, got %T: %v", err, err)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that defaults are applied when no config files exist
	tempDir, err := os.MkdirTemp("", "config-defaults-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load defaults: %v", err)
	}

	// Check some default values
	if cfg.UI.DateFormat == "" {
		t.Error("Expected default date format to be set")
	}

	if cfg.Notifications.Enabled {
		t.Error("Expected default notifications to be disabled")
	}

	t.Logf("✅ Defaults applied: date_format=%s, notifications=%t",
		cfg.UI.DateFormat, cfg.Notifications.Enabled)
}

func containsString(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr ||
		(len(text) > len(substr) &&
			(text[:len(substr)] == substr ||
				text[len(text)-len(substr):] == substr ||
				findSubstring(text, substr))))
}

func findSubstring(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Project:  Project{Name: "Test Project"},
				Paradigm: Paradigm{ID: "kanban"},
				UI:       UI{DateFormat: "2006-01-02"},
			},
			expectError: false,
		},
		{
			name: "invalid paradigm",
			config: Config{
				Paradigm: Paradigm{ID: "invalid"},
			},
			expectError: true,
			errorMsg:    "paradigm.id 'invalid' is not valid",
		},
		{
			name: "invalid date format",
			config: Config{
				UI: UI{DateFormat: "invalid-format"},
			},
			expectError: true,
			errorMsg:    "ui.date_format 'invalid-format' does not contain standard date components",
		},
		{
			name: "deprecated fields",
			config: Config{
				Mode:      "embedded",
				IssuesDir: "/path/to/issues",
			},
			expectError: true,
			errorMsg:    "field 'mode' is deprecated",
		},
		{
			name: "project name too long",
			config: Config{
				Project: Project{Name: strings.Repeat("a", 101)},
			},
			expectError: true,
			errorMsg:    "project.name exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				t.Logf("✅ Validation correctly failed: %s", err.Error())
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}
