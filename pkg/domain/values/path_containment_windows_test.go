package values_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

// shortName returns the 8.3 form of an existing path (C:\Users\RUNNER~1\...),
// which is how GitHub's Windows runners spell TEMP. It skips when the volume has
// 8.3 name generation disabled, because then no short form exists to test with.
func shortName(t *testing.T, long string) string {
	t.Helper()
	in, err := syscall.UTF16PtrFromString(long)
	if err != nil {
		t.Fatalf("UTF16PtrFromString(%q): %v", long, err)
	}
	buf := make([]uint16, syscall.MAX_LONG_PATH)
	n, err := syscall.GetShortPathName(in, &buf[0], uint32(len(buf)))
	if err != nil {
		t.Fatalf("GetShortPathName(%q): %v", long, err)
	}
	short := syscall.UTF16ToString(buf[:n])
	if strings.EqualFold(short, long) {
		t.Skip("8.3 short names are disabled on this volume")
	}
	return short
}

// TestNewSafePath_Windows83ShortNames is the regression test for the Windows CI
// failure: t.TempDir() is C:\Users\RUNNER~1\AppData\Local\Temp\..., the candidate
// was resolved to C:\Users\runneradmin\..., the root was not, and the two never
// matched.
func TestNewSafePath_Windows83ShortNames(t *testing.T) {
	tempDir := testutil.TempDir(t)
	longRoot := filepath.Join(tempDir, "averylongdirectoryname")
	testutil.AssertNoError(t, os.MkdirAll(filepath.Join(longRoot, "existing"), 0o755))
	shortRoot := shortName(t, longRoot)
	wantRoot, err := filepath.EvalSymlinks(longRoot)
	testutil.AssertNoError(t, err)

	tests := []struct {
		name         string
		path         string
		root         string
		wantAbsolute string
	}{
		{"short root, short candidate", shortRoot, shortRoot, wantRoot},
		{"short root, long candidate", longRoot, shortRoot, wantRoot},
		{"long root, short candidate", shortRoot, longRoot, wantRoot},
		{"short root, existing child", filepath.Join(shortRoot, "existing"), shortRoot, filepath.Join(wantRoot, "existing")},
		{"short root, new child", filepath.Join(shortRoot, "new", "file.txt"), shortRoot, filepath.Join(wantRoot, "new", "file.txt")},
		{"upper-cased root, lower-cased candidate", strings.ToLower(longRoot), strings.ToUpper(longRoot), wantRoot},
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
		})
	}
}
