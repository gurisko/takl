package jira

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const workflowCacheFilename = "jira-workflow.json"

// LoadWorkflowCache loads the workflow cache from .takl/jira-workflow.json
func LoadWorkflowCache(projectPath string) (*WorkflowCache, error) {
	cachePath := filepath.Join(projectPath, ".takl", workflowCacheFilename)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache doesn't exist yet, return empty cache
			return NewWorkflowCache(), nil
		}
		return nil, fmt.Errorf("failed to read workflow cache: %w", err)
	}

	var cache WorkflowCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse workflow cache: %w", err)
	}

	// Initialize the map if it's nil
	if cache.Statuses == nil {
		cache.Statuses = make(map[string]*StatusInfo)
	}

	return &cache, nil
}

// SaveWorkflowCache saves the workflow cache to .takl/jira-workflow.json
// Uses atomic write (temp file + rename) to prevent corruption
func SaveWorkflowCache(projectPath string, cache *WorkflowCache) error {
	taklDir := filepath.Join(projectPath, ".takl")

	// Ensure .takl directory exists with restrictive permissions
	if err := os.MkdirAll(taklDir, 0700); err != nil {
		return fmt.Errorf("failed to create .takl directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow cache: %w", err)
	}

	// Write atomically via temp file in the same directory
	tmpFile, err := os.CreateTemp(taklDir, ".jira-workflow-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on any error
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Set permissions
	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	// Write content
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomically rename temp file to final location
	finalPath := filepath.Join(taklDir, workflowCacheFilename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Success - prevent cleanup from removing the file
	tmpFile = nil
	return nil
}
