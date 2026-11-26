package bridge

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

// TestCLIToGUIFlagCoverage is a strict schema guard using reflection over fuzznexus.ConfigOptions
// and bridge.ScanRequestDTO. If a new CLI flag is added to pkg/ffuf/optionsparser.go or
// pkg/ffuf/flags.go, CI fails unless it is mapped into ScanRequestDTO or explicitly
// justified in cliOnlyAllowlist.
func TestCLIToGUIFlagCoverage(t *testing.T) {
	// Collect every flag registered by ffuf CLI
	fs := flag.NewFlagSet("parity-check", flag.ContinueOnError)
	opts := fuzznexus.NewConfigOptions()
	reg := fuzznexus.RegisterFlags(fs, opts)

	// Map every CLI flag name -> backing ScanRequestDTO JSON field(s)
	cliToGUIDTO := map[string]string{
		// HTTP
		"u":                      "url",
		"X":                      "method",
		"H":                      "headers",
		"d":                      "data",
		"r":                      "follow_redirects",
		"timeout":                "timeout",
		"x":                      "proxy_url",
		"replay-proxy":           "replay_proxy_url",
		"raw":                    "raw",
		"http2":                  "http2",
		"ignore-body":            "ignore_body",
		"recursion":              "recursion",
		"recursion-depth":        "recursion_depth",
		"recursion-strategy":     "recursion_strategy",
		"sni":                    "sni",
		"cc":                     "client_cert",
		"ck":                     "client_key",
		"tls-profile":            "tls_profile",
		"close-conn-on-throttle": "close_conn_on_throttle",
		"detect-soft-block":      "detect_soft_block",
		"preflight":              "preflights",
		"preflight-var":          "preflights",
		"preflight-var-auto":     "preflights",
		"postflight":             "postflights",
		"postflight-var":         "postflights",
		"postflight-var-auto":    "postflights",
		"preflight-mode":         "preflight_mode",
		"preflight-error":        "preflight_error",
		"preflight-anyhost":      "preflight_anyhost",
		"b":                      "headers", // Cookie header in GUI

		// General
		"t":           "threads",
		"rate":        "rate",
		"p":           "delay",
		"maxtime":     "max_time",
		"maxtime-job": "max_time_job",
		"ac":          "auto_calibration",
		"ach":         "auto_calibration_per_host",
		"ack":         "auto_calibration_keyword",
		"acs":         "auto_calibration_strategies",
		"acc":         "auto_calibration_strings",
		"sf":          "stop_on_403",
		"se":          "stop_on_errors",
		"sa":          "stop_on_all",
		"scrapers":    "scrapers",
		"scraperfile": "scraper_file",
		"config":      "ImportTOMLConfig", // Dedicated bridge method
		"search":      "SearchFNHASH",     // Dedicated bridge method

		// Matcher
		"mc":    "matcher_status",
		"ms":    "matcher_size",
		"mw":    "matcher_words",
		"ml":    "matcher_lines",
		"mr":    "matcher_regexp",
		"mt":    "matcher_time",
		"mmode": "matcher_mode",

		// Filter
		"fc":    "filter_status",
		"fs":    "filter_size",
		"fw":    "filter_words",
		"fl":    "filter_lines",
		"fr":    "filter_regexp",
		"ft":    "filter_time",
		"fmode": "filter_mode",

		// Input
		"w":             "wordlists",
		"mode":          "input_mode",
		"e":             "extensions",
		"D":             "dirsearch_compat",
		"ic":            "ignore_wordlist_comments",
		"enc":           "encoders",
		"request":       "request_file",
		"request-proto": "request_proto",
		"input-cmd":     "input_commands",
		"input-num":     "input_num",
		"input-shell":   "input_shell",

		// Output
		"audit-log": "audit_log",
		"debug-log": "debug_log",
		"od":        "output_directory",
		"o":         "ExportResults", // Dedicated bridge method
		"of":        "ExportResults", // Dedicated bridge method
	}

	// Explicitly CLI/stdout-only flags (either terminal UX or cURL compat dummies)
	cliOnlyAllowlist := map[string]bool{
		"V":              true, // Version shown natively in TitleBar / AboutModal
		"c":              true, // ANSI terminal colors (GUI uses theme palettes)
		"json":           true, // Terminal NDJSON stdout stream (GUI uses live SQLite/events + JSON export)
		"noninteractive": true, // Terminal interactive console toggle
		"s":              true, // Terminal silent mode
		"v":              true, // Terminal verbose stdout (GUI Inspector displays full URL & redirects)
		"or":             true, // Skip empty CLI output file (GUI Export warns if empty)
		// Compat aliases & cURL dummies
		"cookie":      true,
		"data":        true,
		"data-ascii":  true,
		"data-binary": true,
		"compressed":  true,
		"i":           true,
		"k":           true,
	}

	// Build set of actual JSON field tags on ScanRequestDTO via reflection
	dtoFields := map[string]bool{}
	rt := reflect.TypeOf(ScanRequestDTO{})
	for i := 0; i < rt.NumField(); i++ {
		tag := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			dtoFields[tag] = true
		}
	}
	svcType := reflect.TypeOf(&Service{})
	for i := 0; i < svcType.NumMethod(); i++ {
		dtoFields[svcType.Method(i).Name] = true
	}

	fs.VisitAll(func(f *flag.Flag) {
		if cliOnlyAllowlist[f.Name] {
			return
		}
		target, mapped := cliToGUIDTO[f.Name]
		if !mapped {
			t.Errorf("CLI flag -%s (section=%q) is NOT mapped to GUI or allowlisted!", f.Name, func() string { s, _ := reg.SectionOf(f.Name); return s }())
			return
		}
		if !dtoFields[target] {
			t.Errorf("CLI flag -%s maps to %q, which does not exist on ScanRequestDTO or Service!", f.Name, target)
		}
	})
}

