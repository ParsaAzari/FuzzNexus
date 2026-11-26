package scan

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	schemeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)
)

// TargetAnalysis holds parsed components and metadata of a target URL.
type TargetAnalysis struct {
	Original     string `json:"original"`
	Normalized   string `json:"normalized"`
	Scheme       string `json:"scheme"`
	Host         string `json:"host"`
	Path         string `json:"path"`
	HasFuzz      bool   `json:"has_fuzz"`
	FuzzLocation string `json:"fuzz_location"` // "path", "query", "subdomain", "extension", "headers", "body", "none"
	AutoAppended bool   `json:"auto_appended"`
}

// NormalizeTarget cleans and formats user target input into a safe ffuf URL.
func NormalizeTarget(rawURL string, headers map[string]string, body string) (*TargetAnalysis, error) {
	return NormalizeTargetWithOptions(rawURL, headers, body, "", "", nil)
}

// NormalizeTargetWithOptions cleans and formats target input while respecting attack mode (e.g. sniper §...§)
// and custom multi-wordlist keywords (so /FUZZ is never spuriously appended when custom keywords exist).
func NormalizeTargetWithOptions(rawURL string, headers map[string]string, body string, method string, inputMode string, keywords []string) (*TargetAnalysis, error) {
	trimmed := strings.TrimSpace(rawURL)
	// Strip surrounding double or single quotes
	trimmed = strings.Trim(trimmed, `"'`)
	if trimmed == "" {
		return nil, errors.New("target URL cannot be empty")
	}

	isSniper := strings.EqualFold(strings.TrimSpace(inputMode), "sniper")

	analysis := &TargetAnalysis{
		Original:     rawURL,
		FuzzLocation: "none",
	}

	containsCustomToken := func(s string) bool {
		if isSniper && strings.Contains(s, "§") {
			return true
		}
		for _, kw := range keywords {
			if kw != "" && kw != "FUZZ" && strings.Contains(s, kw) {
				return true
			}
		}
		return false
	}

	// 1. Check if FUZZ or custom keyword is present in headers, request body, or method
	hasFuzzInHeaders := false
	for k, v := range headers {
		if (!isSniper && (strings.Contains(strings.ToLower(k), "fuzz") || strings.Contains(strings.ToLower(v), "fuzz"))) ||
			containsCustomToken(k) || containsCustomToken(v) {
			hasFuzzInHeaders = true
			analysis.FuzzLocation = "headers"
			break
		}
	}
	hasFuzzInBody := (!isSniper && strings.Contains(strings.ToLower(body), "fuzz")) || containsCustomToken(body)
	if hasFuzzInBody && !hasFuzzInHeaders {
		analysis.FuzzLocation = "body"
	}
	hasFuzzInMethod := containsCustomToken(method) || (!isSniper && strings.Contains(strings.ToLower(method), "fuzz"))

	// 2. Ensure scheme
	target := trimmed
	if !isSniper {
		// Normalize any casing of 'fuzz' in the URL to uppercase 'FUZZ' for the engine.
		target = normalizeFuzzKeyword(target)
	}

	if !schemeRegex.MatchString(target) {
		lower := strings.ToLower(target)
		if strings.HasPrefix(lower, "localhost") ||
			strings.HasPrefix(lower, "127.0.0.1") ||
			strings.HasPrefix(lower, "[::1]") ||
			strings.Contains(lower, ":8000") ||
			strings.Contains(lower, ":8080") ||
			strings.Contains(lower, ":3000") ||
			strings.Contains(lower, ":5000") {
			target = "http://" + target
			analysis.Scheme = "http"
		} else {
			target = "https://" + target
			analysis.Scheme = "https"
		}
	}

	// 3. Check FUZZ / custom keyword in URL
	hasFuzzInURL := (!isSniper && strings.Contains(target, "FUZZ")) || containsCustomToken(target)
	analysis.HasFuzz = hasFuzzInURL || hasFuzzInHeaders || hasFuzzInBody || hasFuzzInMethod

	if hasFuzzInURL {
		if strings.Contains(target, "://FUZZ.") || strings.Contains(target, ".FUZZ.") {
			analysis.FuzzLocation = "subdomain"
		} else if strings.Contains(target, "?") && strings.Contains(strings.Split(target, "?")[1], "FUZZ") {
			analysis.FuzzLocation = "query"
		} else if strings.Contains(target, ".FUZZ") {
			analysis.FuzzLocation = "extension"
		} else {
			analysis.FuzzLocation = "path"
		}
	} else if !isSniper && !hasFuzzInHeaders && !hasFuzzInBody && !hasFuzzInMethod {
		// Safely auto-append FUZZ
		analysis.AutoAppended = true
		analysis.FuzzLocation = "path"
		if strings.Contains(target, "?") {
			parts := strings.SplitN(target, "?", 2)
			query := parts[1]
			if strings.HasSuffix(query, "=") || strings.HasSuffix(query, "&") || query == "" {
				target = target + "FUZZ"
				analysis.FuzzLocation = "query"
			} else {
				if strings.HasSuffix(parts[0], "/") {
					target = parts[0] + "FUZZ?" + query
				} else {
					target = parts[0] + "/FUZZ?" + query
				}
			}
		} else {
			if strings.HasSuffix(target, "/") {
				target = target + "FUZZ"
			} else {
				target = target + "/FUZZ"
			}
		}
		analysis.HasFuzz = true
	}

	// 4. Validate URI structure
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "" {
		return nil, errors.New("target URL must contain a valid host name or IP address")
	}

	analysis.Normalized = target
	analysis.Host = parsed.Host
	analysis.Path = parsed.Path
	if analysis.Scheme == "" {
		analysis.Scheme = parsed.Scheme
	}

	return analysis, nil
}

// normalizeFuzzKeyword replaces any casing variation of 'fuzz' (e.g. fuzz, Fuzz, fUzZ) with uppercase FUZZ.
func normalizeFuzzKeyword(s string) string {
	if !strings.Contains(strings.ToLower(s), "fuzz") {
		return s
	}
	re := regexp.MustCompile(`(?i)fuzz`)
	return re.ReplaceAllString(s, "FUZZ")
}
