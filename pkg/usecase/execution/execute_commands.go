package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
)

// ContainerExecutor is an alias for the container's Executor interface
type ContainerExecutor = container.Executor

// ExecuteCommandsInput represents the input for executing commands
type ExecuteCommandsInput struct {
	Commands   []*executor.Command
	WorkingDir string
	Mode       executor.ExecutionMode
}

// ExecuteCommandsOutput represents the output of executing commands
type ExecuteCommandsOutput struct {
	ExecutedCommands []*executor.Command
	FailedCommands   []*executor.Command
	SkippedCommands  []*executor.Command
}

// ExecuteCommandsUseCase handles command execution
type ExecuteCommandsUseCase struct {
	container  *container.Container
	output     executor.OutputHandler
	progressCB ProgressCallback
	aiClient   AIClient
	maxAIFixes int
}

// ProgressCallback is called for progress updates
type ProgressCallback func(projectID string, stage string, message string, percentage int)

// NewExecuteCommandsUseCase creates a new ExecuteCommandsUseCase
func NewExecuteCommandsUseCase(cont *container.Container, outputHandler executor.OutputHandler) *ExecuteCommandsUseCase {
	return &ExecuteCommandsUseCase{
		container:  cont,
		output:     outputHandler,
		maxAIFixes: 3, // Max AI fix attempts per command
	}
}

// SetProgressCallback sets the progress callback
func (uc *ExecuteCommandsUseCase) SetProgressCallback(callback ProgressCallback) {
	uc.progressCB = callback
}

// SetAIClient sets the AI client for auto-fix functionality
func (uc *ExecuteCommandsUseCase) SetAIClient(client AIClient) {
	uc.aiClient = client
}

// Execute executes the commands use case
func (uc *ExecuteCommandsUseCase) Execute(ctx context.Context, input ExecuteCommandsInput) error {
	if len(input.Commands) == 0 {
		return nil
	}

	exec := uc.container.CreateExecutor(input.WorkingDir, input.Mode)

	output := &ExecuteCommandsOutput{
		ExecutedCommands: []*executor.Command{},
		FailedCommands:   []*executor.Command{},
		SkippedCommands:  []*executor.Command{},
	}

	for i, cmd := range input.Commands {
		// Calculate progress range for this command
		// Installation step ranges from 60% to 90%
		progressStart := 60 + (i * 30 / len(input.Commands))
		progressEnd := 60 + ((i + 1) * 30 / len(input.Commands))
		if progressEnd > 90 {
			progressEnd = 90
		}

		// Clear any progress indicator before showing command info
		if uc.output != nil {
			// Clear line before writing command header (send ANSI escape code through output handler)
			uc.output("\r\033[2K")
			uc.output(fmt.Sprintf("   [%d/%d] %s\n", i+1, len(input.Commands), cmd.Description))
			uc.output(fmt.Sprintf("   $ %s\n", cmd.Command))
		}

		// Only send progress update at start for first command or when stage changes
		// This reduces flickering significantly
		if uc.progressCB != nil && i == 0 {
			uc.progressCB("", "executing", cmd.Description, progressStart)
		}

		// Execute with AI auto-fix retry loop
		err := uc.executeWithAutoFix(ctx, exec, cmd, input.WorkingDir)

		if err != nil {
			if cmd.Required {
				output.FailedCommands = append(output.FailedCommands, cmd)
				if uc.progressCB != nil {
					uc.progressCB("", "executing", fmt.Sprintf("Failed: %s", cmd.Description), progressEnd)
				}
				return fmt.Errorf("required command failed: %w", err)
			}
			output.FailedCommands = append(output.FailedCommands, cmd)
			if uc.output != nil {
				uc.output(fmt.Sprintf("   ⚠️  Command failed (optional): %v\n", err))
			}
			// Don't send progress update for optional failures to reduce flicker
		} else {
			output.ExecutedCommands = append(output.ExecutedCommands, cmd)
			if uc.output != nil {
				uc.output("   ✓ Command completed successfully\n")
			}
			// Only update progress when command completes (and only for significant milestones)
			// Update every 2nd command or on last command to reduce flickering
			if uc.progressCB != nil && (i%2 == 1 || i == len(input.Commands)-1) {
				uc.progressCB("", "executing", fmt.Sprintf("Installing dependencies (%d/%d)", i+1, len(input.Commands)), progressEnd)
			}
		}

		if uc.output != nil {
			uc.output("\n")
		}
	}

	return nil
}

