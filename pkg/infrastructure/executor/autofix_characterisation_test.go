package executor_test

import (
	"context"
	"sync"
	"testing"

	"github.com/glive/infrastructure/executor"
)

// Characterisation of the auto-fix loop - GL0 T1b, step 6.
//
// The loop lives here rather than in the use case: executor.go:198 (a command
// that will not start) and executor.go:343 (a command that exits non-zero) each
// guard recovery with
//
//	if e.aiClient != nil && e.mode == ModeAuto { ... maxRetries := 5 ... }
//
// so the two properties worth pinning are the attempt ceiling and the mode gate.
// Both are easy to lose in a refactor and neither is visible from the use case,
// which only sees "the executor returned an error".
//
// No AI and no network: the fake client counts calls and always declines to
// supply a fix, which is what drives the loop to its ceiling.

// countingAI implements executor.AIClient. It never returns a usable fix, so the
// recovery loop exhausts its attempts rather than succeeding early.
type countingAI struct {
	mu      sync.Mutex
	debug   int
	autoFix int
	fixCmd  string
	fixable bool

	// declineAfter caps how many fixes are offered before the fake gives up.
	// Without a cap the executor recurses without bound (see the ceiling test
	// below), so this is what keeps the test terminating. 0 means no cap.
	declineAfter int
}

func (c *countingAI) DebugError(command, output, errorMsg string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.debug++
	return "", nil
}

func (c *countingAI) AutoFixError(command, output, errorMsg, workingDir string) (string, string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoFix++
	if c.declineAfter > 0 && c.autoFix >= c.declineAfter {
		return "", "", false, nil // stop offering fixes so the recursion unwinds
	}
	return c.fixCmd, "", c.fixable, nil
}

func (c *countingAI) counts() (debug, autoFix int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.debug, c.autoFix
}

// failingCommand is an allow-listed executable given an argument that makes it
// exit non-zero on every platform, so the recovery path at executor.go:343 runs.
//
// `git rev-parse --verify <bad ref>` is used rather than anything that invokes a
// compiler: the ceiling test drives five recovery attempts, and five toolchain
// invocations under -race take minutes. This fails in milliseconds.
func failingCommand(workingDir string) *executor.Command {
	return &executor.Command{
		ID:          "fail-1",
		Description: "a command that always fails",
		Command:     "git rev-parse --verify definitely-not-a-ref-xyzzy",
		WorkingDir:  workingDir,
		Stage:       "setup",
		Required:    true,
	}
}

// TestAutoFix_AttemptCeilingIsPerRecursionLevel_NotGlobal pins a DEFECT found by
// writing this test.
//
// executor.go:346 reads `maxRetries := 5`, and the loop it guards looks like a
// hard ceiling of five recovery attempts per command. It is not. When the AI
// offers a fix, executor.go:393 RETRIES BY CALLING Execute RECURSIVELY - and the
// recursive call enters the same block and starts a fresh `maxRetries := 5` loop
// of its own. The budget is therefore 5 per level with no bound on depth, not 5
// overall.
//
// With an AI that keeps returning plausible-but-still-failing commands - the
// normal failure mode of an LLM asked to repair a broken build - this recurses
// until the stack is exhausted. Measured before this test was capped: the run
// did not terminate within 240s and the panic trace was a solid column of
// Execute frames at executor.go:393.
//
// Why it matters beyond a hang: every level re-executes commands in the user's
// project, so an unbounded recursion means unbounded execution of AI-suggested
// commands, and unbounded provider spend, on code GLive already treats as
// untrusted. The ceiling that is supposed to contain that does not hold.
//
// This test does NOT fix it - T1b is characterisation. It caps the fake so the
// recursion unwinds, then asserts the count EXCEEDS five, which is the proof that
// the ceiling is per-level. When the recursion is replaced by a bounded loop with
// a global budget, this assertion flips and whoever fixes it updates the pin.
func TestAutoFix_AttemptCeilingIsPerRecursionLevel_NotGlobal(t *testing.T) {
	const cap = 12 // > 5, and small enough to unwind quickly

	ai := &countingAI{
		fixable:      true,
		fixCmd:       "git rev-parse --verify also-not-a-ref-xyzzy",
		declineAfter: cap,
	}
	exec := executor.New(t.TempDir(), executor.ModeAuto, ai)

	cmd := failingCommand(t.TempDir())
	if err := exec.Execute(context.Background(), cmd, func(string) {}); err == nil {
		t.Fatal("Execute() succeeded on a command that cannot succeed")
	}

	_, autoFix := ai.counts()
	if autoFix <= 5 {
		t.Errorf("AutoFixError called %d times; expected MORE than 5 because the retry at "+
			"executor.go:393 recurses and each level restarts maxRetries. If this now stops "+
			"at 5, the ceiling has been made global - update this characterisation.", autoFix)
	}
	t.Logf("AutoFixError called %d time(s) before the fake stopped offering fixes "+
		"(the documented ceiling is 5)", autoFix)
}

