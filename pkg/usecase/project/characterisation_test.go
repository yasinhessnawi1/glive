package project_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glive/domain/entities"
	"github.com/glive/infrastructure/analyzer"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/infrastructure/scanner"
	"github.com/glive/usecase/project"
)

// Characterisation of SetupProjectUseCase.Execute - GL0 T1/T1b.
//
// These pin the CURRENT behaviour of the seven-step flow so the T6 collapse (the
// agent routed through this use case) cannot change it by accident. Where the
// behaviour is wrong, the test asserts the wrong behaviour and says so; the
// clearest case is the security scan, which warns and proceeds. T9 inverts that,
// and these assertions are what make the inversion a visible diff.
//
// Step lines in pkg/usecase/project/setup_project.go:
//
//	Step 1 parse            :105
//	Step 2 AI analysis      :173 (deferred) / :289 (kicked off after clone)
//	Step 3 clone            :193
//	Step 4 security scan    :310  (warn-only at :318 and :322)
//	Step 5 analyse          :346  (AI result merged at :370)
//	Step 6 execute+auto-fix :414
//	Step 7 ready            :501
//
// This file replaces the previous setup_test.go, which constructed an empty
// container, never called Execute and ended with `_ = uc` - it asserted nothing.

func setupInput(mode executor.ExecutionMode) project.SetupProjectInput {
	return project.SetupProjectInput{
		GitHubURL: "https://github.com/octocat/hello-world",
		Mode:      mode,
	}
}

func runExecute(t *testing.T, h *harness, mode executor.ExecutionMode) (*project.SetupProjectOutput, error) {
	t.Helper()
	uc := project.NewSetupProjectUseCase(h.repo, h.container, discard{})
	in := setupInput(mode)
	in.WorkspaceDir = h.localPath
	return uc.Execute(context.Background(), in)
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

// TestExecute_StepOrdering is the headline characterisation: the seven steps run
// in this order, and the deferred AI analysis (step 2) is kicked off AFTER the
// clone and its result is merged AFTER the basic analysis.
//
// setup_project.go:171 decides whether AI runs at all; :289 starts the goroutine
// once localPath exists; :370 blocks on the channel after Analyze() returned.
// That "announced early, started after clone, merged after analyse" shape is
// easy to lose in a refactor and impossible to see from the step numbering alone.
func TestExecute_StepOrdering(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey:     "test-key",
		aiResponse: `{"description":"from ai"}`,
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands:    []analyzer.Command{cmd("c1", "npm install")},
		},
	})

	out, err := runExecute(t, h, executor.ModeAuto)
	if err != nil {
		t.Fatalf("Execute() = error %v, want success", err)
	}
	if out == nil {
		t.Fatal("Execute() returned a nil output with no error")
	}

	events := h.rec.snapshot()
	t.Logf("trace: %v", events)

	clone := h.rec.indexOf("clone")
	aiStart := h.rec.indexOf("ai.AnalyzeProject")
	scan := h.rec.indexOf("scan")
	analyse := h.rec.indexOf("analyze")
	exec := h.rec.indexOf("execute:npm install")

	for name, idx := range map[string]int{
		"clone": clone, "ai.AnalyzeProject": aiStart, "scan": scan,
		"analyze": analyse, "execute": exec,
	} {
		if idx < 0 {
			t.Fatalf("%s never happened; trace = %v", name, events)
		}
	}

	if clone >= scan {
		t.Errorf("clone (%d) must precede scan (%d) - setup_project.go:193 then :310", clone, scan)
	}
	if scan >= analyse {
		t.Errorf("scan (%d) must precede analyse (%d) - setup_project.go:310 then :346", scan, analyse)
	}
	if analyse >= exec {
		t.Errorf("analyse (%d) must precede execute (%d) - setup_project.go:346 then :414", analyse, exec)
	}
	// Step 2 is announced before the clone but only STARTS after it: the
	// goroutine at :289 needs localPath, which does not exist until :193 ran.
	if clone >= aiStart {
		t.Errorf("AI analysis (%d) must start after the clone (%d) - setup_project.go:289", aiStart, clone)
	}
	// ...and its result is consumed after the basic analysis, at :370.
	if aiStart >= exec {
		t.Errorf("AI analysis (%d) must be merged before execution (%d)", aiStart, exec)
	}
}

