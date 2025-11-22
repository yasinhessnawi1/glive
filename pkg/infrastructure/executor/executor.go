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

// Executor executes commands and manages their lifecycle
type Executor struct {
	workingDir string
	mode       ExecutionMode
	aiClient   AIClient
	validator  *security.CommandValidator
	mu         sync.Mutex
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

// OutputHandler is called for each line of output
type OutputHandler func(line string)

// Execute runs a command and streams output
// All long-running operations MUST accept context for cancellation
func (e *Executor) Execute(ctx context.Context, cmd *Command, outputHandler OutputHandler) error {
	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
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
		e.mu.Unlock()
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

		// Try to auto-fix error with AI if available
		if e.aiClient != nil && e.mode == ModeAuto {
			e.mu.Unlock() // Unlock before calling AI

			fixedCmd, explanation, canFix, aiErr := e.aiClient.AutoFixError(
				cmd.Command,
				cmd.Output,
				cmd.Error,
				workDir,
			)

			if aiErr == nil && canFix && fixedCmd != "" {
				// Validate the fixed command before retrying
				if err := e.validator.Validate(fixedCmd); err != nil {
					if outputHandler != nil {
						outputHandler(fmt.Sprintf("   ⚠️  AI suggested fix failed security validation: %v", err))
					}
					// Command validation failed, return error
					return fmt.Errorf("command failed with exit code %d", cmd.ExitCode)
				}

				// AI found a fix - inform user and retry
				if outputHandler != nil {
					outputHandler(fmt.Sprintf("   🤖 AI detected an issue: %s", explanation))
					outputHandler(fmt.Sprintf("   🔧 Auto-fixing command: %s", fixedCmd))
					outputHandler("")
				}

				// Update the command and retry
				originalCmd := cmd.Command
				cmd.Command = fixedCmd
				cmd.Status = CommandPending

				// Recursive retry with fixed command (only once to avoid infinite loops)
				retryErr := e.Execute(ctx, cmd, outputHandler)
				if retryErr == nil {
					// Fixed successfully!
					if outputHandler != nil {
						outputHandler("   ✅ Auto-fix successful!")
					}
					return nil
				}

				// Auto-fix didn't work, restore original command and continue with error
				cmd.Command = originalCmd
				if outputHandler != nil {
					outputHandler("   ⚠️  Auto-fix didn't resolve the issue")
				}
			}

			e.mu.Lock()
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
	for _, cmd := range commands {
		// Check for context cancellation before each command
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !cmd.Required && e.mode == ModeManual {
			cmd.Status = CommandSkipped
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
