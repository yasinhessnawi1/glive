package values_test

import (
	"os"
	"path/filepath"
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

	tests := []struct {
		name       string
		path       string
		root       string
		wantErr    bool
		errContains string
	}{
		{"valid path under root", "subdir/file.txt", allowedRoot, false, ""},
		{"path traversal", "../../../etc/passwd", allowedRoot, true, "PATH_TRAVERSAL"},
		{"windows reserved name", "con", allowedRoot, true, "PATH_DANGEROUS"},
		{"control character", "file\x00.txt", allowedRoot, true, "PATH_INVALID_CHAR"},
		{"empty path", "", allowedRoot, true, "PATH_EMPTY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(tt.root, tt.path)
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

	// Verify it's still under root
	if !filepath.HasPrefix(joined.Absolute(), tempDir) {
		t.Errorf("joined path %q should be under root %q", joined.Absolute(), tempDir)
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

	// The absolute path should resolve the symlink
	if safePath.Absolute() != realDir {
		t.Errorf("expected symlink to resolve to %q, got %q", realDir, safePath.Absolute())
	}
}


