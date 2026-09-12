package executor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/glive/core/ai"
	"github.com/glive/core/types"
)

// Executor executes commands and manages their lifecycle
type Executor struct {
	workingDir string
	mode       types.ExecutionMode
	aiClient   AIClient
	mu         sync.Mutex
	maxRetries int // Added maxRetries field
}

// AIClient interface for debugging errors
type AIClient interface {
	DebugError(command string, output string, errorMsg string) (string, error)
	AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error)
	ClassifyError(command, output, errorMsg string) (*ai.ErrorContext, error)
	AutoFixWithStrategies(command, output, errorMsg, workingDir string) ([]ai.RecoveryStrategy, *ai.ErrorContext, error)
}

// New creates a new Executor
func New(workingDir string, mode types.ExecutionMode, aiClient AIClient) *Executor {
	return &Executor{
		workingDir: workingDir,
		mode:       mode,
		aiClient:   aiClient,
		maxRetries: 3, // Allow up to 3 retry attempts
	}
}

// OutputHandler is called for each line of output
type OutputHandler func(line string)

// maxRecoveryAttempts caps how many AI recovery strategies may be applied to a
// single command in TOTAL, across the whole recovery.
const maxRecoveryAttempts = 5

// ErrRecoveryExhausted is returned when recovery used its whole attempt budget
// without producing a command that runs.
var ErrRecoveryExhausted = errors.New("auto-fix recovery exhausted")

// recoveryBudget is the shared, decrementing attempt count for one call to
// Execute.
//
// The recovery loops below retry by calling Execute RECURSIVELY, and each
// recursive call fetched a fresh list of strategies and worked through all of
// them. Nothing bounded the depth - the maxRetries field on Executor was set to
// 3 and then never read - so an AI that keeps offering plausible-but-still-
// failing commands drove recovery without limit, re-executing commands in the
// user's project at every level.
//
// This is the same defect as pkg/infrastructure/executor, fixed the same way.
// The agent server runs on THIS copy until GL0 T6 routes it through the use
// case, so it cannot wait for the twin to be deleted.
type recoveryBudget struct {
	remaining int
}

// use consumes one attempt, reporting whether one was available.
func (b *recoveryBudget) use() bool {
	if b.remaining <= 0 {
		return false
	}
	b.remaining--
	return true
}

// Execute runs a command and streams output
// All long-running operations MUST accept context for cancellation
func (e *Executor) Execute(ctx context.Context, cmd *types.Command, outputHandler OutputHandler) error {
	return e.execute(ctx, cmd, outputHandler, &recoveryBudget{remaining: maxRecoveryAttempts})
}

