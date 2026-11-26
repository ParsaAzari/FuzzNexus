package wordlists_test

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"fuzznexus-desktop/internal/wordlists"

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
		category TEXT NOT NULL DEFAULT 'general',
		is_builtin INTEGER NOT NULL DEFAULT 0,
		builtin_id TEXT,
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
	defer os.RemoveAll(tempDir)
	defer db.Close()

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

	// Check file mode 0600 on POSIX systems (Windows uses ACLs / synthetic 0666)
	info, err := os.Stat(wl.Path)
	if err != nil {
		t.Fatalf("os.Stat failed: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
	}

	// 2. List wordlists
	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	var foundCustom bool
	for _, item := range list {
		if item.ID == wl.ID {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Fatalf("created wordlist not found in list: %+v", list)
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
	var foundUpdated *wordlists.WordlistDTO
	for _, item := range list {
		if item.ID == wl.ID {
			cpy := item
			foundUpdated = &cpy
			break
		}
	}
	if foundUpdated == nil || foundUpdated.Name != "custom.txt" {
		t.Fatalf("name update failed: %+v", foundUpdated)
	}

	// 5. Save content
	newContent := "root\nadmin\nuser\n"
	if err := mgr.SaveWordlistContent(wl.ID, newContent); err != nil {
		t.Fatalf("SaveWordlistContent failed: %v", err)
	}
	list, _ = mgr.ListWordlists()
	foundUpdated = nil
	for _, item := range list {
		if item.ID == wl.ID {
			cpy := item
			foundUpdated = &cpy
			break
		}
	}
	if foundUpdated == nil || foundUpdated.LineCount != 3 {
		t.Fatalf("expected 3 lines after save, got %+v", foundUpdated)
	}

	// 6. Delete wordlist
	if err := mgr.DeleteWordlist(wl.ID); err != nil {
		t.Fatalf("DeleteWordlist failed: %v", err)
	}
	list, _ = mgr.ListWordlists()
	for _, item := range list {
		if item.ID == wl.ID {
			t.Fatalf("expected wl to be deleted, but found in list")
		}
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

	page2, _, err := wordlists.ReadLinesPage(filePath, 100, 5)
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
	var customRemaining []wordlists.WordlistDTO
	for _, item := range list {
		if !item.IsBuiltin {
			customRemaining = append(customRemaining, item)
		}
	}
	if len(customRemaining) != 1 || customRemaining[0].ID != wl3.ID {
		t.Fatalf("expected only wl3 remaining among custom wordlists, got: %+v", customRemaining)
	}
}

func TestWordlistVerificationAndAutoCleaning(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Create with dirty lines (empty lines, leading/trailing whitespace, CRLF)
	dirtyContent := "  admin  \r\n\r\n   \r\napi\r\n\r\nlogin   \r\n"
	wl, err := mgr.CreateWordlist("dirty.txt", dirtyContent)
	if err != nil {
		t.Fatalf("CreateWordlist failed: %v", err)
	}

	// Should have auto-cleaned to 3 valid lines ("admin", "api", "login")
	if wl.LineCount != 3 {
		t.Fatalf("expected 3 clean lines after auto-clean on create, got %d", wl.LineCount)
	}

	page, err := mgr.ReadWordlistPage(wl.ID, 0, 10)
	if err != nil {
		t.Fatalf("ReadWordlistPage failed: %v", err)
	}
	expected := []string{"admin", "api", "login"}
	if len(page.Lines) != 3 || page.Lines[0] != "admin" || page.Lines[1] != "api" || page.Lines[2] != "login" {
		t.Fatalf("unexpected lines after auto-clean: %+v (expected %+v)", page.Lines, expected)
	}

	// 2. Verification of a clean wordlist
	health, err := mgr.VerifyWordlist(wl.ID)
	if err != nil {
		t.Fatalf("VerifyWordlist failed: %v", err)
	}
	if !health.IsClean || health.BlankLinesCount != 0 {
		t.Fatalf("expected clean health, got: %+v", health)
	}

	// 3. Manually create an external dirty file to test LinkWordlistPath & Verify/Clean
	externalDirtyPath := filepath.Join(tempDir, "external_dirty.txt")
	rawDirtyData := "route1\n\n   \nroute2\r\n  route3  \n\n"
	if err := os.WriteFile(externalDirtyPath, []byte(rawDirtyData), 0600); err != nil {
		t.Fatalf("failed to write external dirty file: %v", err)
	}

	linkedWl, err := mgr.LinkWordlistPath(externalDirtyPath)
	if err != nil {
		t.Fatalf("LinkWordlistPath failed: %v", err)
	}

	// Verify should report issues
	dirtyHealth, err := mgr.VerifyWordlist(linkedWl.ID)
	if err != nil {
		t.Fatalf("VerifyWordlist failed: %v", err)
	}
	if dirtyHealth.IsClean {
		t.Fatalf("expected IsClean == false for dirty wordlist")
	}
	if dirtyHealth.BlankLinesCount != 3 {
		t.Fatalf("expected 3 blank lines, got: %d", dirtyHealth.BlankLinesCount)
	}
	if !dirtyHealth.HasWhitespace || !dirtyHealth.HasCRLF {
		t.Fatalf("expected whitespace and CRLF flags to be true: %+v", dirtyHealth)
	}

	// 4. Test Lock protection during CleanWordlist
	mgr.LockWordlist(linkedWl.Path)
	_, err = mgr.CleanWordlist(linkedWl.ID)
	if err != wordlists.ErrWordlistLocked {
		t.Fatalf("expected ErrWordlistLocked on CleanWordlist, got: %v", err)
	}
	mgr.UnlockWordlist(linkedWl.Path)

	// 5. CleanWordlist execution
	cleanRes, err := mgr.CleanWordlist(linkedWl.ID)
	if err != nil {
		t.Fatalf("CleanWordlist failed: %v", err)
	}
	if cleanRes.EmptyLinesRemoved != 3 {
		t.Fatalf("expected 3 empty lines removed, got: %d", cleanRes.EmptyLinesRemoved)
	}
	if cleanRes.LinesAfter != 3 {
		t.Fatalf("expected 3 lines after cleaning, got: %d", cleanRes.LinesAfter)
	}

	// Verify health again after clean
	postCleanHealth, err := mgr.VerifyWordlist(linkedWl.ID)
	if err != nil {
		t.Fatalf("post-clean VerifyWordlist failed: %v", err)
	}
	if !postCleanHealth.IsClean || postCleanHealth.BlankLinesCount != 0 {
		t.Fatalf("expected clean health after CleanWordlist, got: %+v", postCleanHealth)
	}
}

func TestWordlistMetadataAndMerge(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Create two wordlists with initial content
	wl1, err := mgr.CreateWordlist("api_endpoints.txt", "users\nadmin\nlogin\n")
	if err != nil {
		t.Fatalf("CreateWordlist 1 failed: %v", err)
	}
	if wl1.Category != "general" {
		t.Fatalf("expected default category 'general', got: %s", wl1.Category)
	}

	wl2, err := mgr.CreateWordlist("custom_params.txt", "admin\npass\ntoken\n")
	if err != nil {
		t.Fatalf("CreateWordlist 2 failed: %v", err)
	}

	// 2. Test UpdateWordlistMetadata
	if err := mgr.UpdateWordlistMetadata(wl1.ID, "renamed_api.txt", "api"); err != nil {
		t.Fatalf("UpdateWordlistMetadata failed: %v", err)
	}

	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	var foundRenamed bool
	for _, w := range list {
		if w.ID == wl1.ID {
			foundRenamed = true
			if w.Name != "renamed_api.txt" || w.Category != "api" {
				t.Fatalf("expected renamed_api.txt and category 'api', got: %+v", w)
			}
		}
	}
	if !foundRenamed {
		t.Fatalf("renamed wordlist not found in list")
	}

	// 3. Test MergeWordlists
	// wl1 has: users, admin, login (3 lines)
	// wl2 has: admin, pass, token (3 lines)
	// Deduplicated merged: users, admin, login, pass, token (5 lines)
	merged, err := mgr.MergeWordlists([]int64{wl1.ID, wl2.ID}, "merged_suite.txt", "fuzzing")
	if err != nil {
		t.Fatalf("MergeWordlists failed: %v", err)
	}
	if merged.Name != "merged_suite.txt" {
		t.Fatalf("expected name 'merged_suite.txt', got: %s", merged.Name)
	}
	if merged.Category != "fuzzing" {
		t.Fatalf("expected category 'fuzzing', got: %s", merged.Category)
	}
	if merged.LineCount != 5 {
		t.Fatalf("expected 5 deduplicated lines, got: %d", merged.LineCount)
	}

	// Verify merged lines page
	page, err := mgr.ReadWordlistPage(merged.ID, 0, 10)
	if err != nil {
		t.Fatalf("ReadWordlistPage failed: %v", err)
	}
	if len(page.Lines) != 5 {
		t.Fatalf("expected 5 lines in page, got: %d (%v)", len(page.Lines), page.Lines)
	}
}

func TestCategoryCRUD(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// 1. Initial seeded categories
	cats, err := mgr.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories failed: %v", err)
	}
	if len(cats) < 6 {
		t.Fatalf("expected at least 6 seeded categories, got %d", len(cats))
	}

	// 2. Create custom category
	custom, err := mgr.CreateCategory("custom_infra", "Custom Infra Services", "#3b82f6")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if custom.ID != "custom_infra" || custom.Name != "Custom Infra Services" || custom.Color != "#3b82f6" {
		t.Fatalf("unexpected category created: %+v", custom)
	}

	// Create wordlist with this category
	wl, err := mgr.CreateWordlistWithCategory("aws_endpoints.txt", "s3\nec2\nlambda\n", "custom_infra")
	if err != nil {
		t.Fatalf("CreateWordlistWithCategory failed: %v", err)
	}
	if wl.Category != "custom_infra" {
		t.Fatalf("expected category 'custom_infra', got %s", wl.Category)
	}

	// 3. Update category
	err = mgr.UpdateCategory("custom_infra", "Custom Infra & Cloud", "#60a5fa")
	if err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}

	// 4. Delete category (should fall back wordlist to general)
	err = mgr.DeleteCategory("custom_infra")
	if err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	// Verify wordlist is now in general category
	allWls, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	found := false
	for _, w := range allWls {
		if w.ID == wl.ID {
			found = true
			if w.Category != "general" {
				t.Fatalf("expected wordlist category to fall back to 'general', got %s", w.Category)
			}
		}
	}
	if !found {
		t.Fatalf("created wordlist not found in list")
	}

	// Cannot delete general
	err = mgr.DeleteCategory("general")
	if err == nil {
		t.Fatalf("expected error when deleting general category, got nil")
	}
}

func TestSmartDeduplicationPatterns(t *testing.T) {
	// Test CleanStringContent with critical patterns:
	// 1. Case sensitivity: /admin vs /Admin must be preserved as distinct
	// 2. Trailing/leading slash semantics: /api vs api/ must be preserved as distinct
	// 3. Exact duplicates: repeated items removed, first occurrence preserved
	// 4. CRLF and outer whitespace trimmed, blank lines skipped
	input := "/admin\r\n/Admin\r\n/admin\n  /api  \n/api\napi/\n\n\r\nlogin\nlogin\n"
	cleaned, emptyCount, trimmedCount, dupCount := wordlists.CleanStringContent(input)

	if emptyCount != 3 {
		t.Errorf("expected 3 empty lines removed, got: %d", emptyCount)
	}
	if trimmedCount != 1 { // "  /api  " had outer whitespace
		t.Errorf("expected 1 trimmed line, got: %d", trimmedCount)
	}
	if dupCount != 3 {
		t.Errorf("expected 3 duplicates, got: %d", dupCount)
	}

	expectedLines := []string{
		"/admin",
		"/Admin",
		"/api",
		"api/",
		"login",
	}
	expectedOutput := strings.Join(expectedLines, "\n") + "\n"
	if cleaned != expectedOutput {
		t.Fatalf("unexpected cleaned output:\nGot:\n%s\nExpected:\n%s", cleaned, expectedOutput)
	}

	// Test CleanWordlistFile on disk
	tmpFile, err := os.CreateTemp("", "test-dedup-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(input); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	_ = tmpFile.Close()

	// Verify before cleaning
	preHealth, err := wordlists.VerifyWordlistFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("VerifyWordlistFile failed: %v", err)
	}
	if preHealth.IsClean {
		t.Fatalf("expected file to be dirty before cleaning")
	}
	if preHealth.DuplicateCount != 3 {
		t.Errorf("expected 3 duplicates detected, got: %d", preHealth.DuplicateCount)
	}

	// Clean file
	cleanRes, err := wordlists.CleanWordlistFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("CleanWordlistFile failed: %v", err)
	}
	if cleanRes.DuplicatesRemoved != 3 {
		t.Errorf("expected 3 duplicates removed, got: %d", cleanRes.DuplicatesRemoved)
	}
	if cleanRes.LinesAfter != 5 {
		t.Errorf("expected 5 lines after cleaning, got: %d", cleanRes.LinesAfter)
	}

	// Verify after cleaning
	postHealth, err := wordlists.VerifyWordlistFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("VerifyWordlistFile post-clean failed: %v", err)
	}
	if !postHealth.IsClean {
		t.Fatalf("expected file to be clean, got: %+v", postHealth)
	}
	if postHealth.DuplicateCount != 0 {
		t.Errorf("expected 0 duplicates after cleaning, got: %d", postHealth.DuplicateCount)
	}
}

