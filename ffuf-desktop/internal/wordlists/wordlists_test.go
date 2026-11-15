package wordlists_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"ffuf-desktop/internal/wordlists"

	"go.uber.org/goleak"
	_ "modernc.org/sqlite"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func setupTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ffuf-wordlists-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE wordlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		line_count INTEGER NOT NULL DEFAULT 0,
		size_bytes INTEGER NOT NULL DEFAULT 0,
		is_linked INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create schema: %v", err)
	}

	return db, tempDir
}

func TestWordlistCRUD(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Create wordlist
	content := "admin\nlogin\ndashboard\napi\n"
	wl, err := mgr.CreateWordlist("common.txt", content)
	if err != nil {
		t.Fatalf("CreateWordlist failed: %v", err)
	}
	if wl.ID <= 0 || wl.Name != "common.txt" || wl.LineCount != 4 {
		t.Fatalf("unexpected wordlist: %+v", wl)
	}

	// Check file mode 0600
	info, err := os.Stat(wl.Path)
	if err != nil {
		t.Fatalf("os.Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
	}

	// 2. List wordlists
	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != wl.ID {
		t.Fatalf("unexpected list: %+v", list)
	}

	// 3. Read page
	page, err := mgr.ReadWordlistPage(wl.ID, 1, 2)
	if err != nil {
		t.Fatalf("ReadWordlistPage failed: %v", err)
	}
	if len(page.Lines) != 2 || page.Lines[0] != "login" || page.Lines[1] != "dashboard" {
		t.Fatalf("unexpected page: %+v", page)
	}

	// 4. Update name
	if err := mgr.UpdateWordlist(wl.ID, "custom.txt"); err != nil {
		t.Fatalf("UpdateWordlist failed: %v", err)
	}
	list, _ = mgr.ListWordlists()
	if list[0].Name != "custom.txt" {
		t.Fatalf("name update failed: %s", list[0].Name)
	}

	// 5. Save content
	newContent := "root\nadmin\nuser\n"
	if err := mgr.SaveWordlistContent(wl.ID, newContent); err != nil {
		t.Fatalf("SaveWordlistContent failed: %v", err)
	}
	list, _ = mgr.ListWordlists()
	if list[0].LineCount != 3 {
		t.Fatalf("expected 3 lines after save, got %d", list[0].LineCount)
	}

	// 6. Delete wordlist
	if err := mgr.DeleteWordlist(wl.ID); err != nil {
		t.Fatalf("DeleteWordlist failed: %v", err)
	}
	list, _ = mgr.ListWordlists()
	if len(list) != 0 {
		t.Fatalf("expected 0 wordlists after delete, got %d", len(list))
	}
	if _, err := os.Stat(wl.Path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted from disk")
	}
}

func TestActiveScanLock(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	wl, err := mgr.CreateWordlist("locked.txt", "one\ntwo\nthree\n")
	if err != nil {
		t.Fatalf("CreateWordlist failed: %v", err)
	}

	// Lock the wordlist as if an active scan session is reading it
	mgr.LockWordlist(wl.Path)
	if !mgr.IsLocked(wl.Path) {
		t.Fatalf("expected wordlist to be locked")
	}

	// Attempt to delete should fail with ErrWordlistLocked
	err = mgr.DeleteWordlist(wl.ID)
	if err != wordlists.ErrWordlistLocked {
		t.Fatalf("expected ErrWordlistLocked on delete, got: %v", err)
	}

	// Attempt to save content should fail with ErrWordlistLocked
	err = mgr.SaveWordlistContent(wl.ID, "modified content")
	if err != wordlists.ErrWordlistLocked {
		t.Fatalf("expected ErrWordlistLocked on save, got: %v", err)
	}

	// Unlock the wordlist
	mgr.UnlockWordlist(wl.Path)
	if mgr.IsLocked(wl.Path) {
		t.Fatalf("expected wordlist to be unlocked")
	}

	// Now deletion should succeed
	if err := mgr.DeleteWordlist(wl.ID); err != nil {
		t.Fatalf("expected successful delete after unlock: %v", err)
	}
}

func TestPathSanitization(t *testing.T) {
	// Rejection of empty path
	if _, err := wordlists.ValidatePath(""); err != wordlists.ErrInvalidPath {
		t.Fatalf("expected ErrInvalidPath for empty path, got: %v", err)
	}

	// Rejection of /dev files
	if _, err := wordlists.ValidatePath("/dev/urandom"); err == nil {
		t.Fatalf("expected error for /dev path, got nil")
	}

	// Rejection of directory path
	tempDir, err := os.MkdirTemp("", "ffuf-sanitization-*")
	if err != nil {
		t.Fatalf("temp dir creation failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if _, err := wordlists.ValidatePath(tempDir); err == nil {
		t.Fatalf("expected error for directory target, got nil")
	}

	// Rejection of non-existent path
	nonExistent := filepath.Join(tempDir, "does-not-exist.txt")
	if _, err := wordlists.ValidatePath(nonExistent); err != wordlists.ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound, got: %v", err)
	}
}

func TestStreamingLinePaginationAndStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ffuf-stream-test-*")
	if err != nil {
		t.Fatalf("temp dir creation failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "large.txt")

	// Generate 5,000 lines
	var sb strings.Builder
	for i := 1; i <= 5000; i++ {
		sb.WriteString(fmt.Sprintf("item_payload_%d\n", i))
	}
	if err := wordlists.AtomicWriteFile(filePath, []byte(sb.String())); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	// Test CountLines
	lines, size, err := wordlists.CountLines(filePath)
	if err != nil {
		t.Fatalf("CountLines failed: %v", err)
	}
	if lines != 5000 || size <= 0 {
		t.Fatalf("expected 5000 lines, got %d (size: %d)", lines, size)
	}

	// Test ReadLinesPage pagination
	page1, total, err := wordlists.ReadLinesPage(filePath, 0, 10)
	if err != nil {
		t.Fatalf("ReadLinesPage page 1 failed: %v", err)
	}
	if len(page1) != 10 || page1[0] != "item_payload_1" || total != 5000 {
		t.Fatalf("unexpected page 1: %+v", page1)
	}

	page2, total, err := wordlists.ReadLinesPage(filePath, 100, 5)
	if err != nil {
		t.Fatalf("ReadLinesPage page 2 failed: %v", err)
	}
	if len(page2) != 5 || page2[0] != "item_payload_101" {
		t.Fatalf("unexpected page 2: %+v", page2)
	}

	// Test ComputeStats
	stats, err := wordlists.ComputeStats(filePath)
	if err != nil {
		t.Fatalf("ComputeStats failed: %v", err)
	}
	if stats.LineCount != 5000 || stats.UniqueLines != 5000 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestConcurrentWordlistReads(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	wl, err := mgr.CreateWordlist("concurrent.txt", "line1\nline2\nline3\nline4\nline5\n")
	if err != nil {
		t.Fatalf("CreateWordlist failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				page, err := mgr.ReadWordlistPage(wl.ID, 0, 3)
				if err != nil || len(page.Lines) != 3 {
					t.Errorf("concurrent read error: %v, len: %d", err, len(page.Lines))
				}
			}
		}()
	}
	wg.Wait()
}

func TestDeleteWordlistsBatch(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	wl1, err := mgr.CreateWordlist("batch1.txt", "a\nb\n")
	if err != nil {
		t.Fatalf("CreateWordlist 1 failed: %v", err)
	}
	wl2, err := mgr.CreateWordlist("batch2.txt", "c\nd\n")
	if err != nil {
		t.Fatalf("CreateWordlist 2 failed: %v", err)
	}
	wl3, err := mgr.CreateWordlist("batch3.txt", "e\nf\n")
	if err != nil {
		t.Fatalf("CreateWordlist 3 failed: %v", err)
	}

	// 1. Empty batch
	deleted, err := mgr.DeleteWordlistsBatch([]int64{})
	if err != nil || deleted != 0 {
		t.Fatalf("expected (0, nil) for empty batch, got (%d, %v)", deleted, err)
	}

	// 2. Lock wl2, attempt batch delete of [wl1, wl2]
	mgr.LockWordlist(wl2.Path)
	deleted, err = mgr.DeleteWordlistsBatch([]int64{wl1.ID, wl2.ID})
	if err == nil || !strings.Contains(err.Error(), wordlists.ErrWordlistLocked.Error()) {
		t.Fatalf("expected ErrWordlistLocked error, got: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected 0 deleted due to lock pre-check, got: %d", deleted)
	}

	// Ensure wl1 was NOT deleted
	if _, err := os.Stat(wl1.Path); os.IsNotExist(err) {
		t.Fatalf("expected wl1 file to still exist after failed batch delete")
	}

	// 3. Unlock wl2, batch delete [wl1, wl2]
	mgr.UnlockWordlist(wl2.Path)
	deleted, err = mgr.DeleteWordlistsBatch([]int64{wl1.ID, wl2.ID})
	if err != nil {
		t.Fatalf("batch delete failed after unlock: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted, got: %d", deleted)
	}

	// Files 1 & 2 should be gone
	if _, err := os.Stat(wl1.Path); !os.IsNotExist(err) {
		t.Fatalf("expected wl1 file to be deleted")
	}
	if _, err := os.Stat(wl2.Path); !os.IsNotExist(err) {
		t.Fatalf("expected wl2 file to be deleted")
	}

	// wl3 should remain
	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != wl3.ID {
		t.Fatalf("expected only wl3 remaining, got: %+v", list)
	}
}
