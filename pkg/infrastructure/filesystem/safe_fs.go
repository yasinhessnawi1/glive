package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/glive/domain/values"
)

const (
	// MaxFileSize is the maximum file size allowed (10MB)
	MaxFileSize = 10 * 1024 * 1024
)

var (
	ErrPathTraversal = fmt.Errorf("path traversal detected")
	ErrFileTooLarge  = fmt.Errorf("file exceeds maximum size")
	ErrOutsideRoot   = fmt.Errorf("path is outside allowed root")
)

// SafeFileSystem provides secure file operations with path validation
type SafeFileSystem struct {
	allowedRoots []string
	mu           sync.RWMutex
}

// NewSafeFileSystem creates a new SafeFileSystem with allowed root directories
func NewSafeFileSystem(allowedRoots ...string) (*SafeFileSystem, error) {
	if len(allowedRoots) == 0 {
		return nil, fmt.Errorf("at least one allowed root required")
	}

	normalized := make([]string, 0, len(allowedRoots))
	for _, root := range allowedRoots {
		// Roots and candidates are canonicalised by the same function so that a
		// symlinked or 8.3-short-named root compares equal to paths below it.
		abs, err := values.CanonicalPath(root)
		if err != nil {
			return nil, fmt.Errorf("invalid root path %s: %w", root, err)
		}
		normalized = append(normalized, abs)
	}

	return &SafeFileSystem{allowedRoots: normalized}, nil
}

// ValidatePath validates that a path is within allowed roots
func (fs *SafeFileSystem) ValidatePath(path string) error {
	// Canonical form: symlinks resolved through the nearest existing ancestor, so
	// a path that does not exist yet (the state directory before MkdirAll) and the
	// same path once created are compared in the same spelling as the root.
	abs, err := values.CanonicalPath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	for _, root := range fs.allowedRoots {
		if values.IsWithinRoot(root, abs) {
			return nil
		}
	}

	return fmt.Errorf("%w: %s is not under allowed roots", ErrPathTraversal, path)
}

// ReadFile reads a file securely
func (fs *SafeFileSystem) ReadFile(path string) ([]byte, error) {
	if err := fs.ValidatePath(path); err != nil {
		return nil, err
	}

	// Check file size before reading
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, info.Size(), MaxFileSize)
	}

	return os.ReadFile(path)
}

// WriteFile writes a file securely
func (fs *SafeFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := fs.ValidatePath(path); err != nil {
		return err
	}

	// Enforce maximum file size
	if len(data) > MaxFileSize {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, len(data), MaxFileSize)
	}

	return os.WriteFile(path, data, perm)
}

// MkdirAll creates directories securely
func (fs *SafeFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err := fs.ValidatePath(path); err != nil {
		return err
	}
	return os.MkdirAll(path, perm)
}

// Stat returns file info securely
func (fs *SafeFileSystem) Stat(path string) (os.FileInfo, error) {
	if err := fs.ValidatePath(path); err != nil {
		return nil, err
	}
	return os.Stat(path)
}

// Remove removes a file securely
func (fs *SafeFileSystem) Remove(path string) error {
	if err := fs.ValidatePath(path); err != nil {
		return err
	}
	return os.Remove(path)
}

// RemoveAll removes a directory tree securely
func (fs *SafeFileSystem) RemoveAll(path string) error {
	if err := fs.ValidatePath(path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

// ReadDir reads a directory securely
func (fs *SafeFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	if err := fs.ValidatePath(path); err != nil {
		return nil, err
	}
	return os.ReadDir(path)
}

// Join joins path elements securely
func (fs *SafeFileSystem) Join(elem ...string) string {
	joined := filepath.Join(elem...)
	// Validate the joined path
	if err := fs.ValidatePath(joined); err == nil {
		return joined
	}
	// If validation fails, return cleaned path anyway (caller should validate)
	return filepath.Clean(joined)
}

// AddAllowedRoot adds an additional allowed root directory
func (fs *SafeFileSystem) AddAllowedRoot(root string) error {
	abs, err := values.CanonicalPath(root)
	if err != nil {
		return fmt.Errorf("invalid root path: %w", err)
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, existing := range fs.allowedRoots {
		if existing == abs {
			return nil // Already exists
		}
	}

	fs.allowedRoots = append(fs.allowedRoots, abs)
	return nil
}
