package executor

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/glive/infrastructure/security"
)

// ApprovalCallback is called for assisted mode to get user approval
// Returns true if approved, false if rejected
type ApprovalCallback func(cmd *Command) (approved bool)

// Executor executes commands and manages their lifecycle
type Executor struct {
	workingDir       string
	mode             ExecutionMode
	aiClient         AIClient
	validator        *security.CommandValidator
	approvalCallback ApprovalCallback
	mu               sync.Mutex
}

// ExecutionMode defines how commands should be executed
type ExecutionMode string

const (
	ModeAuto     ExecutionMode = "auto"
	ModeAssisted ExecutionMode = "assisted"
	ModeManual   ExecutionMode = "manual"
)

// AIClient interface for debugging errors
type AIClient interface {
	DebugError(command string, output string, errorMsg string) (string, error)
	AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error)
}

// Command represents a command to execute
type Command struct {
	ID          string
	Description string
	Command     string
	WorkingDir  string
	Stage       string
	Required    bool
	Status      CommandStatus
	Output      string
	Error       string
	ExitCode    int
}

// CommandStatus represents the execution status of a command
type CommandStatus string

const (
	CommandPending   CommandStatus = "pending"
	CommandRunning   CommandStatus = "running"
	CommandCompleted CommandStatus = "completed"
	CommandFailed    CommandStatus = "failed"
	CommandSkipped   CommandStatus = "skipped"
)

// New creates a new Executor
func New(workingDir string, mode ExecutionMode, aiClient AIClient) *Executor {
	return &Executor{
		workingDir: workingDir,
		mode:       mode,
		aiClient:   aiClient,
		validator:  security.NewCommandValidator(),
	}
}

// NewWithApproval creates a new Executor with an approval callback for assisted mode
func NewWithApproval(workingDir string, mode ExecutionMode, aiClient AIClient, approvalCb ApprovalCallback) *Executor {
	return &Executor{
		workingDir:       workingDir,
		mode:             mode,
		aiClient:         aiClient,
		validator:        security.NewCommandValidator(),
		approvalCallback: approvalCb,
	}
}

// SetApprovalCallback sets the approval callback for assisted mode
func (e *Executor) SetApprovalCallback(cb ApprovalCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.approvalCallback = cb
}

// OutputHandler is called for each line of output
type OutputHandler func(line string)

// ErrCommandRejected is returned when a command is rejected in assisted mode
var ErrCommandRejected = fmt.Errorf("command rejected by user")

