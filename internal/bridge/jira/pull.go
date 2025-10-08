package jira

import (
	"context"
	"fmt"
	"log"
)

// PullResult represents the result of pulling issues
type PullResult struct {
	Fetched int      `json:"fetched"`
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Errors  []string `json:"errors"`
}

// Pull fetches issues from Jira and saves them to local storage
func Pull(ctx context.Context, client *Client, storage *Storage, config *JiraConfig) (*PullResult, error) {
	result := &PullResult{
		Errors: make([]string, 0),
	}

	// Search for all issues in the project
	jql := fmt.Sprintf("project=%s ORDER BY updated DESC", config.Project)
	log.Printf("[DEBUG] Pull: Searching Jira with JQL: %s", jql)
	issues, err := client.SearchIssues(ctx, jql, MaxSearchResults)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	log.Printf("[DEBUG] Pull: Fetched %d issues from Jira", len(issues))
	result.Fetched = len(issues)

	// Get list of existing local issues
	localIssues, err := storage.ListIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list local issues: %w", err)
	}

	localMap := make(map[string]bool)
	for _, key := range localIssues {
		localMap[key] = true
	}

	// Process each issue
	for _, issue := range issues {
		isNew := !localMap[issue.JiraKey]

		log.Printf("[DEBUG] Pull: Processing issue %s (new=%v)", issue.JiraKey, isNew)

		// Check if the issue is unchanged
		if !isNew {
			// Compute hash to compare with existing
			newHash := storage.ComputeHash(&issue)
			if oldHash, ok := storage.ReadExistingHash(issue.JiraKey); ok && oldHash == newHash {
				log.Printf("[DEBUG] Pull: Skipping %s (unchanged)", issue.JiraKey)
				continue // Skip unchanged issues
			}
		}

		// Save the issue
		if err := storage.SaveIssue(&issue); err != nil {
			log.Printf("[ERROR] Pull: Failed to save %s: %v", issue.JiraKey, err)
			result.Errors = append(result.Errors, fmt.Sprintf("failed to save %s: %v", issue.JiraKey, err))
			continue
		}

		if isNew {
			result.Created++
		} else {
			result.Updated++
		}
	}

	log.Printf("[DEBUG] Pull: Complete - Created: %d, Updated: %d, Errors: %d", result.Created, result.Updated, len(result.Errors))

	// Return error if all issues failed to save
	if len(result.Errors) > 0 && result.Created == 0 && result.Updated == 0 {
		return result, fmt.Errorf("failed to save any issues (%d errors)", len(result.Errors))
	}

	return result, nil
}
