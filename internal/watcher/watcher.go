package watcher

import (
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventType represents the type of filesystem event
type EventType int

const (
	Upsert EventType = iota // Create/Write/Chmod
	Delete
	Rename
)

// Event represents a filesystem event that affects issue indexing
type Event struct {
	ProjectID string
	Path      string // absolute path
	Type      EventType
	Timestamp time.Time
}

// Watcher monitors filesystem changes for issue files
type Watcher struct {
	log      *slog.Logger
	out      chan Event
	fsw      *fsnotify.Watcher
	quit     chan struct{}
	rules    *ignoreRules
	deb      *debouncer
	mu       sync.RWMutex
	projects map[string]string // projectID -> issuesDir
}

// New creates a new filesystem watcher
func New(log *slog.Logger, out chan Event) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Use default debouncer configuration
	debConfig := DefaultDebouncerConfig()

	w := &Watcher{
		log:      log,
		out:      out,
		fsw:      fsw,
		quit:     make(chan struct{}),
		rules:    newIgnoreRules(),
		deb:      newDebouncer(debConfig, log),
		projects: make(map[string]string),
	}

	// Wire debouncer output to our event channel
	go func() {
		for event := range w.deb.output {
			select {
			case w.out <- event:
			case <-w.quit:
				return
			}
		}
	}()

	return w, nil
}

// AddProject adds a project's issues directory to the watch list
func (w *Watcher) AddProject(projectID, issuesDir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.projects[projectID] = issuesDir

	// Watch the main issues directory
	if err := w.fsw.Add(issuesDir); err != nil {
		return err
	}

	// Watch issue type subdirectories (bug, feature, task, epic)
	issueTypes := []string{"bug", "feature", "task", "epic"}
	for _, issueType := range issueTypes {
		typeDir := filepath.Join(issuesDir, issueType)
		// Ignore error if directory doesn't exist yet
		_ = w.fsw.Add(typeDir)
	}

	w.log.Info("Added project to watcher", "project_id", projectID, "issues_dir", issuesDir)
	return nil
}

// GetWatchedPaths returns the list of watched issue directories
func (w *Watcher) GetWatchedPaths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.projects))
	for _, path := range w.projects {
		paths = append(paths, path)
	}
	return paths
}

// Run starts the filesystem watcher event loop
func (w *Watcher) Run() error {
	w.log.Info("Starting filesystem watcher")

	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			w.handleFSEvent(event)

		case err := <-w.fsw.Errors:
			w.log.Error("fsnotify error", "error", err)

		case <-w.quit:
			w.log.Info("Stopping filesystem watcher")
			return nil
		}
	}
}

// Stop gracefully shuts down the watcher
func (w *Watcher) Stop() error {
	close(w.quit)
	w.deb.stop()
	return w.fsw.Close()
}

// handleFSEvent processes a filesystem event
func (w *Watcher) handleFSEvent(event fsnotify.Event) {
	// Ignore if not a markdown file
	if !strings.HasSuffix(strings.ToLower(event.Name), ".md") {
		return
	}

	// Apply ignore rules
	if w.rules.shouldIgnore(event.Name) {
		return
	}

	// Find which project this event belongs to
	projectID := w.findProjectForPath(event.Name)
	if projectID == "" {
		w.log.Debug("No project found for path", "path", event.Name)
		return
	}

	var eventType EventType
	switch {
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = Delete
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = Rename
	default:
		// Create, Write, Chmod all treated as Upsert
		eventType = Upsert
	}

	watcherEvent := Event{
		ProjectID: projectID,
		Path:      event.Name,
		Type:      eventType,
		Timestamp: time.Now(),
	}

	w.deb.push(watcherEvent)
	w.log.Debug("Queued filesystem event",
		"project_id", projectID,
		"path", event.Name,
		"type", eventType,
		"fs_op", event.Op)
}

// findProjectForPath determines which project owns a given file path
func (w *Watcher) findProjectForPath(path string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}

	for projectID, issuesDir := range w.projects {
		absIssuesDir, err := filepath.Abs(issuesDir)
		if err != nil {
			continue
		}

		// Check if path is within this project's issues directory
		if strings.HasPrefix(absPath, absIssuesDir+string(filepath.Separator)) ||
			absPath == absIssuesDir {
			return projectID
		}
	}

	return ""
}

// WatcherStats holds statistics about watcher performance
type WatcherStats struct {
	ProjectCount   int               `json:"project_count"`
	WatchedDirs    map[string]string `json:"watched_dirs"` // projectID -> issuesDir
	DebouncerStats DebouncerStats    `json:"debouncer_stats"`
	LastError      string            `json:"last_error,omitempty"`
	LastErrorTime  *time.Time        `json:"last_error_time,omitempty"`
}

// GetStats returns current watcher statistics
func (w *Watcher) GetStats() WatcherStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Copy projects map for safe access
	watchedDirs := make(map[string]string)
	for projectID, issuesDir := range w.projects {
		watchedDirs[projectID] = issuesDir
	}

	return WatcherStats{
		ProjectCount:   len(w.projects),
		WatchedDirs:    watchedDirs,
		DebouncerStats: w.deb.getStats(),
		// LastError fields would be populated by error tracking in Run()
	}
}