// execute is Execute's body, carrying the recovery budget shared by the
// recursive retries. See recoveryBudget.
func (e *Executor) execute(ctx context.Context, cmd *types.Command, outputHandler OutputHandler, budget *recoveryBudget) error {
	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	e.mu.Lock()
	cmd.Status = types.CommandRunning
	e.mu.Unlock()

	// Build the command
	workDir := cmd.WorkingDir
	if workDir == "" {
		workDir = e.workingDir
	}

	// Parse command into parts
	parts := strings.Fields(cmd.Command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Use CommandContext for proper cancellation support
	execCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	execCmd.Dir = workDir

	// Apply environment variables
	execCmd.Env = os.Environ() // Inherit parent environment
	if cmd.Env != nil {
		for k, v := range cmd.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture stdout and stderr
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := execCmd.Start(); err != nil {
		e.mu.Lock()
		cmd.Status = types.CommandFailed
		cmd.Error = err.Error()
		originalCmd := cmd.Command
		e.mu.Unlock()

		// Try multi-strategy auto-fix for command start failures (e.g., command not found)
		if e.aiClient != nil && e.mode == types.ModeAuto {
			// Use new multi-strategy recovery
			strategies, errorCtx, aiErr := e.aiClient.AutoFixWithStrategies(
				cmd.Command,
				"",
				err.Error(),
				workDir,
			)

			if aiErr == nil && len(strategies) > 0 {
				if outputHandler != nil {
					outputHandler(fmt.Sprintf("   🤖 AI classified error as: %s (confidence: %.0f%%)",
						errorCtx.Category, errorCtx.Confidence*100))
					outputHandler(fmt.Sprintf("   🔧 Attempting %d recovery %s...",
						len(strategies),
						map[bool]string{true: "strategy", false: "strategies"}[len(strategies) == 1]))
				}

				// Try each strategy in order, while the shared budget lasts. The
				// budget is what bounds the recursion: without it each recursive
				// retry fetched a fresh strategy list and started over.
				for i, strategy := range strategies {
					if !budget.use() {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⛔ Recovery budget of %d attempts exhausted; giving up",
								maxRecoveryAttempts))
						}
						break
					}
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ▶️  Strategy %d/%d: %s (%d of %d attempts left)",
							i+1, len(strategies), strategy.Name(), budget.remaining, maxRecoveryAttempts))
					}

					// Apply the strategy
					fixedCmd, explanation, applyErr := strategy.Apply(cmd, "", err.Error())
					if applyErr != nil {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⏭️  Skipped: %s", applyErr.Error()))
						}
						continue
					}

					// Check if this is a skip strategy
					if fixedCmd == nil && strings.Contains(explanation, "Skipping") {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⏭️  %s", explanation))
						}
						e.mu.Lock()
						cmd.Status = types.CommandSkipped
						e.mu.Unlock()
						return nil // Consider this a success
					}

					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   💡 %s", explanation))
					}

					// Save original env
					originalEnv := cmd.Env

					// Apply the fix
					cmd.Command = fixedCmd.Command
					cmd.Env = fixedCmd.Env
					cmd.Status = types.CommandPending

					// Retry with fixed command
					retryErr := e.execute(ctx, cmd, outputHandler, budget)
					if retryErr == nil {
						// Success!
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ✅ Strategy '%s' succeeded!", strategy.Name()))
						}
						return nil
					}

					// This strategy didn't work, restore and try next
					cmd.Command = originalCmd
					cmd.Env = originalEnv
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ❌ Strategy '%s' didn't resolve the issue", strategy.Name()))
					}
				}

				// All strategies failed
				if outputHandler != nil {
					outputHandler("   ⚠️  All recovery strategies failed")
				}
			}
		}

		return fmt.Errorf("failed to start command: %w", err)
	}

	// Read output concurrently
	var outputBuilder, errorBuilder strings.Builder
	var wg sync.WaitGroup

	// Read stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")
			if outputHandler != nil {
				outputHandler(line)
			}
		}
	}()

	// Read stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			errorBuilder.WriteString(line + "\n")
			if outputHandler != nil {
				outputHandler("[ERROR] " + line)
			}
		}
	}()

	// Wait for output readers
	wg.Wait()

	// Wait for command to complete
	err = execCmd.Wait()

	e.mu.Lock()
	cmd.Output = outputBuilder.String()
	cmd.Error = errorBuilder.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			cmd.ExitCode = exitErr.ExitCode()
		}
		cmd.Status = types.CommandFailed

		// Try multi-strategy auto-fix if available
		if e.aiClient != nil && e.mode == types.ModeAuto {
			e.mu.Unlock() // Unlock before calling AI

			// Use new multi-strategy recovery
			strategies, errorCtx, aiErr := e.aiClient.AutoFixWithStrategies(
				cmd.Command,
				cmd.Output,
				cmd.Error,
				workDir,
			)

			if aiErr == nil && len(strategies) > 0 {
				if outputHandler != nil {
					outputHandler(fmt.Sprintf("   🤖 AI classified error as: %s (confidence: %.0f%%)",
						errorCtx.Category, errorCtx.Confidence*100))
					outputHandler(fmt.Sprintf("   🔧 Attempting %d recovery %s...",
						len(strategies),
						map[bool]string{true: "strategy", false: "strategies"}[len(strategies) == 1]))
				}

				// Try each strategy in order, while the shared budget lasts. The
				// budget is what bounds the recursion: without it each recursive
				// retry fetched a fresh strategy list and started over.
				for i, strategy := range strategies {
					if !budget.use() {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⛔ Recovery budget of %d attempts exhausted; giving up",
								maxRecoveryAttempts))
						}
						break
					}
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ▶️  Strategy %d/%d: %s (%d of %d attempts left)",
							i+1, len(strategies), strategy.Name(), budget.remaining, maxRecoveryAttempts))
					}

					// Apply the strategy
					fixedCmd, explanation, applyErr := strategy.Apply(cmd, cmd.Output, cmd.Error)
					if applyErr != nil {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⏭️  Skipped: %s", applyErr.Error()))
						}
						continue
					}

					// Check if this is a skip strategy
					if fixedCmd == nil && strings.Contains(explanation, "Skipping") {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⏭️  %s", explanation))
						}
						e.mu.Lock()
						cmd.Status = types.CommandSkipped
						e.mu.Unlock()
						return nil // Consider this a success
					}

					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   💡 %s", explanation))
					}

					// Save original command
					originalCmd := cmd.Command
					originalEnv := cmd.Env

					// Apply the fix
					cmd.Command = fixedCmd.Command
					cmd.Env = fixedCmd.Env
					cmd.Status = types.CommandPending

					// Retry with fixed command
					retryErr := e.execute(ctx, cmd, outputHandler, budget)
					if retryErr == nil {
						// Success!
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ✅ Strategy '%s' succeeded!", strategy.Name()))
						}
						return nil
					}

					// This strategy didn't work, restore and try next
					cmd.Command = originalCmd
					cmd.Env = originalEnv
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ❌ Strategy '%s' didn't resolve the issue", strategy.Name()))
					}
				}

				// All strategies failed
				if outputHandler != nil {
					outputHandler("   ⚠️  All recovery strategies failed")
				}
			}

			e.mu.Lock()
		}

		e.mu.Unlock()
		return fmt.Errorf("command failed with exit code %d", cmd.ExitCode)
	}

	cmd.Status = types.CommandCompleted
	cmd.ExitCode = 0
	e.mu.Unlock()

	return nil
}

// ExecuteMultiple executes multiple commands in sequence
func (e *Executor) ExecuteMultiple(ctx context.Context, commands []*types.Command, outputHandler OutputHandler) error {
	for _, cmd := range commands {
		// Check for context cancellation before each command
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !cmd.Required && e.mode == types.ModeManual {
			cmd.Status = types.CommandSkipped
			continue
		}

		if outputHandler != nil {
			outputHandler(fmt.Sprintf("\n=== Executing: %s ===\n", cmd.Description))
		}

		if err := e.Execute(ctx, cmd, outputHandler); err != nil {
			if cmd.Required {
				return fmt.Errorf("required command failed: %w", err)
			}
			// Non-required command failed, continue
			if outputHandler != nil {
				outputHandler(fmt.Sprintf("Optional command failed: %v\n", err))
			}
		}
	}

	return nil
}

// CheckCommandAvailable checks if a command is available on the system
func CheckCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetInstalledVersion gets the version of an installed tool
func GetInstalledVersion(command string, versionFlag string) (string, error) {
	if versionFlag == "" {
		versionFlag = "--version"
	}

	cmd := exec.Command(command, versionFlag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
