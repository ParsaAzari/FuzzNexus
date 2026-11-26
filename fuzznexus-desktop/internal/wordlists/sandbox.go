package wordlists

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrSandboxLocked    = errors.New("sandbox file is actively locked by a running scan")
	ErrSandboxTraversal = errors.New("potential path traversal out of sandbox boundary")
)

// SandboxManager orchestrates isolated temporary wordlists and active scan locks.
type SandboxManager struct {
	dir            string
	lockedPaths    map[string]int
	ephemeralPaths map[string]bool
	mu             sync.RWMutex
}

// NewSandboxManager initializes the sandbox directory with restricted owner-only permissions.
func NewSandboxManager(baseDataDir string) (*SandboxManager, error) {
	sandboxDir := filepath.Join(baseDataDir, "sandboxes", "wordlists")
	if err := os.MkdirAll(sandboxDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create sandbox directory: %w", err)
	}

	sm := &SandboxManager{
		dir:            sandboxDir,
		lockedPaths:    make(map[string]int),
		ephemeralPaths: make(map[string]bool),
	}

	// Prune orphaned temp files older than 1 hour from any previous unclean exit
	_ = sm.PruneOrphans(1 * time.Hour)

	return sm, nil
}

// Dir returns the absolute path to the sandbox directory.
func (s *SandboxManager) Dir() string {
	return s.dir
}

// ValidateSandboxPath verifies that a path is strictly inside the sandbox directory.
func (s *SandboxManager) ValidateSandboxPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	evalSandbox, err := filepath.EvalSymlinks(s.dir)
	if err != nil {
		evalSandbox = filepath.Clean(s.dir)
	}

	evalTarget, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		evalTarget = cleanPath
	}

	// Enforce prefix containment
	rel, err := filepath.Rel(evalSandbox, evalTarget)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: path %s escapes %s", ErrSandboxTraversal, cleanPath, s.dir)
	}

	return evalTarget, nil
}

// CreateSandboxFile generates a secure unique file inside the sandbox directory with 0600 permissions.
func (s *SandboxManager) CreateSandboxFile(prefix string) (*os.File, string, error) {
	if prefix == "" {
		prefix = "sb_wordlist"
	}

	// Generate random token for non-guessable file name
	tokenBytes := make([]byte, 8)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	timestamp := time.Now().UTC().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s.tmp", prefix, timestamp, token)
	targetPath := filepath.Join(s.dir, filename)

	// O_EXCL ensures we fail if file somehow already exists
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		return nil, "", fmt.Errorf("failed creating sandbox file: %w", err)
	}

	return f, targetPath, nil
}

// RegisterEphemeral marks a sandbox file as ephemeral so it gets cleaned up after scan.
func (s *SandboxManager) RegisterEphemeral(path string) {
	clean := filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ephemeralPaths[clean] = true
}

// IsEphemeral checks if a path was registered as an ephemeral sandbox file.
func (s *SandboxManager) IsEphemeral(path string) bool {
	clean := filepath.Clean(path)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ephemeralPaths[clean]
}

// Lock marks a sandbox path as actively held by a scan session.
func (s *SandboxManager) Lock(path string) {
	clean := filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lockedPaths[clean]++
}

// Unlock decrements the lock count on a sandbox path.
func (s *SandboxManager) Unlock(path string) {
	clean := filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	if count, ok := s.lockedPaths[clean]; ok {
		if count <= 1 {
			delete(s.lockedPaths, clean)
		} else {
			s.lockedPaths[clean] = count - 1
		}
	}
}

// IsLocked checks whether the sandbox path is currently locked.
func (s *SandboxManager) IsLocked(path string) bool {
	clean := filepath.Clean(path)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lockedPaths[clean] > 0
}

// CleanupPath safely deletes a sandbox file if it is not currently locked.
func (s *SandboxManager) CleanupPath(path string) error {
	clean := filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockedPaths[clean] > 0 {
		return ErrSandboxLocked
	}

	delete(s.ephemeralPaths, clean)
	delete(s.lockedPaths, clean)

	if err := os.Remove(clean); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed removing sandbox file %s: %w", clean, err)
	}
	return nil
}

// CleanupAllUnlockedEphemeral removes all registered ephemeral files that are not locked.
func (s *SandboxManager) CleanupAllUnlockedEphemeral() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []string
	for path := range s.ephemeralPaths {
		if s.lockedPaths[path] == 0 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			} else {
				delete(s.ephemeralPaths, path)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during sandbox cleanup: %s", strings.Join(errs, "; "))
	}
	return nil
}

// PruneOrphans deletes temp files in the sandbox directory older than maxAge.
func (s *SandboxManager) PruneOrphans(maxAge time.Duration) error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			fullPath := filepath.Join(s.dir, entry.Name())
			_ = os.Remove(fullPath)
		}
	}
	return nil
}
