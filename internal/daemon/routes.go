//go:build unix

package daemon

import (
	"encoding/json"
	"net/http"
	"time"
)

func (d *Daemon) setupRoutes(mux *http.ServeMux) {
	// Health endpoint
	mux.HandleFunc("/health", d.handleHealth)

	// Project registry endpoints
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			d.handleRegisterProject(w, r)
		case http.MethodGet:
			d.handleListProjects(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/projects/", d.handleProjectByID)
}

func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(d.startTime).Seconds(),
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
