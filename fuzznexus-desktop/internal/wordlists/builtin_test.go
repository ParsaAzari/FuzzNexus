package wordlists_test

import (
	"os"
	"strings"
	"testing"

	"fuzznexus-desktop/internal/wordlists"
)

func TestBuiltinSeedingAndIdempotency(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Verify 11 built-ins seeded
	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}

	var builtins []wordlists.WordlistDTO
	for _, w := range list {
		if w.IsBuiltin {
			builtins = append(builtins, w)
		}
	}

	if len(builtins) != 11 {
		t.Fatalf("expected 11 built-in wordlists, got %d", len(builtins))
	}

	// Verify categories and non-empty keyword counts
	expectedCategories := map[string]bool{
		"admin":      false,
		"api":        false,
		"ai":         false,
		"frameworks": false,
		"cms":        false,
		"cloud":      false,
		"auth":       false,
		"leaks":      false,
		"bypasses":   false,
		"params":     false,
		"extensions": false,
	}

	for _, b := range builtins {
		if b.BuiltinID == "" {
			t.Errorf("builtin %s has empty BuiltinID", b.Name)
		}
		if b.LineCount <= 0 {
			t.Errorf("builtin %s has zero line count", b.Name)
		}
		if b.SizeBytes <= 0 {
			t.Errorf("builtin %s has zero size bytes", b.Name)
		}
		expectedCategories[b.Category] = true

		// Check file mode 0600
		fi, err := os.Stat(b.Path)
		if err != nil {
			t.Errorf("file for %s not found on disk: %v", b.Name, err)
		} else if fi.Mode().Perm() != 0600 {
			t.Errorf("expected file mode 0600, got %o for %s", fi.Mode().Perm(), b.Name)
		}
	}

	for cat, found := range expectedCategories {
		if !found {
			t.Errorf("expected built-in wordlist in category %s was not found", cat)
		}
	}

	// 2. Test Idempotency
	if err := mgr.EnsureBuiltinSeeded(); err != nil {
		t.Fatalf("second EnsureBuiltinSeeded failed: %v", err)
	}

	list2, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	var builtins2 []wordlists.WordlistDTO
	for _, w := range list2 {
		if w.IsBuiltin {
			builtins2 = append(builtins2, w)
		}
	}
	if len(builtins2) != 11 {
		t.Fatalf("expected still 11 built-in wordlists after second seed, got %d", len(builtins2))
	}
}

func TestResetBuiltinWordlist(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}

	var target *wordlists.WordlistDTO
	for _, w := range list {
		if w.IsBuiltin && w.BuiltinID == "frameworks_backend" {
			target = &w
			break
		}
	}
	if target == nil {
		t.Fatalf("frameworks_backend built-in wordlist not found")
	}

	originalLineCount := target.LineCount

	// Modify content of this built-in wordlist
	newContent := "custom_route_1\ncustom_route_2\n"
	if err := mgr.SaveWordlistContent(target.ID, newContent); err != nil {
		t.Fatalf("SaveWordlistContent failed: %v", err)
	}

	// Verify it was modified
	page, err := mgr.ReadWordlistPage(target.ID, 0, 10)
	if err != nil {
		t.Fatalf("ReadWordlistPage failed: %v", err)
	}
	if page.TotalLines != 2 {
		t.Fatalf("expected 2 lines after modification, got %d", page.TotalLines)
	}

	// Now reset back to factory default
	restored, err := mgr.ResetBuiltinWordlist(target.ID)
	if err != nil {
		t.Fatalf("ResetBuiltinWordlist failed: %v", err)
	}
	if restored.LineCount != originalLineCount {
		t.Fatalf("expected line count restored to %d, got %d", originalLineCount, restored.LineCount)
	}

	// Verify reading page again has original keywords
	pageAfter, err := mgr.ReadWordlistPage(target.ID, 0, 10)
	if err != nil {
		t.Fatalf("ReadWordlistPage after reset failed: %v", err)
	}
	if pageAfter.TotalLines != originalLineCount {
		t.Fatalf("expected page total lines %d, got %d", originalLineCount, pageAfter.TotalLines)
	}
	if len(pageAfter.Lines) == 0 || strings.HasPrefix(pageAfter.Lines[0], "custom_route") {
		t.Fatalf("first line should not be custom_route after reset: %v", pageAfter.Lines)
	}
}

func TestBuiltinScanLockSafety(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}

	var target *wordlists.WordlistDTO
	for _, w := range list {
		if w.IsBuiltin {
			target = &w
			break
		}
	}
	if target == nil {
		t.Fatalf("no built-in wordlist found")
	}

	// Lock the wordlist for an active scan
	mgr.LockWordlist(target.Path)

	// Attempting to reset while locked must fail with ErrWordlistLocked
	_, err = mgr.ResetBuiltinWordlist(target.ID)
	if err != wordlists.ErrWordlistLocked {
		t.Fatalf("expected ErrWordlistLocked when resetting locked wordlist, got %v", err)
	}

	// Unlock
	mgr.UnlockWordlist(target.Path)

	// Now reset succeeds
	_, err = mgr.ResetBuiltinWordlist(target.ID)
	if err != nil {
		t.Fatalf("expected reset to succeed after unlocking, got %v", err)
	}
}

func TestRestoreAllBuiltinWordlists(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	list, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}

	var targetID int64
	for _, w := range list {
		if w.IsBuiltin && w.BuiltinID == "bypasses_heuristics" {
			targetID = w.ID
			break
		}
	}
	if targetID == 0 {
		t.Fatalf("bypasses_heuristics built-in wordlist not found")
	}

	// Delete from database
	if err := mgr.DeleteWordlist(targetID); err != nil {
		t.Fatalf("DeleteWordlist failed: %v", err)
	}

	// Verify only 10 built-ins remain
	listAfterDelete, err := mgr.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	var builtinsAfterDelete int
	for _, w := range listAfterDelete {
		if w.IsBuiltin {
			builtinsAfterDelete++
		}
	}
	if builtinsAfterDelete != 10 {
		t.Fatalf("expected 10 built-ins after delete, got %d", builtinsAfterDelete)
	}

	// Call RestoreAllBuiltinWordlists
	restoredList, err := mgr.RestoreAllBuiltinWordlists()
	if err != nil {
		t.Fatalf("RestoreAllBuiltinWordlists failed: %v", err)
	}
	if len(restoredList) != 11 {
		t.Fatalf("expected 11 built-ins restored, got %d", len(restoredList))
	}
}
