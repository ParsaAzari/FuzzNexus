package wordlists

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager handles wordlist lifecycle, database persistence, and file operations.
type Manager struct {
	baseDir     string
	db          *sql.DB
	lockedPaths map[string]int
	sandbox     *SandboxManager
	mu          sync.RWMutex
}

// NewManager initializes the wordlist manager with database backing and sandbox environment.
func NewManager(dataDir string, db *sql.DB) (*Manager, error) {
	wordlistsDir := filepath.Join(dataDir, "wordlists")
	if err := os.MkdirAll(wordlistsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create wordlists directory: %w", err)
	}

	sm, err := NewSandboxManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox manager: %w", err)
	}

	mgr := &Manager{
		baseDir:     wordlistsDir,
		db:          db,
		lockedPaths: make(map[string]int),
		sandbox:     sm,
	}
	if err := mgr.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize wordlist tables: %w", err)
	}
	return mgr, nil
}

func (m *Manager) initSchema() error {
	if m.db == nil {
		return nil
	}
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS wordlists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			line_count INTEGER NOT NULL DEFAULT 0,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			is_linked INTEGER NOT NULL DEFAULT 0,
			category TEXT NOT NULL DEFAULT 'general',
			is_builtin INTEGER NOT NULL DEFAULT 0,
			builtin_id TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		CREATE TABLE IF NOT EXISTS wordlist_categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			color TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL
		);

		INSERT OR IGNORE INTO wordlist_categories (id, name, color, is_default, created_at) VALUES
			('admin', 'Admin & Portals', '#f43f5e', 1, CURRENT_TIMESTAMP),
			('api', 'Modern APIs', '#06b6d4', 1, CURRENT_TIMESTAMP),
			('ai', 'AI & LLMs', '#a855f7', 1, CURRENT_TIMESTAMP),
			('frameworks', 'Frameworks', '#3b82f6', 1, CURRENT_TIMESTAMP),
			('cms', 'CMS Platforms', '#8b5cf6', 1, CURRENT_TIMESTAMP),
			('cloud', 'Cloud & DevOps', '#10b981', 1, CURRENT_TIMESTAMP),
			('auth', 'Auth & SSO', '#ec4899', 1, CURRENT_TIMESTAMP),
			('leaks', 'Leaks & Dumps', '#f59e0b', 1, CURRENT_TIMESTAMP),
			('bypasses', '403 Bypasses', '#6366f1', 1, CURRENT_TIMESTAMP),
			('general', 'General', '#64748b', 1, CURRENT_TIMESTAMP);
	`)
	if err != nil {
		return err
	}

	return m.EnsureBuiltinSeeded()
}

// Sandbox returns the underlying SandboxManager instance.
func (m *Manager) Sandbox() *SandboxManager {
	return m.sandbox
}

// LockWordlist marks a path as actively in-use by a scan session.
func (m *Manager) LockWordlist(path string) {
	clean := filepath.Clean(path)
	m.mu.Lock()
	m.lockedPaths[clean]++
	m.mu.Unlock()
	if m.sandbox != nil {
		m.sandbox.Lock(clean)
	}
}

// UnlockWordlist releases an active scan lock on a wordlist path.
// Note: Ephemeral sandbox files are retained after unlock so the user can re-run
// or adjust scan settings without losing their merged wordlist.
func (m *Manager) UnlockWordlist(path string) {
	clean := filepath.Clean(path)
	m.mu.Lock()
	if count, ok := m.lockedPaths[clean]; ok {
		if count <= 1 {
			delete(m.lockedPaths, clean)
		} else {
			m.lockedPaths[clean] = count - 1
		}
	}
	m.mu.Unlock()
	if m.sandbox != nil {
		m.sandbox.Unlock(clean)
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
		SELECT id, name, path, line_count, size_bytes, is_linked, category, is_builtin, COALESCE(builtin_id, ''), created_at, updated_at
		FROM wordlists
		ORDER BY is_builtin DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query wordlists: %w", err)
	}
	defer rows.Close()

	var list []WordlistDTO
	for rows.Next() {
		var w WordlistDTO
		var isLinkedInt, isBuiltinInt int
		var createdStr, updatedStr string
		err := rows.Scan(
			&w.ID,
			&w.Name,
			&w.Path,
			&w.LineCount,
			&w.SizeBytes,
			&isLinkedInt,
			&w.Category,
			&isBuiltinInt,
			&w.BuiltinID,
			&createdStr,
			&updatedStr,
		)
		if err != nil {
			return nil, err
		}
		if w.Category == "" {
			w.Category = "general"
		}
		w.IsLinked = (isLinkedInt == 1)
		w.IsBuiltin = (isBuiltinInt == 1)
		w.CreatedAt = parseTimeHelper(createdStr)
		w.UpdatedAt = parseTimeHelper(updatedStr)
		list = append(list, w)
	}

	return list, nil
}

// CreateWordlist creates a new empty or initial-content wordlist file and registers it.
func (m *Manager) CreateWordlist(name, initialContent string) (*WordlistDTO, error) {
	return m.CreateWordlistWithCategory(name, initialContent, "general")
}

// CreateWordlistWithCategory creates a new wordlist with a specific category.
func (m *Manager) CreateWordlistWithCategory(name, initialContent, category string) (*WordlistDTO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		name = "untitled.txt"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".txt") {
		name += ".txt"
	}

	category = strings.TrimSpace(category)
	if category == "" {
		category = "general"
	}

	safeFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(name))
	targetPath := filepath.Join(m.baseDir, safeFileName)

	cleanedContent, _, _, _ := CleanStringContent(initialContent)
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
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?)
	`, name, targetPath, lines, size, category, nowStr, nowStr)
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
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetCategories returns all wordlist categories.
func (m *Manager) GetCategories() ([]CategoryDTO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.db.Query(`
		SELECT id, name, color, is_default
		FROM wordlist_categories
		ORDER BY is_default DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []CategoryDTO
	for rows.Next() {
		var c CategoryDTO
		var isDefInt int
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &isDefInt); err != nil {
			return nil, err
		}
		c.IsDefault = (isDefInt == 1)
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// CreateCategory adds a new wordlist category.
func (m *Manager) CreateCategory(id, name, color string) (*CategoryDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name cannot be empty")
	}

	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		var b strings.Builder
		for _, r := range strings.ToLower(name) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteRune('-')
			}
		}
		id = strings.Trim(b.String(), "-")
		if id == "" {
			id = fmt.Sprintf("cat-%d", time.Now().UnixNano()%100000)
		}
	}

	color = strings.TrimSpace(color)
	if color == "" {
		color = "#64748b"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := m.db.Exec(`
		INSERT INTO wordlist_categories (id, name, color, is_default, created_at)
		VALUES (?, ?, ?, 0, ?)
	`, id, name, color, nowStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return &CategoryDTO{
		ID:        id,
		Name:      name,
		Color:     color,
		IsDefault: false,
	}, nil
}

// UpdateCategory updates category name and color.
func (m *Manager) UpdateCategory(id, name, color string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("category name cannot be empty")
	}
	color = strings.TrimSpace(color)
	if color == "" {
		color = "#64748b"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	res, err := m.db.Exec(`
		UPDATE wordlist_categories
		SET name = ?, color = ?
		WHERE id = ?
	`, name, color, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("category not found")
	}
	return nil
}

// DeleteCategory removes a custom category and moves associated wordlists to 'general'.
func (m *Manager) DeleteCategory(id string) error {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "general" {
		return errors.New("the general category cannot be deleted")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Move existing wordlists to general
	_, err := m.db.Exec(`UPDATE wordlists SET category = 'general' WHERE category = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to reassign wordlists: %w", err)
	}

	// Delete category
	_, err = m.db.Exec(`DELETE FROM wordlist_categories WHERE id = ?`, id)
	return err
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

// UpdateWordlistMetadata updates both the display name and category of a wordlist.
func (m *Manager) UpdateWordlistMetadata(id int64, name string, category string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	category = strings.TrimSpace(category)
	if category == "" {
		category = "general"
	}
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := m.db.Exec(`
		UPDATE wordlists
		SET name = ?, category = ?, updated_at = ?
		WHERE id = ?
	`, name, category, nowStr, id)
	return err
}

// MergeWordlists combines multiple wordlists into a single deduplicated new wordlist.
func (m *Manager) MergeWordlists(ids []int64, targetName string, category string) (*WordlistDTO, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one wordlist ID is required to merge")
	}

	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		targetName = "merged_wordlist.txt"
	}
	if !strings.HasSuffix(strings.ToLower(targetName), ".txt") {
		targetName += ".txt"
	}

	category = strings.TrimSpace(category)
	if category == "" {
		category = "general"
	}

	// Fetch file paths for all requested wordlist IDs
	m.mu.RLock()
	var paths []string
	for _, id := range ids {
		var path string
		err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
		if err == nil && path != "" {
			paths = append(paths, path)
		}
	}
	m.mu.RUnlock()

	if len(paths) == 0 {
		return nil, errors.New("none of the specified wordlists could be found")
	}

	safeFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(targetName))
	targetPath := filepath.Join(m.baseDir, safeFileName)

	destFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create merged wordlist file: %w", err)
	}

	writer := bufio.NewWriter(destFile)
	seen := make(map[string]struct{})

	for _, srcPath := range paths {
		f, err := os.Open(srcPath)
		if err != nil {
			continue // Skip unreadable source file
		}

		scanner := bufio.NewScanner(f)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r\n")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, exists := seen[line]; !exists {
				seen[line] = struct{}{}
				if _, err := writer.WriteString(line + "\n"); err != nil {
					f.Close()
					destFile.Close()
					_ = os.Remove(targetPath)
					return nil, fmt.Errorf("failed writing to merged wordlist: %w", err)
				}
			}
		}
		_ = f.Close()
	}

	if err := writer.Flush(); err != nil {
		destFile.Close()
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed flushing merged wordlist: %w", err)
	}
	if err := destFile.Sync(); err != nil {
		destFile.Close()
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed syncing merged wordlist: %w", err)
	}
	_ = destFile.Close()

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
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?)
	`, targetName, targetPath, lines, size, category, nowStr, nowStr)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed to register merged wordlist record: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WordlistDTO{
		ID:        id,
		Name:      targetName,
		Path:      targetPath,
		LineCount: lines,
		SizeBytes: size,
		IsLinked:  false,
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// MergeWordlistsToSandbox combines multiple wordlists with high-precision sanitization into an isolated sandbox file or persistent library wordlist.
func (m *Manager) MergeWordlistsToSandbox(ids []int64, opts SanitizeOptions, isEphemeral bool, targetName string, category string) (*WordlistDTO, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one wordlist ID is required to merge")
	}

	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		targetName = "merged_wordlist.txt"
	}
	if !strings.HasSuffix(strings.ToLower(targetName), ".txt") {
		targetName += ".txt"
	}

	category = strings.TrimSpace(category)
	if category == "" {
		category = "general"
	}

	// Fetch file paths for all requested wordlist IDs
	m.mu.RLock()
	var paths []string
	for _, id := range ids {
		var path string
		err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
		if err == nil && path != "" {
			paths = append(paths, path)
		}
	}
	m.mu.RUnlock()

	if len(paths) == 0 {
		return nil, errors.New("none of the specified wordlists could be found")
	}

	// Open readers for all available source paths
	var readers []io.Reader
	var openFiles []*os.File
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		openFiles = append(openFiles, f)
		readers = append(readers, f)
	}
	defer func() {
		for _, f := range openFiles {
			_ = f.Close()
		}
	}()

	if len(readers) == 0 {
		return nil, errors.New("failed to open any of the source wordlist files")
	}

	now := time.Now().UTC()

	if isEphemeral {
		// Create ephemeral sandbox file
		destFile, targetPath, err := m.sandbox.CreateSandboxFile("sb_merged")
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox file: %w", err)
		}

		report, err := ProcessAndDeduplicate(readers, destFile, opts)
		if err != nil {
			_ = destFile.Close()
			_ = os.Remove(targetPath)
			return nil, fmt.Errorf("failed processing wordlists into sandbox: %w", err)
		}

		if err := destFile.Sync(); err != nil {
			_ = destFile.Close()
			_ = os.Remove(targetPath)
			return nil, fmt.Errorf("failed syncing sandbox file: %w", err)
		}
		_ = destFile.Close()

		m.sandbox.RegisterEphemeral(targetPath)

		return &WordlistDTO{
			ID:        0, // 0 indicates an ephemeral sandbox wordlist
			Name:      targetName,
			Path:      targetPath,
			LineCount: report.DeduplicatedLines,
			SizeBytes: report.SizeBytes,
			IsLinked:  false,
			Category:  "sandbox",
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	// Persistent target in wordlists directory
	safeFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(targetName))
	targetPath := filepath.Join(m.baseDir, safeFileName)

	destFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed creating persistent wordlist file: %w", err)
	}

	report, err := ProcessAndDeduplicate(readers, destFile, opts)
	if err != nil {
		_ = destFile.Close()
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed processing persistent wordlists: %w", err)
	}

	if err := destFile.Sync(); err != nil {
		_ = destFile.Close()
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed syncing persistent wordlist: %w", err)
	}
	_ = destFile.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	nowStr := now.Format(time.RFC3339Nano)
	res, err := m.db.Exec(`
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?)
	`, targetName, targetPath, report.DeduplicatedLines, report.SizeBytes, category, nowStr, nowStr)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed registering persistent wordlist: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WordlistDTO{
		ID:        id,
		Name:      targetName,
		Path:      targetPath,
		LineCount: report.DeduplicatedLines,
		SizeBytes: report.SizeBytes,
		IsLinked:  false,
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// PreviewMergeStats computes a dry-run analysis of combining multiple wordlists without writing output.
func (m *Manager) PreviewMergeStats(ids []int64, opts SanitizeOptions) (*MergeReportDTO, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one wordlist ID is required to preview merge")
	}

	m.mu.RLock()
	var paths []string
	for _, id := range ids {
		var path string
		err := m.db.QueryRow("SELECT path FROM wordlists WHERE id = ?", id).Scan(&path)
		if err == nil && path != "" {
			paths = append(paths, path)
		}
	}
	m.mu.RUnlock()

	if len(paths) == 0 {
		return nil, errors.New("none of the specified wordlists could be found")
	}

	var readers []io.Reader
	var openFiles []*os.File
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		openFiles = append(openFiles, f)
		readers = append(readers, f)
	}
	defer func() {
		for _, f := range openFiles {
			_ = f.Close()
		}
	}()

	return PreviewDeduplication(readers, opts)
}

// CleanupSandboxWordlist removes an unlocked sandbox wordlist file.
func (m *Manager) CleanupSandboxWordlist(path string) error {
	if m.sandbox == nil {
		return nil
	}
	return m.sandbox.CleanupPath(path)
}

// CleanupAllSandboxes deletes all unlocked ephemeral files.
func (m *Manager) CleanupAllSandboxes() error {
	if m.sandbox == nil {
		return nil
	}
	return m.sandbox.CleanupAllUnlockedEphemeral()
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

	cleanedContent, _, _, _ := CleanStringContent(content)
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
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, 'general', ?, ?)
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
		Category:  "general",
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
		INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, 'general', ?, ?)
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
		Category:  "general",
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

// CreateSanitizedSandboxWordlist creates a non-destructive ephemeral sandbox copy of sourcePath
// with traversal, bypass, and WAF-triggering patterns removed so the original wordlist remains untouched.
func (m *Manager) CreateSanitizedSandboxWordlist(sourcePath string) (*WordlistDTO, error) {
	cleanPath, err := ValidatePath(sourcePath)
	if err != nil {
		return nil, err
	}

	srcFile, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source wordlist: %w", err)
	}
	defer srcFile.Close()

	destFile, targetPath, err := m.sandbox.CreateSandboxFile("sb_sanitized")
	if err != nil {
		return nil, fmt.Errorf("failed to create sanitized sandbox file: %w", err)
	}

	opts := DefaultSanitizeOptions()
	opts.StripTraversalPayloads = true

	report, err := ProcessAndDeduplicate([]io.Reader{srcFile}, destFile, opts)
	if err != nil {
		_ = destFile.Close()
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed sanitizing wordlist: %w", err)
	}

	if report.DeduplicatedLines == 0 {
		_ = destFile.Close()
		_ = os.Remove(targetPath)
		return nil, errors.New("all lines in this wordlist contain traversal or WAF-triggering patterns; please select a standard directory or API wordlist instead")
	}

	if err := destFile.Sync(); err != nil {
		_ = destFile.Close()
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed syncing sanitized sandbox file: %w", err)
	}
	_ = destFile.Close()

	m.sandbox.RegisterEphemeral(targetPath)
	now := time.Now().UTC()

	return &WordlistDTO{
		ID:        0,
		Name:      fmt.Sprintf("Sanitized (%s)", filepath.Base(cleanPath)),
		Path:      targetPath,
		LineCount: report.DeduplicatedLines,
		SizeBytes: report.SizeBytes,
		IsLinked:  false,
		Category:  "sandbox",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
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
