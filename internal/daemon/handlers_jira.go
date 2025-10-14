//go:build unix

package daemon

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"

	"github.com/gurisko/takl/internal/bridge/jira"
	"github.com/gurisko/takl/internal/limits"
)

// jiraPullRequest is the JSON payload for pull requests
type jiraPullRequest struct {
	ProjectPath string          `json:"project_path"`
	Config      jira.JiraConfig `json:"config"`
}

// handleJiraPull handles POST /api/jira/pull
func (d *Daemon) handleJiraPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req jiraPullRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, limits.JSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request - all fields required
	if req.ProjectPath == "" {
		writeError(w, "project_path is required", http.StatusBadRequest)
		return
	}
	if req.Config.BaseURL == "" {
		writeError(w, "config.base_url is required", http.StatusBadRequest)
		return
	}
	if req.Config.Email == "" {
		writeError(w, "config.email is required", http.StatusBadRequest)
		return
	}
	if req.Config.APIToken == "" {
		writeError(w, "config.api_token is required", http.StatusBadRequest)
		return
	}
	if req.Config.Project == "" {
		writeError(w, "config.project is required", http.StatusBadRequest)
		return
	}

	// Create Jira client
	client := jira.NewClient(req.Config.BaseURL, req.Config.Email, req.Config.APIToken)

	// Create storage
	storage, err := jira.NewStorage(req.ProjectPath)
	if err != nil {
		writeError(w, "failed to initialize storage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute pull
	result, err := jira.Pull(r.Context(), client, storage, &req.Config)
	if err != nil {
		writeError(w, "pull failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleJiraMembers handles POST /api/jira/members
func (d *Daemon) handleJiraMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req jiraPullRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, limits.JSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ProjectPath == "" {
		writeError(w, "project_path is required", http.StatusBadRequest)
		return
	}
	if req.Config.BaseURL == "" {
		writeError(w, "config.base_url is required", http.StatusBadRequest)
		return
	}
	if req.Config.Email == "" {
		writeError(w, "config.email is required", http.StatusBadRequest)
		return
	}
	if req.Config.APIToken == "" {
		writeError(w, "config.api_token is required", http.StatusBadRequest)
		return
	}
	if req.Config.Project == "" {
		writeError(w, "config.project is required", http.StatusBadRequest)
		return
	}

	// Create Jira client
	client := jira.NewClient(req.Config.BaseURL, req.Config.Email, req.Config.APIToken)

	// Refresh member cache
	cache, err := jira.RefreshMemberCache(r.Context(), client, req.ProjectPath, req.Config.Project)
	if err != nil {
		writeError(w, "failed to refresh cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert cache to members slice for response
	members := make([]*jira.Member, 0, len(cache.Members))
	for _, member := range cache.Members {
		members = append(members, member)
	}

	// Return members list
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(members)
}

// handleJiraWorkflow handles POST /api/jira/workflow
func (d *Daemon) handleJiraWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req jiraPullRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, limits.JSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ProjectPath == "" {
		writeError(w, "project_path is required", http.StatusBadRequest)
		return
	}
	if req.Config.BaseURL == "" {
		writeError(w, "config.base_url is required", http.StatusBadRequest)
		return
	}
	if req.Config.Email == "" {
		writeError(w, "config.email is required", http.StatusBadRequest)
		return
	}
	if req.Config.APIToken == "" {
		writeError(w, "config.api_token is required", http.StatusBadRequest)
		return
	}
	if req.Config.Project == "" {
		writeError(w, "config.project is required", http.StatusBadRequest)
		return
	}

	// Create Jira client
	client := jira.NewClient(req.Config.BaseURL, req.Config.Email, req.Config.APIToken)

	// Refresh workflow cache
	cache, err := jira.RefreshWorkflowCache(r.Context(), client, req.ProjectPath, req.Config.Project)
	if err != nil {
		writeError(w, "failed to refresh cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert cache to statuses slice for response
	statuses := make([]*jira.StatusInfo, 0, len(cache.Statuses))
	for _, status := range cache.Statuses {
		statuses = append(statuses, status)
	}

	// Sort statuses deterministically by name, then by ID
	sortStatuses(statuses)

	// Return statuses list
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(statuses)
}

// sortStatuses sorts statuses by category order (To Do → In Progress → Done → Undefined),
// then by name alphabetically, then by ID for stability
func sortStatuses(statuses []*jira.StatusInfo) {
	// Define workflow category order
	categoryOrder := map[string]int{
		"new":           0, // To Do
		"indeterminate": 1, // In Progress
		"done":          2, // Done
		"undefined":     3, // Undefined
	}

	sort.Slice(statuses, func(i, j int) bool {
		catI := categoryOrder[statuses[i].Category]
		catJ := categoryOrder[statuses[j].Category]

		// First: sort by category
		if catI != catJ {
			return catI < catJ
		}

		// Second: sort by name within category
		if statuses[i].Name != statuses[j].Name {
			return statuses[i].Name < statuses[j].Name
		}

		// Third: sort by ID for stability (when name and category are equal)
		return statuses[i].ID < statuses[j].ID
	})
}
