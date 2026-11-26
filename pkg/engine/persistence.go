package engine

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ResumeState holds the complete state needed to safely pause and resume execution.
// Raw IPs are strictly prohibited and never persisted.
type ResumeState struct {
	SessionID        int64             `json:"session_id,omitempty"`
	TargetURL        string            `json:"target_url,omitempty"`
	Method           string            `json:"method,omitempty"`
	Wordlist         string            `json:"wordlist,omitempty"`
	BrowserProfile   string            `json:"browser_profile,omitempty"`
	TLSProfile       string            `json:"tls_profile,omitempty"`
	CurrentIndex     int               `json:"current_index"`
	IPHash           string            `json:"ip_hash"` // Only 8-char SHA-256 hash
	Headers          map[string]string `json:"headers"`
	EncryptedCookies string            `json:"encrypted_cookies"` // AES-256-GCM hex
	CookiesNonce     string            `json:"cookies_nonce"`     // AES-256-GCM nonce hex
	Stats            RollingStats      `json:"stats"`
	Timestamp        time.Time         `json:"timestamp"`
}

// GetDefaultResumeFilePath returns ~/.ffuf/resume_state.json.
func GetDefaultResumeFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".ffuf", "resume_state.json")
}

// EncryptCookies encrypts cookies plaintext using AES-256-GCM with machine-derived key.
func EncryptCookies(cookies string) (ciphertextHex, nonceHex string, err error) {
	if cookies == "" {
		return "", "", nil
	}

	key := deriveMachineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(cookies), nil)
	return hex.EncodeToString(ciphertext), hex.EncodeToString(nonce), nil
}

// DecryptCookies decrypts AES-256-GCM ciphertext using machine-derived key.
func DecryptCookies(ciphertextHex, nonceHex string) (string, error) {
	if ciphertextHex == "" || nonceHex == "" {
		return "", nil
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return "", err
	}

	key := deriveMachineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("cookie decryption failed (machine mismatch): %w", err)
	}

	return string(plaintext), nil
}

// SaveResumeState writes the ResumeState to disk ensuring parent directories exist.
func SaveResumeState(filePath string, state ResumeState) error {
	if filePath == "" {
		filePath = GetDefaultResumeFilePath()
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Double-check no raw IP is ever written
	if len(state.IPHash) > 16 {
		state.IPHash = HashString8(state.IPHash)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0600)
}

// LoadResumeState reads and parses the ResumeState file from disk.
func LoadResumeState(filePath string) (*ResumeState, error) {
	if filePath == "" {
		filePath = GetDefaultResumeFilePath()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var state ResumeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// DeleteResumeState removes the ResumeState file upon successful resumption.
func DeleteResumeState(filePath string) error {
	if filePath == "" {
		filePath = GetDefaultResumeFilePath()
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(filePath)
}

// deriveMachineKey derives a 32-byte AES-256 key from host machine identity.
func deriveMachineKey() []byte {
	seed := runtime.GOOS + runtime.GOARCH
	if host, err := os.Hostname(); err == nil {
		seed += host
	}
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		seed += string(data)
	} else if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		seed += string(data)
	}

	hash := sha256.Sum256([]byte(seed))
	return hash[:]
}
