package wordlists

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fuzznexus-desktop/internal/wordlists/data"
)

// PackManifest represents metadata and content of an embedded pack.
type PackManifest struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Color       string   `json:"color"`
	Count       int      `json:"count"`
	Keywords    []string `json:"keywords"`
}

var builtinPackIDs = []string{
	"admin_portals",
	"api_modern",
	"ai_llm",
	"frameworks_backend",
	"cms_platforms",
	"cloud_devops",
	"auth_sso",
	"leaks_artifacts",
	"bypasses_heuristics",
	"params_fuzzing",
	"extensions_backups",
}

// loadEmbeddedPack decodes and uncompresses an embedded pack by its ID.
func loadEmbeddedPack(packID string) (*PackManifest, error) {
	filename := fmt.Sprintf("packs/%s.json.gz", packID)
	f, err := data.BuiltinPacksFS.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed opening embedded pack %s: %w", filename, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed creating gzip reader for %s: %w", filename, err)
	}
	defer gzr.Close()

	var pack PackManifest
	if err := json.NewDecoder(gzr).Decode(&pack); err != nil {
		return nil, fmt.Errorf("failed decoding pack JSON %s: %w", filename, err)
	}

	return &pack, nil
}

// EnsureBuiltinSeeded verifies and seeds all 8 curated wordlist packs into SQLite and disk.
func (m *Manager) EnsureBuiltinSeeded() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db == nil {
		return nil
	}

	if err := os.MkdirAll(m.baseDir, 0700); err != nil {
		return fmt.Errorf("failed creating wordlists base directory: %w", err)
	}

	// Migrate any legacy .ffuf-desktop paths to current .fuzznexus path in database
	_, _ = m.db.Exec(`
		UPDATE wordlists
		SET path = REPLACE(path, '/.fuzznexus-desktop/', '/.fuzznexus/')
		WHERE path LIKE '%/.fuzznexus-desktop/%'
	`)

	for _, packID := range builtinPackIDs {
		targetFileName := fmt.Sprintf("builtin_%s.txt", packID)
		expectedPath := filepath.Join(m.baseDir, targetFileName)

		var existingID int64
		var existingPath string
		var existingLines int64
		err := m.db.QueryRow("SELECT id, path, line_count FROM wordlists WHERE builtin_id = ?", packID).Scan(&existingID, &existingPath, &existingLines)

		// Check if file exists on disk and has content
		fileExists := false
		if err == nil && existingID > 0 {
			if fi, statErr := os.Stat(existingPath); statErr == nil && fi.Size() > 0 {
				fileExists = true
			} else if fi, statErr := os.Stat(expectedPath); statErr == nil && fi.Size() > 0 {
				// File exists at expectedPath in m.baseDir, heal database path!
				fileExists = true
				_, _ = m.db.Exec("UPDATE wordlists SET path = ?, size_bytes = ? WHERE id = ?", expectedPath, fi.Size(), existingID)
			}
		}

		pack, err := loadEmbeddedPack(packID)
		if err != nil {
			return err
		}

		// Skip rewriting if already present and up-to-date (allow upgrades when embedded pack grows)
		if fileExists && existingLines >= int64(len(pack.Keywords)) {
			continue
		}

		// Write keywords to file
		file, err := os.OpenFile(expectedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("failed creating built-in wordlist file %s: %w", expectedPath, err)
		}

		bw := bufio.NewWriter(file)
		var totalLines int64 = 0
		for _, kw := range pack.Keywords {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			_, _ = bw.WriteString(kw)
			_ = bw.WriteByte('\n')
			totalLines++
		}
		if err := bw.Flush(); err != nil {
			_ = file.Close()
			return err
		}
		_ = file.Close()

		fi, err := os.Stat(expectedPath)
		var sizeBytes int64 = 0
		if err == nil {
			sizeBytes = fi.Size()
		}

		nowStr := time.Now().UTC().Format(time.RFC3339Nano)
		if existingID > 0 {
			_, err = m.db.Exec(`
				UPDATE wordlists
				SET path = ?, line_count = ?, size_bytes = ?, updated_at = ?
				WHERE id = ?
			`, expectedPath, totalLines, sizeBytes, nowStr, existingID)
		} else {
			_, err = m.db.Exec(`
				INSERT INTO wordlists (name, path, line_count, size_bytes, is_linked, category, is_builtin, builtin_id, created_at, updated_at)
				VALUES (?, ?, ?, ?, 0, ?, 1, ?, ?, ?)
			`, pack.Name, expectedPath, totalLines, sizeBytes, pack.Category, pack.ID, nowStr, nowStr)
		}
		if err != nil {
			return fmt.Errorf("failed registering built-in wordlist %s in db: %w", pack.Name, err)
		}
	}

	return nil
}

