package mcp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store handles the persistence of Tool resources using SQLite.
type Store struct {
	db *sql.DB
}

// NewStore initializes a new SQLite store at the specified path.
func NewStore(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS tools (
		name TEXT,
		version TEXT,
		module TEXT,
		data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (name, version)
	);
	CREATE INDEX IF NOT EXISTS idx_tools_module ON tools(module);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}

// Save stores or updates a tool in the database.
func (s *Store) Save(t *Tool) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal tool: %w", err)
	}

	query := `
	INSERT INTO tools (name, version, module, data, updated_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(name, version) DO UPDATE SET
		module = excluded.module,
		data = excluded.data,
		updated_at = CURRENT_TIMESTAMP;
	`
	_, err = s.db.Exec(query, t.Metadata.Name, t.Metadata.Version, t.Metadata.Module, string(data))
	if err != nil {
		return fmt.Errorf("save tool: %w", err)
	}
	return nil
}

// List returns all tools stored in the database.
func (s *Store) List() ([]*Tool, error) {
	query := `SELECT data FROM tools ORDER BY name, version DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query tools: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tools []*Tool
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scan tool: %w", err)
		}

		var t Tool
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			return nil, fmt.Errorf("unmarshal tool: %w", err)
		}
		tools = append(tools, &t)
	}

	return tools, nil
}

// GetStateHash returns a string representing the current state of the tools table.
// This is used to short-circuit reconciliation if no changes have occurred.
func (s *Store) GetStateHash() (string, error) {
	query := `SELECT COUNT(*), COALESCE(MAX(updated_at), '') FROM tools`
	var count int
	var maxUpdated string
	err := s.db.QueryRow(query).Scan(&count, &maxUpdated)
	if err != nil {
		return "", fmt.Errorf("get state hash: %w", err)
	}
	return fmt.Sprintf("%d-%s", count, maxUpdated), nil
}

// Get retrieves a specific version of a tool.
func (s *Store) Get(name, version string) (*Tool, error) {
	query := `SELECT data FROM tools WHERE name = ? AND version = ?`
	var data string
	err := s.db.QueryRow(query, name, version).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tool %s@%s not found", name, version)
	}
	if err != nil {
		return nil, fmt.Errorf("get tool: %w", err)
	}

	var t Tool
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, fmt.Errorf("unmarshal tool: %w", err)
	}
	return &t, nil
}

// Delete removes a tool version from the database.
func (s *Store) Delete(name, version string) error {
	query := `DELETE FROM tools WHERE name = ? AND version = ?`
	_, err := s.db.Exec(query, name, version)
	if err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
