package handlers

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/takl/takl/internal/indexer"
	"github.com/takl/takl/internal/watcher"
)

// IndexStatusHandler handles indexer monitoring endpoints
type IndexStatusHandler struct {
	indexer       *indexer.Indexer
	watcher       *watcher.Watcher
	eventsChannel <-chan watcher.Event
	// Statistics tracking
	totalEvents     atomic.Uint64
	lastEventTime   atomic.Value // stores time.Time
	eventsProcessed atomic.Uint64
	eventsFailed    atomic.Uint64
}

// NewIndexStatusHandler creates a new index status handler
func NewIndexStatusHandler(ix *indexer.Indexer, w *watcher.Watcher, events <-chan watcher.Event) *IndexStatusHandler {
	h := &IndexStatusHandler{
		indexer:       ix,
		watcher:       w,
		eventsChannel: events,
	}
	h.lastEventTime.Store(time.Time{})
	return h
}

// HandleIndexStatus handles GET /api/index/status
func (h *IndexStatusHandler) HandleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Calculate queue depth (approximate based on channel buffer)
	queueDepth := 0
	if h.eventsChannel != nil {
		queueDepth = len(h.eventsChannel)
	}

	// Get last event time
	var lastEvent interface{}
	if lastTime, ok := h.lastEventTime.Load().(time.Time); ok && !lastTime.IsZero() {
		lastEvent = map[string]interface{}{
			"timestamp": lastTime.UTC(),
			"age":       time.Since(lastTime).String(),
		}
	}

	// Build status response
	status := map[string]interface{}{
		"status":           h.getIndexerStatus(),
		"queue_depth":      queueDepth,
		"last_event":       lastEvent,
		"events_total":     h.totalEvents.Load(),
		"events_processed": h.eventsProcessed.Load(),
		"events_failed":    h.eventsFailed.Load(),
		"timestamp":        time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getIndexerStatus determines the overall indexer status
func (h *IndexStatusHandler) getIndexerStatus() string {
	if h.indexer == nil {
		return "not_initialized"
	}

	// If we haven't seen events in a while, might be idle
	if lastTime, ok := h.lastEventTime.Load().(time.Time); ok {
		if time.Since(lastTime) > 5*time.Minute {
			return "idle"
		}
	}

	return "running"
}

// HandleWatcherStatus handles GET /api/watcher/status
func (h *IndexStatusHandler) HandleWatcherStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get watcher status
	watcherStatus := "not_initialized"
	watchedPaths := []string{}

	if h.watcher != nil {
		watcherStatus = "running"
		// Get watched projects from watcher
		watchedPaths = h.watcher.GetWatchedPaths()
	}

	// Get last event time
	var lastEvent interface{}
	if lastTime, ok := h.lastEventTime.Load().(time.Time); ok && !lastTime.IsZero() {
		lastEvent = map[string]interface{}{
			"timestamp": lastTime.UTC(),
			"age":       time.Since(lastTime).String(),
		}
	}

	status := map[string]interface{}{
		"status":          watcherStatus,
		"watched_paths":   watchedPaths,
		"events_received": h.totalEvents.Load(),
		"last_event":      lastEvent,
		"timestamp":       time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// RecordEvent tracks that an event was received (called by daemon)
func (h *IndexStatusHandler) RecordEvent(event watcher.Event) {
	h.totalEvents.Add(1)
	h.lastEventTime.Store(time.Now())
}

// RecordProcessed tracks that an event was successfully processed
func (h *IndexStatusHandler) RecordProcessed() {
	h.eventsProcessed.Add(1)
}

// RecordFailed tracks that an event processing failed
func (h *IndexStatusHandler) RecordFailed() {
	h.eventsFailed.Add(1)
}
