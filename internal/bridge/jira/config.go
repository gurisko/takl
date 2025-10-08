package jira

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadConfig loads and validates Jira configuration from .takl/jira.json
// in the specified project directory. Returns the config and project path.
func LoadConfig(projectPath string) (*JiraConfig, error) {
	configPath := filepath.Join(projectPath, ".takl", "jira.json")

	// Check file permissions (should be 0600 to protect API token)
	fi, statErr := os.Stat(configPath)
	if statErr == nil && (fi.Mode().Perm()&0o077) != 0 {
		return nil, fmt.Errorf("insecure permissions on %s; please run: chmod 600 %s", configPath, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read jira config at %s: %w\nPlease create .takl/jira.json with your Jira credentials", configPath, err)
	}

	var config JiraConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse jira config: %w", err)
	}

	// Validate required fields
	if config.BaseURL == "" || config.Email == "" || config.APIToken == "" || config.Project == "" {
		return nil, fmt.Errorf("jira config is incomplete: base_url, email, api_token, and project are required")
	}

	return &config, nil
}

// LoadConfigFromCwd loads Jira configuration from the current working directory.
// Returns the config and the current working directory path.
func LoadConfigFromCwd() (*JiraConfig, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current directory: %w", err)
	}

	config, err := LoadConfig(cwd)
	if err != nil {
		return nil, "", err
	}

	return config, cwd, nil
}
