package engine

import (
	"context"
	"fmt"
	"time"
)

// BanCategory defines the classification of network response anomalies.
type BanCategory int

const (
	BanNone BanCategory = iota
	BanRateLimit
	BanOutage
	BanIPBlock
)

// String returns the human-readable name of the BanCategory.
func (b BanCategory) String() string {
	switch b {
	case BanRateLimit:
		return "RateLimit"
	case BanOutage:
		return "Outage"
	case BanIPBlock:
		return "IPBlock"
	default:
		return "None"
	}
}

// BanDetectorConfig configures thresholds for ban and anomaly detection.
type BanDetectorConfig struct {
	WindowSize      int           // Rolling window size (default 50)
	CooloffDuration time.Duration // Mandatory cooloff before probe evaluation (default 30s)
}

// DefaultBanDetectorConfig returns the locked defaults per Phase 5 spec.
func DefaultBanDetectorConfig() BanDetectorConfig {
	return BanDetectorConfig{
		WindowSize:      50,
		CooloffDuration: 30 * time.Second,
	}
}

// BanDetector evaluates window metrics against definitive Phase 5 rules.
type BanDetector struct {
	config  BanDetectorConfig
	probeFn func(ctx context.Context) (isBlocked bool, err error)
}

// NewBanDetector creates a new BanDetector instance.
func NewBanDetector(cfg BanDetectorConfig, probeFn func(ctx context.Context) (isBlocked bool, err error)) *BanDetector {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 50
	}
	if cfg.CooloffDuration <= 0 {
		cfg.CooloffDuration = 30 * time.Second
	}
	return &BanDetector{
		config:  cfg,
		probeFn: probeFn,
	}
}

// SetProbeFunc configures the single probe executor used for condition (3).
func (bd *BanDetector) SetProbeFunc(fn func(ctx context.Context) (isBlocked bool, err error)) {
	bd.probeFn = fn
}

// SetCooloffDuration allows overriding cooloff duration for fast testing.
func (bd *BanDetector) SetCooloffDuration(d time.Duration) {
	bd.config.CooloffDuration = d
}

// CooloffDuration returns the configured cooloff duration.
func (bd *BanDetector) CooloffDuration() time.Duration {
	return bd.config.CooloffDuration
}

// ClassifyMetrics evaluates rolling window metrics against exact locked rules.
// Rules (Strict):
// 1. RateLimit: >= 3 429s in 50-window OR 429 with Retry-After -> RateLimit (no popup).
// 2. Outage: >= 3 5xx in 50-window AND 0 403s in same window -> Outage (no popup).
// 3. IPBlock Candidate: >= 5 403s or connection-resets in 50-window AND ratio >= 50%.
func (bd *BanDetector) ClassifyMetrics(stats RollingStats, hasRetryAfter bool) BanCategory {
	count403OrReset := stats.Count403 + stats.CountErrors
	ratio403OrReset := 0.0
	if stats.TotalSamples > 0 {
		ratio403OrReset = float64(count403OrReset) / float64(stats.TotalSamples)
	}

	// Rule 3 (IPBlock Preconditions 1 & 2):
	if count403OrReset >= 5 && ratio403OrReset >= 0.50 {
		return BanIPBlock
	}

	// Rule 1 (RateLimit): >= 3 429s in window OR Retry-After present
	if stats.Count429 >= 3 || hasRetryAfter {
		return BanRateLimit
	}

	// Rule 2 (Outage): >= 3 5xx in window AND 0 403s in same window
	if stats.Count5xx >= 3 && stats.Count403 == 0 {
		return BanOutage
	}

	return BanNone
}

// VerifyCondition3 executes a single probe after cooloff to definitively confirm IPBlock.
// Condition (3): After cooloff, single probe is still 403 or reset.
func (bd *BanDetector) VerifyCondition3(ctx context.Context) (isBlocked bool, reason string) {
	if bd.probeFn == nil {
		return false, "no probe function configured"
	}

	blocked, err := bd.probeFn(ctx)
	if err != nil {
		return true, fmt.Sprintf("probe connection error/reset: %v", err)
	}
	if blocked {
		return true, "probe returned 403 Forbidden / block response"
	}
	return false, "probe succeeded with 200 OK"
}