// TestExecute_Step2_SkippedWithoutAPIKey pins the other half of the step-2
// branch at setup_project.go:171: with no AI client the flow proceeds and never
// calls the provider.
func TestExecute_Step2_SkippedWithoutAPIKey(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey: "", // no AI configured
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands:    []analyzer.Command{cmd("c1", "npm install")},
		},
	})

	if _, err := runExecute(t, h, executor.ModeAuto); err != nil {
		t.Fatalf("Execute() = error %v, want success without AI", err)
	}

	if idx := h.rec.indexOf("ai.AnalyzeProject"); idx >= 0 {
		t.Errorf("AI was called with no API key configured; trace = %v", h.rec.snapshot())
	}
	if got := h.rec.indexOf("execute:npm install"); got < 0 {
		t.Error("execution did not happen; the flow must continue without AI")
	}
}

// TestExecute_Step2_AIFailureIsNonFatal pins that a failing AI analysis degrades
// to the basic analysis rather than failing the run (setup_project.go:376).
func TestExecute_Step2_AIFailureIsNonFatal(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey: "test-key",
		aiErr:  errors.New("provider unavailable"),
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands:    []analyzer.Command{cmd("c1", "npm install")},
		},
	})

	out, err := runExecute(t, h, executor.ModeAuto)
	if err != nil {
		t.Fatalf("Execute() = error %v; a failed AI analysis must not fail the run", err)
	}
	if out.Project.Status() != entities.StatusReady {
		t.Errorf("Status = %v, want Ready despite the AI failure", out.Project.Status())
	}
}

// TestExecute_Step4_SuspiciousScanWarnsAndProceeds is the characterisation T9
// deliberately inverts.
//
// setup_project.go:322 appends warnings for a suspicious result and carries on to
// step 5. Nothing blocks. When T9's fail-closed ScanGate lands, this test must be
// edited - which is the point.
func TestExecute_Step4_SuspiciousScanWarnsAndProceeds(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey: "",
		scanResult: &scanner.ScanResult{
			IsSuspicious:      true,
			SuspiciousReasons: []string{"[CRITICAL] AWS Access Key ID detected at x.py:1"},
			Findings:          []scanner.Finding{{Severity: "critical", File: "x.py", Line: 1}},
		},
		analysis: &analyzer.AnalysisResult{
			ProjectType: "python",
			Commands:    []analyzer.Command{cmd("c1", "pip install -r requirements.txt")},
		},
	})

	out, err := runExecute(t, h, executor.ModeAuto)
	if err != nil {
		t.Fatalf("Execute() = error %v; today a suspicious scan does NOT fail the run", err)
	}

	// It executed anyway. This is the behaviour T9 changes.
	if len(h.exec.ran()) == 0 {
		t.Error("no command ran; today a suspicious repository is still executed (setup_project.go:322)")
	}
	if out.Project.Status() != entities.StatusReady {
		t.Errorf("Status = %v, want Ready - the scan is advisory today", out.Project.Status())
	}

	var found bool
	for _, w := range out.Warnings {
		if w == "Potentially suspicious code detected" {
			found = true
		}
	}
	if !found {
		t.Errorf("Warnings = %v, want the suspicious-code warning (setup_project.go:322)", out.Warnings)
	}
}

// TestExecute_Step4_ScanErrorIsSwallowed pins setup_project.go:318: a scan that
// FAILS becomes a warning and the run proceeds. Under T9 an unparseable or failed
// scan must block instead.
func TestExecute_Step4_ScanErrorIsSwallowed(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey:  "",
		scanErr: errors.New("scanner exploded"),
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands:    []analyzer.Command{cmd("c1", "npm install")},
		},
	})

	out, err := runExecute(t, h, executor.ModeAuto)
	if err != nil {
		t.Fatalf("Execute() = error %v; today a scan ERROR does not fail the run", err)
	}
	if len(h.exec.ran()) == 0 {
		t.Error("no command ran; today execution proceeds after a scan error")
	}

	var found bool
	for _, w := range out.Warnings {
		if len(w) >= 20 && w[:20] == "Security scan failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("Warnings = %v, want a 'Security scan failed' warning (setup_project.go:318)", out.Warnings)
	}
}