// executeWithAutoFix executes a command with AI auto-fix retry on failure
func (uc *ExecuteCommandsUseCase) executeWithAutoFix(ctx context.Context, exec ContainerExecutor, cmd *executor.Command, workingDir string) error {
	var lastErr error
	var lastOutput string
	currentCmd := cmd

	for attempt := 0; attempt <= uc.maxAIFixes; attempt++ {
		// Execute the command
		err := exec.Execute(ctx, currentCmd, uc.output)

		if err == nil {
			// Command succeeded
			if attempt > 0 {
				// This was an AI-fixed command that succeeded
				if uc.output != nil {
					uc.output("   🤖 AI fix successful!\n")
				}
			}
			return nil
		}

		// Command failed
		lastErr = err
		lastOutput = currentCmd.Output

		// If no AI client or we've exceeded max attempts, return the error
		if uc.aiClient == nil || attempt >= uc.maxAIFixes {
			break
		}

		// Try AI auto-fix
		if uc.output != nil {
			uc.output(fmt.Sprintf("   🤖 Command failed, attempting AI auto-fix (attempt %d/%d)...\n", attempt+1, uc.maxAIFixes))
		}

		fixedCmd, explanation, canFix, aiErr := uc.aiClient.AutoFixError(
			currentCmd.Command,
			lastOutput,
			err.Error(),
			workingDir,
		)

		if aiErr != nil || !canFix {
			if uc.output != nil {
				if aiErr != nil {
					uc.output(fmt.Sprintf("   ⚠️  AI auto-fix failed: %v\n", aiErr))
				} else {
					uc.output("   ⚠️  AI determined this error cannot be auto-fixed\n")
				}
			}
			break
		}

		// AI provided a fix
		if uc.output != nil {
			uc.output(fmt.Sprintf("   🔧 AI fix: %s\n", explanation))
			uc.output(fmt.Sprintf("   $ %s\n", fixedCmd))
		}

		// Check if the fix is just a command or includes additional steps
		// Some fixes might be multi-step (e.g., "create file then run command")
		fixedCommands := uc.parseAIFixCommands(fixedCmd, cmd, workingDir)

		// Execute any prerequisite commands first
		for _, prereq := range fixedCommands[:len(fixedCommands)-1] {
			if uc.output != nil {
				uc.output(fmt.Sprintf("   🔧 Running prerequisite: %s\n", prereq.Command))
			}
			if prereqErr := exec.Execute(ctx, prereq, uc.output); prereqErr != nil {
				if uc.output != nil {
					uc.output(fmt.Sprintf("   ⚠️  Prerequisite failed: %v\n", prereqErr))
				}
				// Continue trying even if prerequisite fails
			}
		}

		// Update current command for next attempt
		currentCmd = fixedCommands[len(fixedCommands)-1]
	}

	return lastErr
}

// parseAIFixCommands parses AI fix response which might contain multiple commands
func (uc *ExecuteCommandsUseCase) parseAIFixCommands(fixedCmd string, originalCmd *executor.Command, workingDir string) []*executor.Command {
	commands := []*executor.Command{}

	// Split by common command separators (&&, ;, newline)
	// But be careful with commands that have these in strings
	lines := strings.Split(fixedCmd, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by && if present
		parts := strings.Split(line, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			commands = append(commands, &executor.Command{
				Command:     part,
				Description: originalCmd.Description + " (AI-fixed)",
				WorkingDir:  workingDir,
				Required:    originalCmd.Required,
				Status:      executor.CommandPending,
			})
		}
	}

	// If no commands were parsed, return the original fixed command
	if len(commands) == 0 {
		commands = append(commands, &executor.Command{
			Command:     fixedCmd,
			Description: originalCmd.Description + " (AI-fixed)",
			WorkingDir:  workingDir,
			Required:    originalCmd.Required,
			Status:      executor.CommandPending,
		})
	}

	return commands
}