type wireRecord struct {
	Method string
	URI    string
	Body   string
	Header http.Header
}

// TestDifferential_CLIvsGUI_ConfigAndWireParity runs identical fuzz scenarios through:
// (A) Pure CLI flag parsing + ConfigFromOptions + filter.FromConfig + assembly.BuildJob
// (B) GUI bridge.StartScan(ScanRequestDTO)
// and verifies exact wire-level request equivalence and identical matched results.
func TestDifferential_CLIvsGUI_ConfigAndWireParity(t *testing.T) {
	tempDir := t.TempDir()

	var mu sync.Mutex
	var cliRequests []wireRecord
	var guiRequests []wireRecord
	recordingToGUI := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		rec := wireRecord{
			Method: r.Method,
			URI:    r.RequestURI,
			Body:   string(b),
			Header: r.Header.Clone(),
		}
		if recordingToGUI {
			guiRequests = append(guiRequests, rec)
		} else {
			cliRequests = append(cliRequests, rec)
		}
		mu.Unlock()

		if r.URL.Path == "/api/hit.php" && r.URL.Query().Get("q") == "aGl0" && r.Header.Get("X-Token") == "t1" {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("MATCH_UNIQUE_12345 OK"))
			return
		}
		if r.URL.Path == "/api/noise" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("NOISE_WORD"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404"))
	}))
	defer srv.Close()

	wlPath := filepath.Join(tempDir, "words.txt")
	_ = os.WriteFile(wlPath, []byte("# comment\nhit\nnoise\n"), 0600)

	// --- (A) Pure CLI Run ---
	cliCtx, cliCancel := context.WithCancel(context.Background())
	defer cliCancel()

	cliOpts := fuzznexus.NewConfigOptions()
	fs := flag.NewFlagSet("cli-diff", flag.ContinueOnError)
	fuzznexus.RegisterFlags(fs, cliOpts)
	err := fs.Parse([]string{
		"-u", srv.URL + "/api/FUZZ?q=ENC",
		"-X", "POST",
		"-H", "X-Token: t1",
		"-d", `{"k":"FUZZ"}`,
		"-w", wlPath + ":FUZZ",
		"-w", wlPath + ":ENC",
		"-mode", "clusterbomb",
		"-e", ".php",
		"-ic",
		"-enc", "ENC:b64encode",
		"-mc", "200,201",
		"-mr", "MATCH_UNIQUE",
		"-mmode", "and",
		"-fw", "1",
		"-t", "1",
		"-noninteractive",
		"-s",
	})
	if err != nil {
		t.Fatalf("CLI flag parse failed: %v", err)
	}
	cliConf, err := fuzznexus.ConfigFromOptions(cliOpts, cliCtx, cliCancel)
	if err != nil {
		t.Fatalf("CLI ConfigFromOptions failed: %v", err)
	}
	cliMM, err := filter.FromConfig(cliOpts, true)
	if err != nil {
		t.Fatalf("CLI filter.FromConfig failed: %v", err)
	}
	cliConf.MatcherManager = cliMM
	cliJob, err := assembly.BuildJob(cliConf)
	if err != nil {
		t.Fatalf("CLI BuildJob failed: %v", err)
	}
	cliJob.Start()
	cliResults := cliJob.Output.GetCurrentResults()

	// --- (B) GUI Bridge Run ---
	svc, err := NewService(filepath.Join(tempDir, "gui_data"), nil)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()

	mu.Lock()
	recordingToGUI = true
	mu.Unlock()

	sid, err := svc.StartScan(ScanRequestDTO{
		URL:                    srv.URL + "/api/FUZZ?q=ENC",
		Method:                 "POST",
		Headers:                map[string]string{"X-Token": "t1"},
		Data:                   `{"k":"FUZZ"}`,
		Wordlists:              []string{wlPath + ":FUZZ", wlPath + ":ENC"},
		InputMode:              "clusterbomb",
		Extensions:             ".php",
		IgnoreWordlistComments: true,
		Encoders:               []string{"ENC:b64encode"},
		MatcherStatus:          "200,201",
		MatcherRegexp:          "MATCH_UNIQUE",
		MatcherMode:            "and",
		FilterWords:            "1",
		Threads:                1,
		AutoCalibration:        false,
	})
	if err != nil {
		t.Fatalf("GUI StartScan failed: %v", err)
	}
	if st := waitForBridgeSession(t, svc, sid, 8*time.Second); st != "Completed" {
		t.Fatalf("GUI scan expected Completed, got %s", st)
	}
	_ = svc.StopScan()
	guiPage, err := svc.GetResultsSince(sid, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	// Compare wire requests (normalized/sorted by URI+Body)
	normReqs := func(in []wireRecord) []string {
		var out []string
		for _, r := range in {
			out = append(out, fmt.Sprintf("%s %s body=%s xtok=%s", r.Method, r.URI, r.Body, r.Header.Get("X-Token")))
		}
		sort.Strings(out)
		return out
	}
	cliWire := normReqs(cliRequests)
	guiWire := normReqs(guiRequests)
	if !reflect.DeepEqual(cliWire, guiWire) {
		t.Fatalf("CLI vs GUI wire request mismatch!\nCLI (%d): %v\nGUI (%d): %v", len(cliWire), cliWire, len(guiWire), guiWire)
	}

	// Compare matched results
	if len(cliResults) != 1 || len(guiPage.Results) != 1 {
		t.Fatalf("CLI vs GUI result count mismatch: CLI=%d GUI=%d", len(cliResults), len(guiPage.Results))
	}
	if cliResults[0].StatusCode != guiPage.Results[0].Status ||
		cliResults[0].Url != guiPage.Results[0].URL ||
		string(cliResults[0].Input["FUZZ"]) != guiPage.Results[0].Inputs["FUZZ"] ||
		string(cliResults[0].Input["ENC"]) != guiPage.Results[0].Inputs["ENC"] {
		t.Fatalf("CLI vs GUI matched result mismatch:\nCLI=%+v\nGUI=%+v", cliResults[0], guiPage.Results[0])
	}
}

