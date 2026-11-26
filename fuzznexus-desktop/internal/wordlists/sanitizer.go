package wordlists

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/parsaazari/fuzznexus/pkg/engine"
)

// SanitizeLine cleans a raw line according to the provided SanitizeOptions.
// Returns the cleaned line, whether the line should be dropped, and the count of empty/comment/trimmed events.
func SanitizeLine(raw string, opts SanitizeOptions, isFirstLine bool) (string, bool, string, bool) {
	line := raw

	// 1. Strip UTF-8 BOM if present on the first line
	if isFirstLine && opts.StripBOM {
		line = strings.TrimPrefix(line, "\xEF\xBB\xBF")
		line = strings.TrimPrefix(line, "\uFEFF")
	}

	// 2. Remove CR carriage return and Null-bytes
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, "\x00", "")

	// 3. Trim whitespace
	isTrimmed := false
	if opts.TrimWhitespace {
		trimmed := strings.TrimFunc(line, isCustomWhitespace)
		if trimmed != line {
			isTrimmed = true
			line = trimmed
		}
	}

	// 4. Empty check
	if line == "" {
		if opts.RemoveEmpty {
			return "", true, "empty", isTrimmed
		}
		return "", false, "", isTrimmed
	}

	// 5. Comment check
	if opts.StripComments {
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			return "", true, "comment", isTrimmed
		}
	}

	// 6. Leading slash normalization
	if opts.StripLeadingSlash {
		slashed := strings.TrimLeft(line, "/")
		if slashed != line {
			isTrimmed = true
			line = slashed
			if line == "" && opts.RemoveEmpty {
				return "", true, "empty", isTrimmed
			}
		}
	}

	// 7. Traversal / WAF-signature payload stripping (used during non-destructive safe copy)
	if opts.StripTraversalPayloads && engine.IsRiskyTraversalPayload(line) {
		return "", true, "traversal", isTrimmed
	}

	// 8. Max line length check
	if opts.MaxLineLen > 0 && len(line) > opts.MaxLineLen {
		return "", true, "length", isTrimmed
	}

	return line, false, "", isTrimmed
}

// ProcessAndDeduplicate streams multiple readers, sanitizes lines, deduplicates them,
// writes the clean output to writer, and returns a detailed MergeReportDTO.
func ProcessAndDeduplicate(readers []io.Reader, writer io.Writer, opts SanitizeOptions) (*MergeReportDTO, error) {
	report := &MergeReportDTO{}
	bufWriter := bufio.NewWriter(writer)

	// Hash map to track seen lines.
	// If CaseInsensitive is true, the lookup key is lowercase, but we preserve the original first-seen casing.
	seen := make(map[string]struct{})
	var writtenBytes int64 = 0

	for _, r := range readers {
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024) // Allow lines up to 1MB

		firstLine := true
		for scanner.Scan() {
			report.TotalSourceLines++
			raw := scanner.Text()

			cleaned, drop, reason, isTrimmed := SanitizeLine(raw, opts, firstLine)
			firstLine = false

			if isTrimmed {
				report.LinesTrimmed++
			}

			if drop {
				switch reason {
				case "empty":
					report.EmptyLinesRemoved++
				case "comment":
					report.CommentsRemoved++
				case "traversal":
					report.TraversalRemoved++
				case "length":
					report.EmptyLinesRemoved++ // Group length drops with filtered
				}
				continue
			}

			// Key for deduplication
			lookupKey := cleaned
			if opts.CaseInsensitive {
				lookupKey = strings.ToLower(cleaned)
			}

			if _, exists := seen[lookupKey]; exists {
				report.DuplicatesRemoved++
				continue
			}
			seen[lookupKey] = struct{}{}

			lineBytes := []byte(cleaned + "\n")
			n, err := bufWriter.Write(lineBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to write deduplicated line: %w", err)
			}
			writtenBytes += int64(n)
			report.DeduplicatedLines++
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading wordlist stream: %w", err)
		}
	}

	if err := bufWriter.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush buffer writer: %w", err)
	}

	report.SizeBytes = writtenBytes
	return report, nil
}

// PreviewDeduplication runs the deduplication logic in dry-run mode without writing to disk.
func PreviewDeduplication(readers []io.Reader, opts SanitizeOptions) (*MergeReportDTO, error) {
	var discard bytes.Buffer
	return ProcessAndDeduplicate(readers, &discard, opts)
}

func isCustomWhitespace(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '\u00A0', // Non-breaking space
		'\u200B', // Zero-width space
		'\u200C', // Zero-width non-joiner
		'\u200D', // Zero-width joiner
		'\uFEFF': // Byte order mark / zero-width no-break space
		return true
	default:
		return false
	}
}
