package values_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestNewPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid path", "/tmp/test", false},
		{"relative path", "test/path", false},
		{"empty path", "", true},
		{"path with control char", "/tmp/test\x00", true},
		{"path with newline", "/tmp/test\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := values.NewPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if path.Value() == "" {
				t.Error("path value should not be empty")
			}
		})
	}
}

func TestNewSafePath(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedRoot := tempDir

	// joinRoot says whether the case's path is relative to root (and so must be
	// joined to it) or is passed to NewSafePath verbatim. The "empty path" case
	// needs the latter: filepath.Join(root, "") returns root, a perfectly valid
	// non-empty directory, so joining silently turned the empty-path case into a
	// valid-path case that could never produce PATH_EMPTY.
	tests := []struct {
		name        string
		path        string
		root        string
		joinRoot    bool
		wantErr     bool
		errContains string
	}{
		{"valid path under root", "subdir/file.txt", allowedRoot, true, false, ""},
		{"path traversal", "../../../etc/passwd", allowedRoot, true, true, "PATH_TRAVERSAL"},
		{"windows reserved name", "con", allowedRoot, true, true, "PATH_DANGEROUS"},
		{"control character", "file\x00.txt", allowedRoot, true, true, "PATH_INVALID_CHAR"},
		{"empty path", "", allowedRoot, false, true, "PATH_EMPTY"},
		{"whitespace-only path", "   ", allowedRoot, false, true, "PATH_EMPTY"},
		{"path with trailing newline", "subdir/file.txt\n", allowedRoot, false, true, "PATH_CONTROL_CHAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := tt.path
			if tt.joinRoot {
				fullPath = filepath.Join(tt.root, tt.path)
			}
			safePath, err := values.NewSafePath(fullPath, tt.root)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errContains != "" {
					testutil.AssertContains(t, err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if safePath.Absolute() == "" {
				t.Error("absolute path should not be empty")
			}
		})
	}
}

func TestSafePath_Join(t *testing.T) {
	tempDir := testutil.TempDir(t)
	safePath, err := values.NewSafePath(tempDir, tempDir)
	testutil.AssertNoError(t, err)

	joined, err := safePath.Join("subdir", "file.txt")
	testutil.AssertNoError(t, err)

	if joined.Absolute() == "" {
		t.Error("joined path should not be empty")
	}

	// Verify it's still under root. filepath.Rel is used rather than the deprecated
	// filepath.HasPrefix, which compares raw strings: it does not respect path
	// boundaries (so "/rootx" counts as being under "/root") and ignores case rules.
	// A relative path that does not start with ".." proves real containment.
	rel, err := filepath.Rel(tempDir, joined.Absolute())
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Errorf("joined path %q should be under root %q (rel=%q, err=%v)",
			joined.Absolute(), tempDir, rel, err)
	}
}

func TestSafePath_SymlinkResolution(t *testing.T) {
	tempDir := testutil.TempDir(t)
	realDir := filepath.Join(tempDir, "real")
	err := os.MkdirAll(realDir, 0755)
	testutil.AssertNoError(t, err)

	linkPath := filepath.Join(tempDir, "link")
	err = os.Symlink(realDir, linkPath)
	if err != nil {
		// Symlinks may not be supported on Windows, skip test
		t.Skip("symlinks not supported")
	}

	safePath, err := values.NewSafePath(linkPath, tempDir)
	testutil.AssertNoError(t, err)

	// The absolute path should resolve the symlink - to the real directory in its
	// canonical spelling. The temp dir itself may be reached through a symlink
	// (macOS: /var -> /private/var) or an 8.3 short name (Windows: RUNNER~1), so
	// the expectation is canonicalised the same way.
	wantDir, err := filepath.EvalSymlinks(realDir)
	testutil.AssertNoError(t, err)
	if safePath.Absolute() != wantDir {
		t.Errorf("expected symlink to resolve to %q, got %q", wantDir, safePath.Absolute())
	}
}
