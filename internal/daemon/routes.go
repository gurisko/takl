//go:build unix
// +build unix

package daemon

import (
	"encoding/json"
	"net/http"
	"time"
)

func (d *Daemon) setupRoutes(mux *http.ServeMux) {
	// Health endpoint
	mux.HandleFunc("/health", d.handleHealth)
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
