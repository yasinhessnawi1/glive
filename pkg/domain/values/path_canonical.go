package values

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// CanonicalPath returns the one spelling of path that every containment check
// compares: absolute, cleaned, with every symbolic link resolved and, on Windows,
// 8.3 short names expanded and on-disk case restored (filepath.EvalSymlinks does
// all three).
//
// A path that does not exist yet is resolved through its nearest existing
// ancestor and the remaining, already-cleaned tail is appended lexically. That
// keeps a root and a candidate in the same form whether the candidate has been
// created or not — the CI failures this fixes came from resolving one side and
// not the other (/tmp vs /private/tmp on macOS, RUNNER~1 vs runneradmin on
// Windows, a state directory validated before and after MkdirAll).
//
// It fails closed: any resolution error other than "does not exist" (a
// component that is a file, an invalid name, a missing volume) is returned.
func CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path %q: %w", path, err)
	}

	existing := abs
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(existing)
		if err == nil {
			return filepath.Join(append([]string{resolved}, tail...)...), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("resolve %q: %w", existing, err)
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", fmt.Errorf("resolve %q: no existing ancestor: %w", abs, err)
		}
		tail = append([]string{filepath.Base(existing)}, tail...)
		existing = parent
	}
}

// IsWithinRoot reports whether path is root itself or a descendant of it.
//
// Both arguments must already be canonical (see CanonicalPath) — comparing an
// unresolved spelling against a resolved one is exactly the defect this package
// exists to prevent. The test is by path component, never by string prefix:
// "/tmp2" is not under "/tmp", but "/tmp/..hidden" is. On Windows, filepath.Rel
// compares case-insensitively and refuses to relate paths on different volumes.
func IsWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
