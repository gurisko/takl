package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/takl/takl/internal/app"
	"github.com/takl/takl/internal/http/middleware"
	"github.com/takl/takl/internal/registry"
)

// Handlers aggregates all HTTP handlers
type Handlers struct {
	Registry     *RegistryHandler
	IssuesRead   *IssuesReadHandler
	IssuesWrite  *IssuesWriteHandler
	Search       *SearchHandler
	Health       *HealthHandler
	IndexStatus  *IndexStatusHandler
	registry     *registry.Registry
	issueService *app.IssueService
}

// writeError writes a structured error response using the centralized error handler
func writeError(w http.ResponseWriter, err error) {
	if eh, ok := middleware.GetErrorHandler(w); ok {
		eh.WriteError(err)
	} else {
		// Fallback for handlers not using error middleware
		httpErr := middleware.MapAppError(err)
		middleware.WriteErr(w, httpErr)
	}
}

// NewHandlers creates a new handlers collection
func NewHandlers(registry *registry.Registry, issueService *app.IssueService) *Handlers {
	return &Handlers{
		Registry:     NewRegistryHandler(registry),
		IssuesRead:   NewIssuesReadHandler(registry, issueService),
		IssuesWrite:  NewIssuesWriteHandler(issueService),
		Search:       NewSearchHandler(registry, issueService),
		Health:       NewHealthHandler(registry),
		IndexStatus:  NewIndexStatusHandler(nil, nil, nil), // Will be set by daemon if available
		registry:     registry,
		issueService: issueService,
	}
}

// SetIndexStatusHandler updates the index status handler with real dependencies
func (h *Handlers) SetIndexStatusHandler(handler *IndexStatusHandler) {
	h.IndexStatus = handler
}

// HandleProjectAPI routes project-scoped API requests
func (h *Handlers) HandleProjectAPI(w http.ResponseWriter, r *http.Request) {
	// Extract project ID and subpath from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, fmt.Errorf("invalid: project ID required"))
		return
	}

	projectID := parts[0]
	subPath := parts[1:]

	// Get project from registry
	project, found := h.registry.GetProject(projectID)
	if !found {
		writeError(w, fmt.Errorf("not found: project not found"))
		return
	}

	// Route to appropriate sub-handler based on subpath
	if len(subPath) == 0 {
		// GET /api/projects/{id} -> project info/status
		h.IssuesRead.HandleProjectStatus(w, r, project)
		return
	}

	fmt.Printf("DEBUG: HandleProjectAPI routing subPath[0]='%s'\n", subPath[0])
	switch subPath[0] {
	case "issues":
		h.handleProjectIssues(w, r, project, subPath[1:])
	case "search":
		fmt.Printf("DEBUG: Routing to HandleProjectSearch\n")
		h.Search.HandleProjectSearch(w, r, project)
	default:
		writeError(w, fmt.Errorf("not found: unknown project endpoint"))
	}
}

// handleProjectIssues routes issue-related requests for a specific project
func (h *Handlers) handleProjectIssues(w http.ResponseWriter, r *http.Request, project *registry.Project, subPath []string) {
	if len(subPath) == 0 {
		// /api/projects/{id}/issues
		switch r.Method {
		case http.MethodGet:
			h.IssuesRead.HandleListIssues(w, r, project.ID)
		case http.MethodPost:
			h.IssuesWrite.HandleCreateIssue(w, r, project.ID)
		default:
			writeError(w, fmt.Errorf("method not allowed"))
		}
		return
	}

	// /api/projects/{id}/issues/{issueID}
	issueID := subPath[0]
	switch r.Method {
	case http.MethodGet:
		h.IssuesRead.HandleGetIssue(w, r, project.ID, issueID)
	case http.MethodPut:
		h.IssuesWrite.HandleUpdateIssue(w, r, project.ID, issueID)
	case http.MethodPatch:
		h.IssuesWrite.HandlePatchIssue(w, r, project.ID, issueID)
	case http.MethodDelete:
		h.IssuesWrite.HandleDeleteIssue(w, r, project.ID, issueID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
