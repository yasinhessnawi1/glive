package values

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/glive/domain/errors"
)

// Path represents a validated file system path
type Path struct {
	value string
}

var (
	// Dangerous path components
	dangerousNames = map[string]bool{
		"..": true,
		"con": true, "prn": true, "aux": true, "nul": true, // Windows reserved
		"com1": true, "com2": true, "com3": true, "com4": true,
		"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	}

	// Dangerous characters
	dangerousChars = []rune{'\x00', '\n', '\r', '|', '<', '>', '"', '?', '*'}
)

// SafePath represents a validated safe path with root boundary enforcement
type SafePath struct {
	absolute string
	relative string
	root     string
}

// NewPath creates a new validated Path
func NewPath(value string) (*Path, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.NewUserError("PATH_EMPTY", "Path cannot be empty")
	}

	// Check for dangerous characters
	for _, char := range dangerousChars {
		if strings.ContainsRune(value, char) {
			return nil, errors.NewUserError("PATH_INVALID_CHAR",
				fmt.Sprintf("Path contains invalid character: %q", char))
		}
	}

	// Check for control characters
	for _, r := range value {
		if unicode.IsControl(r) && r != '\t' {
			return nil, errors.NewUserError("PATH_CONTROL_CHAR", "Path contains control characters")
		}
	}

	return &Path{value: value}, nil
}

// NewSafePath creates a new SafePath with root boundary enforcement
func NewSafePath(path string, allowedRoot string) (*SafePath, error) {
	// Trim and validate
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.NewUserError("PATH_EMPTY", "Path cannot be empty")
	}

	// Check for dangerous characters
	for _, char := range dangerousChars {
		if strings.ContainsRune(path, char) {
			return nil, errors.NewUserError("PATH_INVALID_CHAR",
				fmt.Sprintf("Path contains invalid character: %q", char))
		}
	}

	// Check for control characters
	for _, r := range path {
		if unicode.IsControl(r) && r != '\t' {
			return nil, errors.NewUserError("PATH_CONTROL_CHAR", "Path contains control characters")
		}
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WrapSystem(err, "failed to resolve absolute path")
	}

	// Resolve symlinks to prevent symlink attacks
	realPath, err := resolveSymlinks(absPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.WrapSystem(err, "failed to resolve symlinks")
	}
	if realPath != "" {
		absPath = realPath
	}

	// Ensure path is under allowed root
	absRoot, err := filepath.Abs(allowedRoot)
	if err != nil {
		return nil, errors.WrapSystem(err, "failed to resolve root path")
	}

	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		return nil, errors.NewSecurityError("PATH_TRAVERSAL",
			fmt.Sprintf("Path %s is outside allowed root %s", path, allowedRoot))
	}

	// Check each path component
	parts := strings.Split(absPath, string(filepath.Separator))
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if dangerous, exists := dangerousNames[lower]; exists && dangerous {
			return nil, errors.NewSecurityError("PATH_DANGEROUS",
				fmt.Sprintf("Path contains dangerous component: %s", part))
		}
	}

	// Calculate relative path
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return nil, errors.WrapSystem(err, "failed to calculate relative path")
	}

	return &SafePath{
		absolute: absPath,
		relative: relPath,
		root:     absRoot,
	}, nil
}

func resolveSymlinks(path string) (string, error) {
	// Check if path exists
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	// If it's a symlink, resolve it
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		return target, nil
	}

	return path, nil
}

// Value returns the string value of the Path
func (p *Path) Value() string {
	return p.value
}

// String returns the string representation of the Path
func (p *Path) String() string {
	return p.value
}

// SafePath methods

// Absolute returns the absolute path
func (p *SafePath) Absolute() string {
	return p.absolute
}

// Relative returns the relative path from root
func (p *SafePath) Relative() string {
	return p.relative
}

// Root returns the root directory
func (p *SafePath) Root() string {
	return p.root
}

// Join joins path elements and returns a new SafePath
func (p *SafePath) Join(elem ...string) (*SafePath, error) {
	joined := filepath.Join(append([]string{p.absolute}, elem...)...)
	return NewSafePath(joined, p.root)
}
