package engine

import (
	"bytes"
	"strings"
)

// SoftBlockDetector inspects response headers and body for silent WAF challenge pages.
type SoftBlockDetector struct {
	Enabled bool
}

// NewSoftBlockDetector returns a new SoftBlockDetector.
func NewSoftBlockDetector(enabled bool) *SoftBlockDetector {
	return &SoftBlockDetector{Enabled: enabled}
}

// Check evaluates whether a response is a soft block (200 OK containing challenge HTML or challenge headers).
func (s *SoftBlockDetector) Check(statusCode int, headerMap map[string][]string, body []byte) (bool, string) {
	if !s.Enabled {
		return false, ""
	}

	var hasSucuriID bool

	// 1. Check for deterministic WAF mitigation headers
	for k, vs := range headerMap {
		lowerK := strings.ToLower(k)
		if lowerK == "ar-challenge" {
			for _, v := range vs {
				if strings.TrimSpace(v) != "" {
					return true, "ArvanCloud WAF challenge header detected (ar-challenge)"
				}
			}
		}
		if lowerK == "x-sucuri-id" {
			hasSucuriID = true
		}
		for _, v := range vs {
			lowerV := strings.ToLower(v)
			switch lowerK {
			case "cf-mitigated":
				if strings.Contains(lowerV, "challenge") {
					return true, "WAF mitigation header detected (cf-mitigated: challenge)"
				}
			case "x-arvan-waf":
				if strings.Contains(lowerV, "block") || strings.Contains(lowerV, "challenge") || strings.Contains(lowerV, "captcha") {
					return true, "ArvanCloud WAF mitigation header detected (x-arvan-waf)"
				}
			case "x-amzn-waf-action":
				if strings.Contains(lowerV, "captcha") || strings.Contains(lowerV, "challenge") {
					return true, "AWS WAF challenge header detected (x-amzn-waf-action)"
				}
			case "x-datadome":
				if strings.Contains(lowerV, "protected") {
					return true, "DataDome WAF protection header detected (x-datadome: protected)"
				}
			}
		}
	}

	// 2. Check for active challenge platform keywords inside body (especially on 200 OK).
	// Note: Cloudflare injects benign `/cdn-cgi/challenge-platform/scripts/jsd/main.js` telemetry
	// into normal 200 OK pages without `cf-mitigated`; do NOT treat passive JSD beacons as blocks.
	if statusCode == 200 && len(body) > 0 {
		lowerBody := bytes.ToLower(body)

		// Cloudflare
		isActiveOrchestrate := bytes.Contains(lowerBody, []byte("challenge-platform/h/")) ||
			bytes.Contains(lowerBody, []byte("orchestrate/chl_page")) ||
			bytes.Contains(lowerBody, []byte("_cf_chl_opt")) ||
			bytes.Contains(lowerBody, []byte(`id="challenge-platform"`)) ||
			bytes.Contains(lowerBody, []byte(`id='challenge-platform'`))
		if isActiveOrchestrate {
			return true, "HTML challenge platform signature detected in body"
		}
		if bytes.Contains(lowerBody, []byte("just a moment...")) &&
			(bytes.Contains(lowerBody, []byte("verify you are human")) ||
				bytes.Contains(lowerBody, []byte("enable javascript and cookies")) ||
				bytes.Contains(lowerBody, []byte("challenge-error-text"))) {
			return true, "Cloudflare turnstile/managed challenge HTML detected in body"
		}

		// Imperva / Incapsula
		if bytes.Contains(lowerBody, []byte("_incapsula_resource")) ||
			(bytes.Contains(lowerBody, []byte("incapsula incident id")) && bytes.Contains(lowerBody, []byte("request unsuccessful"))) {
			return true, "Imperva/Incapsula WAF challenge HTML detected in body"
		}

		// Sucuri
		if (hasSucuriID && bytes.Contains(lowerBody, []byte("access denied - sucuri"))) ||
			(bytes.Contains(lowerBody, []byte("sucuri website firewall")) && bytes.Contains(lowerBody, []byte("access denied"))) {
			return true, "Sucuri WAF block HTML detected in body"
		}

		// Akamai
		if bytes.Contains(lowerBody, []byte("reference #")) &&
			(bytes.Contains(lowerBody, []byte("request blocked by")) ||
				(bytes.Contains(lowerBody, []byte("akamai")) && bytes.Contains(lowerBody, []byte("access denied")))) {
			return true, "Akamai WAF block HTML detected in body"
		}

		// AWS WAF JS/Captcha challenge
		if (bytes.Contains(lowerBody, []byte("awswaf")) || bytes.Contains(lowerBody, []byte("token.awswaf.com"))) &&
			(bytes.Contains(lowerBody, []byte("captcha")) || bytes.Contains(lowerBody, []byte("challenge")) || bytes.Contains(lowerBody, []byte("human verification"))) {
			return true, "AWS WAF challenge HTML detected in body"
		}

		// ArvanCloud
		if len(body) < 40*1024 &&
			(bytes.Contains(lowerBody, []byte("arvancloud")) || bytes.Contains(lowerBody, []byte("arvan cloud"))) &&
			(bytes.Contains(lowerBody, []byte("captcha")) || bytes.Contains(lowerBody, []byte("access denied")) || bytes.Contains(lowerBody, []byte("challenge"))) {
			return true, "ArvanCloud WAF challenge HTML detected in body"
		}

		// ModSecurity
		if len(body) < 20*1024 &&
			(bytes.Contains(lowerBody, []byte("mod_security")) || bytes.Contains(lowerBody, []byte("modsecurity"))) &&
			(bytes.Contains(lowerBody, []byte("not acceptable")) || bytes.Contains(lowerBody, []byte("access denied")) || bytes.Contains(lowerBody, []byte("blocked"))) {
			return true, "ModSecurity WAF block HTML detected in body"
		}

		// Generic CAPTCHA interstitial (< 30KB to avoid false positives on full login/contact pages)
		if len(body) < 30*1024 {
			hasCaptchaWidget := bytes.Contains(lowerBody, []byte("g-recaptcha")) ||
				bytes.Contains(lowerBody, []byte("h-captcha")) ||
				bytes.Contains(lowerBody, []byte("cf-turnstile")) ||
				bytes.Contains(lowerBody, []byte("captcha-container"))
			hasBlockPhrase := bytes.Contains(lowerBody, []byte("verify you are human")) ||
				bytes.Contains(lowerBody, []byte("automated requests")) ||
				bytes.Contains(lowerBody, []byte("bot detection")) ||
				bytes.Contains(lowerBody, []byte("unusual traffic")) ||
				bytes.Contains(lowerBody, []byte("complete the security check"))
			if hasCaptchaWidget && hasBlockPhrase {
				return true, "Generic CAPTCHA interstitial block detected in 200 OK body"
			}
		}
	}

	return false, ""
}
