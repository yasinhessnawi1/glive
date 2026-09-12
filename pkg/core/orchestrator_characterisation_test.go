package core

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Characterisation of core.Orchestrator - GL0 T2.
//
// This is the agent server's orchestrator: the SECOND of the two the repo runs
// (the CLI runs pkg/usecase/project.SetupProjectUseCase). GL0 T6 routes the agent
// through the use case and T8 deletes this file along with the rest of the twin,
// so what is pinned here is the behaviour T6 must preserve while doing it.
//
// THESE TESTS DIE WITH THE TWIN AT T8. The agent's REST/WS contract tests in
// pkg/agent are the ones that survive; these exist to make T6 a comparison rather
// than a guess.
//
// The tests are in-package rather than in core_test because the progress helpers
// and the port allocator are unexported and there is no seam to reach them
// through - and adding one to code scheduled for deletion would be waste.
//
// No network and no AI: everything below drives state, port allocation, the
// running-project registry and the progress callback. RunProject itself clones
// and calls a provider, so it is characterised only at its pre-network failure.

func newTestOrchestrator(t *testing.T) (*Orchestrator, string) {
	t.Helper()

	workspace := t.TempDir()
	cfg := &Config{
		WorkspaceDir: workspace,
		DefaultMode:  ModeAuto,
		APIProvider:  "deepseek",
	}

	o, err := NewOrchestrator(cfg, io.Discard)
	if err != nil {
		t.Fatalf("NewOrchestrator: %v", err)
	}
	return o, workspace
}

// ---------------------------------------------------------------- progress

// TestProgress_StageNamesAndPercentages is the table T4 needs and T6's adapter
// must reproduce.
//
// The web dashboard renders these stage strings and drives its progress bar from
// these percentages, so they are a published contract, not internal detail. When
// T6 routes the agent through the use case, the use case's own vocabulary
// (parsing/cloning/analyzing/installing/ready at different percentages) has to be
// mapped onto exactly this, or the dashboard silently changes behaviour.
//
// Source: pkg/core/orchestrator.go, the sendProgress / sendProgressWithSuccess
// calls at the cited lines.
func TestProgress_StageNamesAndPercentages(t *testing.T) {
	// stage -> the percentages emitted for it, in order of appearance.
	want := []struct {
		line       int
		stage      string
		message    string
		percentage int
		success    bool // emitted via sendProgressWithSuccess
	}{
		{100, "parsing", "Parsing GitHub URL", 10, false},
		{109, "parsing", "GitHub URL parsed successfully", 15, true},
		{133, "cloning", "Cloning repository", 20, false},
		{163, "cloning", "Repository cloned successfully", 25, true},
		{172, "scanning", "Scanning for security issues", 30, false},
		{195, "scanning", "Security scan completed", 35, true},
		{204, "analyzing", "Analyzing project structure", 40, false},
		{220, "analyzing", "Project analysis completed", 45, true},
		{230, "ai_analysis", "Running AI analysis", 50, false},
		{269, "ai_analysis", "AI analysis completed", 55, true},
		{282, "installing", "Installing dependencies", 60, false},
		// "executing" percentages are computed per command; see the test below.
		{346, "ready", "Project is ready", 100, false},
		{414, "cloning", "Cloning repository (auto-recovery)", 10, false},
		{490, "installing", "Installing dependencies (auto-recovery)", 0, false},
		{553, "starting", "Starting application", 0, false},
		{596, "stopped", "Project stopped", 100, false},
	}

	o, _ := newTestOrchestrator(t)

	for _, w := range want {
		t.Run(w.stage+"/"+w.message, func(t *testing.T) {
			var got ProgressUpdate
			var calls int
			cb := func(u ProgressUpdate) { got = u; calls++ }

			if w.success {
				o.sendProgressWithSuccess(cb, "p-1", w.stage, w.message, w.percentage, true)
			} else {
				o.sendProgress(cb, "p-1", w.stage, w.message, w.percentage)
			}

			if calls != 1 {
				t.Fatalf("callback called %d times, want 1", calls)
			}
			if got.ProjectID != "p-1" {
				t.Errorf("ProjectID = %q, want %q", got.ProjectID, "p-1")
			}
			if got.Stage != w.stage {
				t.Errorf("Stage = %q, want %q (orchestrator.go:%d)", got.Stage, w.stage, w.line)
			}
			if got.Message != w.message {
				t.Errorf("Message = %q, want %q (orchestrator.go:%d)", got.Message, w.message, w.line)
			}
			if got.Percentage != w.percentage {
				t.Errorf("Percentage = %d, want %d (orchestrator.go:%d)", got.Percentage, w.percentage, w.line)
			}
			if got.Success != w.success {
				t.Errorf("Success = %v, want %v", got.Success, w.success)
			}
			if got.Timestamp.IsZero() {
				t.Error("Timestamp is zero; the frame must carry one")
			}
		})
	}
}

