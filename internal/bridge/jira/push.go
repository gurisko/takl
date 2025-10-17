package jira

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// PushResult represents the result of pushing local changes to Jira
type PushResult struct {
	Scanned   int            `json:"scanned"`   // Total local issues scanned
	Pushed    int            `json:"pushed"`    // Successfully pushed to Jira
	Skipped   int            `json:"skipped"`   // No local changes
	Conflicts []ConflictInfo `json:"conflicts"` // Issues with conflicts
	Errors    []string       `json:"errors"`    // Other errors
}

// ConflictInfo describes a conflict for a specific issue
type ConflictInfo struct {
	IssueKey string    `json:"issue_key"`
	Updated  time.Time `json:"updated"` // When the remote was last updated
}

// Push pushes local changes to Jira with strict conflict detection
// Option 1 (Strict): Fails if remote has ANY changes since last pull
// If issueKey is provided, only that issue will be pushed
func Push(ctx context.Context, client *Client, storage *Storage, config *JiraConfig, issueKey string) (*PushResult, error) {
	result := &PushResult{
		Conflicts: make([]ConflictInfo, 0),
		Errors:    make([]string, 0),
	}

	// Load member cache for user resolution (non-fatal if missing)
	memberCache, err := LoadMembersCache(storage.projectPath)
	if err != nil {
		log.Printf("[WARN] Push: Failed to load member cache: %v", err)
		memberCache = NewMemberCache()
	}

	// Get local issues (all or specific one)
	var localIssues []*Issue
	if issueKey != "" {
		// Push only the specified issue
		issue, err := storage.ReadIssue(issueKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read issue %s: %w", issueKey, err)
		}
		localIssues = []*Issue{issue}
		log.Printf("[DEBUG] Push: Pushing single issue %s", issueKey)
	} else {
		// Push all issues
		localIssues, err = storage.ListAllIssues()
		if err != nil {
			return nil, fmt.Errorf("failed to list local issues: %w", err)
		}
		log.Printf("[DEBUG] Push: Scanned %d local issues", len(localIssues))
	}

	result.Scanned = len(localIssues)

	// Process each local issue
	for _, localIssue := range localIssues {
		log.Printf("[DEBUG] Push: Processing issue %s", localIssue.JiraKey)

		// Compute local hash
		localHash := storage.ComputeHash(localIssue)

		// Compare with base hash (from file)
		baseHash := localIssue.Hash

		// If local == base, no changes to push
		if localHash == baseHash {
			log.Printf("[DEBUG] Push: Skipping %s (no local changes)", localIssue.JiraKey)
			result.Skipped++
			continue
		}

		log.Printf("[DEBUG] Push: Issue %s has local changes (base=%s, local=%s)",
			localIssue.JiraKey, baseHash[:8], localHash[:8])

		// Fetch current remote version for conflict detection
		remoteIssue, err := client.GetIssue(ctx, localIssue.JiraKey, memberCache)
		if err != nil {
			log.Printf("[ERROR] Push: Failed to fetch remote %s: %v", localIssue.JiraKey, err)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to fetch remote: %v", localIssue.JiraKey, err))
			continue
		}

		// Compute remote hash
		remoteHash := storage.ComputeHash(remoteIssue)

		log.Printf("[DEBUG] Push: Issue %s remote hash=%s", localIssue.JiraKey, remoteHash[:8])

		// Check for conflict: remote != base (remote was modified)
		if remoteHash != baseHash {
			log.Printf("[WARN] Push: Conflict detected for %s (remote modified)", localIssue.JiraKey)
			result.Conflicts = append(result.Conflicts, ConflictInfo{
				IssueKey: localIssue.JiraKey,
				Updated:  remoteIssue.Updated,
			})
			continue
		}

		// No conflict: safe to push
		log.Printf("[DEBUG] Push: Pushing changes for %s", localIssue.JiraKey)
		if err := pushIssue(ctx, client, storage, localIssue, remoteIssue); err != nil {
			log.Printf("[ERROR] Push: Failed to push %s: %v", localIssue.JiraKey, err)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", localIssue.JiraKey, err))
			continue
		}

		result.Pushed++
		log.Printf("[DEBUG] Push: Successfully pushed %s", localIssue.JiraKey)
	}

	log.Printf("[DEBUG] Push: Complete - Scanned: %d, Pushed: %d, Skipped: %d, Conflicts: %d, Errors: %d",
		result.Scanned, result.Pushed, result.Skipped, len(result.Conflicts), len(result.Errors))

	// Return error if conflicts detected
	if len(result.Conflicts) > 0 {
		return result, fmt.Errorf("cannot push - %d issue(s) have conflicts", len(result.Conflicts))
	}

	return result, nil
}

