package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/glive/infrastructure/ai"
	"github.com/glive/infrastructure/monitoring"
	"github.com/glive/infrastructure/security"
)

// CommandExecutor is an interface for executing commands
type CommandExecutor interface {
	Execute(ctx context.Context, cmd *Command, outputHandler OutputHandler) error
	ExecuteMultiple(ctx context.Context, commands []*Command, outputHandler OutputHandler) error
}

// ExecutorFactory interface to avoid import cycle
type ExecutorFactory interface {
	Create(workingDir string, mode ExecutionMode) CommandExecutor
}

// AIClientFactory interface to avoid import cycle
type AIClientFactory interface {
	Create() interface{}
}

// SelfHealingExecutor wraps normal executor with recovery capabilities
type SelfHealingExecutor struct {
	baseExecutor  CommandExecutor
	recoveryAgent *ai.RecoveryAgent
	config        SelfHealingConfig
	observer      ai.RecoveryObserver
	projectInfo   *ai.ProjectInfo
}

// SelfHealingConfig configures self-healing behavior
type SelfHealingConfig struct {
	Enabled             bool
	MaxRecoveryAttempts int
	AutoApproveRisk     ai.RiskLevel // Auto-approve up to this risk level
	RequireConfirmation bool         // Require user confirmation
	LearnFromSuccesses  bool         // Add to knowledge base
}

// DefaultSelfHealingConfig returns default configuration
func DefaultSelfHealingConfig() SelfHealingConfig {
	return SelfHealingConfig{
		Enabled:             true,
		MaxRecoveryAttempts: 3,
		AutoApproveRisk:     ai.RiskLow,
		RequireConfirmation: false,
		LearnFromSuccesses:  true,
	}
}

// NewSelfHealingExecutor creates a self-healing executor
func NewSelfHealingExecutor(baseExecutor CommandExecutor, aiClient *ai.Client, config SelfHealingConfig) *SelfHealingExecutor {
	if !config.Enabled {
		return &SelfHealingExecutor{
			baseExecutor: baseExecutor,
			config:       config,
		}
	}

	validator := security.NewCommandValidator()
	recoveryAgent := ai.NewRecoveryAgent(aiClient, validator)

	return &SelfHealingExecutor{
		baseExecutor:  baseExecutor,
		recoveryAgent: recoveryAgent,
		config:        config,
		projectInfo:   &ai.ProjectInfo{},
	}
}

// SetProjectInfo sets project information for recovery context
func (she *SelfHealingExecutor) SetProjectInfo(info *ai.ProjectInfo) {
	she.projectInfo = info
}

// SetObserver sets the recovery observer
func (she *SelfHealingExecutor) SetObserver(observer ai.RecoveryObserver) {
	she.observer = observer
}

// Execute runs a command with self-healing
func (she *SelfHealingExecutor) Execute(ctx context.Context, cmd *Command, outputHandler OutputHandler) error {
	if !she.config.Enabled {
		return she.baseExecutor.Execute(ctx, cmd, outputHandler)
	}

	attempts := 0
	maxAttempts := she.config.MaxRecoveryAttempts

	for attempts < maxAttempts {
		// Execute command with monitoring
		result, err := she.executeWithMonitoring(ctx, cmd, outputHandler)

		if err == nil && result.ExitCode == 0 {
			// Success!
			return nil
		}

		// Failure detected
		attempts++

		if attempts >= maxAttempts {
			// Max attempts reached
			return fmt.Errorf("max recovery attempts (%d) exceeded: %w", maxAttempts, err)
		}

		// Attempt recovery
		recoveryPlan, err := she.recoveryAgent.AnalyzeFailure(ctx, result, she.projectInfo)
		if err != nil {
			return fmt.Errorf("recovery analysis failed: %w", err)
		}

		// Check if recovery is viable
		if recoveryPlan.Confidence < 0.5 {
			return fmt.Errorf("low confidence in recovery plan (%.2f) - manual intervention needed", recoveryPlan.Confidence)
		}

		// Check if approval needed
		if recoveryPlan.RequiresApproval && she.config.RequireConfirmation {
			if she.observer == nil || !she.observer.RequestApproval(recoveryPlan) {
				return fmt.Errorf("recovery plan requires approval but was declined")
			}
		}

		// Execute recovery
		recoveryResult, err := she.recoveryAgent.ExecuteRecovery(ctx, recoveryPlan, cmd.WorkingDir, she.observer)
		if err != nil {
			return fmt.Errorf("recovery execution failed: %w", err)
		}

		if recoveryResult.Status != ai.StatusSuccess {
			// Recovery failed
			if outputHandler != nil {
				outputHandler(fmt.Sprintf("   ⚠️  Recovery attempt %d/%d failed: %s\n", attempts, maxAttempts, recoveryResult.Message))
			}
			continue // Try again
		}

		// Recovery succeeded, retry original command
		if outputHandler != nil {
			outputHandler(fmt.Sprintf("   ✅ Recovery successful, retrying command...\n"))
		}
		continue
	}

	return fmt.Errorf("unexpected execution flow")
}

