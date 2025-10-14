//go:build unix

package daemon

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gurisko/takl/internal/bridge/jira"
)

// Request/Response types

type ListIssuesRequest struct {
	ProjectPath string   `json:"project_path"`
	Status      string   `json:"status,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Search      string   `json:"search,omitempty"`
}

type ListIssuesResponse struct {
	Issues []*jira.Issue `json:"issues"`
	Count  int           `json:"count"`
}

type ShowIssueRequest struct {
	ProjectPath string `json:"project_path"`
	IssueKey    string `json:"issue_key"`
}

type ShowIssueResponse struct {
	Issue *jira.Issue `json:"issue"`
}

// Handler methods

// handleListIssues handles GET /api/issues
// Accepts project_path and filter parameters via query string
func (d *Daemon) handleListIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	projectPath := query.Get("project_path")
	if projectPath == "" {
		writeError(w, "project_path query parameter is required", http.StatusBadRequest)
		return
	}

	// Open storage (read-only)
	storage, err := jira.OpenStorage(projectPath)
	if err != nil {
		writeError(w, "failed to open storage: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build filter from query parameters
	filter := jira.IssueFilter{
		Status:   query.Get("status"),
		Assignee: query.Get("assignee"),
		Search:   query.Get("search"),
	}

	// Parse labels (comma-separated) and trim whitespace
	if labelsParam := query.Get("labels"); labelsParam != "" {
		rawLabels := strings.Split(labelsParam, ",")
		filter.Labels = make([]string, len(rawLabels))
		for i, label := range rawLabels {
			filter.Labels[i] = strings.TrimSpace(label)
		}
	}

	// List issues with filters
	issues, err := storage.ListFilteredIssues(filter)
	if err != nil {
		writeError(w, "failed to list issues: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort by Updated desc, then by JiraKey for stable ordering
	sort.Slice(issues, func(i, j int) bool {
		if !issues[i].Updated.Equal(issues[j].Updated) {
			return issues[i].Updated.After(issues[j].Updated)
		}
		return issues[i].JiraKey < issues[j].JiraKey
	})

	resp := ListIssuesResponse{
		Issues: issues,
		Count:  len(issues),
	}
	writeJSON(w, resp, http.StatusOK)
}

// handleShowIssue handles GET /api/issues/{key}
// Expects project_path as query parameter and issue key in URL path
func (d *Daemon) handleShowIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract issue key from path: /api/issues/{key}
	issueKey := strings.TrimPrefix(r.URL.Path, "/api/issues/")
	if issueKey == "" || issueKey == r.URL.Path {
		writeError(w, "issue key is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	projectPath := query.Get("project_path")
	if projectPath == "" {
		writeError(w, "project_path query parameter is required", http.StatusBadRequest)
		return
	}

	// Open storage (read-only)
	storage, err := jira.OpenStorage(projectPath)
	if err != nil {
		writeError(w, "failed to open storage: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Read issue
	issue, err := storage.ReadIssue(issueKey)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, "issue not found: "+issueKey, http.StatusNotFound)
			return
		}
		writeError(w, "failed to read issue: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ShowIssueResponse{
		Issue: issue,
	}
	writeJSON(w, resp, http.StatusOK)
}
