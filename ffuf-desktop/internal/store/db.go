package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ffuf-desktop/internal/output"

	_ "modernc.org/sqlite"
)

type SessionDTO struct {
	ID                 int64      `json:"id"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	State              string     `json:"state"`
	ConfigJSON         string     `json:"config_json"`
	ConfigRedactedJSON string     `json:"config_redacted_json"`
	TotalRequests      int64      `json:"total_requests"`
	LastError          string     `json:"last_error,omitempty"`
}

type Store struct {
	dbPath string
	writer *sql.DB
	reader *sql.DB
	mu     sync.Mutex
}

// NewStore initializes the SQLite database with WAL mode and runs pending migrations.
func NewStore(dataDir string) (*Store, error) {
	// Create directory with secure 0700 permissions
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "ffuf.db")

	// Single writer connection pool
	dsnWriter := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	writer, err := sql.Open("sqlite", dsnWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to open writer connection: %w", err)
	}
	writer.SetMaxOpenConns(1)

	// Concurrent reader connection pool
	dsnReader := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	reader, err := sql.Open("sqlite", dsnReader)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to open reader connection: %w", err)
	}
	reader.SetMaxOpenConns(4)

	s := &Store{
		dbPath: dbPath,
		writer: writer,
		reader: reader,
	}

	if err := s.runMigrations(); err != nil {
		s.Close()
		return nil, fmt.Errorf("migration failure: %w", err)
	}

	return s, nil
}

func (s *Store) runMigrations() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.writer.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	var version int
	row := s.writer.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&version); err != nil {
		return err
	}

	if version < 1 {
		tx, err := s.writer.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		schema := `
		CREATE TABLE sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			started_at TIMESTAMP NOT NULL,
			finished_at TIMESTAMP,
			state TEXT NOT NULL,
			config_json TEXT NOT NULL,
			config_redacted_json TEXT NOT NULL,
			total_requests INTEGER DEFAULT 0,
			last_error TEXT
		);

		CREATE TABLE results (
			session_id INTEGER NOT NULL,
			id INTEGER NOT NULL,
			position INTEGER NOT NULL,
			inputs_json TEXT NOT NULL,
			status INTEGER NOT NULL,
			length INTEGER NOT NULL,
			words INTEGER NOT NULL,
			lines INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			url TEXT NOT NULL,
			redirect_location TEXT,
			content_type TEXT,
			hash TEXT,
			host TEXT,
			scraper_data_json TEXT,
			cancelled INTEGER DEFAULT 0,
			PRIMARY KEY (session_id, id),
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);

		CREATE INDEX idx_results_status ON results(session_id, status);
		CREATE INDEX idx_results_length ON results(session_id, length);
		CREATE INDEX idx_results_words  ON results(session_id, words);

		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		INSERT INTO schema_migrations (version, applied_at) VALUES (1, CURRENT_TIMESTAMP);
		`
		if _, err := tx.Exec(schema); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func parseDBTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	return time.Parse(time.DateTime, s)
}

// CreateSession registers a new scan session in SQLite.
func (s *Store) CreateSession(configJSON, configRedactedJSON string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.writer.Exec(`
		INSERT INTO sessions (started_at, state, config_json, config_redacted_json)
		VALUES (?, 'Running', ?, ?)
	`, nowStr, configJSON, configRedactedJSON)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateSessionState updates the status and completion time of a session.
func (s *Store) UpdateSessionState(sessionID int64, state string, lastError string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var nowStr *string
	if state == "Completed" || state == "Failed" || state == "Stopped" {
		tStr := time.Now().UTC().Format(time.RFC3339Nano)
		nowStr = &tStr
	}

	_, err := s.writer.Exec(`
		UPDATE sessions
		SET state = ?, finished_at = ?, last_error = ?
		WHERE id = ?
	`, state, nowStr, lastError, sessionID)
	return err
}

// InsertResults commits a batch of ResultDTOs inside an isolated SQLite transaction.
func (s *Store) InsertResults(sessionID int64, batch []output.ResultDTO) error {
	if len(batch) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.writer.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO results (
			session_id, id, position, inputs_json, status, length, words, lines,
			duration_ms, url, redirect_location, content_type, hash, host,
			scraper_data_json, cancelled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range batch {
		inputsJSON, _ := json.Marshal(item.Inputs)
		scraperJSON, _ := json.Marshal(item.ScraperData)
		cancelledInt := 0
		if item.Cancelled {
			cancelledInt = 1
		}

		_, err := stmt.Exec(
			sessionID,
			item.ID,
			item.Position,
			string(inputsJSON),
			item.Status,
			item.Length,
			item.Words,
			item.Lines,
			item.DurationMs,
			item.URL,
			item.RedirectLocation,
			item.ContentType,
			item.Hash,
			item.Host,
			string(scraperJSON),
			cancelledInt,
		)
		if err != nil {
			return err
		}
	}

	_ = stmt.Close()

	// Update session total counter
	_, err = tx.Exec(`UPDATE sessions SET total_requests = total_requests + ? WHERE id = ?`, len(batch), sessionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetResultsSince retrieves results for a session after lastID, up to limit.
// Clamps limit between 1 and 5000 (default 1000) and lastID to >= 0.
func (s *Store) GetResultsSince(sessionID int64, lastID int64, limit int) ([]output.ResultDTO, bool, error) {
	if lastID < 0 {
		lastID = 0
	}
	if limit <= 0 {
		limit = 1000
	} else if limit > 5000 {
		limit = 5000
	}

	// Query limit+1 to determine if there are more rows available (hasMore)
	queryLimit := limit + 1
	rows, err := s.reader.Query(`
		SELECT id, position, inputs_json, status, length, words, lines,
		       duration_ms, url, redirect_location, content_type, hash, host,
		       scraper_data_json, cancelled
		FROM results
		WHERE session_id = ? AND id > ?
		ORDER BY id ASC
		LIMIT ?
	`, sessionID, lastID, queryLimit)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var results []output.ResultDTO
	for rows.Next() {
		var item output.ResultDTO
		var inputsJSON, scraperJSON string
		var redirectLoc, contentType, hash, host sql.NullString
		var cancelledInt int

		err := rows.Scan(
			&item.ID,
			&item.Position,
			&inputsJSON,
			&item.Status,
			&item.Length,
			&item.Words,
			&item.Lines,
			&item.DurationMs,
			&item.URL,
			&redirectLoc,
			&contentType,
			&hash,
			&host,
			&scraperJSON,
			&cancelledInt,
		)
		if err != nil {
			return nil, false, err
		}

		if inputsJSON != "" {
			_ = json.Unmarshal([]byte(inputsJSON), &item.Inputs)
		}
		if scraperJSON != "" {
			_ = json.Unmarshal([]byte(scraperJSON), &item.ScraperData)
		}
		item.RedirectLocation = redirectLoc.String
		item.ContentType = contentType.String
		item.Hash = hash.String
		item.Host = host.String
		item.Cancelled = (cancelledInt == 1)

		results = append(results, item)
	}

	hasMore := false
	if len(results) > limit {
		hasMore = true
		results = results[:limit]
	}

	return results, hasMore, nil
}

// GetSession reads session metadata by ID.
func (s *Store) GetSession(sessionID int64) (*SessionDTO, error) {
	row := s.reader.QueryRow(`
		SELECT id, started_at, finished_at, state, config_json, config_redacted_json, total_requests, last_error
		FROM sessions
		WHERE id = ?
	`, sessionID)

	var sess SessionDTO
	var startedAtStr string
	var finishedAtStr sql.NullString
	var lastErr sql.NullString
	err := row.Scan(
		&sess.ID,
		&startedAtStr,
		&finishedAtStr,
		&sess.State,
		&sess.ConfigJSON,
		&sess.ConfigRedactedJSON,
		&sess.TotalRequests,
		&lastErr,
	)
	if err != nil {
		return nil, err
	}
	if t, err := parseDBTime(startedAtStr); err == nil {
		sess.StartedAt = t
	}
	if finishedAtStr.Valid && finishedAtStr.String != "" {
		if t, err := parseDBTime(finishedAtStr.String); err == nil {
			sess.FinishedAt = &t
		}
	}
	sess.LastError = lastErr.String
	return &sess, nil
}

// GetSessions lists all scan sessions ordered by id DESC.
func (s *Store) GetSessions(limit int) ([]SessionDTO, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	rows, err := s.reader.Query(`
		SELECT id, started_at, finished_at, state, config_json, config_redacted_json, total_requests, last_error
		FROM sessions
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionDTO
	for rows.Next() {
		var sess SessionDTO
		var startedAtStr string
		var finishedAtStr sql.NullString
		var lastErr sql.NullString
		if err := rows.Scan(
			&sess.ID,
			&startedAtStr,
			&finishedAtStr,
			&sess.State,
			&sess.ConfigJSON,
			&sess.ConfigRedactedJSON,
			&sess.TotalRequests,
			&lastErr,
		); err != nil {
			return nil, err
		}
		if t, err := parseDBTime(startedAtStr); err == nil {
			sess.StartedAt = t
		}
		if finishedAtStr.Valid && finishedAtStr.String != "" {
			if t, err := parseDBTime(finishedAtStr.String); err == nil {
				sess.FinishedAt = &t
			}
		}
		sess.LastError = lastErr.String
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

// CheckpointWAL forces a WAL checkpoint into the primary database file.
func (s *Store) CheckpointWAL() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkpointWALLocked()
}

func (s *Store) checkpointWALLocked() error {
	_, err := s.writer.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	return err
}

// Close gracefully checkpoints the WAL and closes all reader and writer connections.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.checkpointWALLocked()
	rErr := s.reader.Close()
	wErr := s.writer.Close()
	if wErr != nil {
		return wErr
	}
	return rErr
}