// TestExecute_Step6_RunsPlannedCommandsInOrder pins that the commands the
// analysis planned are the commands that run, in order (setup_project.go:422).
func TestExecute_Step6_RunsPlannedCommandsInOrder(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey: "",
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands: []analyzer.Command{
				cmd("c1", "npm install"),
				cmd("c2", "npm run build"),
			},
		},
	})

	if _, err := runExecute(t, h, executor.ModeAuto); err != nil {
		t.Fatalf("Execute() = error %v", err)
	}

	got := h.exec.ran()
	want := []string{"npm install", "npm run build"}
	if len(got) != len(want) {
		t.Fatalf("ran %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("command %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestExecute_Step6_NoCommandsIsNotAnError pins setup_project.go:495: an analysis
// that plans nothing still reaches step 7.
func TestExecute_Step6_NoCommandsIsNotAnError(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey:   "",
		analysis: &analyzer.AnalysisResult{ProjectType: "unknown"},
	})

	out, err := runExecute(t, h, executor.ModeAuto)
	if err != nil {
		t.Fatalf("Execute() = error %v, want success with no commands", err)
	}
	if len(h.exec.ran()) != 0 {
		t.Errorf("ran %v, want nothing", h.exec.ran())
	}
	if out.Project.Status() != entities.StatusReady {
		t.Errorf("Status = %v, want Ready", out.Project.Status())
	}
}

// TestExecute_Step7_ReachesReadyAndPersists pins the terminal state and the
// status trail Execute drives through the repository (:501 plus the Save calls
// at each step).
func TestExecute_Step7_ReachesReadyAndPersists(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey: "",
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands:    []analyzer.Command{cmd("c1", "npm install")},
		},
	})

	out, err := runExecute(t, h, executor.ModeAuto)
	if err != nil {
		t.Fatalf("Execute() = error %v", err)
	}

	if out.Project.Status() != entities.StatusReady {
		t.Errorf("final Status = %v, want Ready", out.Project.Status())
	}

	trail := h.repo.statusTrail()
	if len(trail) == 0 {
		t.Fatal("no project was persisted")
	}
	if trail[len(trail)-1] != entities.StatusReady {
		t.Errorf("last persisted status = %v, want Ready", trail[len(trail)-1])
	}

	// The flow moves Pending -> Cloning -> Analyzing -> Installing -> Ready, and
	// persists at each transition. Assert the ones that must appear, in order,
	// rather than the exact number of Saves.
	want := []entities.ProjectStatus{
		entities.StatusPending, entities.StatusCloning,
		entities.StatusAnalyzing, entities.StatusInstalling, entities.StatusReady,
	}
	var i int
	for _, s := range trail {
		if i < len(want) && s == want[i] {
			i++
		}
	}
	if i != len(want) {
		t.Errorf("status trail %v does not contain %v in order (reached %d of %d)",
			trail, want, i, len(want))
	}
}

// TestExecute_Step3_CloneFailureStopsTheRun pins that a failed clone aborts
// rather than proceeding to scan or analyse.
func TestExecute_Step3_CloneFailureStopsTheRun(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey:   "",
		cloneErr: errors.New("clone refused"),
	})

	if _, err := runExecute(t, h, executor.ModeAuto); err == nil {
		t.Fatal("Execute() succeeded despite a failed clone")
	}
	if idx := h.rec.indexOf("scan"); idx >= 0 {
		t.Errorf("scan ran after a failed clone; trace = %v", h.rec.snapshot())
	}
	if idx := h.rec.indexOf("analyze"); idx >= 0 {
		t.Errorf("analyse ran after a failed clone; trace = %v", h.rec.snapshot())
	}
}

// TestExecute_Step5_AnalyseFailureStopsTheRun pins setup_project.go:356.
func TestExecute_Step5_AnalyseFailureStopsTheRun(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey:     "",
		analyzeErr: errors.New("analyzer exploded"),
	})

	if _, err := runExecute(t, h, executor.ModeAuto); err == nil {
		t.Fatal("Execute() succeeded despite a failed analysis")
	}
	if len(h.exec.ran()) != 0 {
		t.Errorf("commands ran after a failed analysis: %v", h.exec.ran())
	}
}

// TestExecute_ContextCancellation pins the cancellation checks Execute performs
// between steps (setup_project.go:98, :184, :302, :405).
func TestExecute_ContextCancellation(t *testing.T) {
	h := newHarness(t, harnessOptions{apiKey: ""})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the first step

	uc := project.NewSetupProjectUseCase(h.repo, h.container, discard{})
	in := setupInput(executor.ModeAuto)
	in.WorkspaceDir = h.localPath

	_, err := uc.Execute(ctx, in)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() = %v, want context.Canceled", err)
	}
	if idx := h.rec.indexOf("clone"); idx >= 0 {
		t.Error("clone ran despite a cancelled context")
	}
}

