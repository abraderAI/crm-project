// Package upload provides file upload handling with pluggable storage providers.
package upload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// StorageProvider abstracts file storage for testability and swappability.
type StorageProvider interface {
	// Store saves file content and returns the storage path.
	Store(filename string, content io.Reader) (string, error)
	// Get returns a reader for the stored file.
	Get(storagePath string) (io.ReadCloser, error)
	// Delete removes a stored file.
	Delete(storagePath string) error
}

// LocalStorage implements StorageProvider using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a local filesystem storage provider.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating upload directory: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

// Store saves a file to the local filesystem with a unique name.
func (s *LocalStorage) Store(filename string, content io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	storageName := uuid.New().String() + ext
	storagePath := filepath.Join(s.baseDir, storageName)

	f, err := os.Create(storagePath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, content); err != nil {
		_ = os.Remove(storagePath)
		return "", fmt.Errorf("writing file: %w", err)
	}

	return storageName, nil
}

// Get returns a reader for a stored file.
func (s *LocalStorage) Get(storagePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, storagePath)
	// Prevent directory traversal.
	clean := filepath.Clean(fullPath)
	if !isSubpath(s.baseDir, clean) {
		return nil, fmt.Errorf("invalid storage path")
	}
	f, err := os.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return f, nil
}

// Delete removes a stored file.
func (s *LocalStorage) Delete(storagePath string) error {
	fullPath := filepath.Join(s.baseDir, storagePath)
	clean := filepath.Clean(fullPath)
	if !isSubpath(s.baseDir, clean) {
		return fmt.Errorf("invalid storage path")
	}
	if err := os.Remove(clean); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file: %w", err)
	}
	return nil
}

// isSubpath checks if child is under parent directory.
func isSubpath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && rel[0] != '.'
}
