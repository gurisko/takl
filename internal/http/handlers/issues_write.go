package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/takl/takl/internal/app"
	"github.com/takl/takl/internal/http/dto"
)

// IssuesWriteHandler handles write operations for issues
type IssuesWriteHandler struct {
	issueService *app.IssueService
}

// NewIssuesWriteHandler creates a new issues write handler
func NewIssuesWriteHandler(issueService *app.IssueService) *IssuesWriteHandler {
	return &IssuesWriteHandler{
		issueService: issueService,
	}
}

// HandleCreateIssue handles POST /api/projects/{id}/issues
func (h *IssuesWriteHandler) HandleCreateIssue(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	var req dto.CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Convert DTO to domain request (NO validation here - service layer validates)
	domainReq := req.ToDomain()

	// Create issue via service
	issue, err := h.issueService.CreateIssue(context.Background(), projectID, domainReq)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dto.FromDomainIssue(issue)); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleUpdateIssue handles PUT /api/projects/{id}/issues/{issueID}
func (h *IssuesWriteHandler) HandleUpdateIssue(w http.ResponseWriter, r *http.Request, projectID, issueID string) {
	if r.Method != http.MethodPut {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	var req dto.UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Convert DTO to domain request
	domainReq := req.ToDomain()

	// Update issue via service
	issue, err := h.issueService.UpdateIssue(context.Background(), projectID, issueID, domainReq)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto.FromDomainIssue(issue)); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandlePatchIssue handles PATCH /api/projects/{id}/issues/{issueID}
func (h *IssuesWriteHandler) HandlePatchIssue(w http.ResponseWriter, r *http.Request, projectID, issueID string) {
	if r.Method != http.MethodPatch {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	var req dto.PatchIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Convert patch request to update request
	domainReq := dto.UpdateIssueRequest(req).ToDomain()

	// Patch issue via service with write-through indexing
	issue, etag, err := h.issueService.PatchIssue(context.Background(), projectID, issueID, domainReq)
	if err != nil {
		writeError(w, err)
		return
	}

	// Set index status header if indexing failed
	if etag == "stale" {
		w.Header().Set("X-Index", "stale")
		etag = fmt.Sprintf(`"%d-%d"`, issue.Version, issue.Updated.Unix())
	}

	// Set ETag header
	w.Header().Set("ETag", etag)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto.FromDomainIssue(issue)); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleDeleteIssue handles DELETE /api/projects/{id}/issues/{issueID}
func (h *IssuesWriteHandler) HandleDeleteIssue(w http.ResponseWriter, r *http.Request, projectID, issueID string) {
	if r.Method != http.MethodDelete {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	if h.issueService == nil {
		writeError(w, fmt.Errorf("issue service not initialized - CLI->daemon convergence in progress"))
		return
	}

	// Delete issue via service
	if err := h.issueService.DeleteIssue(context.Background(), projectID, issueID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