func TestHighPrecisionSanitizer(t *testing.T) {
	// 1. Test SanitizeLine specifics
	opts := wordlists.DefaultSanitizeOptions()

	// Test BOM stripping on first line
	cleaned, drop, reason, _ := wordlists.SanitizeLine("\xEF\xBB\xBFadmin", opts, true)
	if drop || cleaned != "admin" {
		t.Fatalf("expected 'admin', got '%s', drop=%v, reason=%s", cleaned, drop, reason)
	}

	// Test CRLF and invisible whitespace
	cleaned, drop, _, trimmed := wordlists.SanitizeLine("  \t\u00A0secret\u200B\r\n", opts, false)
	if drop || cleaned != "secret" || !trimmed {
		t.Fatalf("expected 'secret' with trimmed=true, got '%s', drop=%v, trimmed=%v", cleaned, drop, trimmed)
	}

	// Test Null-byte removal
	cleaned, drop, _, _ = wordlists.SanitizeLine("api\x00v1", opts, false)
	if drop || cleaned != "apiv1" {
		t.Fatalf("expected 'apiv1', got '%s', drop=%v", cleaned, drop)
	}

	// Test comments
	_, drop, reason, _ = wordlists.SanitizeLine("# SecLists header comment", opts, false)
	if !drop || reason != "comment" {
		t.Fatalf("expected drop with reason=comment, got drop=%v, reason=%s", drop, reason)
	}
	_, drop, reason, _ = wordlists.SanitizeLine("// inline comment", opts, false)
	if !drop || reason != "comment" {
		t.Fatalf("expected drop with reason=comment, got drop=%v, reason=%s", drop, reason)
	}

	// Test Leading Slash stripping option
	slashOpts := opts
	slashOpts.StripLeadingSlash = true
	cleaned, drop, _, trimmed = wordlists.SanitizeLine("/api/v2", slashOpts, false)
	if drop || cleaned != "api/v2" || !trimmed {
		t.Fatalf("expected 'api/v2', got '%s', drop=%v, trimmed=%v", cleaned, drop, trimmed)
	}

	// 2. Test Stream Processing with Deduplication (Case sensitive vs Case insensitive)
	rawSource1 := "\xEF\xBB\xBF# Discovery header\nadmin\nlogin\n\n  users  \r\n"
	rawSource2 := "ADMIN\nlogin\n/api/test\n# footer\n"

	// Case sensitive run
	var outBuf bytes.Buffer
	report, err := wordlists.ProcessAndDeduplicate(
		[]io.Reader{strings.NewReader(rawSource1), strings.NewReader(rawSource2)},
		&outBuf,
		opts,
	)
	if err != nil {
		t.Fatalf("ProcessAndDeduplicate failed: %v", err)
	}

	// Expected lines: admin, login, users, ADMIN, /api/test (5 lines)
	// 'login' is duplicated, empty line dropped, 2 comments dropped
	if report.DuplicatesRemoved != 1 {
		t.Errorf("expected 1 duplicate removed, got: %d", report.DuplicatesRemoved)
	}
	if report.EmptyLinesRemoved != 1 {
		t.Errorf("expected 1 empty line removed, got: %d", report.EmptyLinesRemoved)
	}
	if report.CommentsRemoved != 2 {
		t.Errorf("expected 2 comments removed, got: %d", report.CommentsRemoved)
	}
	if report.DeduplicatedLines != 5 {
		t.Errorf("expected 5 deduplicated lines, got: %d", report.DeduplicatedLines)
	}

	// Case-insensitive run
	caseOpts := opts
	caseOpts.CaseInsensitive = true
	var caseBuf bytes.Buffer
	caseReport, err := wordlists.ProcessAndDeduplicate(
		[]io.Reader{strings.NewReader(rawSource1), strings.NewReader(rawSource2)},
		&caseBuf,
		caseOpts,
	)
	if err != nil {
		t.Fatalf("ProcessAndDeduplicate case-insensitive failed: %v", err)
	}
	// 'ADMIN' matches 'admin' when case-insensitive, so 2 duplicates removed (login, ADMIN)
	if caseReport.DuplicatesRemoved != 2 {
		t.Errorf("expected 2 duplicates removed in case-insensitive, got: %d", caseReport.DuplicatesRemoved)
	}
	if caseReport.DeduplicatedLines != 4 {
		t.Errorf("expected 4 deduplicated lines in case-insensitive, got: %d", caseReport.DeduplicatedLines)
	}
}

func TestSandboxLifecycleAndSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ffuf-sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm, err := wordlists.NewSandboxManager(tempDir)
	if err != nil {
		t.Fatalf("NewSandboxManager failed: %v", err)
	}

	// Check sandbox directory exists
	info, err := os.Stat(sm.Dir())
	if err != nil {
		t.Fatalf("sandbox dir stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected sandbox to be a directory")
	}

	// Create sandbox file
	f, path, err := sm.CreateSandboxFile("test_sb")
	if err != nil {
		t.Fatalf("CreateSandboxFile failed: %v", err)
	}
	if _, err := f.WriteString("payload1\npayload2\n"); err != nil {
		t.Fatalf("failed to write sandbox file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close sandbox file: %v", err)
	}

	// Check file permissions
	fInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("sandbox file stat failed: %v", err)
	}
	if fInfo.Size() == 0 {
		t.Fatalf("expected non-empty file")
	}

	// Validate path containment
	validPath, err := sm.ValidateSandboxPath(path)
	if err != nil {
		t.Fatalf("ValidateSandboxPath failed for valid file: %v", err)
	}
	evalExpected, _ := filepath.EvalSymlinks(path)
	if validPath != evalExpected {
		t.Fatalf("expected %s, got %s", evalExpected, validPath)
	}

	// Test path traversal rejection
	_, err = sm.ValidateSandboxPath(filepath.Join(sm.Dir(), "..", "escaped.txt"))
	if err == nil {
		t.Fatalf("expected path traversal error for escaping path")
	}

	// Test locking and cleanup
	sm.RegisterEphemeral(path)
	if !sm.IsEphemeral(path) {
		t.Fatalf("expected path to be registered as ephemeral")
	}

	sm.Lock(path)
	if !sm.IsLocked(path) {
		t.Fatalf("expected path to be locked")
	}

	// Attempting cleanup while locked should fail with ErrSandboxLocked
	if err := sm.CleanupPath(path); err != wordlists.ErrSandboxLocked {
		t.Fatalf("expected ErrSandboxLocked, got: %v", err)
	}

	// Unlock and cleanup
	sm.Unlock(path)
	if sm.IsLocked(path) {
		t.Fatalf("expected path to be unlocked")
	}

	if err := sm.CleanupPath(path); err != nil {
		t.Fatalf("CleanupPath failed after unlock: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted from disk, err: %v", err)
	}
}

