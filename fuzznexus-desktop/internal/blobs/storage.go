package blobs

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"
)

var (
	ErrQuotaExceeded = errors.New("blobs: session disk quota exceeded")
	ErrNotFound      = errors.New("blobs: payload not found")
)

const (
	// DefaultSessionQuota is 2GB per scan session
	DefaultSessionQuota int64 = 2 * 1024 * 1024 * 1024
	// DefaultMaxPreviewBytes is 500KB
	DefaultMaxPreviewBytes int = 500 * 1024
)

// PayloadDTO represents a request/response payload preview for the UI inspector.
type PayloadDTO struct {
	SessionID     int64  `json:"session_id"`
	ResultID      int64  `json:"result_id"`
	RequestText   string `json:"request_text"`
	ResponseText  string `json:"response_text"`
	Truncated     bool   `json:"truncated"`
	FullSizeBytes int64  `json:"full_size_bytes"`
	ContentType   string `json:"content_type"`
}

// Storage manages compressed raw request/response blobs on disk with quota enforcement.
type Storage struct {
	baseDir      string
	quota        int64
	sessionSizes sync.Map // sessionID (int64) -> *atomic.Int64
	encoder      *zstd.Encoder
	decoder      *zstd.Decoder
	mu           sync.Mutex
}

// NewStorage creates or initializes blob storage at baseDir.
func NewStorage(baseDir string, quota int64) (*Storage, error) {
	if quota <= 0 {
		quota = DefaultSessionQuota
	}

	blobsDir := filepath.Join(baseDir, "blobs")
	if err := os.MkdirAll(blobsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create blobs directory: %w", err)
	}

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		enc.Close()
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}

	return &Storage{
		baseDir: blobsDir,
		quota:   quota,
		encoder: enc,
		decoder: dec,
	}, nil
}

// sessionDir returns the session folder path and ensures it exists with 0700 permissions.
func (s *Storage) sessionDir(sessionID int64) (string, error) {
	dir := filepath.Join(s.baseDir, fmt.Sprintf("%d", sessionID))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Storage) getSessionCounter(sessionID int64) *atomic.Int64 {
	val, loaded := s.sessionSizes.Load(sessionID)
	if loaded {
		return val.(*atomic.Int64)
	}
	counter := &atomic.Int64{}
	actual, _ := s.sessionSizes.LoadOrStore(sessionID, counter)
	return actual.(*atomic.Int64)
}

// SavePayload compresses and writes request and response blobs to disk for a result.
// If the session exceeds its quota, ErrQuotaExceeded is returned and nothing is written.
func (s *Storage) SavePayload(sessionID int64, resultID int64, req []byte, resp []byte) error {
	s.mu.Lock()
	compressedReq := s.encoder.EncodeAll(req, make([]byte, 0, len(req)/2+64))
	compressedResp := s.encoder.EncodeAll(resp, make([]byte, 0, len(resp)/2+64))
	s.mu.Unlock()

	totalNewBytes := int64(len(compressedReq) + len(compressedResp))
	counter := s.getSessionCounter(sessionID)

	// Check quota before disk write
	current := counter.Load()
	if current+totalNewBytes > s.quota {
		return ErrQuotaExceeded
	}

	dir, err := s.sessionDir(sessionID)
	if err != nil {
		return err
	}

	reqPath := filepath.Join(dir, fmt.Sprintf("%d_req.zst", resultID))
	respPath := filepath.Join(dir, fmt.Sprintf("%d_resp.zst", resultID))

	if err := os.WriteFile(reqPath, compressedReq, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(respPath, compressedResp, 0600); err != nil {
		_ = os.Remove(reqPath)
		return err
	}

	counter.Add(totalNewBytes)
	return nil
}

// GetResultPayload retrieves and decompresses request and response blobs, truncating
// the response at maxBytes on safe UTF-8 boundaries.
func (s *Storage) GetResultPayload(sessionID int64, resultID int64, maxBytes int) (*PayloadDTO, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxPreviewBytes
	}

	dir := filepath.Join(s.baseDir, fmt.Sprintf("%d", sessionID))
	reqPath := filepath.Join(dir, fmt.Sprintf("%d_req.zst", resultID))
	respPath := filepath.Join(dir, fmt.Sprintf("%d_resp.zst", resultID))

	var rawReq, rawResp []byte

	// Read request if available
	if compReq, err := os.ReadFile(reqPath); err == nil {
		s.mu.Lock()
		decompReq, decErr := s.decoder.DecodeAll(compReq, nil)
		s.mu.Unlock()
		if decErr == nil {
			rawReq = decompReq
		}
	}

	// Read response if available
	compResp, err := os.ReadFile(respPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	s.mu.Lock()
	decompResp, decErr := s.decoder.DecodeAll(compResp, nil)
	s.mu.Unlock()
	if decErr != nil {
		return nil, fmt.Errorf("decompression failed: %w", decErr)
	}
	rawResp = decompResp

	fullSize := int64(len(rawResp))
	truncated := false
	previewRespBytes := rawResp

	if len(rawResp) > maxBytes {
		truncated = true
		previewRespBytes = safeTruncateUTF8(rawResp[:maxBytes])
	}

	// Detect content type from initial bytes
	contentType := "text/plain; charset=utf-8"
	if len(rawResp) > 0 {
		probeLen := 512
		if len(rawResp) < 512 {
			probeLen = len(rawResp)
		}
		trimmed := bytes.TrimSpace(rawResp[:probeLen])
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			contentType = "application/json; charset=utf-8"
		} else {
			contentType = http.DetectContentType(rawResp[:probeLen])
		}
	}

	return &PayloadDTO{
		SessionID:     sessionID,
		ResultID:      resultID,
		RequestText:   string(rawReq),
		ResponseText:  string(previewRespBytes),
		Truncated:     truncated,
		FullSizeBytes: fullSize,
		ContentType:   contentType,
	}, nil
}

// safeTruncateUTF8 cuts a byte slice safely on a UTF-8 rune boundary.
func safeTruncateUTF8(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	// Step back up to 3 bytes to find valid UTF-8 boundary
	for i := 0; i < 4 && len(b) > 0; i++ {
		if utf8.Valid(b) {
			return b
		}
		b = b[:len(b)-1]
	}
	return b
}

// DeleteSessionBlobs removes all blob files for a completed or deleted session.
func (s *Storage) DeleteSessionBlobs(sessionID int64) error {
	dir := filepath.Join(s.baseDir, fmt.Sprintf("%d", sessionID))
	s.sessionSizes.Delete(sessionID)
	return os.RemoveAll(dir)
}

// Close closes zstd decoder and encoder resources.
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.encoder != nil {
		_ = s.encoder.Close()
	}
	if s.decoder != nil {
		s.decoder.Close()
	}
	return nil
}
