package characterisation

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/glive/infrastructure/git"
)

// Step 3 - clone. setup_project.go:193. git.Client.Clone shells out to
// `git clone --depth 1 <URL> <LocalPath>`, so pointing URL at a repository in a
// temp directory exercises the real code path with no network: git treats a
// local path as a perfectly good remote.
//
// The behaviours pinned here are the ones the T6 collapse could change without
// anyone noticing: re-cloning over an existing clone is a no-op, and colliding
// with a non-repository directory is a typed ConflictError rather than a
// clobber.

// newLocalOriginRepo creates a git repository in a temp dir and returns its path,
// for use as a clone source. Skips the test if git is unavailable.
func newLocalOriginRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed; skipping clone characterisation")
	}

	origin := t.TempDir()
	write(t, filepath.Join(origin, "README.md"), "# origin fixture\n")
	write(t, filepath.Join(origin, "package.json"), `{"name":"origin-fixture","version":"1.0.0"}`+"\n")

	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "fixture@example.invalid"},
		{"config", "user.name", "Fixture"},
		{"add", "."},
		{"commit", "-m", "initial"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = origin
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in origin: %v\n%s", args, err, out)
		}
	}
	return origin
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestStep3_Clone_FromLocalOrigin(t *testing.T) {
	origin := newLocalOriginRepo(t)
	workspace := t.TempDir()

	// Fields are exported, so the client can be built directly with a local
	// origin instead of going through ParseGitHubURL (which requires a known
	// provider host). This is the same struct Execute clones with.
	client := &git.Client{URL: origin, Owner: "fixture", Name: "origin-fixture"}

	if err := client.Clone(workspace, nil); err != nil {
		t.Fatalf("Clone() = error %v, want success", err)
	}

	wantPath := filepath.Join(workspace, "origin-fixture")
	if got := client.GetLocalPath(); got != wantPath {
		t.Errorf("GetLocalPath() = %q, want %q", got, wantPath)
	}
	if !client.IsCloned() {
		t.Error("IsCloned() = false after a successful clone")
	}
	for _, f := range []string{"README.md", "package.json", ".git"} {
		if _, err := os.Stat(filepath.Join(wantPath, f)); err != nil {
			t.Errorf("expected %s in the clone: %v", f, err)
		}
	}
}

// TestStep3_Clone_ExistingCloneIsANoOp pins the idempotency Execute relies on:
// a second Clone over an existing git repository succeeds and leaves it alone.
func TestStep3_Clone_ExistingCloneIsANoOp(t *testing.T) {
	origin := newLocalOriginRepo(t)
	workspace := t.TempDir()
	client := &git.Client{URL: origin, Owner: "fixture", Name: "origin-fixture"}

	if err := client.Clone(workspace, nil); err != nil {
		t.Fatalf("first Clone() = error %v", err)
	}

	// A marker file proves the second call reuses the directory rather than
	// re-cloning over it.
	marker := filepath.Join(client.GetLocalPath(), "marker.txt")
	write(t, marker, "kept")

	if err := client.Clone(workspace, nil); err != nil {
		t.Fatalf("second Clone() = error %v, want a no-op success", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("second Clone() destroyed the existing clone: %v", err)
	}
}

// TestStep3_Clone_ConflictOnNonRepoDirectory pins that colliding with a
// non-repository directory yields a typed *git.ConflictError and does NOT touch
// the directory. Execute routes this to its conflict callback rather than
// failing outright, so the type matters, not just the error.
func TestStep3_Clone_ConflictOnNonRepoDirectory(t *testing.T) {
	origin := newLocalOriginRepo(t)
	workspace := t.TempDir()

	// Pre-create the destination with content but no .git.
	dest := filepath.Join(workspace, "origin-fixture")
	if err := os.MkdirAll(dest, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	occupied := filepath.Join(dest, "not-a-repo.txt")
	write(t, occupied, "pre-existing")

	client := &git.Client{URL: origin, Owner: "fixture", Name: "origin-fixture"}
	err := client.Clone(workspace, nil)

	if err == nil {
		t.Fatal("Clone() over a non-repository directory succeeded, want a conflict")
	}

	var conflict *git.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("Clone() = %T (%v), want *git.ConflictError", err, err)
	}
	if conflict.IsGitRepo {
		t.Error("ConflictError.IsGitRepo = true, want false for a plain directory")
	}
	if conflict.Path != dest {
		t.Errorf("ConflictError.Path = %q, want %q", conflict.Path, dest)
	}
	if _, statErr := os.Stat(occupied); statErr != nil {
		t.Errorf("the conflicting directory was modified: %v", statErr)
	}
}

// mkdirAll is a small helper shared by the step tests.
func mkdirAll(path string) error { return os.MkdirAll(path, 0o750) }