// TestAutoFix_StopsEarlyWhenNoFixIsOffered pins the other exit from the loop: an
// AI that reports canFix=false ends recovery immediately instead of burning the
// remaining attempts (executor.go:407, the `break`).
func TestAutoFix_StopsEarlyWhenNoFixIsOffered(t *testing.T) {
	ai := &countingAI{fixable: false}
	exec := executor.New(t.TempDir(), executor.ModeAuto, ai)

	cmd := failingCommand(t.TempDir())
	if err := exec.Execute(context.Background(), cmd, func(string) {}); err == nil {
		t.Fatal("Execute() succeeded on a command that cannot succeed")
	}

	if _, autoFix := ai.counts(); autoFix != 1 {
		t.Errorf("AutoFixError called %d times, want 1 - the loop must stop once no fix is offered", autoFix)
	}
}

// TestAutoFix_OnlyInAutoMode pins the mode gate. assisted and manual must not
// auto-fix, whatever the AI client is: the condition at executor.go:198 and :343
// is `e.mode == ModeAuto`, not merely "an AI client exists".
//
// Manual mode does not run the command at all (executor.go:114), so the command
// is marked skipped; assisted mode runs it but must not recover from a failure.
func TestAutoFix_OnlyInAutoMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       executor.ExecutionMode
		wantStatus executor.CommandStatus
	}{
		{"assisted does not auto-fix", executor.ModeAssisted, executor.CommandFailed},
		{"manual does not even run", executor.ModeManual, executor.CommandSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := &countingAI{}
			exec := executor.New(t.TempDir(), tt.mode, ai)

			cmd := failingCommand(t.TempDir())
			_ = exec.Execute(context.Background(), cmd, func(string) {})

			if _, autoFix := ai.counts(); autoFix != 0 {
				t.Errorf("AutoFixError called %d times in %v mode, want 0", autoFix, tt.mode)
			}
			if cmd.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", cmd.Status, tt.wantStatus)
			}
		})
	}
}

// TestAutoFix_RequiresAnAIClient pins the other half of the guard: in auto mode
// with a nil AI client the command fails without any recovery attempt.
func TestAutoFix_RequiresAnAIClient(t *testing.T) {
	exec := executor.New(t.TempDir(), executor.ModeAuto, nil)

	cmd := failingCommand(t.TempDir())
	err := exec.Execute(context.Background(), cmd, func(string) {})

	if err == nil {
		t.Fatal("Execute() succeeded on a command that cannot succeed")
	}
	if cmd.Status != executor.CommandFailed {
		t.Errorf("Status = %q, want %q", cmd.Status, executor.CommandFailed)
	}
}

// TestAutoFix_SucceedingCommandNeverAsksForAFix pins that the recovery path is
// only reached on failure - a command that works must not consult the AI.
func TestAutoFix_SucceedingCommandNeverAsksForAFix(t *testing.T) {
	ai := &countingAI{}
	dir := t.TempDir()
	exec := executor.New(dir, executor.ModeAuto, ai)

	cmd := &executor.Command{
		ID:          "ok-1",
		Description: "a command that succeeds",
		Command:     "git --version",
		WorkingDir:  dir,
		Stage:       "setup",
		Required:    true,
	}

	if err := exec.Execute(context.Background(), cmd, func(string) {}); err != nil {
		t.Fatalf("Execute(git --version) = error %v, want success", err)
	}
	if cmd.Status != executor.CommandCompleted {
		t.Errorf("Status = %q, want %q", cmd.Status, executor.CommandCompleted)
	}

	debug, autoFix := ai.counts()
	if debug != 0 || autoFix != 0 {
		t.Errorf("AI consulted on a successful command: DebugError=%d AutoFixError=%d", debug, autoFix)
	}
}
