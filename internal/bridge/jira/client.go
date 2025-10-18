package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Client is a lightweight Jira REST API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	email      string
	apiToken   string
}

// NewClient creates a new Jira API client
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		email:      email,
		apiToken:   apiToken,
	}
}

// doRequest executes an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	endpoint := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.apiToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, MaxErrorBodySize))
		resp.Body.Close()
		return nil, fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// SearchIssues searches for issues using JQL with pagination
// If cache is provided, formats users as "Display Name <email>", otherwise uses display name only
func (c *Client) SearchIssues(ctx context.Context, jql string, maxResults int, cache *MemberCache) ([]Issue, error) {
	var allIssues []Issue
	nextPageToken := ""
	pageNum := 1

	for {
		// Calculate page size: request minimum of (SearchPageSize, remaining results)
		// This avoids over-fetching when maxResults is smaller than page size
		want := SearchPageSize
		if maxResults > 0 && maxResults-len(allIssues) < want {
			want = maxResults - len(allIssues)
			if want <= 0 {
				break
			}
		}

		reqBody := map[string]interface{}{
			"jql":        jql,
			"maxResults": want,
			"fields":     []string{"summary", "description", "status", "assignee", "reporter", "created", "updated", "labels", "comment", "attachment"},
		}

		if nextPageToken != "" {
			reqBody["nextPageToken"] = nextPageToken
		}

		tokenDisplay := "none"
		if len(nextPageToken) > 8 {
			tokenDisplay = nextPageToken[:8] + "..."
		}
		log.Printf("[DEBUG] SearchIssues: Fetching page %d (token: %s)", pageNum, tokenDisplay)

		resp, err := c.doRequest(ctx, "POST", "/rest/api/3/search/jql", reqBody)
		if err != nil {
			return nil, err
		}

		var searchResp jiraSearchResponse
		if err := json.NewDecoder(io.LimitReader(resp.Body, MaxSearchResponseSize)).Decode(&searchResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode search response: %w", err)
		}
		resp.Body.Close()

		nextTokenDisplay := "none"
		if len(searchResp.NextPageToken) > 8 {
			nextTokenDisplay = searchResp.NextPageToken[:8] + "..."
		}
		log.Printf("[DEBUG] SearchIssues: Page %d returned %d issues (total so far: %d, nextToken: %s)",
			pageNum, len(searchResp.Issues), len(allIssues)+len(searchResp.Issues), nextTokenDisplay)

		// Convert and append issues (skip archived ones)
		for _, jiraIssue := range searchResp.Issues {
			// Skip archived issues
			if jiraIssue.Archived {
				log.Printf("[DEBUG] SearchIssues: Skipping archived issue %s", jiraIssue.Key)
				continue
			}
			allIssues = append(allIssues, convertJiraIssue(jiraIssue, cache))
		}

		// Check if we have more pages
		// Treat maxResults <= 0 as unlimited
		if searchResp.NextPageToken == "" || (maxResults > 0 && len(allIssues) >= maxResults) {
			break
		}

		nextPageToken = searchResp.NextPageToken
		pageNum++
	}

	log.Printf("[DEBUG] SearchIssues: Complete - fetched %d total issues", len(allIssues))
	return allIssues, nil
}

// convertJiraIssue converts Jira API response to our Issue type
// If cache is provided, formats users as "Display Name <email>", otherwise uses display name only
func convertJiraIssue(jr jiraIssueResponse, cache *MemberCache) Issue {
	// Convert description from ADF to Markdown
	description, err := ADFToMarkdown(jr.Fields.Description)
	if err != nil {
		log.Printf("[WARN] Failed to convert description for %s: %v", jr.Key, err)
		description = "" // Fallback to empty string
	}

	// Helper to format user from accountId
	formatUser := func(accountID, displayName string) string {
		if cache != nil {
			if member := cache.FindByAccountID(accountID); member != nil {
				return member.FormatMember()
			}
		}
		// Fallback to display name only
		return displayName
	}

	issue := Issue{
		JiraKey:     jr.Key,
		JiraID:      jr.ID,
		Title:       jr.Fields.Summary,
		Description: description,
		Status:      jr.Fields.Status.Name,
		Reporter:    formatUser(jr.Fields.Reporter.AccountID, jr.Fields.Reporter.DisplayName),
		Created:     jr.Fields.Created.Time,
		Updated:     jr.Fields.Updated.Time,
		Labels:      jr.Fields.Labels,
	}

	if jr.Fields.Assignee != nil {
		issue.Assignee = formatUser(jr.Fields.Assignee.AccountID, jr.Fields.Assignee.DisplayName)
	}

	// Convert comments
	issue.Comments = make([]Comment, 0, len(jr.Fields.Comment.Comments))
	for _, jc := range jr.Fields.Comment.Comments {
		// Convert comment body from ADF to Markdown
		body, err := ADFToMarkdown(jc.Body)
		if err != nil {
			log.Printf("[WARN] Failed to convert comment body for %s: %v", jr.Key, err)
			body = "" // Fallback to empty string
		}

		issue.Comments = append(issue.Comments, Comment{
			ID:      jc.ID,
			Author:  formatUser(jc.Author.AccountID, jc.Author.DisplayName),
			Body:    body,
			Created: jc.Created.Time,
			Updated: jc.Updated.Time,
		})
	}

	// Convert attachments
	issue.Attachments = make([]Attachment, 0, len(jr.Fields.Attachment))
	for _, ja := range jr.Fields.Attachment {
		issue.Attachments = append(issue.Attachments, Attachment{
			ID:       ja.ID,
			Filename: ja.Filename,
			URL:      ja.Content,
			MimeType: ja.MimeType,
			Size:     ja.Size,
			Created:  ja.Created.Time,
		})
	}

	return issue
}

