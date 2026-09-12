package characterisation

import (
	"testing"

	"github.com/glive/infrastructure/analyzer"
)

// Step 5 - analyse. setup_project.go:346 constructs analyzer.New(localPath) and
// calls Analyze(), then converts the result to a domain Analysis at
// setup_project.go:733 (convertAnalysisToDomain).
//
// What is pinned here is the detection contract the rest of the flow depends on:
// the project type chosen, the package managers found, and - most importantly -
// the fact that Analyze SUCCEEDS on a directory it cannot classify. Step 6 plans
// commands from this result, so a change from "unknown, no error" to "error"
// would reroute the whole flow.

func analyseFixture(t *testing.T, name string) *analyzer.AnalysisResult {
	t.Helper()
	res, err := analyzer.New(fixture(name)).Analyze()
	if err != nil {
		t.Fatalf("Analyze(%s) = error %v", name, err)
	}
	if res == nil {
		t.Fatalf("Analyze(%s) returned a nil result with no error", name)
	}
	return res
}

func TestStep5_Analyse_ProjectTypeDetection(t *testing.T) {
	tests := []struct {
		fixture            string
		wantType           analyzer.ProjectType
		wantPackageManager string // "" means: expect none
	}{
		{fixture: "clean-node", wantType: "nodejs", wantPackageManager: "npm"},
		{fixture: "clean-python", wantType: "python", wantPackageManager: "pip"},
		// The dangerous fixture carries a setup.py, so it classifies as python.
		// Analysis is independent of the scan: a suspicious repository is still
		// analysed, which is what lets step 6 run on it today.
		{fixture: "dangerous", wantType: "python", wantPackageManager: "pip"},
		// An empty directory is classified rather than rejected.
		{fixture: "empty", wantType: "unknown", wantPackageManager: ""},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			res := analyseFixture(t, tt.fixture)

			if res.ProjectType != tt.wantType {
				t.Errorf("ProjectType = %q, want %q", res.ProjectType, tt.wantType)
			}

			if tt.wantPackageManager == "" {
				if len(res.PackageManagers) != 0 {
					t.Errorf("PackageManagers = %v, want none", res.PackageManagers)
				}
				return
			}

			var found bool
			for _, pm := range res.PackageManagers {
				if pm == tt.wantPackageManager {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("PackageManagers = %v, want it to contain %q", res.PackageManagers, tt.wantPackageManager)
			}
		})
	}
}

// TestStep5_Analyse_UnknownProjectIsNotAnError pins the behaviour step 6 depends
// on: an unclassifiable directory yields ProjectType "unknown" and a nil error,
// not a failure. Execute has no branch for "analysis failed" that skips
// execution, so turning this into an error would change the flow's shape.
func TestStep5_Analyse_UnknownProjectIsNotAnError(t *testing.T) {
	res, err := analyzer.New(fixture("empty")).Analyze()

	if err != nil {
		t.Fatalf("Analyze(empty) = error %v, want a successful 'unknown' result", err)
	}
	if res.ProjectType != "unknown" {
		t.Errorf("ProjectType = %q, want %q", res.ProjectType, "unknown")
	}
}

// TestStep5_Analyse_MissingDirectory pins what happens when the path does not
// exist. Execute only reaches step 5 after a successful clone, so this is the
// defensive edge rather than a normal path - but the collapse must not change it
// silently.
func TestStep5_Analyse_MissingDirectory(t *testing.T) {
	res, err := analyzer.New(fixture("does-not-exist")).Analyze()

	// Pin whichever contract holds today, and make the alternative visible.
	if err != nil {
		if res != nil {
			t.Errorf("Analyze() returned both an error (%v) and a non-nil result", err)
		}
		return
	}
	if res == nil {
		t.Fatal("Analyze() returned nil result and nil error")
	}
	if res.ProjectType != "unknown" {
		t.Errorf("Analyze() on a missing directory = type %q with no error, want %q",
			res.ProjectType, "unknown")
	}
}

// TestStep5_Analyse_IsIndependentOfTheScanner pins that Analyze does not consult
// the scanner: the AnalysisResult carries IsSuspicious/SuspiciousReasons fields,
// but analysing a dangerous repository leaves them unset. Execute merges the scan
// result in separately. If analysis ever started populating them, the warn-only
// behaviour pinned in step 4 could change shape without step 4's tests noticing.
func TestStep5_Analyse_IsIndependentOfTheScanner(t *testing.T) {
	res := analyseFixture(t, "dangerous")

	if res.IsSuspicious {
		t.Error("AnalysisResult.IsSuspicious = true; analysis is not supposed to run the scanner")
	}
	if len(res.SuspiciousReasons) != 0 {
		t.Errorf("AnalysisResult.SuspiciousReasons = %v, want empty", res.SuspiciousReasons)
	}
}
