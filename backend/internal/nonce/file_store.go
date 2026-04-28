package nonce

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// FileStore implements Store backed by an append-only log file.
// Each line has the format "nonce|unix-timestamp".
// A mutex provides thread safety for all file operations.
type FileStore struct {
	mu   sync.Mutex
	path string
}

// NewFileStore creates a new FileStore. It creates the parent directory and
// file if they do not exist.
func NewFileStore(path string) (*FileStore, error) {
	dir := ""
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		dir = path[:idx]
	}
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create nonce dir: %w", err)
		}
	}
	// Create file if it does not exist.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("create nonce file: %w", err)
	}
	f.Close()

	return &FileStore{path: path}, nil
}

// CheckAndSet scans the file for an existing entry with the same nonce.
// If found it returns false (duplicate). Otherwise it appends a new line.
func (s *FileStore) CheckAndSet(_ context.Context, nonce string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.path)
	if err != nil {
		return false, fmt.Errorf("open nonce file: %w", err)
	}

	now := time.Now()
	cutoff := now.Add(-ttl)
	found := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == nonce {
			ts, err := time.Parse(time.RFC3339Nano, parts[1])
			if err != nil {
				// If we cannot parse the timestamp, treat it as still valid.
				found = true
				break
			}
			if ts.After(cutoff) {
				found = true
				break
			}
			// Expired entry — treat as not found, will be cleaned later.
		}
	}
	f.Close()

	if found {
		return false, nil
	}

	// Append the new nonce.
	af, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return false, fmt.Errorf("open nonce file for append: %w", err)
	}
	defer af.Close()

	line := fmt.Sprintf("%s|%s\n", nonce, now.Format(time.RFC3339Nano))
	if _, err := af.WriteString(line); err != nil {
		return false, fmt.Errorf("write nonce: %w", err)
	}

	return true, nil
}

// Clean reads all lines, keeps only those within 5 minutes, and rewrites the file.
func (s *FileStore) Clean(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.path)
	if err != nil {
		return fmt.Errorf("open nonce file: %w", err)
	}

	cutoff := time.Now().Add(-5 * time.Minute)
	var kept []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, parts[1])
		if err != nil {
			continue
		}
		if ts.After(cutoff) {
			kept = append(kept, line)
		}
	}
	f.Close()

	// Rewrite file with only the kept lines.
	wf, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("rewrite nonce file: %w", err)
	}
	defer wf.Close()

	for _, line := range kept {
		if _, err := fmt.Fprintln(wf, line); err != nil {
			return fmt.Errorf("write kept nonce: %w", err)
		}
	}
	return nil
}
