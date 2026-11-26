package wordlists_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fuzznexus-desktop/internal/wordlists"
)

func TestAnalyzeWordlistRisks_HighAndLow(t *testing.T) {
	tempDir := t.TempDir()

	highPath := filepath.Join(tempDir, "bypass.txt")
	highContent := strings.Join([]string{
		".//admin",
		".//system",
		".//manager",
		".//internal",
		".//v1",
		".;/api",
		"/%2e/portal",
		"about",
		"contact",
	}, "\n")
	if err := os.WriteFile(highPath, []byte(highContent), 0600); err != nil {
		t.Fatal(err)
	}

	report, err := wordlists.AnalyzeWordlistRisks(highPath)
	if err != nil {
		t.Fatalf("AnalyzeWordlistRisks failed: %v", err)
	}
	if report.RiskLevel != "high" {
		t.Fatalf("expected high risk level, got %q (%+v)", report.RiskLevel, report)
	}
	if report.TraversalCount != 7 {
		t.Fatalf("expected 7 traversal patterns, got %d", report.TraversalCount)
	}
	if len(report.SampleRiskyLines) == 0 {
		t.Fatal("expected sample risky lines to be captured")
	}

	lowPath := filepath.Join(tempDir, "clean.txt")
	lowContent := "about\ncontact\npricing\nblog\ndocs\nstatus\n"
	if err := os.WriteFile(lowPath, []byte(lowContent), 0600); err != nil {
		t.Fatal(err)
	}
	lowReport, err := wordlists.AnalyzeWordlistRisks(lowPath)
	if err != nil {
		t.Fatalf("AnalyzeWordlistRisks low failed: %v", err)
	}
	if lowReport.RiskLevel != "low" {
		t.Fatalf("expected low risk level, got %q (%+v)", lowReport.RiskLevel, lowReport)
	}
}

func TestCreateSanitizedSandboxWordlist_NonDestructive(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll(tempDir)

	mgr, err := wordlists.NewManager(tempDir, db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	srcPath := filepath.Join(tempDir, "mixed_bypasses.txt")
	original := ".//system\n.//manager\n.;/internal\n/%2e/v1\n?format=json\nAPI\nSYSTEM\nabout\n"
	if err := os.WriteFile(srcPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	sanitizedDTO, err := mgr.CreateSanitizedSandboxWordlist(srcPath)
	if err != nil {
		t.Fatalf("CreateSanitizedSandboxWordlist failed: %v", err)
	}

	if sanitizedDTO.Path == srcPath {
		t.Fatalf("expected new sandbox file path, got original path")
	}
	if !strings.Contains(sanitizedDTO.Path, "sb_sanitized") {
		t.Fatalf("expected sb_sanitized sandbox path, got %s", sanitizedDTO.Path)
	}
	if sanitizedDTO.LineCount != 4 {
		t.Fatalf("expected 4 clean lines remaining (?format=json, API, SYSTEM, about), got %d", sanitizedDTO.LineCount)
	}

	// Ensure original file was not altered
	afterSrcBytes, _ := os.ReadFile(srcPath)
	if string(afterSrcBytes) != original {
		t.Fatalf("source wordlist was mutated! expected %q, got %q", original, string(afterSrcBytes))
	}

	// Verify sanitized content has zero .// or .;/ lines
	cleanBytes, err := os.ReadFile(sanitizedDTO.Path)
	if err != nil {
		t.Fatalf("failed reading sanitized file: %v", err)
	}
	if strings.Contains(string(cleanBytes), ".//") || strings.Contains(string(cleanBytes), ".;/") {
		t.Fatalf("traversal pattern leaked into sanitized file:\n%s", string(cleanBytes))
	}

	// Verify all-traversal wordlist returns helpful error instead of empty scan
	allRiskyPath := filepath.Join(tempDir, "only_traversal.txt")
	if err := os.WriteFile(allRiskyPath, []byte(".//a\n.//b\n../c\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.CreateSanitizedSandboxWordlist(allRiskyPath); err == nil {
		t.Fatal("expected error when 100% of wordlist lines are traversal payloads")
	}
}