// FetchProjectMembers fetches all assignable users for a project with pagination
// The Jira API limits results per request, so we paginate using startAt to get all users
func (c *Client) FetchProjectMembers(ctx context.Context, projectKey string) ([]*Member, error) {
	var allMembers []*Member
	startAt := 0
	pageSize := 1_000
	pageNum := 1

	// URL-escape project key for safety
	escapedProjectKey := url.QueryEscape(projectKey)

	for {
		// Build paginated request path
		path := fmt.Sprintf("/rest/api/3/user/assignable/search?project=%s&maxResults=%d&startAt=%d",
			escapedProjectKey, pageSize, startAt)

		log.Printf("[DEBUG] FetchProjectMembers: Fetching page %d (startAt=%d)", pageNum, startAt)

		resp, err := c.doRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch project members: %w", err)
		}

		var users []jiraUserResponse
		if err := json.NewDecoder(io.LimitReader(resp.Body, MaxSearchResponseSize)).Decode(&users); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode users response: %w", err)
		}
		resp.Body.Close()

		log.Printf("[DEBUG] FetchProjectMembers: Page %d returned %d users (total so far: %d)",
			pageNum, len(users), len(allMembers)+len(users))

		// Convert and append users
		for _, user := range users {
			allMembers = append(allMembers, &Member{
				AccountID:    user.AccountID,
				DisplayName:  user.DisplayName,
				EmailAddress: user.EmailAddress,
				Active:       user.Active,
			})
		}

		// Check if we've fetched all users
		// If we got fewer results than requested, we've reached the end
		if len(users) < pageSize {
			break
		}

		startAt += len(users)
		pageNum++
	}

	log.Printf("[DEBUG] FetchProjectMembers: Complete - fetched %d total members for project %s", len(allMembers), projectKey)
	return allMembers, nil
}

// FetchProjectStatuses fetches all statuses for a project grouped by issue type
// Returns a deduplicated list of all unique statuses across all issue types
func (c *Client) FetchProjectStatuses(ctx context.Context, projectKey string) ([]*StatusInfo, error) {
	// URL-escape project key for safety
	escapedProjectKey := url.QueryEscape(projectKey)
	path := fmt.Sprintf("/rest/api/3/project/%s/statuses", escapedProjectKey)

	log.Printf("[DEBUG] FetchProjectStatuses: Fetching statuses for project %s", projectKey)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project statuses: %w", err)
	}
	defer resp.Body.Close()

	var issueTypes []jiraProjectStatusesResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, MaxSearchResponseSize)).Decode(&issueTypes); err != nil {
		return nil, fmt.Errorf("failed to decode statuses response: %w", err)
	}

	// Deduplicate statuses across issue types (same status can appear in multiple issue types)
	statusMap := make(map[string]*StatusInfo)
	for _, issueType := range issueTypes {
		for _, jiraStatus := range issueType.Statuses {
			// Use status ID as key for deduplication (more stable than name)
			if _, exists := statusMap[jiraStatus.ID]; !exists {
				statusMap[jiraStatus.ID] = &StatusInfo{
					ID:       jiraStatus.ID,
					Name:     jiraStatus.Name,
					Category: jiraStatus.StatusCategory.Key,
				}
			}
		}
	}

	// Convert map to slice
	statuses := make([]*StatusInfo, 0, len(statusMap))
	for _, status := range statusMap {
		statuses = append(statuses, status)
	}

	log.Printf("[DEBUG] FetchProjectStatuses: Complete - fetched %d unique statuses for project %s", len(statuses), projectKey)
	return statuses, nil
}