// outputForwarder forwards output to handler and captures it
type outputForwarder struct {
	handler OutputHandler
	output  *strings.Builder
}

func (f *outputForwarder) OnOutputLine(line monitoring.OutputLine) {
	f.output.WriteString(line.Content + "\n")
	prefix := ""
	if line.Stream == "stderr" {
		prefix = "[ERROR] "
	}
	f.handler(prefix + line.Content)
}

func (f *outputForwarder) OnPatternDetected(line monitoring.OutputLine, match monitoring.PatternMatch) {
	// Patterns are detected in OnOutputLine, this is for additional handling
}

// executeWithMonitoring executes a command with monitoring
func (she *SelfHealingExecutor) executeWithMonitoring(ctx context.Context, cmd *Command, outputHandler OutputHandler) (*monitoring.ExecutionResult, error) {
	// Parse command
	parts := strings.Fields(cmd.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// Create exec command
	workDir := cmd.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	execCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	execCmd.Dir = workDir

	// Create monitor
	patternDetector := monitoring.NewPatternDetector()
	monitor := monitoring.NewCommandMonitor(execCmd, patternDetector)

	// Track output for result
	var combinedOutput strings.Builder

	// Add observer to forward output to handler
	if outputHandler != nil {
		observer := &outputForwarder{
			handler: outputHandler,
			output:  &combinedOutput,
		}
		monitor.AddObserver(observer)
	}

	// Start monitoring (this will start the command)
	err := monitor.Start(ctx)

	// Get execution result
	result := monitor.GetExecutionResult()

	// Update command status
	if err == nil && result.ExitCode == 0 {
		cmd.Status = CommandCompleted
		cmd.ExitCode = 0
		cmd.Output = combinedOutput.String()
	} else {
		cmd.Status = CommandFailed
		cmd.ExitCode = result.ExitCode
		cmd.Output = combinedOutput.String()
		if err != nil {
			cmd.Error = err.Error()
		}
	}

	return result, err
}

// ExecuteMultiple executes multiple commands in sequence with self-healing
func (she *SelfHealingExecutor) ExecuteMultiple(ctx context.Context, commands []*Command, outputHandler OutputHandler) error {
	for _, cmd := range commands {
		// Check for context cancellation before each command
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := she.Execute(ctx, cmd, outputHandler); err != nil {
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

// CreateSelfHealingExecutorFactory creates a factory for self-healing executors
func CreateSelfHealingExecutorFactory(baseFactory ExecutorFactory, aiClientFactory AIClientFactory, config SelfHealingConfig) ExecutorFactory {
	return &selfHealingExecutorFactory{
		baseFactory:    baseFactory,
		aiClientFactory: aiClientFactory,
		config:         config,
	}
}

type selfHealingExecutorFactory struct {
	baseFactory    ExecutorFactory
	aiClientFactory AIClientFactory
	config         SelfHealingConfig
}

func (f *selfHealingExecutorFactory) Create(workingDir string, mode ExecutionMode) CommandExecutor {
	baseExecutor := f.baseFactory.Create(workingDir, mode)

	if !f.config.Enabled {
		return baseExecutor
	}

	// Get AI client - we need the underlying *ai.Client
	// The factory returns a container.AIClient which might be wrapped
	aiClientInterface := f.aiClientFactory.Create()

	// Try to get the underlying *ai.Client
	// For LazyAIClient, we need to extract it
	var aiClient *ai.Client

	// Check if it's already a *ai.Client
	if client, ok := aiClientInterface.(*ai.Client); ok {
		aiClient = client
	} else {
		// Try to get from lazy wrapper (this is a bit fragile but necessary)
		// We'll create a new client directly instead
		// For now, disable self-healing if we can't get the client
		// In a real implementation, we'd need a better way to get the client
		return baseExecutor
	}

	return NewSelfHealingExecutor(baseExecutor, aiClient, f.config)
}

