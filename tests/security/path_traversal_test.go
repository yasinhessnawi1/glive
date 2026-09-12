package security

import (
	"os"
	"path/filepath"
	"runtime"
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
		// {"encoded traversal", "..%2f..%2fetc", true}, // URL encoding not supported in file paths
		{"backslash traversal", "..\\..\\etc", true},
		{"mixed separators", "../..\\etc", true},
		{"null byte", "file\x00.txt", true},
		{"absolute path outside", filepath.Join(filepath.VolumeName(allowedRoot)+string(filepath.Separator), "etc", "passwd"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// An absolute candidate is passed verbatim. Joining it under the root
			// would turn "/etc/passwd" into "<root>/etc/passwd" - a path that IS
			// inside the root - and the case would test nothing. The guard itself
			// never joins: an absolute path is accepted only if it is under the
			// root after canonicalisation.
			fullPath := tt.path
			if !filepath.IsAbs(tt.path) {
				fullPath = filepath.Join(allowedRoot, tt.path)
			}
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

	// The resolved path should be the real directory, in its canonical spelling:
	// on macOS the temp dir is itself reached through /var -> /private/var, on
	// Windows through an 8.3 short name, so the expectation is canonicalised too.
	wantDir, err := filepath.EvalSymlinks(realDir)
	testutil.AssertNoError(t, err)
	if safePath.Absolute() != wantDir {
		t.Errorf("expected symlink to resolve to %q, got %q", wantDir, safePath.Absolute())
	}
}

// TestSymlinkEscapeIsBlocked is the fail-closed counterpart of
// TestSymlinkAttackPrevention: a symlink inside the root that points outside it
// must not admit anything below it, whether or not the leaf exists yet.
func TestSymlinkEscapeIsBlocked(t *testing.T) {
	tempDir := testutil.TempDir(t)
	root := filepath.Join(tempDir, "root")
	outside := filepath.Join(tempDir, "outside")
	testutil.AssertNoError(t, os.MkdirAll(root, 0o755))
	testutil.AssertNoError(t, os.MkdirAll(outside, 0o755))
	testutil.AssertNoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o600))

	escape := filepath.Join(root, "escape")
	if err := os.Symlink(outside, escape); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	for _, path := range []string{
		escape,
		filepath.Join(escape, "secret.txt"),
		filepath.Join(escape, "not-yet-created.txt"),
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := values.NewSafePath(path, root); err == nil {
				t.Errorf("expected symlink escape to be blocked: %s", path)
			}
		})
	}
}

// TestSiblingWithSharedPrefixIsOutsideRoot: containment is a path-component
// check, not a string-prefix check. "/tmp2" is not under "/tmp".
func TestSiblingWithSharedPrefixIsOutsideRoot(t *testing.T) {
	tempDir := testutil.TempDir(t)
	root := filepath.Join(tempDir, "ws")
	sibling := root + "2"
	testutil.AssertNoError(t, os.MkdirAll(root, 0o755))
	testutil.AssertNoError(t, os.MkdirAll(sibling, 0o755))

	if _, err := values.NewSafePath(sibling, root); err == nil {
		t.Errorf("expected %q to be outside root %q", sibling, root)
	}
	if _, err := values.NewSafePath(filepath.Join(sibling, "file.txt"), root); err == nil {
		t.Errorf("expected a child of %q to be outside root %q", sibling, root)
	}
}

// TestBackslashIsRejectedOnNonWindows is the regression test for the Ubuntu CI
// failure: on POSIX a backslash is a filename character, so "..\..\etc" joined
// under the root named a file INSIDE the root and was accepted. GLive paths come
// from repositories, URLs and AI output; a backslash on Linux or macOS is never
// legitimate, so the guard fails closed on it.
func TestBackslashIsRejectedOnNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backslash is the native separator on Windows")
	}
	root := testutil.TempDir(t)
	for _, path := range []string{
		root + "/..\\..\\etc",
		root + "/sub\\file.txt",
	} {
		t.Run(path, func(t *testing.T) {
			_, err := values.NewSafePath(path, root)
			if err == nil {
				t.Fatalf("expected backslash path to be rejected: %q", path)
			}
			testutil.AssertContains(t, err.Error(), "PATH_INVALID_CHAR")
		})
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