// pushIssue pushes changes for a single issue to Jira
// Compares local vs remote and updates only what changed
func pushIssue(ctx context.Context, client *Client, storage *Storage, local *Issue, remote *Issue) error {
	updates := make(map[string]interface{})
	hasUpdates := false

	// Check title (summary)
	if local.Title != remote.Title {
		log.Printf("[DEBUG] pushIssue: Title changed for %s", local.JiraKey)
		updates["summary"] = local.Title
		hasUpdates = true
	}

	// Check description
	if local.Description != remote.Description {
		log.Printf("[DEBUG] pushIssue: Description changed for %s", local.JiraKey)
		updates["description"] = local.Description
		hasUpdates = true
	}

	// Check labels
	if !equalStringSlices(local.Labels, remote.Labels) {
		log.Printf("[DEBUG] pushIssue: Labels changed for %s", local.JiraKey)
		updates["labels"] = local.Labels
		hasUpdates = true
	}

	// Update issue fields if any changed
	if hasUpdates {
		if err := client.UpdateIssue(ctx, local.JiraKey, updates); err != nil {
			return fmt.Errorf("failed to update issue fields: %w", err)
		}
	}

	// Check status (requires workflow transition)
	if local.Status != remote.Status {
		log.Printf("[DEBUG] pushIssue: Status changed for %s (%s → %s)", local.JiraKey, remote.Status, local.Status)

		// Validate status against workflow cache
		workflowCache, err := LoadWorkflowCache(storage.projectPath)
		if err != nil {
			log.Printf("[WARN] pushIssue: Failed to load workflow cache: %v", err)
			workflowCache = NewWorkflowCache()
		}

		// Check if target status exists in workflow
		var targetStatusExists bool
		for _, status := range workflowCache.Statuses {
			if status.Name == local.Status {
				targetStatusExists = true
				break
			}
		}

		if !targetStatusExists {
			return fmt.Errorf("invalid status %q: not found in project workflow (run 'takl jira workflow' to see valid statuses)", local.Status)
		}

		// Get available transitions for this issue
		transitions, err := client.GetTransitions(ctx, local.JiraKey)
		if err != nil {
			return fmt.Errorf("failed to get transitions: %w", err)
		}

		// Find transition to target status
		var transitionID string
		for _, t := range transitions {
			if t.ToStatus == local.Status {
				transitionID = t.ID
				log.Printf("[DEBUG] pushIssue: Found transition %q (ID: %s) to status %q", t.Name, t.ID, local.Status)
				break
			}
		}

		if transitionID == "" {
			// List available transitions for error message
			availableStatuses := make([]string, 0, len(transitions))
			for _, t := range transitions {
				availableStatuses = append(availableStatuses, t.ToStatus)
			}
			return fmt.Errorf("cannot transition to %q: no transition available from current status %q (available: %v)",
				local.Status, remote.Status, availableStatuses)
		}

		// Execute transition
		if err := client.TransitionIssue(ctx, local.JiraKey, transitionID); err != nil {
			return fmt.Errorf("failed to transition issue: %w", err)
		}
	}

	// Check for new comments (local has more comments than remote)
	// We only support adding new comments, not editing existing ones
	if len(local.Comments) > len(remote.Comments) {
		log.Printf("[DEBUG] pushIssue: Detected %d new comment(s) for %s",
			len(local.Comments)-len(remote.Comments), local.JiraKey)

		// Push new comments
		for i := len(remote.Comments); i < len(local.Comments); i++ {
			comment := local.Comments[i]
			log.Printf("[DEBUG] pushIssue: Adding comment #%d to %s", i+1, local.JiraKey)
			if err := client.AddComment(ctx, local.JiraKey, comment.Body); err != nil {
				return fmt.Errorf("failed to add comment #%d: %w", i+1, err)
			}
		}
	}

	// After successful push, update the local file with new hash
	// We need to fetch the updated issue from Jira to get the correct hash
	log.Printf("[DEBUG] pushIssue: Fetching updated issue %s from Jira", local.JiraKey)
	memberCache, _ := LoadMembersCache(storage.projectPath)
	updatedIssue, err := client.GetIssue(ctx, local.JiraKey, memberCache)
	if err != nil {
		log.Printf("[WARN] pushIssue: Failed to fetch updated issue, keeping local hash: %v", err)
		// Don't fail - the push succeeded, we just couldn't update the hash
		return nil
	}

	// Save the updated issue to update the hash
	if err := storage.SaveIssue(updatedIssue); err != nil {
		log.Printf("[WARN] pushIssue: Failed to save updated issue: %v", err)
		// Don't fail - the push succeeded
	}

	return nil
}

// equalStringSlices compares two string slices for equality (order matters)
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// FormatConflictError formats conflict information into a user-friendly error message
func FormatConflictError(conflicts []ConflictInfo) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Cannot push - %d issue(s) have conflicts:\n", len(conflicts)))
	for _, c := range conflicts {
		buf.WriteString(fmt.Sprintf("  - %s: Remote modified (last updated: %s)\n",
			c.IssueKey, c.Updated.Format("2006-01-02 15:04")))
	}
	buf.WriteString("\nRun 'takl jira pull' to fetch remote changes, then push again.")
	return buf.String()
}