// TestExecute_Step1_InvalidURLFailsBeforeAnything pins that a URL the domain
// rejects stops the run at step 1 (setup_project.go:109) - nothing is cloned.
func TestExecute_Step1_InvalidURLFailsBeforeAnything(t *testing.T) {
	for _, bad := range []string{"", "not-a-url", "file:///etc/passwd", "http://localhost/repo"} {
		t.Run(bad, func(t *testing.T) {
			h := newHarness(t, harnessOptions{apiKey: ""})
			uc := project.NewSetupProjectUseCase(h.repo, h.container, discard{})
			in := setupInput(executor.ModeAuto)
			in.GitHubURL = bad
			in.WorkspaceDir = h.localPath

			if _, err := uc.Execute(context.Background(), in); err == nil {
				t.Fatalf("Execute(%q) succeeded, want a parse failure", bad)
			}
			if idx := h.rec.indexOf("clone"); idx >= 0 {
				t.Error("clone ran despite an invalid URL")
			}
		})
	}
}

// TestExecute_ExistingProject_ForceRecloned pins the --force branch at
// setup_project.go:125: an existing project is deleted from the repository and
// the flow proceeds to a fresh clone.
func TestExecute_ExistingProject_ForceRecloned(t *testing.T) {
	h := newHarness(t, harnessOptions{
		apiKey: "",
		analysis: &analyzer.AnalysisResult{
			ProjectType: "nodejs",
			Commands:    []analyzer.Command{cmd("c1", "npm install")},
		},
	})
	h.repo.seedExisting(t, "https://github.com/octocat/hello-world")

	uc := project.NewSetupProjectUseCase(h.repo, h.container, discard{})
	in := setupInput(executor.ModeAuto)
	in.WorkspaceDir = h.localPath
	in.Force = true

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("Execute(force) = error %v", err)
	}
	if !h.repo.deleted {
		t.Error("the existing project was not deleted despite Force (setup_project.go:134)")
	}
	if idx := h.rec.indexOf("clone"); idx < 0 {
		t.Error("no clone happened after a forced re-clone")
	}
	if out.Project.Status() != entities.StatusReady {
		t.Errorf("Status = %v, want Ready", out.Project.Status())
	}
}

// TestExecute_Clone_ConflictCallback pins the three conflict resolutions at
// setup_project.go:210-255. A directory conflict is NOT a hard failure when a
// callback is installed: the user chooses.
func TestExecute_Clone_ConflictCallback(t *testing.T) {
	tests := []struct {
		name      string
		action    project.ConflictAction
		wantErr   bool
		wantClone int // total Clone calls, including the retry
	}{
		{"delete then retry", project.ConflictActionDelete, false, 2},
		{"rename then retry", project.ConflictActionRename, false, 2},
		{"cancel", project.ConflictActionCancel, true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, harnessOptions{
				apiKey: "",
				analysis: &analyzer.AnalysisResult{
					ProjectType: "nodejs",
					Commands:    []analyzer.Command{cmd("c1", "npm install")},
				},
			})
			// Conflict on the first Clone only; the retry succeeds.
			h.git.conflictOnce = true

			uc := project.NewSetupProjectUseCase(h.repo, h.container, discard{})
			uc.SetConflictCallback(func(path string) project.ConflictAction { return tt.action })

			in := setupInput(executor.ModeAuto)
			in.WorkspaceDir = h.localPath

			_, err := uc.Execute(context.Background(), in)
			if tt.wantErr && err == nil {
				t.Fatal("Execute() succeeded, want the cancelled-by-user error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Execute() = error %v, want the conflict resolved", err)
			}

			var clones int
			for _, e := range h.rec.snapshot() {
				if e == "clone" {
					clones++
				}
			}
			if clones != tt.wantClone {
				t.Errorf("Clone called %d times, want %d", clones, tt.wantClone)
			}
		})
	}
}

// TestExecute_Clone_ConflictWithoutCallbackFails pins setup_project.go:256: with
// no callback installed a conflict is a plain failure.
func TestExecute_Clone_ConflictWithoutCallbackFails(t *testing.T) {
	h := newHarness(t, harnessOptions{apiKey: ""})
	h.git.conflictOnce = true

	if _, err := runExecute(t, h, executor.ModeAuto); err == nil {
		t.Fatal("Execute() succeeded on a conflict with no callback installed")
	}
}
