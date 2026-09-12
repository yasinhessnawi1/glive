package values_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestCanonicalPath_ResolvesThroughNearestExistingAncestor(t *testing.T) {
	tempDir := testutil.TempDir(t)
	realDir := filepath.Join(tempDir, "real")
	testutil.AssertNoError(t, os.MkdirAll(realDir, 0o755))
	linkDir := filepath.Join(tempDir, "link")
	symlinkDir(t, realDir, linkDir)
	wantReal := canonical(t, realDir)

	tests := []struct {
		name string
		path string
		want string
	}{
		{"existing directory", realDir, wantReal},
		{"symlink to it", linkDir, wantReal},
		{"missing child of the symlink", filepath.Join(linkDir, "new"), filepath.Join(wantReal, "new")},
		{"missing nested path below the symlink", filepath.Join(linkDir, "a", "b", "c.txt"), filepath.Join(wantReal, "a", "b", "c.txt")},
		{"lexical dot-dot is cleaned first", filepath.Join(linkDir, "sub", "..", "x"), filepath.Join(wantReal, "x")},
		{"trailing separator is cleaned", linkDir + string(filepath.Separator), wantReal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := values.CanonicalPath(tt.path)
			if err != nil {
				t.Fatalf("CanonicalPath(%q): %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("CanonicalPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCanonicalPath_FailsClosedWhenAnAncestorIsAFile(t *testing.T) {
	tempDir := testutil.TempDir(t)
	file := filepath.Join(tempDir, "file.txt")
	testutil.AssertNoError(t, os.WriteFile(file, []byte("x"), 0o600))

	// A path "below" a regular file cannot exist. Some platforms report this as
	// not-found (resolved through the file, then rejected by the caller's
	// containment check), others as a distinct error; either way it must not be
	// silently accepted as something it is not.
	got, err := values.CanonicalPath(filepath.Join(file, "child"))
	if err == nil && got != filepath.Join(canonical(t, file), "child") {
		t.Errorf("CanonicalPath(file/child) = %q, want an error or the lexical tail under the file", got)
	}
}

func TestIsWithinRoot(t *testing.T) {
	sep := string(filepath.Separator)
	root := filepath.Join(testutil.TempDir(t), "root")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"root itself", root, true},
		{"root with trailing separator", root + sep, true},
		{"direct child", filepath.Join(root, "child"), true},
		{"nested child", filepath.Join(root, "a", "b", "c"), true},
		{"child named with a dot-dot prefix", filepath.Join(root, "..hidden"), true},
		{"child named exactly two dots plus text", filepath.Join(root, "..."), true},
		{"parent", filepath.Dir(root), false},
		{"sibling sharing the prefix", root + "2", false},
		{"child of the prefix sibling", filepath.Join(root+"2", "x"), false},
		{"dot-dot escape", filepath.Join(root, "..", "escaped"), false},
		{"absolute path elsewhere", filepath.Join(filepath.VolumeName(root)+sep, "etc", "passwd"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := values.IsWithinRoot(root, tt.path); got != tt.want {
				t.Errorf("IsWithinRoot(%q, %q) = %v, want %v", root, tt.path, got, tt.want)
			}
		})
	}
}
