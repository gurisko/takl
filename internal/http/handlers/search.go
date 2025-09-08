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

// SearchHandler handles search operations
type SearchHandler struct {
	registry     *registry.Registry
	issueService *app.IssueService
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(registry *registry.Registry, issueService *app.IssueService) *SearchHandler {
	return &SearchHandler{
		registry:     registry,
		issueService: issueService,
	}
}

// HandleProjectSearch handles GET /api/projects/{id}/search
func (h *SearchHandler) HandleProjectSearch(w http.ResponseWriter, r *http.Request, project *registry.Project) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("service unavailable"))
		return
	}

	query := r.URL.Query()
	searchQuery := query.Get("q")
	if searchQuery == "" {
		writeError(w, fmt.Errorf("invalid: search query 'q' parameter required"))
		return
	}

	// Perform search via service
	fmt.Printf("DEBUG: Handler calling SearchIssues with project.ID=%s, query=%s\n", project.ID, searchQuery)
	issues, err := h.issueService.SearchIssues(context.Background(), project.ID, searchQuery)
	if err != nil {
		fmt.Printf("DEBUG: SearchIssues failed: %v\n", err)
		writeError(w, fmt.Errorf("search failed: %w", err))
		return
	}
	fmt.Printf("DEBUG: SearchIssues returned %d issues\n", len(issues))

	// Apply additional filters if provided
	if status := query.Get("status"); status != "" {
		filtered := make([]*domain.Issue, 0, len(issues))
		for _, issue := range issues {
			if strings.EqualFold(issue.Status, status) {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}

	if issueType := query.Get("type"); issueType != "" {
		filtered := make([]*domain.Issue, 0, len(issues))
		for _, issue := range issues {
			if strings.EqualFold(issue.Type, issueType) {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}

	// Apply pagination
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit < len(issues) {
			issues = issues[:limit]
		}
	}

	// Convert to DTOs
	response := make([]*dto.IssueResponse, len(issues))
	for i, issue := range issues {
		response[i] = dto.FromDomainIssue(issue)
	}

	// Match the SDK expected format
	result := struct {
		Query   string               `json:"query"`
		Results []*dto.IssueResponse `json:"results"`
		Total   int                  `json:"total"`
		Project interface{}          `json:"project,omitempty"`
	}{
		Query:   searchQuery,
		Results: response,
		Total:   len(response),
		Project: dto.FromRegistryProject(project),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleGlobalSearch handles GET /api/search
func (h *SearchHandler) HandleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	query := r.URL.Query()
	searchQuery := query.Get("q")
	if searchQuery == "" {
		writeError(w, fmt.Errorf("invalid: search query 'q' parameter required"))
		return
	}

	// TODO: Get all registered projects and search across them
	// projects := h.registry.ListProjects()

	// Search across all projects via service
	// TODO: Implement global search when we have proper multi-project service support
	allResults := make(map[string][]*dto.IssueResponse)
	totalResults := 0

	// For now, just return empty results as a placeholder
	// This would need proper implementation when we have IndexedSearchService

	// Apply global limit if specified
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			currentCount := 0
			for projectID, results := range allResults {
				if currentCount >= limit {
					delete(allResults, projectID)
					continue
				}

				remainingLimit := limit - currentCount
				if len(results) > remainingLimit {
					allResults[projectID] = results[:remainingLimit]
				}
				currentCount += len(allResults[projectID])
			}
		}
	}

	result := map[string]interface{}{
		"query":          searchQuery,
		"total_results":  totalResults,
		"projects_count": len(allResults),
		"results":        allResults,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleGlobalIssues handles GET /api/issues
func (h *SearchHandler) HandleGlobalIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	query := r.URL.Query()

	// TODO: Get all registered projects and list issues across them
	// projects := h.registry.ListProjects()

	// Parse global filters
	filters := make(map[string]interface{})
	if status := query.Get("status"); status != "" {
		filters["status"] = status
	}
	if issueType := query.Get("type"); issueType != "" {
		filters["type"] = issueType
	}
	if priority := query.Get("priority"); priority != "" {
		filters["priority"] = priority
	}
	if assignee := query.Get("assignee"); assignee != "" {
		filters["assignee"] = assignee
	}

	// Collect issues from all projects
	// TODO: Implement global issues list when we have proper multi-project service support
	allResults := make(map[string][]*dto.IssueResponse)
	totalResults := 0

	// For now, just return empty results as a placeholder
	// This would need proper implementation when we have multi-project support in the service layer

	// Apply global limit if specified
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			currentCount := 0
			for projectID, results := range allResults {
				if currentCount >= limit {
					delete(allResults, projectID)
					continue
				}

				remainingLimit := limit - currentCount
				if len(results) > remainingLimit {
					allResults[projectID] = results[:remainingLimit]
				}
				currentCount += len(allResults[projectID])
			}
		}
	}

	result := map[string]interface{}{
		"filters":        filters,
		"total_results":  totalResults,
		"projects_count": len(allResults),
		"results":        allResults,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}
