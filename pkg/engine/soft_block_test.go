package engine

import (
	"strings"
	"testing"
)

func TestSoftBlockDetectorMultiWAF(t *testing.T) {
	det := NewSoftBlockDetector(true)

	cases := []struct {
		name       string
		statusCode int
		headers    map[string][]string
		body       string
		wantBlock  bool
		wantSubstr string
	}{
		{
			name:       "Cloudflare cf-mitigated header",
			statusCode: 403,
			headers:    map[string][]string{"Cf-Mitigated": {"challenge"}},
			wantBlock:  true,
			wantSubstr: "cf-mitigated",
		},
		{
			name:       "ArvanCloud ar-challenge header",
			statusCode: 200,
			headers:    map[string][]string{"Ar-Challenge": {"1"}},
			wantBlock:  true,
			wantSubstr: "ArvanCloud",
		},
		{
			name:       "ArvanCloud 200 HTML challenge",
			statusCode: 200,
			body:       `<html><title>ArvanCloud Security</title><body>Access Denied - Captcha Required</body></html>`,
			wantBlock:  true,
			wantSubstr: "ArvanCloud",
		},
		{
			name:       "AWS WAF header",
			statusCode: 202,
			headers:    map[string][]string{"X-Amzn-Waf-Action": {"captcha"}},
			wantBlock:  true,
			wantSubstr: "AWS WAF",
		},
		{
			name:       "AWS WAF 200 HTML challenge",
			statusCode: 200,
			body:       `<script src="https://abc.token.awswaf.com/challenge.js"></script><div>Human Verification Captcha</div>`,
			wantBlock:  true,
			wantSubstr: "AWS WAF",
		},
		{
			name:       "Akamai 200 block page",
			statusCode: 200,
			body:       `<html><body>Request blocked by Edge. Reference #18.2a3f11.1700000</body></html>`,
			wantBlock:  true,
			wantSubstr: "Akamai",
		},
		{
			name:       "Imperva Incapsula 200 challenge",
			statusCode: 200,
			body:       `<html><iframe src="/_Incapsula_Resource?SWUDNSAI=31"></iframe></html>`,
			wantBlock:  true,
			wantSubstr: "Imperva",
		},
		{
			name:       "Sucuri 200 block",
			statusCode: 200,
			headers:    map[string][]string{"X-Sucuri-Id": {"12345"}},
			body:       `<html><title>Sucuri WebSite Firewall - Access Denied - Sucuri</title></html>`,
			wantBlock:  true,
			wantSubstr: "Sucuri",
		},
		{
			name:       "DataDome header",
			statusCode: 403,
			headers:    map[string][]string{"X-Datadome": {"protected"}},
			wantBlock:  true,
			wantSubstr: "DataDome",
		},
		{
			name:       "ModSecurity 200 block",
			statusCode: 200,
			body:       `<html><body>ModSecurity: Not Acceptable request blocked</body></html>`,
			wantBlock:  true,
			wantSubstr: "ModSecurity",
		},
		{
			name:       "Generic CAPTCHA interstitial (<30KB)",
			statusCode: 200,
			body:       `<html><body><h1>Verify you are human</h1><div class="g-recaptcha"></div></body></html>`,
			wantBlock:  true,
			wantSubstr: "CAPTCHA",
		},
		{
			name:       "Benign Cloudflare JSD beacon on 200 OK (no false positive)",
			statusCode: 200,
			body:       `<html><body><h1>Welcome to Parsa Azari</h1><script src="/cdn-cgi/challenge-platform/scripts/jsd/main.js"></script></body></html>`,
			wantBlock:  false,
		},
		{
			name:       "Benign large page with recaptcha login form (>30KB, no false positive)",
			statusCode: 200,
			body:       strings.Repeat("<div>Product catalog item</div>", 1200) + `<div class="g-recaptcha"></div><p>Verify you are human to sign up</p>`,
			wantBlock:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotBlock, gotReason := det.Check(tc.statusCode, tc.headers, []byte(tc.body))
			if gotBlock != tc.wantBlock {
				t.Fatalf("Check(%d) = %v (%q), want %v", tc.statusCode, gotBlock, gotReason, tc.wantBlock)
			}
			if tc.wantBlock && tc.wantSubstr != "" && !strings.Contains(gotReason, tc.wantSubstr) {
				t.Fatalf("reason %q does not contain %q", gotReason, tc.wantSubstr)
			}
		})
	}
}