func TestMergeWordlistsToSandbox(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create 2 wordlists across different categories
	wl1, err := mgr.CreateWordlistWithCategory("discovery_paths.txt", "admin\nlogin\ndashboard\n", "discovery")
	if err != nil {
		t.Fatalf("CreateWordlist 1 failed: %v", err)
	}

	wl2, err := mgr.CreateWordlistWithCategory("api_endpoints.txt", "login\n/api/v1/users\n# comment\n", "api")
	if err != nil {
		t.Fatalf("CreateWordlist 2 failed: %v", err)
	}

	// 1. Test PreviewMergeStats
	opts := wordlists.DefaultSanitizeOptions()
	opts.StripLeadingSlash = true
	preview, err := mgr.PreviewMergeStats([]int64{wl1.ID, wl2.ID}, opts)
	if err != nil {
		t.Fatalf("PreviewMergeStats failed: %v", err)
	}
	// wl1: admin, login, dashboard (3 lines)
	// wl2: login (duplicate), api/v1/users (leading slash stripped), comment (stripped)
	// Deduplicated lines: admin, login, dashboard, api/v1/users (4 lines)
	if preview.DeduplicatedLines != 4 {
		t.Errorf("expected 4 deduplicated lines in preview, got: %d", preview.DeduplicatedLines)
	}
	if preview.DuplicatesRemoved != 1 {
		t.Errorf("expected 1 duplicate removed, got: %d", preview.DuplicatesRemoved)
	}
	if preview.CommentsRemoved != 1 {
		t.Errorf("expected 1 comment removed, got: %d", preview.CommentsRemoved)
	}

	// 2. Test Ephemeral Sandbox Merge
	sbDTO, err := mgr.MergeWordlistsToSandbox([]int64{wl1.ID, wl2.ID}, opts, true, "ephemeral_merged.txt", "sandbox")
	if err != nil {
		t.Fatalf("MergeWordlistsToSandbox (ephemeral) failed: %v", err)
	}

	if sbDTO.ID != 0 {
		t.Errorf("expected ID 0 for ephemeral sandbox wordlist, got: %d", sbDTO.ID)
	}
	if sbDTO.Category != "sandbox" {
		t.Errorf("expected category 'sandbox', got: %s", sbDTO.Category)
	}
	if sbDTO.LineCount != 4 {
		t.Errorf("expected 4 lines, got: %d", sbDTO.LineCount)
	}
	if !mgr.Sandbox().IsEphemeral(sbDTO.Path) {
		t.Errorf("expected path %s to be registered as ephemeral", sbDTO.Path)
	}

	// Verify file content on disk
	content, err := os.ReadFile(sbDTO.Path)
	if err != nil {
		t.Fatalf("failed to read sandbox file: %v", err)
	}
	expectedContent := "admin\nlogin\ndashboard\napi/v1/users\n"
	if string(content) != expectedContent {
		t.Errorf("expected content:\n%s\ngot:\n%s", expectedContent, string(content))
	}

	// 3. Test Active Scan Lock: unlock retains sandbox file for re-scanning
	mgr.LockWordlist(sbDTO.Path)
	if !mgr.IsLocked(sbDTO.Path) {
		t.Errorf("expected path to be locked")
	}

	// Unlocking must NOT delete the file; user may re-scan or tweak settings
	mgr.UnlockWordlist(sbDTO.Path)
	if _, err := os.Stat(sbDTO.Path); os.IsNotExist(err) {
		t.Errorf("expected ephemeral file to be retained after unlock for re-use, but it was deleted")
	}

	// Explicit cleanup (e.g. user clicks Clear Sandbox in UI) deletes the file
	if err := mgr.CleanupSandboxWordlist(sbDTO.Path); err != nil {
		t.Fatalf("CleanupSandboxWordlist failed: %v", err)
	}
	if _, err := os.Stat(sbDTO.Path); !os.IsNotExist(err) {
		t.Errorf("expected ephemeral file to be deleted after explicit CleanupSandboxWordlist, err: %v", err)
	}

	// 4. Test Persistent Merge (Save to Library)
	persDTO, err := mgr.MergeWordlistsToSandbox([]int64{wl1.ID, wl2.ID}, opts, false, "persistent_merged.txt", "fuzzing")
	if err != nil {
		t.Fatalf("MergeWordlistsToSandbox (persistent) failed: %v", err)
	}
	if persDTO.ID <= 0 {
		t.Errorf("expected valid DB ID > 0 for persistent wordlist, got: %d", persDTO.ID)
	}
	if persDTO.Category != "fuzzing" {
		t.Errorf("expected category 'fuzzing', got: %s", persDTO.Category)
	}
	// Check DB record exists
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM wordlists WHERE id = ?", persDTO.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected record in DB for persistent wordlist")
	}
}
