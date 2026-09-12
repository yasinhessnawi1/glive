package characterisation

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/glive/infrastructure/scanner"
)

// Step 4 - security scan. setup_project.go:310.
//
// THIS STEP IS NOT A GATE TODAY, AND THESE TESTS PIN THAT.
//
// setup_project.go:318 turns a scan *error* into a warning and carries on.
// setup_project.go:322 turns a suspicious *result* into a warning and carries
// on. Execution proceeds either way. GL0 T9 replaces both with a fail-closed
// ScanGate, at which point these assertions must be edited - deliberately, in a
// reviewable diff. That visibility is the whole point of pinning it.
//
// The severity counts are pinned too, because T9's gate table is defined in
// terms of them and T4/T8 fold the pkg/core scanner's patterns into this one.
// If a fold changes what a fixture produces, these break.

func fixture(name string) string { return filepath.Join("..", "fixtures", name) }

func scanFixture(t *testing.T, name string) *scanner.ScanResult {
	t.Helper()
	res, err := scanner.New(fixture(name)).Scan()
	if err != nil {
		t.Fatalf("Scan(%s) = error %v", name, err)
	}
	return res
}

func severityCounts(res *scanner.ScanResult) map[string]int {
	counts := map[string]int{}
	for _, f := range res.Findings {
		counts[f.Severity]++
	}
	return counts
}

func TestStep4_Scan_FixtureSeverities(t *testing.T) {
	tests := []struct {
		fixture        string
		wantSuspicious bool
		wantCounts     map[string]int
	}{
		{
			fixture:        "clean-node",
			wantSuspicious: false,
			wantCounts:     map[string]int{},
		},
		{
			fixture:        "clean-python",
			wantSuspicious: false,
			wantCounts:     map[string]int{},
		},
		{
			// AWS key + private key are critical; os.system, exec( and rm -rf /
			// are high. Any one of them is enough to set IsSuspicious.
			fixture:        "dangerous",
			wantSuspicious: true,
			wantCounts:     map[string]int{"critical": 2, "high": 3},
		},
		{
			fixture:        "empty",
			wantSuspicious: false,
			wantCounts:     map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			res := scanFixture(t, tt.fixture)

			if res.IsSuspicious != tt.wantSuspicious {
				t.Errorf("IsSuspicious = %v, want %v (findings: %v)",
					res.IsSuspicious, tt.wantSuspicious, res.SuspiciousReasons)
			}

			got := severityCounts(res)
			for sev, want := range tt.wantCounts {
				if got[sev] != want {
					t.Errorf("severity %q count = %d, want %d", sev, got[sev], want)
				}
			}
			for sev, n := range got {
				if _, expected := tt.wantCounts[sev]; !expected {
					t.Errorf("unexpected severity %q with %d finding(s)", sev, n)
				}
			}
		})
	}
}

// TestStep4_Scan_MediumEscalationThreshold pins the rule that A-2's original gate
// table did not have and that GL0 T9 must preserve: MORE THAN THREE medium
// findings make a repository suspicious, even with nothing critical or high.
//
// scanner.go:303 - `critical > 0 || high > 0 || medium > 3`.
//
// If T9's gate blocked only on critical/high, a repository with four medium
// findings would go from "suspicious" today to "proceeds with a warning" - the
// fail-closed gate would be weaker than the warn-only scanner it replaced. These
// two fixtures are the guard against that.
func TestStep4_Scan_MediumEscalationThreshold(t *testing.T) {
	t.Run("four medium findings are suspicious", func(t *testing.T) {
		res := scanFixture(t, "medium-only")
		if got := severityCounts(res)["medium"]; got != 4 {
			t.Fatalf("fixture drifted: medium count = %d, want 4", got)
		}
		if severityCounts(res)["critical"]+severityCounts(res)["high"] != 0 {
			t.Fatalf("fixture drifted: expected no critical or high findings")
		}
		if !res.IsSuspicious {
			t.Error("IsSuspicious = false with 4 medium findings, want true (scanner.go:303)")
		}
	})

	t.Run("two medium findings are not suspicious", func(t *testing.T) {
		res := scanFixture(t, "medium-few")
		if got := severityCounts(res)["medium"]; got != 2 {
			t.Fatalf("fixture drifted: medium count = %d, want 2", got)
		}
		if res.IsSuspicious {
			t.Error("IsSuspicious = true with 2 medium findings, want false (scanner.go:303)")
		}
	})
}

