package values_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

// symlinkDir creates link -> target and skips the test where the platform (or the
// current user's privileges on Windows) does not allow symbolic links.
func symlinkDir(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not supported here: %v", err)
	}
}

// canonical resolves a path the way the OS will see it (symlinks, and on Windows
// 8.3 short names and on-disk case), so expectations do not depend on how the
// temp directory happened to be spelled.
func canonical(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", path, err)
	}
	return resolved
}

// TestNewSafePath_CanonicalisesBothSides is the regression test for the CI
// failures on macOS and Windows, where the temp directory reaches the guard
// through a symlink (/tmp -> /private/tmp, /var -> /private/var) or an 8.3 short
// name (C:\Users\RUNNER~1). The candidate was resolved to its real path while the
// root was compared as typed, so a valid path was reported as outside its own root.
//
// Both sides of the containment check must be canonicalised identically.
func TestNewSafePath_CanonicalisesBothSides(t *testing.T) {
	tempDir := testutil.TempDir(t)
	realRoot := filepath.Join(tempDir, "real")
	testutil.AssertNoError(t, os.MkdirAll(filepath.Join(realRoot, "existing"), 0o755))
	linkRoot := filepath.Join(tempDir, "link")
	symlinkDir(t, realRoot, linkRoot)

	wantRoot := canonical(t, realRoot)

	tests := []struct {
		name         string
		path         string
		root         string
		wantAbsolute string
	}{
		// The macOS case: NewSafePath("/tmp", "/tmp") with /tmp a symlink.
		{"root itself through a symlink", linkRoot, linkRoot, wantRoot},
		{"existing child through a symlinked root", filepath.Join(linkRoot, "existing"), linkRoot, filepath.Join(wantRoot, "existing")},
		{"new child through a symlinked root", filepath.Join(linkRoot, "new", "file.txt"), linkRoot, filepath.Join(wantRoot, "new", "file.txt")},
		{"symlinked candidate under a plain root", linkRoot, tempDir, wantRoot},
		{"symlinked root, plain candidate", realRoot, linkRoot, wantRoot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := values.NewSafePath(tt.path, tt.root)
			if err != nil {
				t.Fatalf("NewSafePath(%q, %q) rejected a path inside its root: %v", tt.path, tt.root, err)
			}
			if got.Absolute() != tt.wantAbsolute {
				t.Errorf("Absolute() = %q, want canonical %q", got.Absolute(), tt.wantAbsolute)
			}
			rel, err := filepath.Rel(got.Root(), got.Absolute())
			if err != nil || rel == ".." || filepath.IsAbs(rel) {
				t.Errorf("Absolute() %q is not under Root() %q (rel=%q, err=%v)", got.Absolute(), got.Root(), rel, err)
			}
		})
	}
}

// TestNewSafePath_RejectsSymlinkEscapes pins the fail-closed side: a symlink
// INSIDE the root that points OUTSIDE it must not smuggle its children in.
//
// Before the fix only a symlink at the leaf was resolved; a regular file or a
// not-yet-existing name below an escaping symlink kept its lexical path and was
// accepted.
func TestNewSafePath_RejectsSymlinkEscapes(t *testing.T) {
	tempDir := testutil.TempDir(t)
	root := filepath.Join(tempDir, "root")
	outside := filepath.Join(tempDir, "outside")
	testutil.AssertNoError(t, os.MkdirAll(root, 0o755))
	testutil.AssertNoError(t, os.MkdirAll(outside, 0o755))
	testutil.AssertNoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o600))
	escape := filepath.Join(root, "escape")
	symlinkDir(t, outside, escape)

	tests := []struct {
		name string
		path string
	}{
		{"the escaping symlink itself", escape},
		{"existing file below the escaping symlink", filepath.Join(escape, "secret.txt")},
		{"new file below the escaping symlink", filepath.Join(escape, "new.txt")},
		{"new nested path below the escaping symlink", filepath.Join(escape, "a", "b", "c.txt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := values.NewSafePath(tt.path, root)
			if err == nil {
				t.Fatalf("NewSafePath(%q, %q) accepted a symlink escape; Absolute()=%q", tt.path, root, got.Absolute())
			}
			testutil.AssertContains(t, err.Error(), "PATH_TRAVERSAL")
		})
	}
}

// TestNewSafePath_ContainmentIsByComponentNotPrefix: "/tmp2" is not under "/tmp",
// and an absolute path outside the root is outside however it is spelled.
func TestNewSafePath_ContainmentIsByComponentNotPrefix(t *testing.T) {
	tempDir := testutil.TempDir(t)
	root := filepath.Join(tempDir, "root")
	sibling := root + "2"
	testutil.AssertNoError(t, os.MkdirAll(root, 0o755))
	testutil.AssertNoError(t, os.MkdirAll(sibling, 0o755))

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"root itself", root, false},
		{"child of root", filepath.Join(root, "child"), false},
		{"dotdot-prefixed name inside root", filepath.Join(root, "..hidden"), false},
		{"sibling sharing the root's prefix", sibling, true},
		{"child of the prefix sibling", filepath.Join(sibling, "file.txt"), true},
		{"parent of root", tempDir, true},
		{"absolute path outside root", filepath.Join(filepath.VolumeName(root)+string(filepath.Separator), "etc", "passwd"), true},
		{"lexical traversal out of root", filepath.Join(root, "..", "escaped"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := values.NewSafePath(tt.path, root)
			if tt.wantErr && err == nil {
				t.Errorf("NewSafePath(%q, %q) accepted a path outside the root", tt.path, root)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("NewSafePath(%q, %q) rejected a path inside the root: %v", tt.path, root, err)
			}
		})
	}
}
