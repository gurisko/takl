// Package sdk provides a clean, high-level interface to the TAKL daemon API
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Client represents a connection to the TAKL daemon
type Client struct {
	httpClient *http.Client
	socketPath string
}

// readErrorResponse attempts to read and parse an error response from the server
func readErrorResponse(resp *http.Response, defaultMsg string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s (status %d)", defaultMsg, resp.StatusCode)
	}

	// Try to parse as JSON error
	var errorResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
		// Just return the error message without prefix if it's clear enough
		return fmt.Errorf("%s", errorResp.Error)
	}

	// Fall back to raw body if JSON parsing fails
	if len(body) > 0 {
		bodyStr := string(body)
		// Remove HTML tags if it's an HTML error page
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return fmt.Errorf("%s: %s", defaultMsg, bodyStr)
	}

	return fmt.Errorf("%s (status %d)", defaultMsg, resp.StatusCode)
}

// NewClient creates a new TAKL SDK client
func NewClient() (*Client, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	socketPath := os.Getenv("TAKL_SOCKET")
	if socketPath == "" {
		socketPath = filepath.Join(homeDir, ".takl", "daemon.sock")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: func(proto, addr string) (conn net.Conn, err error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 30 * time.Second,
	}

	client := &Client{
		httpClient: httpClient,
		socketPath: socketPath,
	}

	return client, nil
}

// EnsureDaemonRunning starts the daemon if it's not already running
func (c *Client) EnsureDaemonRunning() error {
	return c.EnsureDaemonRunningWithContext(context.Background())
}

// EnsureDaemonRunningWithContext starts the daemon if it's not already running with a context
func (c *Client) EnsureDaemonRunningWithContext(ctx context.Context) error {
	if c.isDaemonRunning() {
		return nil
	}

	// Start daemon as a separate process instead of a goroutine
	if err := c.startDaemonProcess(ctx); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Wait for daemon to be ready (max 5 seconds)
	for i := 0; i < 50; i++ {
		if c.isDaemonRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon failed to start within timeout")
}

func (c *Client) isDaemonRunning() bool {
	resp, err := c.httpClient.Get("http://localhost/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Issue represents an issue with simplified fields for SDK usage
type Issue struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Title    string    `json:"title"`
	Status   string    `json:"status"`
	Priority string    `json:"priority"`
	Assignee string    `json:"assignee"`
	Labels   []string  `json:"labels"`
	Content  string    `json:"content"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	FilePath string    `json:"file_path"`
}

// CreateIssueRequest represents a request to create an issue
type CreateIssueRequest struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Content  string   `json:"content,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Assignee string   `json:"assignee,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// UpdateIssueRequest represents a request to update an issue
type UpdateIssueRequest struct {
	Status   *string  `json:"status,omitempty"`
	Title    *string  `json:"title,omitempty"`
	Content  *string  `json:"content,omitempty"`
	Priority *string  `json:"priority,omitempty"`
	Assignee *string  `json:"assignee,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// ListIssuesRequest represents filters for listing issues
type ListIssuesRequest struct {
	Status   string `json:"status,omitempty"`
	Type     string `json:"type,omitempty"`
	Assignee string `json:"assignee,omitempty"`
}

// Project represents a registered project
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Mode        string    `json:"mode"`
	Description string    `json:"description"`
	Registered  time.Time `json:"registered"`
	LastAccess  time.Time `json:"last_access"`
	Active      bool      `json:"active"`
}

// RegisterProjectRequest represents a request to register a project
type RegisterProjectRequest struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Force       bool   `json:"force,omitempty"`
}

// Issue Operations

// CreateIssue creates a new issue in the specified project
func (c *Client) CreateIssue(projectID string, req CreateIssueRequest) (*Issue, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://localhost/api/projects/%s/issues", projectID)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create failed with status %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
		Issue  *Issue `json:"issue"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Issue, nil
}

// GetIssue retrieves an issue by ID
func (c *Client) GetIssue(projectID, issueID string) (*Issue, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost/api/projects/%s/issues/%s", projectID, issueID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get failed with status %d", resp.StatusCode)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(projectID, issueID string, req UpdateIssueRequest) (*Issue, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://localhost/api/projects/%s/issues/%s", projectID, issueID)
	httpReq, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

// ListIssues lists issues with optional filters
func (c *Client) ListIssues(projectID string, req ListIssuesRequest) ([]*Issue, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost/api/projects/%s/issues", projectID)

	// Add query parameters
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := httpReq.URL.Query()
	if req.Status != "" {
		q.Add("status", req.Status)
	}
	if req.Type != "" {
		q.Add("type", req.Type)
	}
	if req.Assignee != "" {
		q.Add("assignee", req.Assignee)
	}
	httpReq.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed with status %d", resp.StatusCode)
	}

	var result struct {
		Issues []*Issue `json:"issues"`
		Total  int      `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Issues, nil
}

// SearchIssues performs full-text search within a project
func (c *Client) SearchIssues(projectID, query string) ([]*Issue, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost/api/projects/%s/search?q=%s", projectID, url.QueryEscape(query))
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp, "search failed")
	}

	var result struct {
		Query   string   `json:"query"`
		Results []*Issue `json:"results"`
		Total   int      `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Results, nil
}

// GlobalSearchIssues searches across all projects
func (c *Client) GlobalSearchIssues(query string) (map[string]interface{}, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost/api/search?q=%s", url.QueryEscape(query))
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp, "global search failed")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Project Operations

// RegisterProject registers a new project
func (c *Client) RegisterProject(req RegisterProjectRequest) (*Project, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post("http://localhost/api/registry/projects", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var conflictResult struct {
			Error           string   `json:"error"`
			ExistingProject *Project `json:"existing_project"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&conflictResult); err == nil {
			return nil, fmt.Errorf("project already registered: %s", conflictResult.ExistingProject.Name)
		}
		return nil, fmt.Errorf("project already registered")
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, readErrorResponse(resp, "register failed")
	}

	var result struct {
		Status  string   `json:"status"`
		Project *Project `json:"project"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Project, nil
}

// ListProjects lists all registered projects
func (c *Client) ListProjects() ([]*Project, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Get("http://localhost/api/registry/projects")
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed with status %d", resp.StatusCode)
	}

	var result struct {
		Projects []*Project `json:"projects"`
		Total    int        `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Projects, nil
}

// GetProject gets a specific project by ID
func (c *Client) GetProject(projectID string) (*Project, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost/api/registry/projects/%s", projectID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("project not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get failed with status %d", resp.StatusCode)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

// UnregisterProject removes a project from the registry
func (c *Client) UnregisterProject(projectID string) error {
	if err := c.EnsureDaemonRunning(); err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost/api/registry/projects/%s", projectID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("project not found")
	}
	if resp.StatusCode != http.StatusOK {
		return readErrorResponse(resp, "unregister failed")
	}

	return nil
}

// CleanupProjects removes stale/inactive projects (older than 7 days)
func (c *Client) CleanupProjects() (int, error) {
	if err := c.EnsureDaemonRunning(); err != nil {
		return 0, err
	}

	projects, err := c.ListProjects()
	if err != nil {
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}

	cleanupCount := 0
	for _, project := range projects {
		// Mark as stale if inactive for more than 7 days
		if !project.Active && time.Since(project.LastAccess) > 7*24*time.Hour {
			if err := c.UnregisterProject(project.ID); err != nil {
				fmt.Printf("Warning: failed to cleanup project %s (%s): %v\n", project.ID, project.Name, err)
				continue
			}
			cleanupCount++
		}
	}

	return cleanupCount, nil
}

// Daemon Operations

// DaemonStatus represents the status of the daemon
type DaemonStatus struct {
	Running        bool   `json:"running"`
	Uptime         string `json:"uptime,omitempty"`
	RequestCount   uint64 `json:"request_count,omitempty"`
	ProjectCount   int    `json:"project_count,omitempty"`
	ActiveProjects int    `json:"active_projects,omitempty"`
}

// GetDaemonStatus returns the current daemon status
func (c *Client) GetDaemonStatus() (*DaemonStatus, error) {
	if !c.isDaemonRunning() {
		return &DaemonStatus{Running: false}, nil
	}

	resp, err := c.httpClient.Get("http://localhost/stats")
	if err != nil {
		return &DaemonStatus{Running: false}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &DaemonStatus{Running: false}, nil
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return &DaemonStatus{Running: true}, nil // Running but can't get stats
	}

	status := &DaemonStatus{
		Running: true,
	}

	if uptime, ok := stats["uptime"].(string); ok {
		status.Uptime = uptime
	}
	if count, ok := stats["request_count"].(float64); ok {
		status.RequestCount = uint64(count)
	}
	if count, ok := stats["project_count"].(float64); ok {
		status.ProjectCount = int(count)
	}

	return status, nil
}

// ReloadConfig triggers a configuration reload in the daemon
func (c *Client) ReloadConfig() error {
	if !c.isDaemonRunning() {
		return fmt.Errorf("daemon is not running")
	}

	req, err := http.NewRequest("POST", "http://localhost/api/reload", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reload failed with status %d", resp.StatusCode)
	}

	return nil
}

// startDaemonProcess starts the daemon as a separate process
func (c *Client) startDaemonProcess(ctx context.Context) error {
	// Get the path to the current executable (takl binary)
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Start daemon as a detached background process using "takl daemon start"
	cmd := exec.CommandContext(ctx, execPath, "daemon", "start")

	// Detach the process from the parent (important for process independence)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	// Redirect output to prevent hanging
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Don't wait for the process - let it run independently
	return nil
}
