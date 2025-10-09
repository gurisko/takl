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
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
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
func (c *Client) SearchIssues(ctx context.Context, jql string, maxResults int) ([]Issue, error) {
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

		// Convert and append issues
		for _, jiraIssue := range searchResp.Issues {
			allIssues = append(allIssues, convertJiraIssue(jiraIssue))
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
func convertJiraIssue(jr jiraIssueResponse) Issue {
	// Convert description from ADF to Markdown
	description, err := ADFToMarkdown(jr.Fields.Description)
	if err != nil {
		log.Printf("[WARN] Failed to convert description for %s: %v", jr.Key, err)
		description = "" // Fallback to empty string
	}

	issue := Issue{
		JiraKey:     jr.Key,
		JiraID:      jr.ID,
		Title:       jr.Fields.Summary,
		Description: description,
		Status:      jr.Fields.Status.Name,
		Reporter:    jr.Fields.Reporter.DisplayName,
		Created:     jr.Fields.Created.Time,
		Updated:     jr.Fields.Updated.Time,
		Labels:      jr.Fields.Labels,
	}

	if jr.Fields.Assignee != nil {
		issue.Assignee = jr.Fields.Assignee.DisplayName
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
			Author:  jc.Author.DisplayName,
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
