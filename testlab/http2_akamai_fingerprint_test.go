//go:build testlab

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type H2FingerprintDetails struct {
	SettingsStr      string   `json:"settings_str"`
	WindowUpdate     string   `json:"window_update"`
	Priority         string   `json:"priority"`
	PseudoHeaders    []string `json:"pseudo_headers"`
	PseudoOrderStr   string   `json:"pseudo_order_str"`
	AkamaiShorthand  string   `json:"akamai_shorthand"`
	ChromeReference  string   `json:"chrome_reference"`
	FirefoxReference string   `json:"firefox_reference"`
	MatchesChrome    bool     `json:"matches_chrome"`
	MatchesFirefox   bool     `json:"matches_firefox"`
	GapAnalysis      []string `json:"gap_analysis"`
}

// SniffHTTP2Fingerprint listens on TLS with ALPN h2 and captures client frames
func SniffHTTP2Fingerprint(t *testing.T, cert tls.Certificate, timeout time.Duration) (string, H2FingerprintDetails, func(), error) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2"},
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		return "", H2FingerprintDetails{}, nil, err
	}

	addr := ln.Addr().String()
	detailsChan := make(chan H2FingerprintDetails, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			return
		}
		if err := tlsConn.Handshake(); err != nil {
			return
		}

		// Read HTTP/2 Client Preface: 24 bytes "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
		preface := make([]byte, 24)
		if _, err := io.ReadFull(tlsConn, preface); err != nil {
			return
		}

		framer := http2.NewFramer(tlsConn, tlsConn)
		// Server sends its SETTINGS frame
		_ = framer.WriteSettings(
			http2.Setting{ID: http2.SettingMaxConcurrentStreams, Val: 100},
			http2.Setting{ID: http2.SettingInitialWindowSize, Val: 65535},
		)

		var capturedDetails H2FingerprintDetails
		capturedDetails.WindowUpdate = "0"
		capturedDetails.Priority = "0"

		var pseudoList []string
		decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
			if strings.HasPrefix(f.Name, ":") {
				switch f.Name {
				case ":method":
					pseudoList = append(pseudoList, "m")
				case ":authority":
					pseudoList = append(pseudoList, "a")
				case ":scheme":
					pseudoList = append(pseudoList, "s")
				case ":path":
					pseudoList = append(pseudoList, "p")
				default:
					pseudoList = append(pseudoList, f.Name)
				}
			}
		})

		deadline := time.Now().Add(timeout)
		_ = conn.SetReadDeadline(deadline)

		gotSettings := false
		gotHeaders := false

		for {
			frame, err := framer.ReadFrame()
			if err != nil {
				break
			}

			switch f := frame.(type) {
			case *http2.SettingsFrame:
				if !f.IsAck() {
					var settingsParts []string
					for i := 0; i < f.NumSettings(); i++ {
						s := f.Setting(i)
						settingsParts = append(settingsParts, fmt.Sprintf("%d:%d", s.ID, s.Val))
					}
					capturedDetails.SettingsStr = strings.Join(settingsParts, ";")
					gotSettings = true
					// Send ACK
					_ = framer.WriteSettingsAck()
				}
			case *http2.WindowUpdateFrame:
				if f.StreamID == 0 {
					capturedDetails.WindowUpdate = fmt.Sprintf("%d", f.Increment)
				}
			case *http2.PriorityFrame:
				capturedDetails.Priority = fmt.Sprintf("%d:%d:%d", f.StreamDep, f.Weight, btoi(f.Exclusive))
			case *http2.HeadersFrame:
				_, _ = decoder.Write(f.HeaderBlockFragment())
				gotHeaders = true

				// Send back a 200 OK headers frame so client is satisfied
				var headerBuf bytes.Buffer
				enc := hpack.NewEncoder(&headerBuf)
				_ = enc.WriteField(hpack.HeaderField{Name: ":status", Value: "200"})
				_ = framer.WriteHeaders(http2.HeadersFrameParam{
					StreamID:      f.StreamID,
					BlockFragment: headerBuf.Bytes(),
					EndHeaders:    true,
					EndStream:     true,
				})
			}

			if gotSettings && gotHeaders {
				break
			}
		}

		capturedDetails.PseudoHeaders = pseudoList
		capturedDetails.PseudoOrderStr = strings.Join(pseudoList, ",")
		capturedDetails.AkamaiShorthand = fmt.Sprintf("%s|%s|%s|%s",
			capturedDetails.SettingsStr,
			capturedDetails.WindowUpdate,
			capturedDetails.Priority,
			capturedDetails.PseudoOrderStr,
		)
		capturedDetails.ChromeReference = "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p"
		capturedDetails.FirefoxReference = "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s"

		capturedDetails.MatchesChrome = capturedDetails.AkamaiShorthand == capturedDetails.ChromeReference
		capturedDetails.MatchesFirefox = capturedDetails.AkamaiShorthand == capturedDetails.FirefoxReference

		// Analyze gaps
		var gaps []string
		if capturedDetails.SettingsStr != "1:65536;2:0;4:6291456;6:262144" {
			gaps = append(gaps, fmt.Sprintf("SETTINGS frame mismatch: Got '%s' vs Chrome '%s'", capturedDetails.SettingsStr, "1:65536;2:0;4:6291456;6:262144"))
		}
		if capturedDetails.WindowUpdate != "15663105" {
			gaps = append(gaps, fmt.Sprintf("WINDOW_UPDATE increment mismatch: Got '%s' vs Chrome '15663105'", capturedDetails.WindowUpdate))
		}
		if capturedDetails.PseudoOrderStr != "m,a,s,p" {
			gaps = append(gaps, fmt.Sprintf("Pseudo-Header order mismatch: Got '%s' vs Chrome 'm,a,s,p'", capturedDetails.PseudoOrderStr))
		}
		capturedDetails.GapAnalysis = gaps

		detailsChan <- capturedDetails
	}()

	cleanup := func() {
		_ = ln.Close()
	}

	// Wait or return wrapper
	return addr, H2FingerprintDetails{}, func() {
		cleanup()
	}, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// -----------------------------------------------------------------------------
