package wordlists

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures the path is clean, points to a regular file, and is not a special device.
func ValidatePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", ErrInvalidPath
	}

	cleanPath := filepath.Clean(path)

	// Reject Unix special device trees
	if strings.HasPrefix(cleanPath, "/dev/") || cleanPath == "/dev" {
		return "", fmt.Errorf("%w: cannot access /dev files", ErrInvalidPath)
	}

	// Resolve symlinks to prevent directory traversal / loop traps
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrFileNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrInvalidPath, err)
	}

	info, err := os.Stat(realPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrFileNotFound
		}
		return "", err
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: target is not a regular file (mode: %s)", ErrNotRegularFile, info.Mode().String())
	}

	return realPath, nil
}

// AtomicWriteFile safely writes content to targetPath using a temporary file, fsync, and atomic rename.
func AtomicWriteFile(targetPath string, content []byte) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".wordlist-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	// Ensure cleanup if rename doesn't complete
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	// Restrict permissions to owner only
	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}

	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Flush buffers to disk
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to fsync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	cleanup = false
	return nil
}

// CountLines streams through a file to count total lines and size in bytes without high memory usage.
func CountLines(path string) (int64, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, 0, err
	}
	sizeBytes := info.Size()

	var lineCount int64 = 0
	buf := make([]byte, 64*1024)
	var lastByte byte = '\n'

	for {
		n, err := file.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					lineCount++
				}
				lastByte = buf[i]
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, 0, err
		}
	}

	// If the file does not end in a newline and has content, count the final line
	if sizeBytes > 0 && lastByte != '\n' {
		lineCount++
	}

	return lineCount, sizeBytes, nil
}

// ReadLinesPage reads a paginated window of lines from path.
func ReadLinesPage(path string, offset int, limit int) ([]string, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 100
	} else if limit > 2000 {
		limit = 2000
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Allow lines up to 64KB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 64*1024)

	var lines []string
	var currentLine int = 0
	var totalLines int64 = 0

	for scanner.Scan() {
		text := scanner.Text()
		if currentLine >= offset && len(lines) < limit {
			lines = append(lines, text)
		}
		currentLine++
		totalLines++
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, 0, err
	}

	return lines, totalLines, nil
}

// ComputeStats gathers analytical metrics for the wordlist editor.
func ComputeStats(path string) (*WordlistStatsDTO, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 64*1024)

	var lineCount int64 = 0
	var totalChars int64 = 0
	var maxLen int = 0
	uniqueMap := make(map[string]struct{})

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		l := len(line)
		totalChars += int64(l)
		if l > maxLen {
			maxLen = l
		}
		// Track unique lines up to 200k to prevent unbounded memory growth
		if len(uniqueMap) < 200000 {
			uniqueMap[line] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, err
	}

	var avgLen float64 = 0
	if lineCount > 0 {
		avgLen = float64(totalChars) / float64(lineCount)
	}

	return &WordlistStatsDTO{
		LineCount:   lineCount,
		SizeBytes:   info.Size(),
		UniqueLines: int64(len(uniqueMap)),
		AvgLineLen:  avgLen,
		MaxLineLen:  maxLen,
	}, nil
}
