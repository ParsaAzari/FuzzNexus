//go:build testlab

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type MatrixResult struct {
	ID         int    `json:"id"`
	UserAgent  string `json:"user_agent"`
	Method     string `json:"method"`
	HasHeaders bool   `json:"has_headers"`
	StatusCode int    `json:"status_code"`
	WAFStatus  string `json:"waf_status"`
	RuleHit    string `json:"rule_hit"`
	Outcome    string `json:"outcome"` // "BYPASS" or "BLOCKED"
}

func TestWAF_FullBypassMatrix(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}

	userAgents := []struct {
		category string
		ua       string
	}{
		{"Modern Chrome", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
		{"Modern Firefox", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0"},
		{"Old IE / Legacy", "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1)"},
		{"Bot (ffuf legacy)", "Fuzz Faster U Fool v2.0.0"},
		{"Bot (sqlmap)", "sqlmap/1.5#stable"},
		{"Bot (gobuster)", "gobuster/3.1.0"},
		{"Empty UA", ""},
		{"Custom Scanner", "Security-Audit-Scanner/1.0"},
	}

	methods := []string{"GET", "POST", "HEAD", "OPTIONS"}

	headerProfiles := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "Browser-Like Headers",
			headers: map[string]string{
				"Referer":         "http://127.0.0.1:8088/",
				"Accept-Language": "en-US,en;q=0.9",
				"X-Forwarded-For": "10.0.0.50",
			},
		},
		{
			name:    "Minimal / Bare Headers",
			headers: map[string]string{},
		},
	}

	var results []MatrixResult
	id := 1

	for _, uaEntry := range userAgents {
		for _, method := range methods {
			for _, hdrProfile := range headerProfiles {
				req, err := http.NewRequest(method, WafEndpoint+"/admin", nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				if uaEntry.ua != "" {
					req.Header.Set("User-Agent", uaEntry.ua)
				} else {
					req.Header.Del("User-Agent")
				}

				for k, v := range hdrProfile.headers {
					req.Header.Set(k, v)
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Skipf("WAF simulator not running on %s: %v", WafEndpoint, err)
					return
				}

				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				wafRule := resp.Header.Get("X-WAF-Rule")
				if wafRule == "" {
					if strings.Contains(string(body), "BOT_SIGNATURE") {
						wafRule = "BOT_SIGNATURE"
					} else if strings.Contains(string(body), "EMPTY_USER_AGENT") {
						wafRule = "EMPTY_USER_AGENT"
					} else if strings.Contains(string(body), "DISALLOWED_METHOD") {
						wafRule = "DISALLOWED_METHOD"
					} else {
						wafRule = "NONE"
					}
				}

				outcome := "BYPASS"
				if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
					outcome = "BLOCKED"
				}

				res := MatrixResult{
					ID:         id,
					UserAgent:  uaEntry.category,
					Method:     method,
					HasHeaders: len(hdrProfile.headers) > 0,
					StatusCode: resp.StatusCode,
					WAFStatus:  resp.Header.Get("X-WAF-Action"),
					RuleHit:    wafRule,
					Outcome:    outcome,
				}
				results = append(results, res)
				id++
				// Small delay to prevent rate limiter interference between tests
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	// Calculate bypass rates
	total := len(results)
	bypassed := 0
	browserTotal := 0
	browserBypassed := 0

	for _, r := range results {
		if r.Outcome == "BYPASS" {
			bypassed++
		}
		if r.UserAgent == "Modern Chrome" || r.UserAgent == "Modern Firefox" {
			// For legitimate methods (GET/POST/HEAD)
			if r.Method != "OPTIONS" {
				browserTotal++
				if r.Outcome == "BYPASS" {
					browserBypassed++
				}
			}
		}
	}

	browserBypassRate := float64(browserBypassed) / float64(browserTotal) * 100

	t.Logf("=== WAF BYPASS MATRIX SUMMARY ===")
	t.Logf("Total Matrix Combinations Tested: %d", total)
	t.Logf("Overall Bypassed: %d / %d (%.1f%%)", bypassed, total, float64(bypassed)/float64(total)*100)
	t.Logf("Browser UA Legitimate Methods Bypass Rate: %d / %d (%.1f%%)", browserBypassed, browserTotal, browserBypassRate)

	// Save results to json file
	data, _ := json.MarshalIndent(results, "", "  ")
	_ = os.WriteFile("waf_matrix_results.json", data, 0644)

	// Print Markdown table for report
	fmt.Println("\n| ID | User-Agent Category | Method | Browser Headers | Status | WAF Rule Hit | Outcome |")
	fmt.Println("| :---: | :--- | :---: | :---: | :---: | :--- | :---: |")
	for _, r := range results {
		headersStr := "No"
		if r.HasHeaders {
			headersStr = "Yes"
		}
		fmt.Printf("| %d | %s | %s | %s | %d | %s | **%s** |\n",
			r.ID, r.UserAgent, r.Method, headersStr, r.StatusCode, r.RuleHit, r.Outcome)
	}

	if browserBypassRate < 90.0 {
		t.Fatalf("Acceptance criteria failed: browser bypass rate %.1f%% < 90%%", browserBypassRate)
	}
}
