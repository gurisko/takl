package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/takl/takl/internal/app"
	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/http/dto"
	"github.com/takl/takl/internal/registry"
)

// IssuesReadHandler handles read operations for issues
type IssuesReadHandler struct {
	registry     *registry.Registry
	issueService *app.IssueService
}

// NewIssuesReadHandler creates a new issues read handler
func NewIssuesReadHandler(registry *registry.Registry, issueService *app.IssueService) *IssuesReadHandler {
	return &IssuesReadHandler{
		registry:     registry,
		issueService: issueService,
	}
}

// HandleListIssues handles GET /api/projects/{id}/issues
func (h *IssuesReadHandler) HandleListIssues(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("service unavailable: issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	filter := domain.IssueFilter{}

	if status := query.Get("status"); status != "" {
		filter.Status = status
	}
	if issueType := query.Get("type"); issueType != "" {
		filter.Type = issueType
	}
	if priority := query.Get("priority"); priority != "" {
		filter.Priority = priority
	}
	if assignee := query.Get("assignee"); assignee != "" {
		filter.Assignee = assignee
	}
	if labels := query.Get("labels"); labels != "" {
		filter.Labels = strings.Split(labels, ",")
	}

	// Parse pagination
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Get issues via service
	issues, err := h.issueService.ListIssues(context.Background(), projectID, filter)
	if err != nil {
		writeError(w, fmt.Errorf("failed to list issues: %w", err))
		return
	}

	// Convert to DTOs
	issueResp := make([]*dto.IssueResponse, len(issues))
	for i, issue := range issues {
		issueResp[i] = dto.FromDomainIssue(issue)
	}

	// Wrap in expected SDK format
	response := struct {
		Issues []*dto.IssueResponse `json:"issues"`
		Total  int                  `json:"total"`
	}{
		Issues: issueResp,
		Total:  len(issueResp),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleGetIssue handles GET /api/projects/{id}/issues/{issueID}
func (h *IssuesReadHandler) HandleGetIssue(w http.ResponseWriter, r *http.Request, projectID, issueID string) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("service unavailable: issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	issue, err := h.issueService.GetIssue(context.Background(), projectID, issueID)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto.FromDomainIssue(issue)); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleProjectStatus handles GET /api/projects/{id}/status
func (h *IssuesReadHandler) HandleProjectStatus(w http.ResponseWriter, r *http.Request, project *registry.Project) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("service unavailable: issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	// Get basic stats via service
	allIssues, err := h.issueService.ListIssues(context.Background(), project.ID, domain.IssueFilter{})
	if err != nil {
		writeError(w, fmt.Errorf("failed to get project stats: %w", err))
		return
	}

	// Count by status
	statusCounts := make(map[string]int)
	typeCounts := make(map[string]int)
	priorityCounts := make(map[string]int)

	for _, issue := range allIssues {
		statusCounts[issue.Status]++
		typeCounts[issue.Type]++
		priorityCounts[issue.Priority]++
	}

	stats := map[string]interface{}{
		"project":         dto.FromRegistryProject(project),
		"total_issues":    len(allIssues),
		"status_counts":   statusCounts,
		"type_counts":     typeCounts,
		"priority_counts": priorityCounts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}
