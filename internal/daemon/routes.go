package daemon

import (
	"net/http"

	"github.com/takl/takl/internal/http/handlers"
	"github.com/takl/takl/internal/http/middleware"
)

func (d *Daemon) setupRoutes(mux *http.ServeMux) {
	// Create IssueService with the new formalized interfaces
	issueService := d.createIssueService()
	h := handlers.NewHandlers(d.registry, issueService)

	// Set up the index status handler with real dependencies
	if d.indexer != nil && d.watcher != nil {
		indexStatusHandler := handlers.NewIndexStatusHandler(d.indexer, d.watcher, d.watcherEvents)
		h.SetIndexStatusHandler(indexStatusHandler)
		// Store reference in daemon for event tracking
		d.indexStatusHandler = indexStatusHandler
	}

	// Set up middleware chain
	stdMiddleware := middleware.StandardMiddleware(d.logger)

	// Helper function to wrap handlers with middleware
	handle := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, stdMiddleware(handler))
	}

	// Health and system endpoints
	handle("/health", h.Health.HandleHealth)
	handle("/stats", h.Health.HandleStats)
	handle("/api/reload", h.Health.HandleReload)

	// Monitoring endpoints
	handle("/api/index/status", h.IndexStatus.HandleIndexStatus)
	handle("/api/watcher/status", h.IndexStatus.HandleWatcherStatus)

	// Registry API
	handle("/api/registry/projects", h.Registry.HandleRegistryProjects)
	handle("/api/registry/projects/", h.Registry.HandleRegistryProject)
	handle("/api/registry/health", h.Registry.HandleRegistryHealth)
	handle("/api/registry/cleanup", h.Registry.HandleRegistryCleanup)

	// Project-scoped API (requires project context)
	handle("/api/projects/", h.HandleProjectAPI)

	// Global operations
	handle("/api/search", h.Search.HandleGlobalSearch)
	handle("/api/issues", h.Search.HandleGlobalIssues)
}
