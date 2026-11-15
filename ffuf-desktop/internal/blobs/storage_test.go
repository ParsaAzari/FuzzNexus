package blobs

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func setupTestStorage(t *testing.T, quota int64) (*Storage, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ffuf-blobs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	st, err := NewStorage(tempDir, quota)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	return st, tempDir
}

func TestSaveAndGetResultPayload(t *testing.T) {
	st, dir := setupTestStorage(t, DefaultSessionQuota)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	sessionID := int64(100)
	resultID := int64(1)
	req := []byte("GET /admin HTTP/1.1\r\nHost: example.com\r\n\r\n")
	resp := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<html><body>Dashboard</body></html>")

	if err := st.SavePayload(sessionID, resultID, req, resp); err != nil {
		t.Fatalf("SavePayload failed: %v", err)
	}

	payload, err := st.GetResultPayload(sessionID, resultID, 1024*1024)
	if err != nil {
		t.Fatalf("GetResultPayload failed: %v", err)
	}

	if payload.RequestText != string(req) {
		t.Errorf("Request text mismatch: got %q, want %q", payload.RequestText, string(req))
	}
	if payload.ResponseText != string(resp) {
		t.Errorf("Response text mismatch: got %q, want %q", payload.ResponseText, string(resp))
	}
	if payload.Truncated {
		t.Error("Expected Truncated to be false")
	}
	if payload.FullSizeBytes != int64(len(resp)) {
		t.Errorf("Expected full size %d, got %d", len(resp), payload.FullSizeBytes)
	}
	if !strings.Contains(payload.ContentType, "text/html") && !strings.Contains(payload.ContentType, "text/plain") {
		t.Errorf("Unexpected ContentType: %s", payload.ContentType)
	}
}

func TestTruncationOnUTF8Boundary(t *testing.T) {
	st, dir := setupTestStorage(t, DefaultSessionQuota)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	sessionID := int64(101)
	resultID := int64(2)
	req := []byte("GET /unicode HTTP/1.1\r\n\r\n")

	// Create a payload with repeating multi-byte UTF-8 emojis and non-ASCII runes: "🚀 Persian سلام Chinese 你好"
	var respBuf bytes.Buffer
	token := "🚀 سلام 你好 "
	for respBuf.Len() < 10000 {
		respBuf.WriteString(token)
	}
	resp := respBuf.Bytes()

	if err := st.SavePayload(sessionID, resultID, req, resp); err != nil {
		t.Fatalf("SavePayload failed: %v", err)
	}

	// Limit to 500 bytes (which will likely slice in the middle of a multi-byte character)
	maxBytes := 500
	payload, err := st.GetResultPayload(sessionID, resultID, maxBytes)
	if err != nil {
		t.Fatalf("GetResultPayload failed: %v", err)
	}

	if !payload.Truncated {
		t.Error("Expected Truncated to be true")
	}
	if payload.FullSizeBytes != int64(len(resp)) {
		t.Errorf("Expected full size %d, got %d", len(resp), payload.FullSizeBytes)
	}
	if len(payload.ResponseText) > maxBytes {
		t.Errorf("ResponseText len %d exceeds maxBytes %d", len(payload.ResponseText), maxBytes)
	}
	if !utf8.ValidString(payload.ResponseText) {
		t.Errorf("ResponseText is NOT valid UTF-8 after truncation!")
	}
}

func TestQuotaEnforcement(t *testing.T) {
	// Set very small quota: 500 bytes
	smallQuota := int64(500)
	st, dir := setupTestStorage(t, smallQuota)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	sessionID := int64(102)
	req := []byte("GET /test HTTP/1.1\r\n\r\n")
	resp := bytes.Repeat([]byte("A"), 1000) // 1000 bytes uncompressed

	// First write might succeed if compressed fits under 500 bytes
	err := st.SavePayload(sessionID, 1, req, resp)
	if err != nil && !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("Unexpected error on write: %v", err)
	}

	// Subsequent writes of large uncompressible data will definitely exceed quota
	randomData := make([]byte, 1024)
	for i := range randomData {
		randomData[i] = byte(i % 256)
	}

	hitQuota := false
	for i := int64(2); i < 20; i++ {
		err := st.SavePayload(sessionID, i, req, randomData)
		if errors.Is(err, ErrQuotaExceeded) {
			hitQuota = true
			break
		}
	}

	if !hitQuota {
		t.Error("Expected ErrQuotaExceeded when exceeding session quota")
	}
}

func TestDeleteSessionBlobs(t *testing.T) {
	st, dir := setupTestStorage(t, DefaultSessionQuota)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	sessionID := int64(103)
	resultID := int64(1)
	req := []byte("GET / HTTP/1.1\r\n\r\n")
	resp := []byte("HTTP/1.1 200 OK\r\n\r\nHello")

	if err := st.SavePayload(sessionID, resultID, req, resp); err != nil {
		t.Fatalf("SavePayload failed: %v", err)
	}

	// Delete
	if err := st.DeleteSessionBlobs(sessionID); err != nil {
		t.Fatalf("DeleteSessionBlobs failed: %v", err)
	}

	// Subsequent read should return ErrNotFound
	_, err := st.GetResultPayload(sessionID, resultID, 1024)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}
}