// 3) Test and Sniff HTTP/2 Akamai Fingerprint for ffuf runner
// -----------------------------------------------------------------------------
func TestPhase4_HTTP2_Akamai_Fingerprinting(t *testing.T) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("=== TESTING HTTP/2 AKAMAI SHORTHAND FINGERPRINTING ===")

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2"},
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	detailsChan := make(chan H2FingerprintDetails, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			return
		}

		preface := make([]byte, 24)
		if _, err := io.ReadFull(tlsConn, preface); err != nil {
			return
		}

		framer := http2.NewFramer(tlsConn, tlsConn)
		_ = framer.WriteSettings(
			http2.Setting{ID: http2.SettingMaxConcurrentStreams, Val: 100},
			http2.Setting{ID: http2.SettingInitialWindowSize, Val: 65535},
		)

		var capturedDetails H2FingerprintDetails
		capturedDetails.WindowUpdate = "0"
		capturedDetails.Priority = "0"

		var pseudoList []string
		decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
			if strings.HasPrefix(f.Name, ":") {
				switch f.Name {
				case ":method":
					pseudoList = append(pseudoList, "m")
				case ":authority":
					pseudoList = append(pseudoList, "a")
				case ":scheme":
					pseudoList = append(pseudoList, "s")
				case ":path":
					pseudoList = append(pseudoList, "p")
				default:
					pseudoList = append(pseudoList, f.Name)
				}
			}
		})

		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		gotSettings := false
		gotHeaders := false

		for {
			frame, err := framer.ReadFrame()
			if err != nil {
				break
			}

			switch f := frame.(type) {
			case *http2.SettingsFrame:
				if !f.IsAck() {
					var settingsParts []string
					for i := 0; i < f.NumSettings(); i++ {
						s := f.Setting(i)
						settingsParts = append(settingsParts, fmt.Sprintf("%d:%d", s.ID, s.Val))
					}
					capturedDetails.SettingsStr = strings.Join(settingsParts, ";")
					gotSettings = true
					_ = framer.WriteSettingsAck()
				}
			case *http2.WindowUpdateFrame:
				if f.StreamID == 0 {
					capturedDetails.WindowUpdate = fmt.Sprintf("%d", f.Increment)
				}
			case *http2.PriorityFrame:
				capturedDetails.Priority = fmt.Sprintf("%d:%d:%d", f.StreamDep, f.Weight, btoi(f.Exclusive))
			case *http2.HeadersFrame:
				_, _ = decoder.Write(f.HeaderBlockFragment())
				gotHeaders = true

				var headerBuf bytes.Buffer
				enc := hpack.NewEncoder(&headerBuf)
				_ = enc.WriteField(hpack.HeaderField{Name: ":status", Value: "200"})
				_ = framer.WriteHeaders(http2.HeadersFrameParam{
					StreamID:      f.StreamID,
					BlockFragment: headerBuf.Bytes(),
					EndHeaders:    true,
					EndStream:     true,
				})
			}

			if gotSettings && gotHeaders {
				break
			}
		}

		capturedDetails.PseudoHeaders = pseudoList
		capturedDetails.PseudoOrderStr = strings.Join(pseudoList, ",")
		capturedDetails.AkamaiShorthand = fmt.Sprintf("%s|%s|%s|%s",
			capturedDetails.SettingsStr,
			capturedDetails.WindowUpdate,
			capturedDetails.Priority,
			capturedDetails.PseudoOrderStr,
		)
		capturedDetails.ChromeReference = "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p"
		capturedDetails.FirefoxReference = "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s"

		capturedDetails.MatchesChrome = capturedDetails.AkamaiShorthand == capturedDetails.ChromeReference
		capturedDetails.MatchesFirefox = capturedDetails.AkamaiShorthand == capturedDetails.FirefoxReference

		var gaps []string
		if capturedDetails.SettingsStr != "1:65536;2:0;4:6291456;6:262144" {
			gaps = append(gaps, fmt.Sprintf("SETTINGS: '%s' (Chrome: '1:65536;2:0;4:6291456;6:262144')", capturedDetails.SettingsStr))
		}
		if capturedDetails.WindowUpdate != "15663105" {
			gaps = append(gaps, fmt.Sprintf("WINDOW_UPDATE: '%s' (Chrome: '15663105')", capturedDetails.WindowUpdate))
		}
		if capturedDetails.PseudoOrderStr != "m,a,s,p" {
			gaps = append(gaps, fmt.Sprintf("Pseudo-Header order: '%s' (Chrome: 'm,a,s,p')", capturedDetails.PseudoOrderStr))
		}
		capturedDetails.GapAnalysis = gaps

		detailsChan <- capturedDetails
	}()

	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "wl_h2.txt")
	_ = os.WriteFile(wlPath, []byte("h2probe\n"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = "https://" + addr + "/FUZZ"
	opts.Input.Wordlists = []string{wlPath}
	opts.HTTP.Http2 = true
	opts.HTTP.TLSProfile = "default"
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.Filter.Status = "200"

	conf, _ := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	mm, _ := filter.FromConfig(opts, true)
	conf.MatcherManager = mm
	job, _ := assembly.BuildJob(conf)
	go job.Start()

	var details H2FingerprintDetails
	select {
	case details = <-detailsChan:
		t.Logf("HTTP/2 Client Fingerprint captured successfully!")
	case <-time.After(4 * time.Second):
		t.Fatalf("Timeout waiting for HTTP/2 client connection")
	}
	cancel()

	t.Logf("Akamai Shorthand Captured: %s", details.AkamaiShorthand)
	t.Logf("Chrome Reference:         %s", details.ChromeReference)
	t.Logf("Matches Chrome:           %v", details.MatchesChrome)
	t.Logf("Gaps Identified:          %v", details.GapAnalysis)

	// Save to JSON
	data, _ := json.MarshalIndent(details, "", "  ")
	_ = os.WriteFile("phase4_http2_fingerprint_results.json", data, 0644)

	fmt.Println("\n========================================================")
	fmt.Println("HTTP/2 AKAMAI SHORTHAND FINGERPRINT REPORT")
	fmt.Println("========================================================")
	fmt.Printf("Captured Shorthand:   %s\n", details.AkamaiShorthand)
	fmt.Printf("Chrome Reference:     %s\n", details.ChromeReference)
	fmt.Printf("Firefox Reference:    %s\n", details.FirefoxReference)
	fmt.Printf("Matches Chrome:       %v\n", details.MatchesChrome)
	fmt.Println("Gap Analysis (Technical Limitations):")
	for _, g := range details.GapAnalysis {
		fmt.Printf("  * %s\n", g)
	}
	fmt.Println("========================================================")
}
