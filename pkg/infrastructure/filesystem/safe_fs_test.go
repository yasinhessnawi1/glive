package filesystem_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glive/infrastructure/filesystem"
)

func symlinkDir(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not supported here: %v", err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

// TestSafeFileSystem_SymlinkedRootAcceptsNewAndExistingPaths reproduces the CI
// failure in every pkg/usecase/project characterisation test:
//
//	NewContainer: failed to create project repository: failed to load projects:
//	path traversal detected: <tmp>/.glive is not under allowed roots
//
// The root was canonicalised (symlinks, 8.3 short names resolved) but a candidate
// that did not exist yet was not, so the state directory passed validation before
// MkdirAll created it and failed validation immediately afterwards.
func TestSafeFileSystem_SymlinkedRootAcceptsNewAndExistingPaths(t *testing.T) {
	tempDir := t.TempDir()
	realDir := filepath.Join(tempDir, "real")
	mustMkdir(t, realDir)
	linkDir := filepath.Join(tempDir, "link")
	symlinkDir(t, realDir, linkDir)

	stateDir := filepath.Join(linkDir, ".glive")
	fs, err := filesystem.NewSafeFileSystem(stateDir)
	if err != nil {
		t.Fatalf("NewSafeFileSystem(%q): %v", stateDir, err)
	}

	// Does not exist yet: must be resolved through its nearest existing ancestor.
	if err := fs.ValidatePath(stateDir); err != nil {
		t.Fatalf("ValidatePath(%q) before creation: %v", stateDir, err)
	}
	if err := fs.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", stateDir, err)
	}
	// Exists now: must still be inside the same root.
	if err := fs.ValidatePath(stateDir); err != nil {
		t.Fatalf("ValidatePath(%q) after creation: %v", stateDir, err)
	}
	if _, err := fs.ReadDir(stateDir); err != nil {
		t.Fatalf("ReadDir(%q): %v", stateDir, err)
	}
	newFile := filepath.Join(stateDir, "projects", "p1.json")
	if err := fs.ValidatePath(newFile); err != nil {
		t.Fatalf("ValidatePath(%q) for a new nested file: %v", newFile, err)
	}
}

// TestSafeFileSystem_RejectsSymlinkEscapes pins fail-closed behaviour: a symlink
// inside the root pointing outside must not admit its (existing or not-yet
// existing) children. Before the fix a non-existent leaf was never resolved.
func TestSafeFileSystem_RejectsSymlinkEscapes(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "root")
	outside := filepath.Join(tempDir, "outside")
	mustMkdir(t, root)
	mustMkdir(t, outside)
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	escape := filepath.Join(root, "escape")
	symlinkDir(t, outside, escape)

	fs, err := filesystem.NewSafeFileSystem(root)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{"escaping symlink itself", escape},
		{"existing file below it", filepath.Join(escape, "secret.txt")},
		{"new file below it", filepath.Join(escape, "new.txt")},
		{"new nested path below it", filepath.Join(escape, "a", "b", "c.txt")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.ValidatePath(tt.path)
			if err == nil {
				t.Fatalf("ValidatePath(%q) accepted a symlink escape from %q", tt.path, root)
			}
			if !errors.Is(err, filesystem.ErrPathTraversal) {
				t.Errorf("want ErrPathTraversal, got %v", err)
			}
		})
	}
}

// TestSafeFileSystem_ContainmentIsByComponentNotPrefix: a sibling that shares the
// root's name as a prefix is outside; a name that merely starts with ".." is inside.
func TestSafeFileSystem_ContainmentIsByComponentNotPrefix(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "root")
	sibling := root + "2"
	mustMkdir(t, root)
	mustMkdir(t, sibling)

	fs, err := filesystem.NewSafeFileSystem(root)
	if err != nil {
		t.Fatal(err)
	}

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
		{"lexical traversal out of root", filepath.Join(root, "..", "escaped"), true},
		{"absolute path outside root", filepath.Join(filepath.VolumeName(root)+string(filepath.Separator), "etc", "passwd"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.ValidatePath(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("ValidatePath(%q) accepted a path outside %q", tt.path, root)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePath(%q) rejected a path inside %q: %v", tt.path, root, err)
			}
		})
	}
}
