package sandbox

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath ensures a path stays within the sandbox root
func ValidatePath(sandboxRoot, path string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Resolve sandbox root
	absRoot, err := filepath.Abs(sandboxRoot)
	if err != nil {
		return fmt.Errorf("invalid sandbox root: %w", err)
	}

	// Check if path is within sandbox root
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return fmt.Errorf("path outside sandbox: %w", err)
	}

	// Prevent directory traversal
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	return nil
}

// SanitizePath ensures a path is safe and within sandbox
func SanitizePath(sandboxRoot, path string) (string, error) {
	// If path is relative, make it relative to sandbox root
	if !filepath.IsAbs(path) {
		path = filepath.Join(sandboxRoot, path)
	}

	// Validate it's within sandbox
	if err := ValidatePath(sandboxRoot, path); err != nil {
		return "", err
	}

	// Clean the path
	return filepath.Clean(path), nil
}

// ValidateMountPoint validates a mount point configuration
func ValidateMountPoint(sandboxRoot string, mount MountPoint) error {
	// Validate source exists (if not empty)
	if mount.Source != "" {
		if !filepath.IsAbs(mount.Source) {
			return fmt.Errorf("mount source must be absolute: %s", mount.Source)
		}
	}

	// Validate destination is within sandbox
	if err := ValidatePath(sandboxRoot, mount.Destination); err != nil {
		return fmt.Errorf("mount destination outside sandbox: %w", err)
	}

	return nil
}