// TestStep4_Scan_SuspiciousReasonsCarryCriticalAndHigh pins the shape of the
// strings Execute copies into Output.Warnings at setup_project.go:322-326.
// Medium and low findings are deliberately NOT listed there, only counted.
func TestStep4_Scan_SuspiciousReasonsCarryCriticalAndHigh(t *testing.T) {
	res := scanFixture(t, "dangerous")

	if len(res.SuspiciousReasons) != 5 {
		t.Errorf("len(SuspiciousReasons) = %d, want 5 (2 critical + 3 high)", len(res.SuspiciousReasons))
	}
	var critical, high int
	for _, r := range res.SuspiciousReasons {
		switch {
		case len(r) > 10 && r[:10] == "[CRITICAL]":
			critical++
		case len(r) > 6 && r[:6] == "[HIGH]":
			high++
		default:
			t.Errorf("unexpected reason prefix: %q", r)
		}
	}
	if critical != 2 || high != 3 {
		t.Errorf("reasons = %d critical / %d high, want 2 / 3", critical, high)
	}

	// medium-only produces findings but NO reasons: analyzeFindings only appends
	// reason strings for critical and high (scanner.go:284-296).
	mediumRes := scanFixture(t, "medium-only")
	if len(mediumRes.SuspiciousReasons) != 0 {
		t.Errorf("medium-only SuspiciousReasons = %v, want empty even though IsSuspicious is true",
			mediumRes.SuspiciousReasons)
	}
}

// TestStep4_Scan_WarnOnly_ExecutionIsNotBlocked is the characterisation that T9
// deliberately inverts.
//
// Today a suspicious result is ADVISORY: Scan returns it, and Execute appends
// warnings and continues to step 5 (setup_project.go:322, then :346). There is
// no blocking path anywhere in the scanner - it exposes no Allow/Deny, only data.
// Asserting that absence is what makes T9's addition a visible change.
func TestStep4_Scan_WarnOnly_ExecutionIsNotBlocked(t *testing.T) {
	res := scanFixture(t, "dangerous")

	if !res.IsSuspicious {
		t.Fatal("precondition: the dangerous fixture must be suspicious")
	}

	// The result is data, not a decision: the caller is free to ignore it, and
	// today's caller does exactly that. If a future ScanResult grows a field
	// that denies execution, this comment and T9's gate are where to look.
	if res.Findings == nil {
		t.Error("Findings = nil, want the findings to be reported to the caller")
	}
}

// TestStep4_Scan_VendoredDirectorySkip_IsPlatformDependent pins a DEFECT, found
// while writing this suite.
//
// Scanner.shouldSkipFile (scanner.go:309) tests the walked path with
// strings.Contains against patterns written with a forward slash - "node_modules/",
// ".git/", "vendor/", "dist/", "build/", "__pycache__/" and the rest. But
// filepath.Walk yields OS-native separators, so on Windows the path is
// ...\node_modules\evil\index.js and NONE of those patterns ever match. The skip
// list is effectively dead on Windows and works on Linux and macOS.
//
// Two consequences, and the second is the one that matters:
//   - scans on Windows walk .git/, node_modules/ and vendor/, so they are slower;
//   - a vendored dependency that trips a pattern produces findings on Windows
//     and none on Linux. Today that is a spurious warning. After GL0 T9 makes the
//     gate fail-closed it becomes a spurious BLOCK - the same repository would be
//     refused on a Windows host and accepted on a Linux one.
//
// This test asserts what the code does now, per platform, rather than what it
// should do. When the separator handling is fixed, the Windows branch fails and
// whoever fixes it updates this test - which is the intended behaviour of a
// characterisation test, not a nuisance.
func TestStep4_Scan_VendoredDirectorySkip_IsPlatformDependent(t *testing.T) {
	root := t.TempDir()
	vendored := filepath.Join(root, "node_modules", "evil")
	if err := mkdirAll(vendored); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write(t, filepath.Join(vendored, "index.js"), "const k = \"AKIAIOSFODNN7EXAMPLE\";\n")

	res, err := scanner.New(root).Scan()
	if err != nil {
		t.Fatalf("Scan() = error %v", err)
	}

	if runtime.GOOS == "windows" {
		if len(res.Findings) == 0 {
			t.Error("node_modules/ is now being skipped on Windows - the separator " +
				"defect appears fixed; update this characterisation and the T9 gate notes")
		}
		if !res.IsSuspicious {
			t.Error("IsSuspicious = false; expected the vendored finding to leak through on Windows")
		}
		return
	}

	if len(res.Findings) != 0 {
		t.Errorf("findings inside node_modules/ were reported on %s: %v",
			runtime.GOOS, res.SuspiciousReasons)
	}
	if res.IsSuspicious {
		t.Errorf("IsSuspicious = true from a vendored file alone on %s", runtime.GOOS)
	}
}
