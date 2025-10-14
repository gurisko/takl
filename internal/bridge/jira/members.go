package jira

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const membersCacheFilename = "jira-members.json"

// LoadMembersCache loads the members cache from .takl/jira-members.json
func LoadMembersCache(projectPath string) (*MemberCache, error) {
	cachePath := filepath.Join(projectPath, ".takl", membersCacheFilename)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache doesn't exist yet, return empty cache
			return NewMemberCache(), nil
		}
		return nil, fmt.Errorf("failed to read members cache: %w", err)
	}

	var cache MemberCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse members cache: %w", err)
	}

	// Initialize the map if it's nil
	if cache.Members == nil {
		cache.Members = make(map[string]*Member)
	}

	return &cache, nil
}

// SaveMembersCache saves the members cache to .takl/jira-members.json
// Uses atomic write (temp file + rename) to prevent corruption
func SaveMembersCache(projectPath string, cache *MemberCache) error {
	taklDir := filepath.Join(projectPath, ".takl")

	// Ensure .takl directory exists with restrictive permissions (contains PII)
	if err := os.MkdirAll(taklDir, 0700); err != nil {
		return fmt.Errorf("failed to create .takl directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal members cache: %w", err)
	}

	// Write atomically via temp file in the same directory
	tmpFile, err := os.CreateTemp(taklDir, ".jira-members-*.json.tmp")
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

	// Set restrictive permissions (contains PII - emails)
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
	finalPath := filepath.Join(taklDir, membersCacheFilename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Success - prevent cleanup from removing the file
	tmpFile = nil
	return nil
}
