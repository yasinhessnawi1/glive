package values

import (
	"fmt"
	"path/filepath"
	"runtime"
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
		"..":  true,
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
	// Checked against the RAW input, before trimming — see rejectTrimmableControlChars.
	if err := rejectTrimmableControlChars(value); err != nil {
		return nil, err
	}

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

// rejectTrimmableControlChars rejects control characters that strings.TrimSpace
// would silently remove: \n, \r, \v and \f. (Tab is excluded — the validator has
// always permitted it, and space is not a control character.)
//
// Those four are the only control characters the post-trim check below could never
// see, because trimming ran first and deleted them. The effect was that a path with
// a leading or trailing newline was accepted and the caller got back a value
// different from the one it passed — silent normalisation of a security-relevant
// input, which is how argument- and log-injection bugs start.
//
// It deliberately does NOT check every control character: NUL and friends are still
// caught by the existing post-trim checks, which keeps their error codes
// (PATH_INVALID_CHAR for the dangerousChars set) unchanged for callers.
func rejectTrimmableControlChars(s string) error {
	for _, r := range s {
		if unicode.IsControl(r) && unicode.IsSpace(r) && r != '\t' {
			return errors.NewUserError("PATH_CONTROL_CHAR", "Path contains control characters")
		}
	}
	return nil
}

// NewSafePath creates a new SafePath with root boundary enforcement
func NewSafePath(path string, allowedRoot string) (*SafePath, error) {
	// Checked against the RAW input, before trimming — same reason as NewPath.
	if err := rejectTrimmableControlChars(path); err != nil {
		return nil, err
	}

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

	if err := rejectForeignSeparator(path); err != nil {
		return nil, err
	}

	// Canonicalise BOTH sides the same way before comparing them. Resolving only
	// the candidate (as this used to) reports /tmp as outside /tmp on macOS, where
	// /tmp is a symlink, and C:\Users\RUNNER~1\... as outside itself on Windows.
	// An absolute candidate is never joined under the root: it is accepted only if
	// it is under the root in canonical form.
	absPath, err := CanonicalPath(path)
	if err != nil {
		return nil, errors.WrapSystem(err, "failed to resolve path")
	}

	absRoot, err := CanonicalPath(allowedRoot)
	if err != nil {
		return nil, errors.WrapSystem(err, "failed to resolve root path")
	}

	if !IsWithinRoot(absRoot, absPath) {
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

// rejectForeignSeparator fails closed on a backslash where it is not a path
// separator. On POSIX a backslash is an ordinary filename character, so
// "..\\..\\etc" joined under a root names a file INSIDE the root and no traversal
// check can see it. GLive paths come from repositories, URLs and AI output; a
// backslash on Linux or macOS is never a legitimate path. On Windows the
// backslash IS the separator and filepath.Clean handles traversal in it.
func rejectForeignSeparator(path string) error {
	if runtime.GOOS != "windows" && strings.ContainsRune(path, '\\') {
		return errors.NewUserError("PATH_INVALID_CHAR",
			fmt.Sprintf("Path contains invalid character: %q", '\\'))
	}
	return nil
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
