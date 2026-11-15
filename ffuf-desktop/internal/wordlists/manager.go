package wordlists

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles wordlist lifecycle, database persistence, and file operations.
type Manager struct {
	baseDir     string
	db          *sql.DB
	lockedPaths map[string]int
	mu          sync.RWMutex
}

// NewManager initializes the wordlist manager with database backing.
func NewManager(dataDir string, db *sql.DB) (*Manager, error) {
	wordlistsDir := filepath.Join(dataDir, "wordlists")
	if err := os.MkdirAll(wordlistsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create wordlists directory: %w", err)
	}

	return &Manager{
		baseDir:     wordlistsDir,
		db:          db,
		lockedPaths: make(map[string]int),
	}, nil
}

// LockWordlist marks a path as actively in-use by a scan session.
func (m *Manager) LockWordlist(path string) {
	clean := filepath.Clean(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lockedPaths[clean]++
}

// UnlockWordlist releases an active scan lock on a wordlist path.
func (m *Manager) UnlockWordlist(path string) {
	clean := filepath.Clean(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if count, ok := m.lockedPaths[clean]; ok {
		if count <= 1 {
			delete(m.lockedPaths, clean)
		} else {
			m.lockedPaths[clean] = count - 1
		}
	}
}

// IsLocked checks whether a wordlist path is currently in use.
func (m *Manager) IsLocked(path string) bool {
	clean := filepath.Clean(path)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lockedPaths[clean] > 0
}

// ListWordlists returns all registered wordlists sorted by name.
func (m *Manager) ListWordlists() ([]WordlistDTO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.db.Query(`
		SELECT id, name, path, line_count, size_bytes, is_linked, created_at, updated_at
		FROM wordlists
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query wordlists: %w", err)
	}
	defer rows.Close()

	var list []WordlistDTO
	for rows.Next() {
		var w WordlistDTO
		var isLinkedInt int
		var createdStr, updatedStr string
		err := rows.Scan(
			&w.ID,
			&w.Name,
			&w.Path,
			&w.LineCount,
			&w.SizeBytes,
			&isLinkedInt,
			&createdStr,
			&updatedStr,
		)
		if err != nil {
			return nil, err
		}
		w.IsLinked = (isLinkedInt == 1)
		w.CreatedAt = parseTimeHelper(createdStr)
		w.UpdatedAt = parseTimeHelper(updatedStr)
		list = append(list, w)
	}

	return list, nil
}

// CreateWordlist creates a new empty or initial-content wordlist file and registers it.
func (m *Manager) CreateWordlist(name, initialContent string) (*WordlistDTO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		name = "untitled.txt"
	}

	safeFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(name))
	targetPath := filepath.Join(m.baseDir, safeFileName)

	cleanedContent, _, _ := CleanStringContent(initialContent)
	contentBytes := []byte(cleanedContent)
	if err := AtomicWriteFile(targetPath, contentBytes); err != nil {
		return nil, err
	}

	lines, size, err := CountLines(targetPath)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, err
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	res, err := m.db.Exec(`
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)
	`, name, targetPath, lines, size, nowStr, nowStr)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed to insert wordlist record: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WordlistDTO{
		ID:        id,
		Name:      name,
		Path:      targetPath,
		LineCount: lines,
		SizeBytes: size,
		IsLinked:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateWordlist renames a wordlist metadata record.
func (m *Manager) UpdateWordlist(id int64, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := m.db.Exec(`
		UPDATE wordlists
		SET name = ?, updated_at = ?
		WHERE id = ?
	`, name, nowStr, id)
	return err
}

// DeleteWordlist removes a wordlist. If not linked, deletes the underlying file.
func (m *Manager) DeleteWordlist(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var path string
	var isLinkedInt int
	err := m.db.QueryRow("SELECT path, is_linked FROM wordlists WHERE id = ?", id).Scan(&path, &isLinkedInt)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrFileNotFound
		}
		return err
	}

	// Check if active scan lock is held
	clean := filepath.Clean(path)
	if m.lockedPaths[clean] > 0 {
		return ErrWordlistLocked
	}

	_, err = m.db.Exec("DELETE FROM wordlists WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Delete file from disk only if it is managed internally (not linked)
	if isLinkedInt == 0 {
		_ = os.Remove(path)
	}

	return nil
}

// DeleteWordlistsBatch deletes multiple wordlists by their IDs.
// Returns the count of successfully deleted wordlists and any error encountered.
func (m *Manager) DeleteWordlistsBatch(ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// Pre-check if any target wordlist is currently locked by an active scan
	for _, id := range ids {
		var path string
		err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
		if err != nil {
			continue
		}
		clean := filepath.Clean(path)
		if m.lockedPaths[clean] > 0 {
			return 0, fmt.Errorf("%w: wordlist %s (id %d) is in use by an active scan", ErrWordlistLocked, filepath.Base(path), id)
		}
	}

	deletedCount := 0
	for _, id := range ids {
		var path string
		var isLinkedInt int
		err := m.db.QueryRow("SELECT path, is_linked FROM wordlists WHERE id = ?", id).Scan(&path, &isLinkedInt)
		if err != nil {
			continue
		}

		_, err = m.db.Exec("DELETE FROM wordlists WHERE id = ?", id)
		if err != nil {
			return deletedCount, err
		}

		if isLinkedInt == 0 {
			_ = os.Remove(path)
		}
		deletedCount++
	}

	return deletedCount, nil
}

// ReadWordlistPage reads a window of lines from the wordlist for virtual rendering.
func (m *Manager) ReadWordlistPage(id int64, offset int, limit int) (WordlistPageDTO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var path string
	var totalLines int64
	err := m.db.QueryRow("SELECT path, line_count FROM wordlists WHERE id = ?", id).Scan(&path, &totalLines)
	if err != nil {
		if err == sql.ErrNoRows {
			return WordlistPageDTO{}, ErrFileNotFound
		}
		return WordlistPageDTO{}, err
	}

	lines, count, err := ReadLinesPage(path, offset, limit)
	if err != nil {
		return WordlistPageDTO{}, err
	}

	return WordlistPageDTO{
		Lines:      lines,
		TotalLines: count,
		Offset:     offset,
		Limit:      limit,
	}, nil
}

// SaveWordlistContent replaces the entire content of a wordlist file atomically.
func (m *Manager) SaveWordlistContent(id int64, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var path string
	err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrFileNotFound
		}
		return err
	}

	clean := filepath.Clean(path)
	if m.lockedPaths[clean] > 0 {
		return ErrWordlistLocked
	}

	cleanedContent, _, _ := CleanStringContent(content)
	contentBytes := []byte(cleanedContent)
	if err := AtomicWriteFile(path, contentBytes); err != nil {
		return err
	}

	lines, size, err := CountLines(path)
	if err != nil {
		return err
	}

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = m.db.Exec(`
		UPDATE wordlists
		SET line_count = ?, size_bytes = ?, updated_at = ?
		WHERE id = ?
	`, lines, size, nowStr, id)

	return err
}

// ImportWordlistFromPath copies an external wordlist file into the app storage.
func (m *Manager) ImportWordlistFromPath(sourcePath string) (*WordlistDTO, error) {
	realSource, err := ValidatePath(sourcePath)
	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(realSource)
	targetPath := filepath.Join(m.baseDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileName))

	srcFile, err := os.Open(realSource)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(destFile, srcFile); err != nil {
		destFile.Close()
		_ = os.Remove(targetPath)
		return nil, err
	}
	_ = destFile.Sync()
	destFile.Close()

	// Automatically sanitize and clean imported wordlists (removes blank lines, \r, and trims whitespace)
	_, _ = CleanWordlistFile(targetPath)

	lines, size, err := CountLines(targetPath)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)
	res, err := m.db.Exec(`
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)
	`, fileName, targetPath, lines, size, nowStr, nowStr)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WordlistDTO{
		ID:        id,
		Name:      fileName,
		Path:      targetPath,
		LineCount: lines,
		SizeBytes: size,
		IsLinked:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// LinkWordlistPath links an external wordlist file directly in place without copying.
func (m *Manager) LinkWordlistPath(sourcePath string) (*WordlistDTO, error) {
	realSource, err := ValidatePath(sourcePath)
	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(realSource)
	lines, size, err := CountLines(realSource)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)
	res, err := m.db.Exec(`
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			line_count = excluded.line_count,
			size_bytes = excluded.size_bytes,
			updated_at = excluded.updated_at
	`, fileName, realSource, lines, size, nowStr, nowStr)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WordlistDTO{
		ID:        id,
		Name:      fileName,
		Path:      realSource,
		LineCount: lines,
		SizeBytes: size,
		IsLinked:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// RelocateWordlist updates the target file path of a wordlist record.
func (m *Manager) RelocateWordlist(id int64, newPath string) error {
	realPath, err := ValidatePath(newPath)
	if err != nil {
		return err
	}

	lines, size, err := CountLines(realPath)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = m.db.Exec(`
		UPDATE wordlists
		SET path = ?, line_count = ?, size_bytes = ?, updated_at = ?
		WHERE id = ?
	`, realPath, lines, size, nowStr, id)
	return err
}

// GetWordlistStats calculates detailed metrics for the editor.
func (m *Manager) GetWordlistStats(id int64) (*WordlistStatsDTO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var path string
	err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	return ComputeStats(path)
}

// ExportWordlist copies a wordlist file to a user-chosen destination.
func (m *Manager) ExportWordlist(id int64, targetPath string) error {
	m.mu.RLock()
	var srcPath string
	err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&srcPath)
	m.mu.RUnlock()

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrFileNotFound
		}
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// VerifyWordlist analyzes the hygiene of a wordlist file for empty lines and formatting issues.
func (m *Manager) VerifyWordlist(id int64) (*WordlistHealthDTO, error) {
	m.mu.RLock()
	var path string
	err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
	m.mu.RUnlock()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	health, err := VerifyWordlistFile(path)
	if err != nil {
		return nil, err
	}
	health.ID = id
	return health, nil
}

// CleanWordlist strips blank/empty lines and trims whitespace from the wordlist file atomically,
// then updates the database records.
func (m *Manager) CleanWordlist(id int64) (*CleanResultDTO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var path string
	err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	clean := filepath.Clean(path)
	if m.lockedPaths[clean] > 0 {
		return nil, ErrWordlistLocked
	}

	res, err := CleanWordlistFile(path)
	if err != nil {
		return nil, err
	}
	res.ID = id

	lines, size, err := CountLines(path)
	if err != nil {
		return nil, err
	}

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = m.db.Exec(`
		UPDATE wordlists
		SET line_count = ?, size_bytes = ?, updated_at = ?
		WHERE id = ?
	`, lines, size, nowStr, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func parseTimeHelper(s string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	return time.Now().UTC()
}
