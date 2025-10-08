//go:build unix

package daemon

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/gurisko/takl/internal/bridge/jira"
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
	if err := json.NewDecoder(io.LimitReader(r.Body, apiclient.MaxJSONPayloadSize)).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ProjectPath == "" || req.Config.BaseURL == "" || req.Config.Project == "" {
		writeError(w, "missing required fields", http.StatusBadRequest)
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
