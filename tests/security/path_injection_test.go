package security

import (
	"strings"
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

// TestNewPath_RejectsTrimmableControlCharacters is the regression test for the
// first of the two security gaps found at GL0 T0a (AUDIT-F-18).
//
// NewPath called strings.TrimSpace before checking for control characters. TrimSpace
// treats \n, \r, \v and \f as whitespace, so a path carrying any of them at either
// end had them silently deleted and the path was ACCEPTED — the post-trim check
// could never see the character that had already been removed.
//
// Two things made that dangerous rather than merely untidy: a path that should have
// been rejected was accepted, and the caller received back a value different from
// the one it passed. A silently-normalised path is how argument smuggling and log
// injection begin, and Path feeds the executor's working directory.
//
// These cases fail against the pre-fix validator.
func TestNewPath_RejectsTrimmableControlCharacters(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"trailing newline", "/tmp/project\n"},
		{"leading newline", "\n/tmp/project"},
		{"trailing carriage return", "/tmp/project\r"},
		{"crlf", "/tmp/project\r\n"},
		{"embedded newline", "/tmp/pro\nject"},
		{"vertical tab", "/tmp/project\v"},
		{"form feed", "/tmp/project\f"},
		{"newline with injected second path", "/tmp/project\n/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := values.NewPath(tt.path)
			if err == nil {
				t.Fatalf("NewPath(%q) accepted a path containing a control character; got value %q",
					tt.path, got.Value())
			}
			testutil.AssertContains(t, err.Error(), "PATH_CONTROL_CHAR")
		})
	}
}

// TestNewSafePath_RejectsTrimmableControlCharacters covers the same defect on the
// root-enforcing constructor, which is the one the executor actually uses.
func TestNewSafePath_RejectsTrimmableControlCharacters(t *testing.T) {
	root := testutil.TempDir(t)

	tests := []struct {
		name string
		path string
	}{
		{"trailing newline", root + "/sub\n"},
		{"leading newline", "\n" + root + "/sub"},
		{"embedded newline", root + "/su\nb"},
		{"carriage return", root + "/sub\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := values.NewSafePath(tt.path, root); err == nil {
				t.Fatalf("NewSafePath(%q) accepted a path containing a control character", tt.path)
			} else {
				testutil.AssertContains(t, err.Error(), "PATH_CONTROL_CHAR")
			}
		})
	}
}

// TestNewPath_DoesNotSilentlyRewriteItsInput states the underlying invariant
// directly: if NewPath returns a Path, its value differs from the caller's input
// only by surrounding ordinary spaces — never by a deleted control character.
func TestNewPath_DoesNotSilentlyRewriteItsInput(t *testing.T) {
	inputs := []string{
		"/tmp/project",
		"  /tmp/project  ",
		"relative/path",
		"/tmp/project\n",
		"/tmp/pro\nject",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			got, err := values.NewPath(in)
			if err != nil {
				return // rejected: nothing was rewritten
			}
			if want := strings.Trim(in, " \t"); got.Value() != want {
				t.Errorf("NewPath(%q) silently rewrote its input to %q (want %q)", in, got.Value(), want)
			}
		})
	}
}

// TestNewSafePath_RejectsEmptyAndWhitespaceOnlyPaths pins the second finding
// reported under AUDIT-F-18.
//
// The audit recorded that "an empty safe path is accepted". It is not — NewSafePath
// has always rejected an empty path. What was broken was the unit test: its harness
// called filepath.Join(root, "") to build the input, and that returns root itself, a
// valid non-empty directory. So the empty-path case never passed an empty path and
// could not have caught a regression. These cases call the constructor directly.
func TestNewSafePath_RejectsEmptyAndWhitespaceOnlyPaths(t *testing.T) {
	root := testutil.TempDir(t)

	tests := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"single space", " "},
		{"spaces", "     "},
		{"tabs", "\t\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := values.NewSafePath(tt.path, root); err == nil {
				t.Fatalf("NewSafePath(%q, root) accepted an empty path", tt.path)
			} else {
				testutil.AssertContains(t, err.Error(), "PATH_EMPTY")
			}
		})
	}
}
