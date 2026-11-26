package wordlists

import (
	"errors"
	"time"
)

var (
	ErrWordlistLocked = errors.New("wordlist is currently locked by an active scan session")
	ErrInvalidPath    = errors.New("invalid or unsafe wordlist path")
	ErrFileNotFound   = errors.New("wordlist file not found")
	ErrNotRegularFile = errors.New("target is not a regular file")
)

// WordlistDTO represents a registered wordlist metadata record.
type WordlistDTO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	LineCount int64     `json:"line_count"`
	SizeBytes int64     `json:"size_bytes"`
	IsLinked  bool      `json:"is_linked"`
	Category  string    `json:"category"`
	IsBuiltin bool      `json:"is_builtin"`
	BuiltinID string    `json:"builtin_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BuiltinPackDTO represents metadata for an embedded curated wordlist pack.
type BuiltinPackDTO struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Color       string   `json:"color"`
	Count       int      `json:"count"`
	Keywords    []string `json:"keywords,omitempty"`
}

// WordlistPageDTO represents a paginated chunk of lines for the virtual editor.
type WordlistPageDTO struct {
	Lines      []string `json:"lines"`
	TotalLines int64    `json:"total_lines"`
	Offset     int      `json:"offset"`
	Limit      int      `json:"limit"`
}

// WordlistStatsDTO contains analytical metrics about wordlist composition.
type WordlistStatsDTO struct {
	LineCount   int64   `json:"line_count"`
	SizeBytes   int64   `json:"size_bytes"`
	UniqueLines int64   `json:"unique_lines"`
	AvgLineLen  float64 `json:"avg_line_len"`
	MaxLineLen  int     `json:"max_line_len"`
}

// CleanResultDTO represents the outcome of a wordlist verification and cleaning operation.
type CleanResultDTO struct {
	ID                int64 `json:"id"`
	LinesBefore       int64 `json:"lines_before"`
	LinesAfter        int64 `json:"lines_after"`
	EmptyLinesRemoved int64 `json:"empty_lines_removed"`
	LinesTrimmed      int64 `json:"lines_trimmed"`
	DuplicatesRemoved int64 `json:"duplicates_removed"`
}

// WordlistHealthDTO summarizes the hygiene of a wordlist file.
type WordlistHealthDTO struct {
	ID              int64 `json:"id"`
	TotalLines      int64 `json:"total_lines"`
	BlankLinesCount int64 `json:"blank_lines_count"`
	DuplicateCount  int64 `json:"duplicate_count"`
	HasWhitespace   bool  `json:"has_whitespace"`
	HasCRLF         bool  `json:"has_crlf"`
	IsClean         bool  `json:"is_clean"`
}

// CategoryDTO represents a wordlist category with custom label and color.
type CategoryDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	IsDefault bool   `json:"is_default"`
}

// SanitizeOptions defines the rules for high-precision wordlist cleaning and normalization.
type SanitizeOptions struct {
	TrimWhitespace         bool `json:"trim_whitespace"`
	StripBOM               bool `json:"strip_bom"`
	RemoveEmpty            bool `json:"remove_empty"`
	StripComments          bool `json:"strip_comments"`
	StripLeadingSlash      bool `json:"strip_leading_slash"`
	StripTraversalPayloads bool `json:"strip_traversal_payloads"`
	CaseInsensitive        bool `json:"case_insensitive"`
	MaxLineLen             int  `json:"max_line_len"`
}

// DefaultSanitizeOptions returns recommended default sanitization options.
func DefaultSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{
		TrimWhitespace:         true,
		StripBOM:               true,
		RemoveEmpty:            true,
		StripComments:          true,
		StripLeadingSlash:      false,
		StripTraversalPayloads: false,
		CaseInsensitive:        false,
		MaxLineLen:             2048,
	}
}

// MergeReportDTO represents the detailed outcome of a wordlist merge and sanitization operation.
type MergeReportDTO struct {
	TotalSourceLines  int64 `json:"total_source_lines"`
	DeduplicatedLines int64 `json:"deduplicated_lines"`
	DuplicatesRemoved int64 `json:"duplicates_removed"`
	EmptyLinesRemoved int64 `json:"empty_lines_removed"`
	CommentsRemoved   int64 `json:"comments_removed"`
	TraversalRemoved  int64 `json:"traversal_removed"`
	LinesTrimmed      int64 `json:"lines_trimmed"`
	SizeBytes         int64 `json:"size_bytes"`
}

// WordlistRiskReportDTO summarizes passive pre-scan WAF risk indicators inside a wordlist.
type WordlistRiskReportDTO struct {
	Path                string   `json:"path"`
	TotalLines          int64    `json:"total_lines"`
	TraversalCount      int64    `json:"traversal_count"`
	WAFSignatureCount   int64    `json:"waf_signature_count"`
	SensitiveRouteCount int64    `json:"sensitive_route_count"`
	RiskLevel           string   `json:"risk_level"` // "low", "medium", "high"
	SampleRiskyLines    []string `json:"sample_risky_lines"`
	Summary             string   `json:"summary"`
}
