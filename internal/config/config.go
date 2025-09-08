package config

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

func DetectMode(repoPath string) (Mode, error) {
	// Check for .takl directory (embedded mode)
	taklDir := filepath.Join(repoPath, ".takl")
	if _, err := os.Stat(taklDir); err == nil {
		return ModeEmbedded, nil
	}

	// Check for .issues directory (standalone mode)
	issuesDir := filepath.Join(repoPath, ".issues")
	if _, err := os.Stat(issuesDir); err == nil {
		return ModeStandalone, nil
	}

	// Default to embedded mode for existing projects with git
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return ModeEmbedded, nil
	}

	// Default to standalone for new projects
	return ModeStandalone, nil
}

func LoadConfig(configPath string) (*Config, error) {
	// Set defaults
	config := &Config{
		WebPort: 3000,
		Git: GitConfig{
			AutoCommit:    true,
			CommitMessage: "Update issue: %s",
		},
	}

	// Try to load config file
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if len(data) > 0 {
			// Use strict decoding to catch typos and unknown fields
			dec := yaml.NewDecoder(bytes.NewReader(data))
			dec.KnownFields(true)
			if err := dec.Decode(config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	return config, nil
}

func (c *Config) GetIssuesDir(repoPath string) string {
	if c.IssuesDir != "" {
		return c.IssuesDir
	}

	switch c.Mode {
	case ModeEmbedded:
		return filepath.Join(repoPath, ".takl", "issues")
	case ModeStandalone:
		return filepath.Join(repoPath, ".issues")
	default:
		return filepath.Join(repoPath, ".issues")
	}
}
