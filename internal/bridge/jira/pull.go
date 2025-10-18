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
	Deleted int      `json:"deleted"`
	Errors  []string `json:"errors"`
}

// RefreshMemberCache fetches project members from Jira and updates the local cache.
// Returns the cache (loaded from disk if fetch fails) and any error.
// This function is non-fatal - it will return a cache even if fetching fails.
func RefreshMemberCache(ctx context.Context, client *Client, projectPath, projectKey string) (*MemberCache, error) {
	log.Printf("[DEBUG] RefreshMemberCache: Fetching project members")

	// Fetch members from Jira
	members, err := client.FetchProjectMembers(ctx, projectKey)
	if err != nil {
		log.Printf("[WARN] RefreshMemberCache: Failed to fetch project members: %v", err)
		// Try to load existing cache as fallback
		cache, loadErr := LoadMembersCache(projectPath)
		if loadErr != nil {
			log.Printf("[WARN] RefreshMemberCache: Failed to load existing cache: %v", loadErr)
			return NewMemberCache(), fmt.Errorf("failed to fetch members: %w", err)
		}
		return cache, fmt.Errorf("failed to fetch members (using cached data): %w", err)
	}

	// Load existing cache (or create new one)
	cache, err := LoadMembersCache(projectPath)
	if err != nil {
		log.Printf("[WARN] RefreshMemberCache: Failed to load existing cache: %v", err)
		cache = NewMemberCache()
	}

	// Update cache with fetched members
	for _, member := range members {
		cache.Add(member)
	}

	// Save cache
	if err := SaveMembersCache(projectPath, cache); err != nil {
		log.Printf("[WARN] RefreshMemberCache: Failed to save cache: %v", err)
		return cache, fmt.Errorf("failed to save cache: %w", err)
	}

	log.Printf("[DEBUG] RefreshMemberCache: Successfully cached %d members", len(members))
	return cache, nil
}

// RefreshWorkflowCache fetches project statuses from Jira and updates the local cache.
// Returns the cache (loaded from disk if fetch fails) and any error.
// This function is non-fatal - it will return a cache even if fetching fails.
func RefreshWorkflowCache(ctx context.Context, client *Client, projectPath, projectKey string) (*WorkflowCache, error) {
	log.Printf("[DEBUG] RefreshWorkflowCache: Fetching project statuses")

	// Fetch statuses from Jira
	statuses, err := client.FetchProjectStatuses(ctx, projectKey)
	if err != nil {
		log.Printf("[WARN] RefreshWorkflowCache: Failed to fetch project statuses: %v", err)
		// Try to load existing cache as fallback
		cache, loadErr := LoadWorkflowCache(projectPath)
		if loadErr != nil {
			log.Printf("[WARN] RefreshWorkflowCache: Failed to load existing cache: %v", loadErr)
			return NewWorkflowCache(), fmt.Errorf("failed to fetch statuses: %w", err)
		}
		return cache, fmt.Errorf("failed to fetch statuses (using cached data): %w", err)
	}

	// Create new cache from fetched data (don't merge with old cache)
	cache := NewWorkflowCache()
	for _, status := range statuses {
		cache.AddStatus(status)
	}

	// Save cache
	if err := SaveWorkflowCache(projectPath, cache); err != nil {
		log.Printf("[WARN] RefreshWorkflowCache: Failed to save cache: %v", err)
		return cache, fmt.Errorf("failed to save cache: %w", err)
	}

	log.Printf("[DEBUG] RefreshWorkflowCache: Successfully cached %d statuses", len(statuses))
	return cache, nil
}

// Pull fetches issues from Jira and saves them to local storage
func Pull(ctx context.Context, client *Client, storage *Storage, config *JiraConfig) (*PullResult, error) {
	result := &PullResult{
		Errors: make([]string, 0),
	}

	// Fetch and cache project members first (non-fatal)
	memberCache, _ := RefreshMemberCache(ctx, client, storage.projectPath, config.Project)

	// Fetch and cache project workflow/statuses (non-fatal)
	_, _ = RefreshWorkflowCache(ctx, client, storage.projectPath, config.Project)

	// Search for all issues in the project (archived filtering handled client-side)
	jql := fmt.Sprintf("project=%s ORDER BY updated DESC", config.Project)
	log.Printf("[DEBUG] Pull: Searching Jira with JQL: %s", jql)
	issues, err := client.SearchIssues(ctx, jql, MaxSearchResults, memberCache)
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

	// Build set of fetched issue keys for efficient lookup
	fetchedKeys := make(map[string]bool)
	for _, issue := range issues {
		fetchedKeys[issue.JiraKey] = true
	}

	// Delete local issues that are no longer in Jira (archived or deleted)
	for _, localKey := range localIssues {
		if !fetchedKeys[localKey] {
			log.Printf("[DEBUG] Pull: Deleting locally archived/removed issue %s", localKey)
			if err := storage.DeleteIssue(localKey); err != nil {
				log.Printf("[ERROR] Pull: Failed to delete %s: %v", localKey, err)
				result.Errors = append(result.Errors, fmt.Sprintf("failed to delete %s: %v", localKey, err))
			} else {
				result.Deleted++
			}
		}
	}

	if result.Deleted > 0 {
		log.Printf("[DEBUG] Pull: Deleted %d locally archived/removed issues", result.Deleted)
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

	log.Printf("[DEBUG] Pull: Complete - Created: %d, Updated: %d, Deleted: %d, Errors: %d", result.Created, result.Updated, result.Deleted, len(result.Errors))

	// Return error if all issues failed to save
	if len(result.Errors) > 0 && result.Created == 0 && result.Updated == 0 {
		return result, fmt.Errorf("failed to save any issues (%d errors)", len(result.Errors))
	}

	return result, nil
}