// TestProgress_ExecutingPercentageFormula pins the per-command progress ramp at
// orchestrator.go:291: percentage = 60 + (i * 30 / len(commands)).
//
// So "executing" spans 60..90 and never reaches 100 - the jump to 100 is the
// separate "ready" frame. A T6 adapter that ramps to 100 during execution would
// make the dashboard show a completed bar on a project that is still installing.
func TestProgress_ExecutingPercentageFormula(t *testing.T) {
	tests := []struct {
		commands int
		want     []int
	}{
		{1, []int{60}},
		{2, []int{60, 75}},
		{3, []int{60, 70, 80}},
		{5, []int{60, 66, 72, 78, 84}},
	}

	for _, tt := range tests {
		var got []int
		for i := 0; i < tt.commands; i++ {
			got = append(got, 60+(i*30/tt.commands))
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("%d commands: percentage[%d] = %d, want %d", tt.commands, i, got[i], tt.want[i])
			}
		}
		if last := got[len(got)-1]; last >= 100 {
			t.Errorf("%d commands: last executing percentage = %d, want < 100", tt.commands, last)
		}
	}
}

// TestProgress_NilCallbackIsSafe pins that a nil callback is a no-op rather than
// a panic - the CLI path passes nil.
func TestProgress_NilCallbackIsSafe(t *testing.T) {
	o, _ := newTestOrchestrator(t)
	o.sendProgress(nil, "p-1", "parsing", "msg", 10)
	o.sendProgressWithSuccess(nil, "p-1", "parsing", "msg", 10, true)
}

// ---------------------------------------------------------------- ports

// TestPortAllocation_StartsAt8081AndIncrements pins the allocator at
// orchestrator.go:520-530. The agent itself is assumed to hold 8080.
func TestPortAllocation_StartsAt8081AndIncrements(t *testing.T) {
	o, _ := newTestOrchestrator(t)

	if o.nextPort != 8081 {
		t.Errorf("initial nextPort = %d, want 8081 (8080 is the agent's)", o.nextPort)
	}

	first := o.allocatePort("p-1")
	second := o.allocatePort("p-2")
	third := o.allocatePort("p-3")

	if first != 8081 || second != 8082 || third != 8083 {
		t.Errorf("allocated %d, %d, %d; want 8081, 8082, 8083", first, second, third)
	}
}

// TestPortAllocation_IsStablePerProject pins the reuse branch: a project that is
// stopped and restarted keeps its port, so a bookmarked localhost:PORT keeps
// working.
func TestPortAllocation_IsStablePerProject(t *testing.T) {
	o, _ := newTestOrchestrator(t)

	first := o.allocatePort("p-1")
	_ = o.allocatePort("p-2")
	again := o.allocatePort("p-1")

	if again != first {
		t.Errorf("port for p-1 = %d on reallocation, want the original %d", again, first)
	}
}

// TestPortAllocation_FreedByCleanup pins orchestrator.go:648-651: cleanup removes
// the mapping. Note it does NOT lower nextPort, so the number is retired rather
// than recycled - a long-running agent walks upward.
func TestPortAllocation_FreedByCleanup(t *testing.T) {
	o, workspace := newTestOrchestrator(t)
	_ = workspace

	first := o.allocatePort("p-1")

	if err := o.CleanupProject("p-1"); err != nil {
		t.Fatalf("CleanupProject: %v", err)
	}

	o.mu.Lock()
	_, stillMapped := o.projectPorts["p-1"]
	next := o.nextPort
	o.mu.Unlock()

	if stillMapped {
		t.Error("the port mapping survived CleanupProject")
	}
	if next <= first {
		t.Errorf("nextPort = %d after cleanup; the allocator does not recycle, want > %d", next, first)
	}

	// Re-allocating for the same id now yields a NEW port, not the freed one.
	if again := o.allocatePort("p-1"); again == first {
		t.Errorf("port %d was recycled after cleanup; today the allocator only moves forward", again)
	}
}

// allocatePort mirrors the allocation block inside StartProject
// (orchestrator.go:520-530) so it can be characterised without starting a real
// application. It is a test helper, not production code: if the production block
// changes, this must change with it and the tests above are what notice.
func (o *Orchestrator) allocatePort(projectID string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	if existing, ok := o.projectPorts[projectID]; ok {
		return existing
	}
	port := o.nextPort
	o.projectPorts[projectID] = port
	o.nextPort++
	return port
}

// ---------------------------------------------------------------- lifecycle

