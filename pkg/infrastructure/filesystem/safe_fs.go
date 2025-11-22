package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("invalid root path %s: %w", root, err)
		}

		// Resolve any symlinks to prevent symlink attacks
		real, err := filepath.EvalSymlinks(abs)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to resolve symlinks for %s: %w", abs, err)
		}
		if real != "" {
			abs = real
		}

		// Normalize path separators
		abs = filepath.Clean(abs)
		normalized = append(normalized, abs)
	}

	return &SafeFileSystem{allowedRoots: normalized}, nil
}

// ValidatePath validates that a path is within allowed roots
func (fs *SafeFileSystem) ValidatePath(path string) error {
	// Get absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Clean the path
	abs = filepath.Clean(abs)

	// Resolve symlinks to prevent symlink attacks
	real, err := filepath.EvalSymlinks(abs)
	if err != nil && !os.IsNotExist(err) {
		// For new files, check parent directory
		parent := filepath.Dir(abs)
		real, err = filepath.EvalSymlinks(parent)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
		real = filepath.Join(real, filepath.Base(abs))
	}
	if real != "" {
		abs = real
	}

	// Normalize path separators
	abs = filepath.Clean(abs)

	// Check if path is under any allowed root
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	for _, root := range fs.allowedRoots {
		// Check if path is within root
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			continue
		}

		// If relative path doesn't start with .., it's within the root
		if !strings.HasPrefix(rel, "..") && rel != ".." {
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
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("invalid root path: %w", err)
	}

	real, err := filepath.EvalSymlinks(abs)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}
	if real != "" {
		abs = real
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	abs = filepath.Clean(abs)
	for _, existing := range fs.allowedRoots {
		if existing == abs {
			return nil // Already exists
		}
	}

	fs.allowedRoots = append(fs.allowedRoots, abs)
	return nil
}

