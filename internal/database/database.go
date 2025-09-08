package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/takl/takl/internal/domain"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn      *sql.DB
	path      string
	projectID string
}

func Open(projectID, dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn:      conn,
		path:      dbPath,
		projectID: projectID,
	}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// UpdateMetadata updates a metadata key-value pair
func (db *DB) UpdateMetadata(key string, value time.Time) error {
	_, err := db.conn.Exec(`
		INSERT OR REPLACE INTO metadata (key, value) 
		VALUES (?, ?)
	`, key, value)
	return err
}

// GetMetadata retrieves a metadata value by key
func (db *DB) GetMetadata(key string) (time.Time, error) {
	var value time.Time
	err := db.conn.QueryRow(`SELECT value FROM metadata WHERE key = ? LIMIT 1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		// Return epoch time for missing keys
		return time.Unix(0, 0), nil
	}
	return value, err
}

// BeginTx starts a database transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.conn.Begin()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS issues (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'open',
		priority TEXT NOT NULL DEFAULT 'medium',
		assignee TEXT,
		labels TEXT, -- JSON array
		created DATETIME NOT NULL,
		updated DATETIME NOT NULL,
		version INTEGER NOT NULL DEFAULT 0,
		file_path TEXT NOT NULL,
		content TEXT,
		project_id TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_issues_type ON issues(type);
	CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
	CREATE INDEX IF NOT EXISTS idx_issues_priority ON issues(priority);
	CREATE INDEX IF NOT EXISTS idx_issues_assignee ON issues(assignee);
	CREATE INDEX IF NOT EXISTS idx_issues_created ON issues(created);
	CREATE INDEX IF NOT EXISTS idx_issues_project ON issues(project_id);

	-- Metadata table for tracking index state
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value DATETIME
	);

	-- Full-text search table
	CREATE VIRTUAL TABLE IF NOT EXISTS issues_fts USING fts5(
		id,
		title,
		content,
		labels,
		content='issues',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS table in sync
	CREATE TRIGGER IF NOT EXISTS issues_fts_insert AFTER INSERT ON issues BEGIN
		INSERT INTO issues_fts(id, title, content, labels) VALUES (NEW.id, NEW.title, NEW.content, NEW.labels);
	END;

	CREATE TRIGGER IF NOT EXISTS issues_fts_update AFTER UPDATE ON issues BEGIN
		UPDATE issues_fts SET title = NEW.title, content = NEW.content, labels = NEW.labels WHERE id = NEW.id;
	END;

	CREATE TRIGGER IF NOT EXISTS issues_fts_delete AFTER DELETE ON issues BEGIN
		DELETE FROM issues_fts WHERE id = OLD.id;
	END;
	`

	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) SaveIssue(issue *domain.Issue) error {
	labelsJSON := "[]"
	if len(issue.Labels) > 0 {
		labelBytes, err := json.Marshal(issue.Labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}
		labelsJSON = string(labelBytes)
	}

	query := `
	INSERT OR REPLACE INTO issues 
	(id, type, title, status, priority, assignee, labels, created, updated, version, file_path, content, project_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query,
		issue.ID,
		issue.Type,
		issue.Title,
		issue.Status,
		issue.Priority,
		issue.Assignee,
		labelsJSON,
		issue.Created,
		issue.Updated,
		issue.Version,
		issue.FilePath,
		issue.Content,
		db.projectID,
	)

	return err
}

func (db *DB) GetIssue(issueID string) (*domain.Issue, error) {
	query := `
	SELECT id, type, title, status, priority, assignee, labels, created, updated, version, file_path, content
	FROM issues WHERE id = ? AND project_id = ?
	`

	row := db.conn.QueryRow(query, issueID, db.projectID)

	var issue domain.Issue
	var labelsJSON string
	var created, updated time.Time

	err := row.Scan(
		&issue.ID,
		&issue.Type,
		&issue.Title,
		&issue.Status,
		&issue.Priority,
		&issue.Assignee,
		&labelsJSON,
		&created,
		&updated,
		&issue.Version,
		&issue.FilePath,
		&issue.Content,
	)

	if err != nil {
		return nil, err
	}

	issue.Created = created
	issue.Updated = updated

	// Parse labels JSON
	if labelsJSON != "[]" && labelsJSON != "" {
		var labels []string
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		issue.Labels = labels
	}

	return &issue, nil
}

func (db *DB) ListIssues(filters ListFilters) ([]*domain.Issue, error) {
	query := "SELECT id, type, title, status, priority, assignee, created, updated, version, file_path FROM issues WHERE project_id = ?"
	args := []interface{}{db.projectID}

	// Add filters
	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}

	if filters.Type != "" {
		query += " AND type = ?"
		args = append(args, filters.Type)
	}

	if filters.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filters.Priority)
	}

	if filters.Assignee != "" {
		query += " AND assignee = ?"
		args = append(args, filters.Assignee)
	}

	if filters.Since != nil {
		query += " AND created >= ?"
		args = append(args, filters.Since)
	}

	if filters.Before != nil {
		query += " AND created < ?"
		args = append(args, filters.Before)
	}

	query += " ORDER BY created DESC"

	// Apply limit and offset
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}
	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []*domain.Issue
	for rows.Next() {
		issue := &domain.Issue{}
		var created, updated time.Time

		err := rows.Scan(
			&issue.ID,
			&issue.Type,
			&issue.Title,
			&issue.Status,
			&issue.Priority,
			&issue.Assignee,
			&created,
			&updated,
			&issue.Version,
			&issue.FilePath,
		)
		if err != nil {
			return nil, err
		}

		issue.Created = created
		issue.Updated = updated
		issues = append(issues, issue)
	}

	return issues, nil
}

func (db *DB) SearchIssues(query string, filters ListFilters) ([]*domain.Issue, error) {
	// Sanitize FTS query
	query = strings.ReplaceAll(query, `"`, `""`)

	searchQuery := `
	SELECT issues.id, issues.type, issues.title, issues.status, issues.priority, issues.assignee, issues.created, issues.updated, issues.file_path
	FROM issues
	JOIN issues_fts ON issues.id = issues_fts.id
	WHERE issues_fts MATCH ? AND issues.project_id = ?
	ORDER BY rank
	`

	rows, err := db.conn.Query(searchQuery, `"`+query+`"`, db.projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []*domain.Issue
	for rows.Next() {
		issue := &domain.Issue{}
		var created, updated time.Time

		err := rows.Scan(
			&issue.ID,
			&issue.Type,
			&issue.Title,
			&issue.Status,
			&issue.Priority,
			&issue.Assignee,
			&created,
			&updated,
			&issue.Version,
			&issue.FilePath,
		)
		if err != nil {
			return nil, err
		}

		issue.Created = created
		issue.Updated = updated
		issues = append(issues, issue)
	}

	return issues, nil
}

func (db *DB) DeleteIssue(issueID string) error {
	query := "DELETE FROM issues WHERE id = ? AND project_id = ?"
	_, err := db.conn.Exec(query, issueID, db.projectID)
	return err
}

func (db *DB) GetIssueCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM issues WHERE project_id = ?", db.projectID).Scan(&count)
	return count, err
}

func (db *DB) GetIssueCountByStatus() (map[string]int, error) {
	query := "SELECT status, COUNT(*) FROM issues WHERE project_id = ? GROUP BY status"
	rows, err := db.conn.Query(query, db.projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, nil
}
