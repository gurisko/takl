package store

import (
	"context"
	"log/slog"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/watcher"
)

// WatcherAdapter adapts the existing watcher.Watcher to domain.Watcher
type WatcherAdapter struct {
	watcher       *watcher.Watcher
	eventsChan    chan domain.FileEvent
	watcherEvents <-chan watcher.Event
	stopChan      chan struct{}
}

// NewWatcherAdapter creates a new adapter for watcher.Watcher
func NewWatcherAdapter(log *slog.Logger) (*WatcherAdapter, error) {
	// Create internal event channel for the watcher
	internalEvents := make(chan watcher.Event, 100)

	// Create the watcher
	w, err := watcher.New(log, internalEvents)
	if err != nil {
		return nil, err
	}

	// Create domain events channel
	domainEvents := make(chan domain.FileEvent, 100)

	adapter := &WatcherAdapter{
		watcher:       w,
		eventsChan:    domainEvents,
		watcherEvents: internalEvents,
		stopChan:      make(chan struct{}),
	}

	// Start event conversion goroutine
	go adapter.convertEvents()

	return adapter, nil
}

// Start implements domain.Watcher.Start
func (a *WatcherAdapter) Start(ctx context.Context) error {
	// Start the watcher in a goroutine
	go func() {
		if err := a.watcher.Run(); err != nil {
			// Log error but don't stop the whole system - watcher runs in background
			_ = err // TODO: Add proper error handling/logging
		}
	}()

	return nil
}

// Stop implements domain.Watcher.Stop
func (a *WatcherAdapter) Stop() error {
	// Signal event conversion goroutine to stop
	close(a.stopChan)

	// Close domain events channel
	close(a.eventsChan)

	// Stop the underlying watcher
	return a.watcher.Stop()
}

// AddProject implements domain.Watcher.AddProject
func (a *WatcherAdapter) AddProject(projectID, issuesDir string) error {
	return a.watcher.AddProject(projectID, issuesDir)
}

// RemoveProject implements domain.Watcher.RemoveProject
func (a *WatcherAdapter) RemoveProject(projectID string) error {
	// The existing watcher doesn't have a RemoveProject method
	// This would need to be implemented in the underlying watcher
	// For now, return nil (no-op)
	return nil
}

// Events implements domain.Watcher.Events
func (a *WatcherAdapter) Events() <-chan domain.FileEvent {
	return a.eventsChan
}

// GetStats implements domain.Watcher.GetStats
func (a *WatcherAdapter) GetStats() map[string]interface{} {
	stats := a.watcher.GetStats()

	// Convert WatcherStats to map[string]interface{}
	return map[string]interface{}{
		"project_count":   stats.ProjectCount,
		"watched_dirs":    stats.WatchedDirs,
		"debouncer_stats": stats.DebouncerStats,
		"last_error":      stats.LastError,
		"last_error_time": stats.LastErrorTime,
	}
}

// convertEvents converts watcher.Event to domain.FileEvent
func (a *WatcherAdapter) convertEvents() {
	for {
		select {
		case event, ok := <-a.watcherEvents:
			if !ok {
				return // Channel closed
			}

			// Convert watcher.Event to domain.FileEvent
			domainEvent := domain.FileEvent{
				ProjectID: event.ProjectID,
				Path:      event.Path,
				Type:      a.convertEventType(event.Type),
				Timestamp: event.Timestamp,
			}

			// Send to domain events channel
			select {
			case a.eventsChan <- domainEvent:
			case <-a.stopChan:
				return
			}

		case <-a.stopChan:
			return
		}
	}
}

// convertEventType converts watcher.EventType to domain.EventType
func (a *WatcherAdapter) convertEventType(watcherType watcher.EventType) domain.EventType {
	switch watcherType {
	case watcher.Upsert:
		return domain.EventUpsert
	case watcher.Delete:
		return domain.EventDelete
	case watcher.Rename:
		return domain.EventRename
	default:
		return domain.EventUpsert // Default fallback
	}
}
