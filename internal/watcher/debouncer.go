package watcher

import (
	"log/slog"
	"sync"
	"time"
)

// DebouncerConfig holds configuration for the debouncer
type DebouncerConfig struct {
	Window         time.Duration // Debounce window (default: 200ms)
	BurstThreshold int           // Events that trigger rescan (default: 100)
	BurstWindow    time.Duration // Window for burst detection (default: 2s)
	MaxQueueSize   int           // Maximum queue size (default: 10000)
}

// DefaultDebouncerConfig returns sensible defaults
func DefaultDebouncerConfig() DebouncerConfig {
	return DebouncerConfig{
		Window:         200 * time.Millisecond,
		BurstThreshold: 100,
		BurstWindow:    2 * time.Second,
		MaxQueueSize:   10000,
	}
}

// debouncer coalesces filesystem events to reduce noise
type debouncer struct {
	config  DebouncerConfig
	log     *slog.Logger
	output  chan Event
	queue   map[string]Event // path -> latest event
	timer   *time.Timer
	mu      sync.Mutex
	stopped bool

	// Burst detection
	burstCount    int
	burstStart    time.Time
	lastFlushTime time.Time

	// Statistics
	totalEvents     int64
	totalFlushes    int64
	burstDetections int64
}

// newDebouncer creates a new event debouncer with the specified configuration
func newDebouncer(config DebouncerConfig, log *slog.Logger) *debouncer {
	if config.Window == 0 {
		config = DefaultDebouncerConfig()
	}

	return &debouncer{
		config: config,
		log:    log,
		output: make(chan Event, config.MaxQueueSize/10), // 10% of max for buffering
		queue:  make(map[string]Event),
	}
}

// push adds an event to the debounce queue with burst detection
func (d *debouncer) push(event Event) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	d.totalEvents++
	now := time.Now()

	// Check for burst detection
	if d.burstStart.IsZero() || now.Sub(d.burstStart) > d.config.BurstWindow {
		// Start new burst window
		d.burstStart = now
		d.burstCount = 1
	} else {
		d.burstCount++
	}

	// Check if we've hit burst threshold
	if d.burstCount >= d.config.BurstThreshold {
		d.burstDetections++
		d.log.Warn("Burst detected - may schedule rescan",
			"events_count", d.burstCount,
			"window_duration", now.Sub(d.burstStart),
			"burst_detections", d.burstDetections)

		// Could trigger rescan logic here in the future
		// For now, just log and continue with normal debouncing
	}

	// Check queue size limits
	if len(d.queue) >= d.config.MaxQueueSize {
		d.log.Warn("Debouncer queue full, dropping oldest events",
			"queue_size", len(d.queue),
			"max_size", d.config.MaxQueueSize)

		// Simple strategy: flush immediately to prevent memory bloat
		d.flushLocked()
	}

	// Coalesce events by path - keep only the latest
	d.queue[event.Path] = event

	// Reset the timer
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.config.Window, d.flush)

	d.log.Debug("Debounced event",
		"path", event.Path,
		"type", event.Type,
		"queue_size", len(d.queue),
		"burst_count", d.burstCount)
}

// flush sends all queued events and clears the queue
func (d *debouncer) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.flushLocked()
}

// flushLocked performs the flush operation while already holding the mutex
func (d *debouncer) flushLocked() {
	if d.stopped || len(d.queue) == 0 {
		return
	}

	eventCount := len(d.queue)
	d.totalFlushes++
	d.lastFlushTime = time.Now()

	// Log summary for bursts
	if eventCount > 10 {
		d.log.Info("Flushing large event batch",
			"count", eventCount,
			"burst_count", d.burstCount,
			"window_duration", d.lastFlushTime.Sub(d.burstStart))
	} else {
		d.log.Debug("Flushing debounced events", "count", eventCount)
	}

	// Send all queued events
	sent := 0
	for _, event := range d.queue {
		select {
		case d.output <- event:
			sent++
		default:
			d.log.Warn("Debouncer output channel full, dropping event",
				"path", event.Path,
				"sent", sent,
				"total", eventCount)
		}
	}

	if sent < eventCount {
		d.log.Warn("Some events dropped due to full output channel",
			"sent", sent,
			"dropped", eventCount-sent)
	}

	// Clear the queue and reset burst detection
	d.queue = make(map[string]Event)
	d.timer = nil
	d.burstCount = 0
	d.burstStart = time.Time{}
}

// stop gracefully shuts down the debouncer
func (d *debouncer) stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stopped = true

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	// Flush any remaining events
	for _, event := range d.queue {
		select {
		case d.output <- event:
		default:
			// Best effort - if channel is full, events will be lost
		}
	}

	close(d.output)
}

// DebouncerStats holds statistics about debouncer performance
type DebouncerStats struct {
	TotalEvents       int64           `json:"total_events"`
	TotalFlushes      int64           `json:"total_flushes"`
	BurstDetections   int64           `json:"burst_detections"`
	CurrentQueueSize  int             `json:"current_queue_size"`
	LastFlushTime     time.Time       `json:"last_flush_time"`
	LastFlushDuration time.Duration   `json:"last_flush_duration"`
	Config            DebouncerConfig `json:"config"`
}

// GetStats returns current debouncer statistics
func (d *debouncer) getStats() DebouncerStats {
	d.mu.Lock()
	defer d.mu.Unlock()

	var lastFlushDuration time.Duration
	if !d.lastFlushTime.IsZero() && !d.burstStart.IsZero() {
		lastFlushDuration = d.lastFlushTime.Sub(d.burstStart)
	}

	return DebouncerStats{
		TotalEvents:       d.totalEvents,
		TotalFlushes:      d.totalFlushes,
		BurstDetections:   d.burstDetections,
		CurrentQueueSize:  len(d.queue),
		LastFlushTime:     d.lastFlushTime,
		LastFlushDuration: lastFlushDuration,
		Config:            d.config,
	}
}
