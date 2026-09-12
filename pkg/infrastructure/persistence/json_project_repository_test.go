package persistence_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glive/infrastructure/persistence"
)

// TestNewJSONProjectRepository_StateDirThroughSymlink is the end-to-end form of
// the CI failure that took down every pkg/usecase/project characterisation test
// on macOS and Windows: the workspace lives under a symlinked (or 8.3 short-named)
// temp directory and the not-yet-existing state directory below it must be
// accepted both before and after it is created.
func TestNewJSONProjectRepository_StateDirThroughSymlink(t *testing.T) {
	tempDir := t.TempDir()
	realDir := filepath.Join(tempDir, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(tempDir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlinks not supported here: %v", err)
	}

	stateDir := filepath.Join(linkDir, ".glive")
	if _, err := persistence.NewJSONProjectRepository(stateDir); err != nil {
		t.Fatalf("NewJSONProjectRepository(%q): %v", stateDir, err)
	}
	if _, err := os.Stat(filepath.Join(realDir, ".glive")); err != nil {
		t.Fatalf("state directory was not created under the real root: %v", err)
	}
}
