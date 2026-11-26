package wordlists

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/parsaazari/fuzznexus/pkg/engine"
)

var (
	wafSignatureTokens = []string{
		"union select",
		"' or ",
		"or 1=1",
		"<script",
		"javascript:",
		"${jndi:",
	}
	sensitiveRouteTokens = []string{
		"admin",
		"wp-admin",
		"wp-login",
		"phpmyadmin",
		".env",
		".git",
	}
)

// AnalyzeWordlistRisks performs passive pre-scan inspection of a wordlist file to detect
// traversal/normalization anomalies and WAF signature payloads before sending traffic.
func AnalyzeWordlistRisks(path string) (*WordlistRiskReportDTO, error) {
	cleanPath, err := ValidatePath(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	report := &WordlistRiskReportDTO{
		Path:      path,
		RiskLevel: "low",
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	seenSamples := make(map[string]struct{})
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r\n"))
		if line == "" {
			continue
		}
		report.TotalLines++
		lower := strings.ToLower(line)

		isSignature := false
		for _, tok := range wafSignatureTokens {
			if strings.Contains(lower, tok) {
				report.WAFSignatureCount++
				isSignature = true
				break
			}
		}

		isTraversal := false
		if !isSignature && engine.IsRiskyTraversalPayload(line) {
			report.TraversalCount++
			isTraversal = true
		}

		if isSignature || isTraversal {
			if len(report.SampleRiskyLines) < 6 {
				if _, ok := seenSamples[line]; !ok {
					seenSamples[line] = struct{}{}
					report.SampleRiskyLines = append(report.SampleRiskyLines, line)
				}
			}
			continue
		}

		trimmedRoute := strings.TrimLeft(lower, "/.")
		for _, route := range sensitiveRouteTokens {
			if trimmedRoute == route || strings.HasPrefix(trimmedRoute, route+"/") {
				report.SensitiveRouteCount++
				break
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning wordlist for risks: %w", err)
	}

	totalHazard := report.TraversalCount + report.WAFSignatureCount
	hazardRatio := 0.0
	if report.TotalLines > 0 {
		hazardRatio = float64(totalHazard) / float64(report.TotalLines)
	}

	switch {
	case report.TraversalCount >= 5 || report.WAFSignatureCount >= 3 || hazardRatio >= 0.20:
		report.RiskLevel = "high"
		report.Summary = fmt.Sprintf(
			"Contains %d traversal/bypass & %d WAF-signature patterns that frequently trigger 403 firewall rules.",
			report.TraversalCount, report.WAFSignatureCount,
		)
	case totalHazard > 0 || report.SensitiveRouteCount >= 5:
		report.RiskLevel = "medium"
		report.Summary = fmt.Sprintf(
			"Contains %d traversal/signature patterns and %d WAF-sensitive admin routes.",
			totalHazard, report.SensitiveRouteCount,
		)
	default:
		report.RiskLevel = "low"
		report.Summary = "Clean wordlist with low WAF heuristic trigger risk."
	}

	return report, nil
}