func generateSelfSignedCertPair(t *testing.T, dir, prefix string) (certPath, keyPath string, certPEM []byte) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: prefix},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost", "custom.sni.local"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	certPath = filepath.Join(dir, prefix+".crt")
	keyPath = filepath.Join(dir, prefix+".key")
	_ = os.WriteFile(certPath, certPEM, 0600)
	_ = os.WriteFile(keyPath, keyPEM, 0600)
	return certPath, keyPath, certPEM
}

// TestCombinatorial_DeepChainBugs executes multi-flag pairwise scenarios specifically designed
// to uncover chained interactions between uTLS, mTLS, Proxy, Recursion, FollowRedirects,
// AutoCalibration (-acc/-ack/-ach), TOML Import, and Session Cloning.
func TestCombinatorial_DeepChainBugs(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Pairwise Matrix: TLSProfile ("default", "chrome", "firefox") x mTLS ClientCert (-cc/-ck) x SNI
	_, _, clientPEM := generateSelfSignedCertPair(t, tempDir, "mtls_client")
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(clientPEM)
	clientCertPath := filepath.Join(tempDir, "mtls_client.crt")
	clientKeyPath := filepath.Join(tempDir, "mtls_client.key")

	mtlsSrv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("NO_CLIENT_CERT"))
			return
		}
		if r.URL.Path == "/mtls_ok" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("MTLS_VERIFIED"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	mtlsSrv.EnableHTTP2 = true
	mtlsSrv.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCAs,
	}
	mtlsSrv.StartTLS()
	defer mtlsSrv.Close()

	svc, err := NewService(tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	wl := filepath.Join(tempDir, "wl_mtls.txt")
	_ = os.WriteFile(wl, []byte("mtls_ok\n"), 0600)

	for _, tlsProf := range []string{"", "chrome", "firefox"} {
		t.Run("uTLS_"+tlsProf+"_with_mTLS", func(t *testing.T) {
			sid, err := svc.StartScan(ScanRequestDTO{
				URL:           mtlsSrv.URL + "/FUZZ",
				Wordlist:      wl,
				TLSProfile:    tlsProf,
				SNI:           "custom.sni.local",
				ClientCert:    clientCertPath,
				ClientKey:     clientKeyPath,
				MatcherStatus: "200",
			})
			if err != nil {
				t.Fatalf("StartScan failed: %v", err)
			}
			waitForBridgeSession(t, svc, sid, 5*time.Second)
			_ = svc.StopScan()
			res, _ := svc.GetResultsSince(sid, 0, 10)
			if len(res.Results) != 1 || res.Results[0].Status != 200 {
				t.Errorf("Chained bug: TLSProfile=%q + mTLS (-cc/-ck) failed to present client cert (results=%+v)", tlsProf, res.Results)
			}
		})
	}

	// 2. Pairwise Matrix: -acc (AutoCalibrationStrings) WITHOUT explicit -ac flag should imply -ac!
	var calibProbeSeen atomic.Bool
	acSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/CUSTOM_CALIB_PROBE" {
			calibProbeSeen.Store(true)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("SOFT404_IDENTICAL_BODY"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("SOFT404_IDENTICAL_BODY"))
	}))
	defer acSrv.Close()

	wlAC := filepath.Join(tempDir, "wl_ac.txt")
	_ = os.WriteFile(wlAC, []byte("test1\ntest2\n"), 0600)
	sidAC, err := svc.StartScan(ScanRequestDTO{
		URL:                    acSrv.URL + "/ENDPOINT",
		Wordlists:              []string{wlAC + ":ENDPOINT"},
		AutoCalibration:        false, // intentionally false: -acc should imply -ac!
		AutoCalibrationKeyword: "ENDPOINT",
		AutoCalibrationStrings: []string{"CUSTOM_CALIB_PROBE"},
		MatcherStatus:          "200",
	})
	if err != nil {
		t.Fatalf("AutoCalibrationStrings StartScan failed: %v", err)
	}
	waitForBridgeSession(t, svc, sidAC, 5*time.Second)
	_ = svc.StopScan()
	if !calibProbeSeen.Load() {
		t.Errorf("Chained bug: setting AutoCalibrationStrings (-acc) without AutoCalibration=true did not trigger calibration!")
	}
	resAC, _ := svc.GetResultsSince(sidAC, 0, 10)
	if len(resAC.Results) != 0 {
		t.Errorf("Expected -acc + -ack=ENDPOINT to auto-filter Soft-404 baseline, got %d results", len(resAC.Results))
	}

	// 3. Round-Trip Parity: Complex ScanRequestDTO -> SQLite Session -> Config JSON -> ImportTOMLConfig
	tomlFile := filepath.Join(tempDir, "roundtrip.fuzznexusrc")
	tomlData := fmt.Sprintf(`[http]
url = "%s/FUZZ"
method = "PUT"
raw = true
http2 = true
ignorebody = true
sni = "custom.sni.local"
tlsprofile = "chrome"
replayproxyurl = "http://127.0.0.1:8080"
recursion = true
recursiondepth = 3
recursionstrategy = "greedy"
[general]
threads = 15
rate = 25
delay = "0.05-0.1"
maxtime = 60
stopon403 = true
autocalibration = true
autocalibrationperhost = true
autocalibrationkeyword = "FUZZ"
[input]
extensions = ".json,.bak"
dirsearchcompat = true
ignorewordlistcomments = true
encoders = ["FUZZ:urlencode b64encode"]
inputmode = "pitchfork"
wordlists = [%s]
`, mtlsSrv.URL, strconv.Quote(wl+":FUZZ"))
	if err := os.WriteFile(tomlFile, []byte(tomlData), 0600); err != nil {
		t.Fatal(err)
	}
	imported, err := svc.ImportTOMLConfig(tomlFile)
	if err != nil {
		t.Fatalf("ImportTOMLConfig failed: %v", err)
	}
	if imported.Method != "PUT" || !imported.Raw || !imported.Http2 || !imported.IgnoreBody ||
		imported.ReplayProxyURL != "http://127.0.0.1:8080" || imported.RecursionDepth != 3 ||
		imported.RecursionStrategy != "greedy" || !imported.AutoCalibrationPerHost ||
		len(imported.Encoders) != 1 || imported.Encoders[0] != "FUZZ:urlencode b64encode" ||
		imported.InputMode != "pitchfork" {
		t.Fatalf("Round-trip TOML import dropped fields: %+v", imported)
	}
	// Verify stored session JSON preserves redacted headers & full options
	sessDTO, err := svc.GetSession(sidAC)
	if err != nil || sessDTO == nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	var restored ScanRequestDTO
	if err := json.Unmarshal([]byte(sessDTO.ConfigRedactedJSON), &restored); err != nil {
		t.Fatalf("Session JSON unmarshal failed: %v", err)
	}
	if restored.AutoCalibrationKeyword != "ENDPOINT" || len(restored.AutoCalibrationStrings) != 1 {
		t.Errorf("Session JSON failed to retain -ack / -acc: %+v", restored)
	}
}
