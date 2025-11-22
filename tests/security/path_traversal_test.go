package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestPathTraversalPrevention_Comprehensive(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedRoot := tempDir

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid relative path", "subdir/file.txt", false},
		{"valid nested path", "a/b/c/file.txt", false},
		{"basic traversal", "../../../etc/passwd", true},
		{"single traversal", "../outside", true},
		{"double traversal", "../../outside", true},
		{"traversal with file", "../../../etc/passwd", true},
		{"traversal at start", "../file.txt", true},
		{"encoded traversal", "..%2f..%2fetc", true},
		{"backslash traversal", "..\\..\\etc", true},
		{"mixed separators", "../..\\etc", true},
		{"null byte", "file\x00.txt", true},
		{"absolute path outside", "/etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(allowedRoot, tt.path)
			_, err := values.NewSafePath(fullPath, allowedRoot)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected path traversal to be blocked: %s", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid path: %v", err)
				}
			}
		})
	}
}

func TestWindowsReservedNames(t *testing.T) {
	tempDir := testutil.TempDir(t)

	reservedNames := []string{
		"con", "prn", "aux", "nul",
		"com1", "com2", "com3", "com4",
		"lpt1", "lpt2", "lpt3", "lpt4",
	}

	for _, name := range reservedNames {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(tempDir, name)
			_, err := values.NewSafePath(path, tempDir)
			if err == nil {
				t.Errorf("expected Windows reserved name to be blocked: %s", name)
			}
		})
	}
}

func TestSymlinkAttackPrevention(t *testing.T) {
	tempDir := testutil.TempDir(t)

	// Create a real directory
	realDir := filepath.Join(tempDir, "real")
	err := os.MkdirAll(realDir, 0755)
	testutil.AssertNoError(t, err)

	// Create a file in real directory
	realFile := filepath.Join(realDir, "file.txt")
	err = os.WriteFile(realFile, []byte("content"), 0644)
	testutil.AssertNoError(t, err)

	// Try to create a symlink (may not work on Windows)
	linkPath := filepath.Join(tempDir, "link")
	err = os.Symlink(realDir, linkPath)
	if err != nil {
		// Symlinks may not be supported, skip test
		t.Skip("symlinks not supported on this platform")
	}

	// Accessing through symlink should resolve to real path
	safePath, err := values.NewSafePath(linkPath, tempDir)
	testutil.AssertNoError(t, err)

	// The resolved path should be the real directory
	if safePath.Absolute() != realDir {
		t.Errorf("expected symlink to resolve to %q, got %q", realDir, safePath.Absolute())
	}
}

func TestRootBoundaryEnforcement(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedRoot := filepath.Join(tempDir, "workspace")
	err := os.MkdirAll(allowedRoot, 0755)
	testutil.AssertNoError(t, err)

	// Paths outside root should be blocked
	outsidePath := filepath.Join(tempDir, "outside")
	_, err = values.NewSafePath(outsidePath, allowedRoot)
	if err == nil {
		t.Error("expected path outside root to be blocked")
	}

	// Paths inside root should be allowed
	insidePath := filepath.Join(allowedRoot, "inside")
	_, err = values.NewSafePath(insidePath, allowedRoot)
	if err != nil {
		t.Errorf("unexpected error for path inside root: %v", err)
	}
}