// GetIssue fetches a single issue by key from Jira
// Used for conflict detection when pushing changes
func (c *Client) GetIssue(ctx context.Context, issueKey string, cache *MemberCache) (*Issue, error) {
	// URL-escape issue key for safety
	escapedKey := url.QueryEscape(issueKey)
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary,description,status,assignee,reporter,created,updated,labels,comment,attachment", escapedKey)

	log.Printf("[DEBUG] GetIssue: Fetching issue %s", issueKey)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}
	defer resp.Body.Close()

	var jiraIssue jiraIssueResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, MaxSearchResponseSize)).Decode(&jiraIssue); err != nil {
		return nil, fmt.Errorf("failed to decode issue response: %w", err)
	}

	issue := convertJiraIssue(jiraIssue, cache)
	log.Printf("[DEBUG] GetIssue: Successfully fetched issue %s", issueKey)
	return &issue, nil
}

// UpdateIssue updates an issue's fields in Jira
// Supports updating: summary (title), description, and labels
// Note: description should be provided as markdown and will be converted to ADF
func (c *Client) UpdateIssue(ctx context.Context, issueKey string, updates map[string]interface{}) error {
	// URL-escape issue key for safety
	escapedKey := url.QueryEscape(issueKey)
	path := fmt.Sprintf("/rest/api/3/issue/%s", escapedKey)

	// Convert description from markdown to ADF if present
	if desc, ok := updates["description"].(string); ok {
		adf, err := MarkdownToADF(desc)
		if err != nil {
			return fmt.Errorf("failed to convert description to ADF: %w", err)
		}
		// Unmarshal the ADF JSON into a map so it serializes correctly
		var adfDoc map[string]interface{}
		if err := json.Unmarshal(adf, &adfDoc); err != nil {
			return fmt.Errorf("failed to unmarshal ADF: %w", err)
		}
		updates["description"] = adfDoc
	}

	// Wrap updates in "fields" object as required by Jira API
	body := map[string]interface{}{
		"fields": updates,
	}

	log.Printf("[DEBUG] UpdateIssue: Updating issue %s", issueKey)

	resp, err := c.doRequest(ctx, "PUT", path, body)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] UpdateIssue: Successfully updated issue %s", issueKey)
	return nil
}

// AddComment adds a comment to an issue in Jira
// The comment body should be in markdown format (will be converted to ADF)
func (c *Client) AddComment(ctx context.Context, issueKey string, commentBody string) error {
	// URL-escape issue key for safety
	escapedKey := url.QueryEscape(issueKey)
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", escapedKey)

	// Convert markdown to ADF for Jira API
	adf, err := MarkdownToADF(commentBody)
	if err != nil {
		return fmt.Errorf("failed to convert comment to ADF: %w", err)
	}

	// Unmarshal the ADF JSON into a map so it serializes correctly
	var adfDoc map[string]interface{}
	if err := json.Unmarshal(adf, &adfDoc); err != nil {
		return fmt.Errorf("failed to unmarshal ADF: %w", err)
	}

	body := map[string]interface{}{
		"body": adfDoc,
	}

	log.Printf("[DEBUG] AddComment: Adding comment to issue %s", issueKey)

	resp, err := c.doRequest(ctx, "POST", path, body)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] AddComment: Successfully added comment to issue %s", issueKey)
	return nil
}

// GetTransitions fetches available workflow transitions for an issue
func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]struct {
	ID         string
	Name       string
	ToStatus   string
	ToStatusID string
}, error) {
	// URL-escape issue key for safety
	escapedKey := url.QueryEscape(issueKey)
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", escapedKey)

	log.Printf("[DEBUG] GetTransitions: Fetching transitions for issue %s", issueKey)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transitions: %w", err)
	}
	defer resp.Body.Close()

	var transitionsResp jiraTransitionsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, MaxSearchResponseSize)).Decode(&transitionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode transitions response: %w", err)
	}

	// Convert to simpler format
	transitions := make([]struct {
		ID         string
		Name       string
		ToStatus   string
		ToStatusID string
	}, 0, len(transitionsResp.Transitions))

	for _, t := range transitionsResp.Transitions {
		transitions = append(transitions, struct {
			ID         string
			Name       string
			ToStatus   string
			ToStatusID string
		}{
			ID:         t.ID,
			Name:       t.Name,
			ToStatus:   t.To.Name,
			ToStatusID: t.To.ID,
		})
	}

	log.Printf("[DEBUG] GetTransitions: Found %d transitions for issue %s", len(transitions), issueKey)
	return transitions, nil
}

// TransitionIssue performs a workflow transition on an issue
func (c *Client) TransitionIssue(ctx context.Context, issueKey string, transitionID string) error {
	// URL-escape issue key for safety
	escapedKey := url.QueryEscape(issueKey)
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", escapedKey)

	body := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	log.Printf("[DEBUG] TransitionIssue: Transitioning issue %s with transition ID %s", issueKey, transitionID)

	resp, err := c.doRequest(ctx, "POST", path, body)
	if err != nil {
		return fmt.Errorf("failed to transition issue: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] TransitionIssue: Successfully transitioned issue %s", issueKey)
	return nil
}
