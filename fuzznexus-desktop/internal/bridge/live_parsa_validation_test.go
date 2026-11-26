package bridge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fuzznexus-desktop/internal/scan"
	"github.com/parsaazari/fuzznexus/pkg/engine"
)

func TestLiveParsaAzari_DifferentialAdvisoryAndSanitize(t *testing.T) {
	if os.Getenv("FUZZNEXUS_LIVE_TEST") != "1" {
		t.Skip("set FUZZNEXUS_LIVE_TEST=1 to run controlled live validation against parsaazari.com")
	}

	tempDir := t.TempDir()
	svc, err := NewService(tempDir, nil)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()

	// 1. Passive Risk Check using real bypass payloads (matching builtin_bypasses_heuristics.txt)
	bypassWL := filepath.Join(tempDir, "live_bypass_slice.txt")
	bypassContent := strings.Join([]string{
		".//system",
		".//manager",
		".//internal",
		".//v1",
		".//portal",
		".//root",
		".//dashboard",
		".//api",
		"?format=json",
		"SYSTEM",
		"INTERNAL",
		"API",
	}, "\n")
	if err := os.WriteFile(bypassWL, []byte(bypassContent), 0600); err != nil {
		t.Fatal(err)
	}

	riskReport, err := svc.AnalyzeWordlistRisk(bypassWL)
	if err != nil {
		t.Fatalf("AnalyzeWordlistRisk failed: %v", err)
	}
	t.Logf("[PASSIVE RISK] level=%s total=%d traversal=%d summary=%q samples=%v",
		riskReport.RiskLevel, riskReport.TotalLines, riskReport.TraversalCount, riskReport.Summary, riskReport.SampleRiskyLines)
	if riskReport.RiskLevel != "high" || riskReport.TraversalCount != 8 {
		t.Fatalf("expected high risk & 8 traversal patterns, got level=%s traversal=%d", riskReport.RiskLevel, riskReport.TraversalCount)
	}

	// 2. Active Live Scan against https://parsaazari.com/FUZZ (safe rate: 2 req/s, threads: 1)
	var falseIPBanCount atomic.Int32
	advisoryCh := make(chan scan.AdvisoryAlertDTO, 4)

	svc.SetEventEmitter(func(name string, data interface{}) {
		switch name {
		case "scan:safe_stop_prompt":
			falseIPBanCount.Add(1)
			t.Logf("[UNEXPECTED SAFE STOP] %+v", data)
		case "scan:advisory_alert":
			if alert, ok := data.(scan.AdvisoryAlertDTO); ok {
				select {
				case advisoryCh <- alert:
				default:
				}
			}
		}
	})

	sid1, err := svc.StartScan(ScanRequestDTO{
		URL:            "https://parsaazari.com/FUZZ",
		Method:         "GET",
		Wordlist:       bypassWL,
		Threads:        1,
		Rate:           2,
		Timeout:        10,
		BrowserProfile: "auto",
		RotateOnBlock:  true,
	})
	if err != nil {
		t.Fatalf("StartScan 1 failed: %v", err)
	}

	// Wait for scan completion cleanly (polling session state up to 35s)
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		sessDTO, gerr := svc.GetSession(sid1)
		if gerr == nil && sessDTO != nil && (sessDTO.State == "Completed" || sessDTO.State == "Failed" || sessDTO.State == "Stopped") {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	_ = svc.StopScan()

	if got := falseIPBanCount.Load(); got != 0 {
		t.Fatalf("expected 0 false IP-ban SafeStop popups, got %d", got)
	}

	select {
	case alert := <-advisoryCh:
		t.Logf("[LIVE ADVISORY] verdict=%s conf=%d root=%d benign=%d header=%q fixedLen=%d samples=%v",
			alert.Verdict, alert.Confidence, alert.Canary.RootStatus, alert.Canary.BenignPathStatus,
			alert.EvidenceHeader, alert.FixedBodyLength, alert.SamplePayloads)
		if alert.Verdict != "WORDLIST_WAF_TRIGGER" {
			t.Fatalf("expected WORDLIST_WAF_TRIGGER on live site, got %s (%s)", alert.Verdict, alert.Summary)
		}
		if alert.Canary.RootStatus != 200 || alert.Canary.BenignPathStatus != 404 {
			t.Fatalf("expected Canary root=200 & benign=404, got root=%d benign=%d", alert.Canary.RootStatus, alert.Canary.BenignPathStatus)
		}
	case <-time.After(4 * time.Second):
		t.Fatalf("expected live scan:advisory_alert for WORDLIST_WAF_TRIGGER on parsaazari.com")
	}

	// 3. Sanitize Copy & Rescan Live Verification
	sanitizedDTO, err := svc.CreateSanitizedSandboxWordlist(bypassWL)
	if err != nil {
		t.Fatalf("CreateSanitizedSandboxWordlist failed: %v", err)
	}
	t.Logf("[SANITIZED COPY] path=%s lines=%d", sanitizedDTO.Path, sanitizedDTO.LineCount)
	if sanitizedDTO.LineCount != 4 {
		t.Fatalf("expected 4 clean lines (?format=json, SYSTEM, INTERNAL, API), got %d", sanitizedDTO.LineCount)
	}

	var sanitizedAdvisory atomic.Int32
	svc.SetEventEmitter(func(name string, data interface{}) {
		if name == "scan:safe_stop_prompt" || name == "scan:advisory_alert" {
			sanitizedAdvisory.Add(1)
		}
	})

	sid2, err := svc.StartScan(ScanRequestDTO{
		URL:            "https://parsaazari.com/FUZZ",
		Method:         "GET",
		Wordlist:       sanitizedDTO.Path,
		Threads:        1,
		Rate:           2,
		Timeout:        10,
		BrowserProfile: "auto",
		RotateOnBlock:  true,
	})
	if err != nil {
		t.Fatalf("StartScan 2 failed: %v", err)
	}

	deadline2 := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline2) {
		sessDTO, gerr := svc.GetSession(sid2)
		if gerr == nil && sessDTO != nil && (sessDTO.State == "Completed" || sessDTO.State == "Failed" || sessDTO.State == "Stopped") {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	_ = svc.StopScan()

	if sanitizedAdvisory.Load() != 0 {
		t.Fatalf("expected 0 block/advisory alerts after sanitizing wordlist, got %d", sanitizedAdvisory.Load())
	}

	res2, err := svc.GetResultsSince(sid2, 0, 20)
	if err != nil {
		t.Fatalf("GetResultsSince 2 failed: %v", err)
	}
	t.Logf("[SANITIZED RESCAN RESULTS] session=%d matchedCount=%d", sid2, len(res2.Results))
	for _, r := range res2.Results {
		t.Logf("  -> %s | status=%d | len=%d", r.URL, r.Status, r.Length)
		if r.Status == 403 {
			t.Fatalf("unexpected 403 after sanitizing traversal payloads: %s", r.URL)
		}
	}

	// 4. Verify SafeStopWorkflow TestConnection succeeds against https://parsaazari.com/ (fix for 404 /health bug)
	cb := engine.NewCircuitBreaker(engine.DefaultCircuitBreakerConfig())
	wf := engine.NewSafeStopWorkflow(cb, engine.NewBanDetector(engine.DefaultBanDetectorConfig(), nil), engine.NewManualVPNProvider(""), nil, "https://parsaazari.com/", filepath.Join(tempDir, "resume.json"))
	wf.SetBrowserProfile("auto")
	wf.SetTargetProbeFlags(false, true)
	_, _ = wf.EnterPausedIPBlock(context.Background(), "live-probe-check")

	ok, ipHash, err := wf.TestConnection(context.Background())
	t.Logf("[LIVE TEST CONNECTION] ok=%v ipHash=%s err=%v", ok, ipHash, err)
	if !ok || err != nil {
		t.Fatalf("expected SafeStopWorkflow.TestConnection to succeed against live https://parsaazari.com/, got ok=%v err=%v", ok, err)
	}
}

func TestLiveParsaAzariAutoCalibrationAndUTLS(t *testing.T) {
	if os.Getenv("FUZZNEXUS_LIVE_TEST") != "1" {
		t.Skip("set FUZZNEXUS_LIVE_TEST=1 to run live network validation")
	}

	tempDir := t.TempDir()
	svc, err := NewService(tempDir, nil)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()

	// Verify 11 builtin packs seeded including params_fuzzing
	packs, err := svc.ListWordlists()
	if err != nil {
		t.Fatalf("ListWordlists failed: %v", err)
	}
	var paramsPackPath string
	for _, p := range packs {
		if p.BuiltinID == "params_fuzzing" {
			paramsPackPath = p.Path
			break
		}
	}
	if paramsPackPath == "" {
		t.Fatalf("expected params_fuzzing builtin wordlist to be seeded")
	}

	// Create a controlled 6-parameter slice from params_fuzzing
	slicePath := filepath.Join(tempDir, "params_slice.txt")
	if err := os.WriteFile(slicePath, []byte("id\nuser_id\ndebug\npage\nq\ntoken\n"), 0600); err != nil {
		t.Fatal(err)
	}

	sid, err := svc.StartScan(ScanRequestDTO{
		URL:             "https://parsaazari.com/?FUZZ=1",
		Method:          "GET",
		Headers:         map[string]string{"X-FuzzNexus-Audit": "2026"},
		Wordlist:        slicePath,
		Threads:         1,
		Rate:            2,
		Timeout:         10,
		AutoCalibration: true,
		TLSProfile:      "chrome",
		BrowserProfile:  "chrome-macos",
	})
	if err != nil {
		t.Fatalf("StartScan failed: %v", err)
	}

	waitForScanCompletion(t, svc, sid)

	sess, err := svc.GetSession(sid)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess.State != "Completed" {
		t.Fatalf("expected Completed state on live scan with uTLS chrome + AutoCalibration, got %s (err=%s)", sess.State, sess.LastError)
	}

	res, err := svc.GetResultsSince(sid, 0, 100)
	if err != nil {
		t.Fatalf("GetResultsSince failed: %v", err)
	}
	t.Logf("[LIVE PARSA AUTOCALIBRATION + UTLS CHROME] state=%s results=%d", sess.State, len(res.Results))
	if len(res.Results) != 0 {
		t.Fatalf("expected AutoCalibration (-ac) to filter baseline 200 OK responses (0 false positives), got %d: %+v", len(res.Results), res.Results)
	}
}