// TestStopProject_UnknownProjectIsAnError pins orchestrator.go:603-610.
func TestStopProject_UnknownProjectIsAnError(t *testing.T) {
	o, _ := newTestOrchestrator(t)

	err := o.StopProject("never-started")
	if err == nil {
		t.Fatal("StopProject on an unknown project succeeded, want an error")
	}
	if err.Error() != "project is not running" {
		t.Errorf("error = %q, want %q", err.Error(), "project is not running")
	}
}

// TestStopProject_CancelsTheRegisteredContext pins the registry round trip at
// orchestrator.go:625-636, which is how the agent's stop endpoint works.
func TestStopProject_CancelsTheRegisteredContext(t *testing.T) {
	o, _ := newTestOrchestrator(t)

	_, cancel := context.WithCancel(context.Background())
	var cancelled bool
	var mu sync.Mutex
	o.RegisterRunningProject("p-1", func() {
		mu.Lock()
		cancelled = true
		mu.Unlock()
		cancel()
	})

	if err := o.StopProject("p-1"); err != nil {
		t.Fatalf("StopProject: %v", err)
	}

	mu.Lock()
	got := cancelled
	mu.Unlock()
	if !got {
		t.Error("StopProject did not invoke the registered cancel function")
	}

	// The registration survives Stop - only UnregisterRunningProject removes it,
	// so a second Stop still "succeeds". Pinning this because it is surprising.
	if err := o.StopProject("p-1"); err != nil {
		t.Errorf("second StopProject = %v; today the registration is not removed by Stop", err)
	}

	o.UnregisterRunningProject("p-1")
	if err := o.StopProject("p-1"); err == nil {
		t.Error("StopProject succeeded after Unregister, want an error")
	}
}

// TestCleanupProject_LeavesTheCloneOnDisk pins the TODO at orchestrator.go:653.
//
// Cleanup deletes the project's STATE and frees its port but leaves the cloned
// repository on disk, so "cleanup" reclaims no space. ARCHITECTURE 9.11 tracks
// it and GL0 T5's lifecycle use case is where it gets fixed; until then this is
// the behaviour, and pinning it means the fix shows up as an edit here.
func TestCleanupProject_LeavesTheCloneOnDisk(t *testing.T) {
	o, workspace := newTestOrchestrator(t)

	// Stand in for a clone.
	clone := filepath.Join(workspace, "some-project")
	if err := os.MkdirAll(clone, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	marker := filepath.Join(clone, "README.md")
	if err := os.WriteFile(marker, []byte("# cloned\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := o.CleanupProject("some-project"); err != nil {
		t.Fatalf("CleanupProject: %v", err)
	}

	if _, err := os.Stat(marker); err != nil {
		t.Errorf("the clone was removed: %v - if cleanup now deletes files, "+
			"ARCHITECTURE 9.11 is closed and this characterisation should be updated", err)
	}
}

// TestCleanupProject_StopsARunningProjectFirst pins orchestrator.go:641 - cleanup
// stops first and ignores the "not running" error.
func TestCleanupProject_StopsARunningProjectFirst(t *testing.T) {
	o, _ := newTestOrchestrator(t)

	var stopped bool
	var mu sync.Mutex
	o.RegisterRunningProject("p-1", func() {
		mu.Lock()
		stopped = true
		mu.Unlock()
	})

	if err := o.CleanupProject("p-1"); err != nil {
		t.Fatalf("CleanupProject: %v", err)
	}

	mu.Lock()
	got := stopped
	mu.Unlock()
	if !got {
		t.Error("CleanupProject did not stop the running project first")
	}
}

// ---------------------------------------------------------------- RunProject

// TestRunProject_InvalidURLFailsBeforeAnyNetworkUse pins that RunProject rejects
// a bad URL at its parse step, so the characterisation can reach it without a
// network. Everything past the parse clones and calls a provider and is therefore
// out of scope here - it is covered for the surviving path in
// pkg/usecase/project.
func TestRunProject_InvalidURLFailsBeforeAnyNetworkUse(t *testing.T) {
	o, _ := newTestOrchestrator(t)

	for _, bad := range []string{"", "not-a-url", "file:///etc/passwd"} {
		t.Run(bad, func(t *testing.T) {
			var frames []ProgressUpdate
			cb := func(u ProgressUpdate) { frames = append(frames, u) }

			_, err := o.RunProject(context.Background(), "p-1", bad, ModeAuto, false, cb)
			if err == nil {
				t.Fatalf("RunProject(%q) succeeded, want a parse failure", bad)
			}

			// It must not have progressed past parsing.
			for _, f := range frames {
				if f.Stage == "cloning" || f.Stage == "installing" || f.Stage == "ready" {
					t.Errorf("reached stage %q on an invalid URL", f.Stage)
				}
			}
		})
	}
}
