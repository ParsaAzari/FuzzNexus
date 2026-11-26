package scan_test

import (
	"testing"

	"fuzznexus-desktop/internal/scan"
)

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		name         string
		rawURL       string
		headers      map[string]string
		body         string
		expectedNorm string
		expectedFuzz bool
		expectedLoc  string
		expectedAuto bool
		shouldError  bool
	}{
		{
			name:         "Bare domain without scheme or FUZZ",
			rawURL:       "example.com",
			expectedNorm: "https://example.com/FUZZ",
			expectedFuzz: true,
			expectedLoc:  "path",
			expectedAuto: true,
		},
		{
			name:         "Localhost with port without scheme or FUZZ",
			rawURL:       "localhost:8080",
			expectedNorm: "http://localhost:8080/FUZZ",
			expectedFuzz: true,
			expectedLoc:  "path",
			expectedAuto: true,
		},
		{
			name:         "127.0.0.1 with trailing slash",
			rawURL:       "http://127.0.0.1:3000/",
			expectedNorm: "http://127.0.0.1:3000/FUZZ",
			expectedFuzz: true,
			expectedLoc:  "path",
			expectedAuto: true,
		},
		{
			name:         "Query parameter ending in =",
			rawURL:       "https://api.target.com/search?q=",
			expectedNorm: "https://api.target.com/search?q=FUZZ",
			expectedFuzz: true,
			expectedLoc:  "query",
			expectedAuto: true,
		},
		{
			name:         "Explicit FUZZ in path",
			rawURL:       "https://target.com/v1/FUZZ/details",
			expectedNorm: "https://target.com/v1/FUZZ/details",
			expectedFuzz: true,
			expectedLoc:  "path",
			expectedAuto: false,
		},
		{
			name:         "Explicit FUZZ in subdomain",
			rawURL:       "https://FUZZ.target.com/",
			expectedNorm: "https://FUZZ.target.com/",
			expectedFuzz: true,
			expectedLoc:  "subdomain",
			expectedAuto: false,
		},
		{
			name:         "Explicit FUZZ in file extension",
			rawURL:       "http://target.com/index.FUZZ",
			expectedNorm: "http://target.com/index.FUZZ",
			expectedFuzz: true,
			expectedLoc:  "extension",
			expectedAuto: false,
		},
		{
			name:         "Lowercase fuzz in path (case-insensitive)",
			rawURL:       "https://target.com/api/v1/fuzz",
			expectedNorm: "https://target.com/api/v1/FUZZ",
			expectedFuzz: true,
			expectedLoc:  "path",
			expectedAuto: false,
		},
		{
			name:         "Mixed case Fuzz in query parameter",
			rawURL:       "https://target.com/search?q=Fuzz&type=active",
			expectedNorm: "https://target.com/search?q=FUZZ&type=active",
			expectedFuzz: true,
			expectedLoc:  "query",
			expectedAuto: false,
		},
		{
			name:         "Mixed case fuZZ in subdomain",
			rawURL:       "https://fuZZ.target.com/dashboard",
			expectedNorm: "https://FUZZ.target.com/dashboard",
			expectedFuzz: true,
			expectedLoc:  "subdomain",
			expectedAuto: false,
		},
		{
			name:         "Lowercase fuzz in headers",
			rawURL:       "https://target.com/api",
			headers:      map[string]string{"X-Custom-Token": "fuzz"},
			expectedNorm: "https://target.com/api",
			expectedFuzz: true,
			expectedLoc:  "headers",
			expectedAuto: false,
		},
		{
			name:         "Lowercase fuzz in body",
			rawURL:       "https://target.com/api",
			body:         `{"search":"fuzz"}`,
			expectedNorm: "https://target.com/api",
			expectedFuzz: true,
			expectedLoc:  "body",
			expectedAuto: false,
		},
		{
			name:         "Accidental whitespace and quotes",
			rawURL:       "  \"http://target.com/api/FUZZ\" \n",
			expectedNorm: "http://target.com/api/FUZZ",
			expectedFuzz: true,
			expectedLoc:  "path",
			expectedAuto: false,
		},
		{
			name:         "FUZZ in header without URL FUZZ",
			rawURL:       "https://target.com/api/users",
			headers:      map[string]string{"Authorization": "Bearer FUZZ"},
			expectedNorm: "https://target.com/api/users", // Should NOT append /FUZZ to URL
			expectedFuzz: true,
			expectedLoc:  "headers",
			expectedAuto: false,
		},
		{
			name:         "FUZZ in body without URL FUZZ",
			rawURL:       "https://target.com/login",
			body:         `{"username":"admin","password":"FUZZ"}`,
			expectedNorm: "https://target.com/login", // Should NOT append /FUZZ to URL
			expectedFuzz: true,
			expectedLoc:  "body",
			expectedAuto: false,
		},
		{
			name:        "Empty target URL",
			rawURL:      "   ",
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := scan.NormalizeTarget(tc.rawURL, tc.headers, tc.body)
			if tc.shouldError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tc.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Normalized != tc.expectedNorm {
				t.Errorf("expected normalized %s, got %s", tc.expectedNorm, res.Normalized)
			}
			if res.HasFuzz != tc.expectedFuzz {
				t.Errorf("expected hasFuzz %v, got %v", tc.expectedFuzz, res.HasFuzz)
			}
			if res.FuzzLocation != tc.expectedLoc {
				t.Errorf("expected fuzzLocation %s, got %s", tc.expectedLoc, res.FuzzLocation)
			}
			if res.AutoAppended != tc.expectedAuto {
				t.Errorf("expected autoAppended %v, got %v", tc.expectedAuto, res.AutoAppended)
			}
		})
	}
}
