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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