// ResetBuiltinWordlist restores a built-in wordlist back to its original factory state.
func (m *Manager) ResetBuiltinWordlist(id int64) (*WordlistDTO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var oldPath, builtinID, name, category, createdStr string
	err := m.db.QueryRow(`
		SELECT path, builtin_id, name, category, created_at
		FROM wordlists
		WHERE id = ? AND is_builtin = 1
	`, id).Scan(&oldPath, &builtinID, &name, &category, &createdStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("built-in wordlist not found")
		}
		return nil, err
	}

	cleanOld := filepath.Clean(oldPath)
	if m.lockedPaths[cleanOld] > 0 {
		return nil, ErrWordlistLocked
	}

	pack, err := loadEmbeddedPack(builtinID)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(m.baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating wordlists dir: %w", err)
	}

	targetPath := filepath.Join(m.baseDir, fmt.Sprintf("builtin_%s.txt", pack.ID))
	cleanTarget := filepath.Clean(targetPath)
	if m.lockedPaths[cleanTarget] > 0 {
		return nil, ErrWordlistLocked
	}

	tmpPath := targetPath + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed creating reset tmp file: %w", err)
	}

	bw := bufio.NewWriter(tmpFile)
	var totalLines int64 = 0
	for _, kw := range pack.Keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		_, _ = bw.WriteString(kw)
		_ = bw.WriteByte('\n')
		totalLines++
	}
	if err := bw.Flush(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return nil, err
	}
	_ = tmpFile.Close()

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("failed atomic reset rename: %w", err)
	}

	// Remove old path if it was different and exists
	if cleanOld != cleanTarget && cleanOld != "" {
		_ = os.Remove(cleanOld)
	}

	fi, err := os.Stat(targetPath)
	var sizeBytes int64 = 0
	if err == nil {
		sizeBytes = fi.Size()
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)
	_, err = m.db.Exec(`
		UPDATE wordlists
		SET name = ?, category = ?, path = ?, line_count = ?, size_bytes = ?, updated_at = ?
		WHERE id = ?
	`, pack.Name, pack.Category, targetPath, totalLines, sizeBytes, nowStr, id)
	if err != nil {
		return nil, fmt.Errorf("failed updating reset metadata in db: %w", err)
	}

	return &WordlistDTO{
		ID:        id,
		Name:      pack.Name,
		Path:      targetPath,
		LineCount: totalLines,
		SizeBytes: sizeBytes,
		IsLinked:  false,
		Category:  pack.Category,
		IsBuiltin: true,
		BuiltinID: builtinID,
		CreatedAt: parseTimeHelper(createdStr),
		UpdatedAt: now,
	}, nil
}

// RestoreAllBuiltinWordlists re-seeds any missing built-in wordlists without touching custom ones.
func (m *Manager) RestoreAllBuiltinWordlists() ([]*WordlistDTO, error) {
	if err := m.EnsureBuiltinSeeded(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.db.Query(`
		SELECT id, name, path, line_count, size_bytes, is_linked, category, is_builtin, COALESCE(builtin_id, ''), created_at, updated_at
		FROM wordlists
		WHERE is_builtin = 1
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*WordlistDTO
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
		w.IsLinked = (isLinkedInt == 1)
		w.IsBuiltin = (isBuiltinInt == 1)
		w.CreatedAt = parseTimeHelper(createdStr)
		w.UpdatedAt = parseTimeHelper(updatedStr)
		results = append(results, &w)
	}

	return results, nil
}
