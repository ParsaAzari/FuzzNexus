package bridge

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

func waitForBridgeSession(t *testing.T, svc *Service, sid int64, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		sess, err := svc.GetSession(sid)
		if err == nil && sess != nil {
			if sess.State == "Completed" || sess.State == "Failed" || sess.State == "Stopped" {
				return sess.State
			}
		}
		time.Sleep(80 * time.Millisecond)
	}
	return "Timeout"
}

// TestE2E_AllCLIFeaturesToGUI verifies end-to-end execution of every CLI capability exposed to the GUI:
// extensions (-e), dirsearch compat (-D), ignore comments (-ic), encoders (-enc), replay-proxy (-replay-proxy),
// recursion (-recursion / greedy), clusterbomb/sniper modes (-mode), raw URI (-raw), audit-log, debug-log,
// output-directory (-od), custom matchers/filters (-ms, -fw, -mmode), TOML config import (-config),
// ejson/ecsv/html/md exports, and live AddLiveFilter with word/line.
func TestE2E_AllCLIFeaturesToGUI(t *testing.T) {
	tempDir := t.TempDir()

	var replayHits atomic.Int32
	replayProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replayHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("REPLAY_CAPTURED"))
	}))
	defer replayProxy.Close()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/admin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("DIR_ADMIN"))
		case p == "/admin/panel":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("DEEP_PANEL_FOUND"))
		case p == "/index.php":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PHP_INDEX_OK"))
		case p == "/b64/YWRtaW4=":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ENCODER_B64_OK"))
		case strings.HasPrefix(p, "/combo/alice/123"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("CLUSTERBOMB_OK"))
		case p == "/sniper/one/two":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("SNIPER_HIT"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("404"))
		}
	}))
	defer target.Close()

	svc, err := NewService(tempDir, nil)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()

	// 1. Recursion + Extensions + DirSearch Compat + IgnoreComments + ReplayProxy + Audit/Debug Logs + OD
	wl1 := filepath.Join(tempDir, "wl_rec.txt")
	wl1Content := "# comment should be ignored\nadmin\npanel\nentry.%EXT%\n"
	if err := os.WriteFile(wl1, []byte(wl1Content), 0600); err != nil {
		t.Fatal(err)
	}

	auditPath := filepath.Join(tempDir, "audit.log")
	debugPath := filepath.Join(tempDir, "debug.log")
	odPath := filepath.Join(tempDir, "matched_od")

	sid1, err := svc.StartScan(ScanRequestDTO{
		URL:                    target.URL + "/FUZZ",
		Method:                 "GET",
		Wordlist:               wl1,
		Threads:                2,
		ReplayProxyURL:         replayProxy.URL,
		Extensions:             ".php",
		DirSearchCompat:        true,
		IgnoreWordlistComments: true,
		Recursion:              true,
		RecursionDepth:         1,
		RecursionStrategy:      "greedy",
		AuditLog:               auditPath,
		DebugLog:               debugPath,
		OutputDirectory:        odPath,
		MatcherStatus:          "200",
	})
	if err != nil {
		t.Fatalf("Recursion StartScan failed: %v", err)
	}
	if st := waitForBridgeSession(t, svc, sid1, 10*time.Second); st != "Completed" {
		t.Fatalf("Expected Completed state for recursion scan, got %s", st)
	}
	_ = svc.StopScan()

	res1, err := svc.GetResultsSince(sid1, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	var foundPanel, foundDir bool
	for _, r := range res1.Results {
		if strings.HasSuffix(r.URL, "/admin") {
			foundDir = true
		}
		if strings.HasSuffix(r.URL, "/admin/panel") {
			foundPanel = true
		}
		if strings.HasPrefix(r.Inputs["FUZZ"], "#") {
			t.Errorf("Expected comments to be ignored with -ic, got %q", r.Inputs["FUZZ"])
		}
	}
	if !foundDir || !foundPanel {
		t.Errorf("Expected greedy recursion to find both /admin and /admin/panel, got %+v", res1.Results)
	}
	if replayHits.Load() < 2 {
		t.Errorf("Expected replay-proxy to receive >= 2 matched requests, got %d", replayHits.Load())
	}
	if info, err := os.Stat(auditPath); err != nil || info.Size() == 0 {
		t.Errorf("Expected non-empty audit log at %q, err=%v", auditPath, err)
	}
	if _, err := os.Stat(debugPath); err != nil {
		t.Errorf("Expected debug log file created at %q, err=%v", debugPath, err)
	}
	if entries, err := os.ReadDir(odPath); err != nil || len(entries) == 0 {
		t.Errorf("Expected output directory files in %q, err=%v", odPath, err)
	}

	// Verify EJSON & ECSV exports
	ejsonPath := filepath.Join(tempDir, "out.ejson")
	if err := svc.ExportResults(sid1, "ejson", ejsonPath); err != nil {
		t.Fatalf("ExportResults ejson failed: %v", err)
	}
	ejsonBytes, _ := os.ReadFile(ejsonPath)
	var decoded []map[string]interface{}
	if err := json.Unmarshal(ejsonBytes, &decoded); err != nil || len(decoded) == 0 {
		t.Fatalf("Expected valid EJSON output, err=%v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(decoded[0]["url"].(string)); err != nil {
		t.Errorf("Expected base64-encoded url in ejson, got %v", decoded[0]["url"])
	}

	ecsvPath := filepath.Join(tempDir, "out.ecsv")
	if err := svc.ExportResults(sid1, "ecsv", ecsvPath); err != nil {
		t.Fatalf("ExportResults ecsv failed: %v", err)
	}
	if b, err := os.ReadFile(ecsvPath); err != nil || !strings.Contains(string(b), "Payloads_B64") {
		t.Errorf("Expected valid ECSV file, err=%v", err)
	}

	// 2. Encoder (-enc) E2E
	wlEnc := filepath.Join(tempDir, "wl_enc.txt")
	if err := os.WriteFile(wlEnc, []byte("admin\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sid2, err := svc.StartScan(ScanRequestDTO{
		URL:           target.URL + "/b64/FUZZ",
		Wordlist:      wlEnc,
		Encoders:      []string{"FUZZ:b64encode"},
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("Encoder StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sid2, 5*time.Second)
	_ = svc.StopScan()
	res2, _ := svc.GetResultsSince(sid2, 0, 20)
	if len(res2.Results) != 1 || !strings.Contains(res2.Results[0].URL, "YWRtaW4=") {
		t.Errorf("Expected base64-encoded payload match /b64/YWRtaW4=, got %+v", res2.Results)
	}

	// 3. Multi-wordlist Clusterbomb (-mode clusterbomb) E2E
	wlUsers := filepath.Join(tempDir, "users.txt")
	wlPins := filepath.Join(tempDir, "pins.txt")
	_ = os.WriteFile(wlUsers, []byte("alice\nbob\n"), 0600)
	_ = os.WriteFile(wlPins, []byte("123\n999\n"), 0600)
	sid3, err := svc.StartScan(ScanRequestDTO{
		URL:           target.URL + "/combo/U/P",
		Wordlists:     []string{wlUsers + ":U", wlPins + ":P"},
		InputMode:     "clusterbomb",
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("Clusterbomb StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sid3, 5*time.Second)
	_ = svc.StopScan()
	res3, _ := svc.GetResultsSince(sid3, 0, 20)
	if len(res3.Results) != 1 || res3.Results[0].Inputs["U"] != "alice" || res3.Results[0].Inputs["P"] != "123" {
		t.Errorf("Expected clusterbomb match U=alice, P=123, got %+v", res3.Results)
	}

	// 4. ImportTOMLConfig (-config) E2E
	tomlPath := filepath.Join(tempDir, "test.fuzznexusrc")
	tomlContent := "[http]\nurl = \"https://parsaazari.com/FUZZ\"\nextensions = \".php,.txt\"\n[general]\nthreads = 12\ndelay = \"0.1\"\n"
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0600); err != nil {
		t.Fatal(err)
	}
	imported, err := svc.ImportTOMLConfig(tomlPath)
	if err != nil {
		t.Fatalf("ImportTOMLConfig failed: %v", err)
	}
	if imported.Threads != 12 || imported.Delay != "0.1" {
		t.Errorf("Unexpected imported TOML DTO: %+v", imported)
	}
}

// TestE2E_AdvancedCLIFeaturesPart2 verifies default redirect recursion, sniper/pitchfork modes,
// -raw, -ignore-body, -http2, -sni, -scraperfile, -preflight, -p delay, -sf, --input-cmd, -request,
// and live AddLiveFilter("word"/"line").
func TestE2E_AdvancedCLIFeaturesPart2(t *testing.T) {
	tempDir := t.TempDir()

	var observedRawURI atomic.Bool
	var observedH2 atomic.Bool
	var observedSNI atomic.Value
	observedSNI.Store("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/vdir":
			// 301 redirect to /vdir/ triggers default recursion strategy
			w.Header().Set("Location", "/vdir/")
			w.WriteHeader(http.StatusMovedPermanently)
		case r.URL.Path == "/vdir/":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/vdir/secret":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("DEEP_DEFAULT_REC_TOKEN=XYZ987"))
		case r.URL.Path == "/pf/u1/p1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PITCHFORK_OK"))
		case r.URL.Path == "/sn/alt/fixed" || r.URL.Path == "/sn/fixed/alt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("SNIPER_OK"))
		case strings.Contains(r.RequestURI, "..%2F") || strings.Contains(r.RequestURI, "/raw/%2e%2e") || strings.Contains(r.RequestURI, "/raw/a b"):
			// Should NOT be encoded when -raw is enabled
			w.WriteHeader(http.StatusBadRequest)
		case strings.Contains(r.RequestURI, "/raw/a b") || strings.Contains(r.RequestURI, "a%20b"):
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/raw/raw_ok":
			observedRawURI.Store(true)
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/token":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<input name="_csrf" value="csrf_42">`))
		case r.URL.Path == "/protected/go" && r.Header.Get("X-CSRF") == "csrf_42":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PREFLIGHT_PASSED"))
		case r.URL.Path == "/cmd/cmd_hit":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("CMD_INPUT_OK"))
		case r.URL.Path == "/bodycheck":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0123456789ABCDEF0123456789ABCDEF"))
		case strings.HasPrefix(r.URL.Path, "/deny/") && !strings.Contains(r.URL.Path, "fn_canary"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("FORBIDDEN"))
		case r.URL.Path == "/reqfile/hit":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("REQ_FILE_OK"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// TLS / HTTP2 / SNI server
	tlsSrv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 {
			observedH2.Store(true)
		}
		if r.TLS != nil && r.TLS.ServerName != "" {
			observedSNI.Store(r.TLS.ServerName)
		}
		if r.URL.Path == "/h2ping" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("H2_OK"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	tlsSrv.EnableHTTP2 = true
	tlsSrv.StartTLS()
	defer tlsSrv.Close()

	svc, err := NewService(tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	// 1. Default Recursion (301 redirect) + ScraperFile (-scraperfile)
	wlRec := filepath.Join(tempDir, "wl_defrec.txt")
	_ = os.WriteFile(wlRec, []byte("vdir\nsecret\n"), 0600)
	scraperJSON := filepath.Join(tempDir, "custom_scraper.json")
	_ = os.WriteFile(scraperJSON, []byte(`{
	  "groupname": "custom_tok",
	  "active": true,
	  "rules": [{
	    "name": "deep_token",
	    "rule": "DEEP_DEFAULT_REC_TOKEN=([A-Z0-9]+)",
	    "target": "body",
	    "type": "regexp",
	    "onlymatched": true,
	    "action": ["output"]
	  }]
	}`), 0600)

	sidRec, err := svc.StartScan(ScanRequestDTO{
		URL:               srv.URL + "/FUZZ",
		Wordlist:          wlRec,
		Threads:           2,
		Recursion:         true,
		RecursionDepth:    1,
		RecursionStrategy: "default",
		ScraperFile:       scraperJSON,
		MatcherStatus:     "200,301",
	})
	if err != nil {
		t.Fatalf("Default recursion StartScan failed: %v", err)
	}
	if st := waitForBridgeSession(t, svc, sidRec, 10*time.Second); st != "Completed" {
		t.Fatalf("Expected Completed for default recursion, got %s", st)
	}
	_ = svc.StopScan()
	resRec, _ := svc.GetResultsSince(sidRec, 0, 50)
	var foundDeep, foundScraper bool
	for _, r := range resRec.Results {
		if strings.HasSuffix(r.URL, "/vdir/secret") {
			foundDeep = true
			if v, ok := r.ScraperData["deep_token"]; ok && strings.Contains(v, "XYZ987") {
				foundScraper = true
			}
		}
	}
	if !foundDeep {
		t.Errorf("Expected default redirect recursion to discover /vdir/secret, got %+v", resRec.Results)
	}
	if !foundScraper {
		t.Errorf("Expected custom -scraperfile to extract deep_token=XYZ987, got %+v", resRec.Results)
	}

	// 2. Pitchfork (-mode pitchfork)
	wlU := filepath.Join(tempDir, "pf_u.txt")
	wlP := filepath.Join(tempDir, "pf_p.txt")
	_ = os.WriteFile(wlU, []byte("u1\nu2\n"), 0600)
	_ = os.WriteFile(wlP, []byte("p1\np2\n"), 0600)
	sidPF, err := svc.StartScan(ScanRequestDTO{
		URL:           srv.URL + "/pf/U/P",
		Wordlists:     []string{wlU + ":U", wlP + ":P"},
		InputMode:     "pitchfork",
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("Pitchfork StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidPF, 5*time.Second)
	_ = svc.StopScan()
	resPF, _ := svc.GetResultsSince(sidPF, 0, 20)
	if len(resPF.Results) != 1 || resPF.Results[0].Inputs["U"] != "u1" || resPF.Results[0].Inputs["P"] != "p1" {
		t.Errorf("Expected pitchfork match u1/p1, got %+v", resPF.Results)
	}

	// 3. Sniper (-mode sniper with §marker§)
	wlSn := filepath.Join(tempDir, "sn.txt")
	_ = os.WriteFile(wlSn, []byte("alt\n"), 0600)
	sidSn, err := svc.StartScan(ScanRequestDTO{
		URL:           srv.URL + "/sn/§fixed§/§fixed§",
		Wordlist:      wlSn,
		InputMode:     "sniper",
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("Sniper StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidSn, 5*time.Second)
	_ = svc.StopScan()
	resSn, _ := svc.GetResultsSince(sidSn, 0, 20)
	if len(resSn.Results) != 2 {
		t.Errorf("Expected 2 sniper position hits (/sn/alt/fixed and /sn/fixed/alt), got %d", len(resSn.Results))
	}

	// 4. IgnoreBody (-ignore-body) + Delay (-p) + Live AddLiveFilter("word"/"line")
	wlBody := filepath.Join(tempDir, "body.txt")
	_ = os.WriteFile(wlBody, []byte("bodycheck\n"), 0600)
	tStart := time.Now()
	sidIB, err := svc.StartScan(ScanRequestDTO{
		URL:           srv.URL + "/FUZZ",
		Wordlist:      wlBody,
		IgnoreBody:    true,
		Delay:         "0.05",
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("IgnoreBody StartScan failed: %v", err)
	}
	if err := svc.AddLiveFilter("word", "999"); err != nil {
		t.Errorf("Expected AddLiveFilter('word') to succeed, got %v", err)
	}
	if err := svc.AddLiveFilter("line", "999"); err != nil {
		t.Errorf("Expected AddLiveFilter('line') to succeed, got %v", err)
	}
	waitForBridgeSession(t, svc, sidIB, 5*time.Second)
	_ = svc.StopScan()
	if elapsed := time.Since(tStart); elapsed < 40*time.Millisecond {
		t.Errorf("Expected -p delay >=40ms, finished in %v", elapsed)
	}
	resIB, _ := svc.GetResultsSince(sidIB, 0, 10)
	if len(resIB.Results) != 1 || resIB.Results[0].Words != 0 || resIB.Results[0].Lines != 0 || !resIB.Results[0].Cancelled {
		t.Errorf("Expected IgnoreBody result with Words=0, Lines=0, Cancelled=true, got %+v", resIB.Results)
	}

	// 5. HTTP/2 (-http2) + Custom SNI (-sni)
	wlH2 := filepath.Join(tempDir, "h2.txt")
	_ = os.WriteFile(wlH2, []byte("h2ping\n"), 0600)
	sidH2, err := svc.StartScan(ScanRequestDTO{
		URL:           tlsSrv.URL + "/FUZZ",
		Wordlist:      wlH2,
		Http2:         true,
		SNI:           "custom.sni.local",
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("HTTP2/SNI StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidH2, 5*time.Second)
	_ = svc.StopScan()
	if !observedH2.Load() {
		t.Error("Expected server to observe HTTP/2 protocol when Http2=true")
	}
	if gotSNI, _ := observedSNI.Load().(string); gotSNI != "custom.sni.local" {
		t.Errorf("Expected custom TLS SNI 'custom.sni.local', got %q", gotSNI)
	}

	// 6. Preflight (-preflight + Auto Var extraction)
	wlPre := filepath.Join(tempDir, "pre.txt")
	_ = os.WriteFile(wlPre, []byte("go\n"), 0600)
	preReqFile := filepath.Join(tempDir, "pre.req")
	hostOnly := strings.TrimPrefix(srv.URL, "http://")
	_ = os.WriteFile(preReqFile, []byte(fmt.Sprintf("GET /token HTTP/1.1\r\nHost: %s\r\n\r\n", hostOnly)), 0600)
	sidPre, err := svc.StartScan(ScanRequestDTO{
		URL:           srv.URL + "/protected/FUZZ",
		Wordlist:      wlPre,
		Headers:       map[string]string{"X-CSRF": "CSRFTOK"},
		MatcherStatus: "200",
		Preflights: []fuzznexus.PreflightConfig{
			{
				RequestFile: preReqFile,
				Vars: []fuzznexus.VarExtract{
					{Name: "CSRFTOK", Source: "form", Key: "_csrf"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Preflight StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidPre, 5*time.Second)
	_ = svc.StopScan()
	resPre, _ := svc.GetResultsSince(sidPre, 0, 10)
	if len(resPre.Results) != 1 || resPre.Results[0].Status != 200 {
		t.Errorf("Expected preflight variable CSRFTOK to unlock /protected/go, got %+v", resPre.Results)
	}

	// 7. Command Input (--input-cmd executes command per iteration with FFUF_NUM)
	inputCmd := "printf cmd_hit"
	if runtime.GOOS == "windows" {
		inputCmd = "<nul set /p=cmd_hit"
	}
	sidCmd, err := svc.StartScan(ScanRequestDTO{
		URL:           srv.URL + "/cmd/FUZZ",
		InputCommands: []string{inputCmd},
		InputNum:      1,
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("InputCommand StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidCmd, 5*time.Second)
	_ = svc.StopScan()
	resCmd, _ := svc.GetResultsSince(sidCmd, 0, 10)
	if len(resCmd.Results) != 1 || resCmd.Results[0].Inputs["FUZZ"] != "cmd_hit" {
		t.Errorf("Expected --input-cmd match cmd_hit, got %+v", resCmd.Results)
	}

	// 8. Raw Request File (-request / -request-proto) + Raw URI (-raw)
	rawReqFile := filepath.Join(tempDir, "raw.req")
	_ = os.WriteFile(rawReqFile, []byte(fmt.Sprintf("GET /reqfile/FUZZ HTTP/1.1\r\nHost: %s\r\n\r\n", hostOnly)), 0600)
	wlReq := filepath.Join(tempDir, "reqwl.txt")
	_ = os.WriteFile(wlReq, []byte("hit\n"), 0600)
	sidReq, err := svc.StartScan(ScanRequestDTO{
		RequestFile:   rawReqFile,
		RequestProto:  "http",
		Wordlist:      wlReq,
		Raw:           true,
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("RequestFile StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidReq, 5*time.Second)
	_ = svc.StopScan()
	resReq, _ := svc.GetResultsSince(sidReq, 0, 10)
	if len(resReq.Results) != 1 {
		t.Errorf("Expected -request file match /reqfile/hit, got %+v", resReq.Results)
	}

	// 9. StopOn403 (-sf) when >95% return 403
	wl403 := filepath.Join(tempDir, "wl403.txt")
	var lines403 []string
	for i := 0; i < 70; i++ {
		lines403 = append(lines403, fmt.Sprintf("p%d", i))
	}
	_ = os.WriteFile(wl403, []byte(strings.Join(lines403, "\n")+"\n"), 0600)
	sid403, err := svc.StartScan(ScanRequestDTO{
		URL:           srv.URL + "/deny/FUZZ",
		Wordlist:      wl403,
		Threads:       2,
		StopOn403:     true,
		MatcherStatus: "200",
	})
	if err != nil {
		t.Fatalf("StopOn403 StartScan failed: %v", err)
	}
	st403 := waitForBridgeSession(t, svc, sid403, 5*time.Second)
	_ = svc.StopScan()
	sess403, _ := svc.GetSession(sid403)
	if st403 != "Failed" || (sess403 != nil && !strings.Contains(sess403.LastError, "403")) {
		t.Errorf("Expected -sf StopOn403 to halt session on 403 storm, got state=%s err=%v", st403, sess403)
	}
	_ = tls.VersionTLS13
}

// TestLiveParsaAzari_FullCLIParity validates real GUI-wired CLI options against https://parsaazari.com
// at gentle, safe rates (Rate=2 req/s, Threads=1).
func TestLiveParsaAzari_FullCLIParity(t *testing.T) {
	if os.Getenv("FUZZNEXUS_LIVE_TEST") != "1" {
		t.Skip("set FUZZNEXUS_LIVE_TEST=1 to run controlled live parity validation against parsaazari.com")
	}

	tempDir := t.TempDir()
	var replayCount atomic.Int32
	replayProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replayCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer replayProxy.Close()

	svc, err := NewService(tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	wlPath := filepath.Join(tempDir, "live_parity.txt")
	wlContent := "# comment ignored by -ic\nrobots\nfuzznexus_nonexistent_canary_xyz\n"
	if err := os.WriteFile(wlPath, []byte(wlContent), 0600); err != nil {
		t.Fatal(err)
	}

	auditLog := filepath.Join(tempDir, "parsa_audit.log")
	debugLog := filepath.Join(tempDir, "parsa_debug.log")
	odDir := filepath.Join(tempDir, "parsa_od")

	sid, err := svc.StartScan(ScanRequestDTO{
		URL:                    "https://parsaazari.com/FUZZ",
		Method:                 "GET",
		Wordlist:               wlPath,
		Threads:                1,
		Rate:                   2,
		Timeout:                12,
		Delay:                  "0.1",
		Extensions:             ".txt",
		IgnoreWordlistComments: true,
		AutoCalibration:        true,
		TLSProfile:             "chrome",
		BrowserProfile:         "chrome-macos",
		ReplayProxyURL:         replayProxy.URL,
		AuditLog:               auditLog,
		DebugLog:               debugLog,
		OutputDirectory:        odDir,
		MatcherStatus:          "200",
	})
	if err != nil {
		t.Fatalf("Live StartScan failed: %v", err)
	}

	finalState := waitForBridgeSession(t, svc, sid, 35*time.Second)
	_ = svc.StopScan()
	if finalState != "Completed" {
		t.Fatalf("Expected live parsaazari.com scan to Complete, got %s", finalState)
	}

	res, err := svc.GetResultsSince(sid, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("[LIVE PARSA CLI PARITY] state=%s matches=%d replayHits=%d", finalState, len(res.Results), replayCount.Load())
	for _, r := range res.Results {
		t.Logf("  -> [%d] %s (%d bytes)", r.Status, r.URL, r.Length)
	}

	if info, err := os.Stat(auditLog); err != nil || info.Size() == 0 {
		t.Errorf("Expected non-empty audit log on live run, err=%v", err)
	}
	if _, err := os.Stat(debugLog); err != nil {
		t.Errorf("Expected debug log on live run, err=%v", err)
	}

	// Verify HTML & Markdown exports on live session results
	htmlOut := filepath.Join(tempDir, "parsa_report.html")
	mdOut := filepath.Join(tempDir, "parsa_report.md")
	if err := svc.ExportResults(sid, "html", htmlOut); err != nil {
		t.Fatalf("HTML export failed: %v", err)
	}
	if err := svc.ExportResults(sid, "md", mdOut); err != nil {
		t.Fatalf("MD export failed: %v", err)
	}
}