// Execute runs a command and streams output
// All long-running operations MUST accept context for cancellation
func (e *Executor) Execute(ctx context.Context, cmd *Command, outputHandler OutputHandler) error {
	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Manual mode: Don't execute, just show the command
	if e.mode == ModeManual {
		if outputHandler != nil {
			outputHandler(fmt.Sprintf("📋 Manual step: %s", cmd.Description))
			outputHandler(fmt.Sprintf("   Run this command manually:"))
			outputHandler(fmt.Sprintf("   $ %s", cmd.Command))
			if cmd.WorkingDir != "" {
				outputHandler(fmt.Sprintf("   (in directory: %s)", cmd.WorkingDir))
			}
			outputHandler("")
		}
		cmd.Status = CommandSkipped
		return nil
	}

	// Assisted mode: Ask for approval before each command
	if e.mode == ModeAssisted && e.approvalCallback != nil {
		if outputHandler != nil {
			outputHandler(fmt.Sprintf("⏸️  Approval required for: %s", cmd.Description))
			outputHandler(fmt.Sprintf("   Command: %s", cmd.Command))
		}

		approved := e.approvalCallback(cmd)
		if !approved {
			if outputHandler != nil {
				outputHandler("   ❌ Command rejected by user")
			}
			cmd.Status = CommandSkipped
			return ErrCommandRejected
		}

		if outputHandler != nil {
			outputHandler("   ✅ Command approved, executing...")
		}
	}

	// Validate command before execution
	if err := e.validator.Validate(cmd.Command); err != nil {
		e.mu.Lock()
		cmd.Status = CommandFailed
		cmd.Error = fmt.Sprintf("command validation failed: %v", err)
		e.mu.Unlock()
		return fmt.Errorf("security validation failed: %w", err)
	}

	e.mu.Lock()
	cmd.Status = CommandRunning
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
		cmd.Status = CommandFailed
		cmd.Error = err.Error()
		originalCmd := cmd.Command
		e.mu.Unlock()

		// Try to auto-fix command start errors with AI (e.g., command not found)
		if e.aiClient != nil && e.mode == ModeAuto {
			maxRetries := 5 // Try up to 5 different recovery strategies
			for attempt := 1; attempt <= maxRetries; attempt++ {
				if outputHandler != nil {
					outputHandler(fmt.Sprintf("   🔄 Recovery attempt %d/%d (command start failure)...", attempt, maxRetries))
				}

				fixedCmd, explanation, canFix, aiErr := e.aiClient.AutoFixError(
					cmd.Command,
					"",
					err.Error(),
					workDir,
				)

				// Check if this is a non-critical command that can be skipped
				if explanation == "Skipping optional command - project can run without it" {
					if outputHandler != nil {
						outputHandler("   ⏭️  Skipping non-critical command - continuing execution")
					}
					e.mu.Lock()
					cmd.Status = CommandSkipped
					e.mu.Unlock()
					return nil // Consider this a success
				}

				if aiErr == nil && canFix && fixedCmd != "" {
					// Validate the fixed command before retrying
					if err := e.validator.Validate(fixedCmd); err != nil {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⚠️  AI suggestion #%d failed security validation: %v", attempt, err))
						}
						continue // Try next strategy
					}

					// AI found a fix - inform user and retry
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   🤖 AI detected an issue: %s", explanation))
						outputHandler(fmt.Sprintf("   🔧 Trying recovery strategy #%d: %s", attempt, fixedCmd))
						outputHandler("")
					}

					// Update the command and retry
					cmd.Command = fixedCmd
					cmd.Status = CommandPending

					// Recursive retry with fixed command
					retryErr := e.Execute(ctx, cmd, outputHandler)
					if retryErr == nil {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ✅ Auto-fix successful after %d attempt(s)!", attempt))
						}
						return nil
					}

					// This strategy didn't work, try next one
					cmd.Command = originalCmd
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ⚠️  Recovery strategy #%d didn't resolve the issue", attempt))
					}
				} else {
					// No more strategies available
					if outputHandler != nil && attempt < maxRetries {
						outputHandler(fmt.Sprintf("   ⚠️  No additional recovery strategies available for attempt #%d", attempt))
					}
					break
				}
			}

			// All recovery attempts failed - provide helpful final message
			if outputHandler != nil {
				outputHandler("")
				outputHandler("   ╔═══════════════════════════════════════════════════════════════╗")
				outputHandler("   ║  ⚠️  GLive couldn't automatically fix this issue             ║")
				outputHandler("   ╚═══════════════════════════════════════════════════════════════╝")
				outputHandler("")
				outputHandler("   📋 What went wrong:")
				outputHandler(fmt.Sprintf("      • Command: %s", originalCmd))
				outputHandler(fmt.Sprintf("      • Error: %s", err.Error()))
				outputHandler("")
				outputHandler("   💡 What you can do:")
				outputHandler("      1. Open the project in VS Code to manually fix the issue")
				outputHandler("      2. Check the project's README for specific setup instructions")
				outputHandler("      3. Review the error details above for clues")
				outputHandler("")
				outputHandler("   🙏 We apologize that GLive couldn't run this project automatically.")
				outputHandler("   Future updates will include fixes for these types of errors.")
				outputHandler("")
			}

			return fmt.Errorf("failed to start command after %d recovery attempts: %w", maxRetries, err)
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
		cmd.Status = CommandFailed

		// Try to auto-fix error with AI if available - try multiple strategies
		if e.aiClient != nil && e.mode == ModeAuto {
			e.mu.Unlock() // Unlock before calling AI

			maxRetries := 5 // Try up to 5 different recovery strategies
			for attempt := 1; attempt <= maxRetries; attempt++ {
				if outputHandler != nil {
					outputHandler(fmt.Sprintf("   🔄 Recovery attempt %d/%d...", attempt, maxRetries))
				}

				fixedCmd, explanation, canFix, aiErr := e.aiClient.AutoFixError(
					cmd.Command,
					cmd.Output,
					cmd.Error,
					workDir,
				)

				// Check if this is a non-critical command that can be skipped
				if explanation == "Skipping optional command - project can run without it" {
					if outputHandler != nil {
						outputHandler("   ⏭️  Skipping non-critical command - continuing execution")
					}
					e.mu.Lock()
					cmd.Status = CommandSkipped
					e.mu.Unlock()
					return nil // Consider this a success
				}

				if aiErr == nil && canFix && fixedCmd != "" {
					// Validate the fixed command before retrying
					if err := e.validator.Validate(fixedCmd); err != nil {
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ⚠️  AI suggestion #%d failed security validation: %v", attempt, err))
						}
						// Try next strategy
						continue
					}

					// AI found a fix - inform user and retry
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   🤖 AI detected an issue: %s", explanation))
						outputHandler(fmt.Sprintf("   🔧 Trying recovery strategy #%d: %s", attempt, fixedCmd))
						outputHandler("")
					}

					// Update the command and retry
					originalCmd := cmd.Command
					cmd.Command = fixedCmd
					cmd.Status = CommandPending

					// Recursive retry with fixed command
					retryErr := e.Execute(ctx, cmd, outputHandler)
					if retryErr == nil {
						// Fixed successfully!
						if outputHandler != nil {
							outputHandler(fmt.Sprintf("   ✅ Auto-fix successful after %d attempt(s)!", attempt))
						}
						return nil
					}

					// This strategy didn't work, try next one
					cmd.Command = originalCmd
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ⚠️  Recovery strategy #%d didn't resolve the issue", attempt))
					}
				} else {
					// No more strategies available
					if outputHandler != nil && attempt < maxRetries {
						outputHandler(fmt.Sprintf("   ⚠️  No additional recovery strategies available for attempt #%d", attempt))
					}
					break
				}
			}

			// All recovery attempts failed - provide helpful final message
			e.mu.Lock()
			e.mu.Unlock()

			if outputHandler != nil {
				outputHandler("")
				outputHandler("   ╔═══════════════════════════════════════════════════════════════╗")
				outputHandler("   ║  ⚠️  GLive couldn't automatically fix this issue             ║")
				outputHandler("   ╚═══════════════════════════════════════════════════════════════╝")
				outputHandler("")
				outputHandler("   📋 What went wrong:")
				outputHandler(fmt.Sprintf("      • Command: %s", cmd.Command))
				outputHandler(fmt.Sprintf("      • Exit code: %d", cmd.ExitCode))
				outputHandler("")
				outputHandler("   🔍 Error details:")
				// Show first few lines of error
				errorLines := strings.Split(strings.TrimSpace(cmd.Error), "\n")
				for i, line := range errorLines {
					if i >= 5 { // Limit to 5 lines
						outputHandler("      ...")
						break
					}
					outputHandler("      " + line)
				}
				outputHandler("")
				outputHandler("   💡 What you can do:")
				outputHandler("      1. Open the project in VS Code to manually fix the issue")
				outputHandler("      2. Check the project's README for specific setup instructions")
				outputHandler("      3. Review the error details above for clues")
				outputHandler("")
				outputHandler("   🙏 We apologize that GLive couldn't run this project automatically.")
				outputHandler("   Future updates will include fixes for these types of errors.")
				outputHandler("")
			}

			return fmt.Errorf("command failed with exit code %d after %d recovery attempts", cmd.ExitCode, maxRetries)
		}

		e.mu.Unlock()
		return fmt.Errorf("command failed with exit code %d", cmd.ExitCode)
	}

	cmd.Status = CommandCompleted
	cmd.ExitCode = 0
	e.mu.Unlock()

	return nil
}

