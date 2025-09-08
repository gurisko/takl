package indexer

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/takl/takl/internal/database"
	"github.com/takl/takl/internal/shared"
	"github.com/takl/takl/internal/watcher"
)

// Indexer handles the consumption of filesystem events and updating the database index
type Indexer struct {
	log       *slog.Logger
	databases map[string]*database.DB  // projectID -> database
	dbMutex   map[string]*sync.RWMutex // per-project mutex for database access
}

// New creates a new indexer instance
func New(log *slog.Logger) *Indexer {
	return &Indexer{
		log:       log,
		databases: make(map[string]*database.DB),
		dbMutex:   make(map[string]*sync.RWMutex),
	}
}

// RegisterDatabase registers a database for a specific project
func (ix *Indexer) RegisterDatabase(projectID string, db *database.DB) {
	if ix.dbMutex[projectID] == nil {
		ix.dbMutex[projectID] = &sync.RWMutex{}
	}

	ix.dbMutex[projectID].Lock()
	defer ix.dbMutex[projectID].Unlock()

	ix.databases[projectID] = db
	ix.log.Info("Registered database for project", "project_id", projectID)
}

// UnregisterDatabase removes a database for a specific project
func (ix *Indexer) UnregisterDatabase(projectID string) {
	if mutex, exists := ix.dbMutex[projectID]; exists {
		mutex.Lock()
		defer mutex.Unlock()
		delete(ix.databases, projectID)
		ix.log.Info("Unregistered database for project", "project_id", projectID)
	}
}

// Consume processes a watcher event and updates the appropriate index
func (ix *Indexer) Consume(event watcher.Event) error {
	ix.log.Debug("Processing watcher event",
		"project_id", event.ProjectID,
		"path", event.Path,
		"type", event.Type,
		"timestamp", event.Timestamp)

	// Get database for this project
	db := ix.getDatabase(event.ProjectID)
	if db == nil {
		ix.log.Warn("No database available for project", "project_id", event.ProjectID)
		return fmt.Errorf("no database available for project %s", event.ProjectID)
	}

	switch event.Type {
	case watcher.Upsert:
		return ix.handleUpsert(db, event)
	case watcher.Delete:
		return ix.handleDelete(db, event)
	case watcher.Rename:
		return ix.handleRename(db, event)
	default:
		return fmt.Errorf("unknown event type: %v", event.Type)
	}
}

// getDatabase safely retrieves a database for a project
func (ix *Indexer) getDatabase(projectID string) *database.DB {
	mutex, exists := ix.dbMutex[projectID]
	if !exists {
		return nil
	}

	mutex.RLock()
	defer mutex.RUnlock()
	return ix.databases[projectID]
}

// handleUpsert processes create/write/chmod events by loading and indexing the issue
func (ix *Indexer) handleUpsert(db *database.DB, event watcher.Event) error {
	// Load issue from filesystem
	issue, err := shared.LoadIssueFromFile(event.Path)
	if err != nil {
		ix.log.Warn("Failed to load issue for indexing",
			"path", event.Path,
			"error", err)
		return fmt.Errorf("failed to load issue from %s: %w", event.Path, err)
	}

	// Upsert to database
	if err := db.SaveIssue(issue); err != nil {
		ix.log.Error("Failed to upsert issue to index",
			"issue_id", issue.ID,
			"path", event.Path,
			"error", err)
		return fmt.Errorf("failed to upsert issue %s: %w", issue.ID, err)
	}

	ix.log.Debug("Upserted issue to index",
		"issue_id", issue.ID,
		"title", issue.Title,
		"path", event.Path)
	return nil
}

// handleDelete processes file deletion events by removing from index
func (ix *Indexer) handleDelete(db *database.DB, event watcher.Event) error {
	// Extract issue ID from file path
	issueID := extractIssueIDFromPath(event.Path)
	if issueID == "" {
		ix.log.Warn("Could not extract issue ID from deleted path", "path", event.Path)
		return fmt.Errorf("could not extract issue ID from path %s", event.Path)
	}

	// Delete from database
	if err := db.DeleteIssue(issueID); err != nil {
		ix.log.Error("Failed to delete issue from index",
			"issue_id", issueID,
			"path", event.Path,
			"error", err)
		return fmt.Errorf("failed to delete issue %s: %w", issueID, err)
	}

	ix.log.Debug("Deleted issue from index",
		"issue_id", issueID,
		"path", event.Path)
	return nil
}

// handleRename processes file rename events - try upsert if file exists, otherwise delete
func (ix *Indexer) handleRename(db *database.DB, event watcher.Event) error {
	// Check if file exists at new location
	if fileExists(event.Path) {
		// File exists, treat as upsert
		ix.log.Debug("Rename event: file exists at new location, upserting", "path", event.Path)
		return ix.handleUpsert(db, event)
	} else {
		// File doesn't exist, treat as delete
		ix.log.Debug("Rename event: file doesn't exist, deleting from index", "path", event.Path)
		return ix.handleDelete(db, event)
	}
}

// extractIssueIDFromPath extracts issue ID from a file path like "bug/iss-abc123-title.md"
func extractIssueIDFromPath(path string) string {
	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, ".md") {
		return ""
	}

	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Issue files have format: {id}-{slug}.md
	// Find first hyphen to separate ID from slug
	if idx := strings.Index(name, "-"); idx > 0 {
		return name[:idx]
	}

	return ""
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		return !stat.IsDir()
	}
	return false
}