// ExecuteMultiple executes multiple commands in sequence
func (e *Executor) ExecuteMultiple(ctx context.Context, commands []*Command, outputHandler OutputHandler) error {
	for i, cmd := range commands {
		// Check for context cancellation before each command
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// In manual mode, show step number
		if e.mode == ModeManual {
			if outputHandler != nil {
				outputHandler(fmt.Sprintf("\n📋 Step %d of %d:", i+1, len(commands)))
			}
		} else if outputHandler != nil {
			outputHandler(fmt.Sprintf("\n=== Executing: %s ===\n", cmd.Description))
		}

		if err := e.Execute(ctx, cmd, outputHandler); err != nil {
			// In assisted mode, if command was rejected, stop execution
			if err == ErrCommandRejected {
				if cmd.Required {
					return fmt.Errorf("required command rejected: execution stopped")
				}
				// Non-required command rejected, continue
				continue
			}

			if cmd.Required {
				return fmt.Errorf("required command failed: %w", err)
			}
			// Non-required command failed, continue
			if outputHandler != nil {
				outputHandler(fmt.Sprintf("Optional command failed: %v\n", err))
			}
		}
	}

	// In manual mode, show completion message
	if e.mode == ModeManual && outputHandler != nil {
		outputHandler("\n✅ All manual steps have been displayed.")
		outputHandler("   Please run the commands above in order to set up your project.")
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
