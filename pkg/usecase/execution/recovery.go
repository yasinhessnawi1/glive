package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/glive/domain/errors"
	"github.com/glive/infrastructure/executor"
)

// RecoveryStrategy defines a strategy for recovering from errors
type RecoveryStrategy interface {
	CanRecover(err error, context *ExecutionContext) bool
	Recover(ctx context.Context, err error, context *ExecutionContext) (*RecoveryAction, error)
}

// RecoveryAction represents an action to recover from an error
type RecoveryAction struct {
	Type        RecoveryType
	Description string
	NewCommand  *executor.Command // For replacement strategies
	Skip        bool              // For skip strategies
}

// RecoveryType indicates the type of recovery action
type RecoveryType int

const (
	RecoveryRetry RecoveryType = iota
	RecoveryReplace
	RecoverySkip
	RecoveryRollback
	RecoveryAbort
)

// ExecutionContext holds context for command execution
type ExecutionContext struct {
	CurrentCommand   *executor.Command
	LastOutput       string
	WorkingDir       string
	AIFixAttempts    int
	RecoveryAttempts int
}

// CompositeRecovery tries multiple strategies
type CompositeRecovery struct {
	strategies  []RecoveryStrategy
	logger      Logger
	maxAttempts int
}

// Logger interface for recovery logging
type Logger interface {
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// NewCompositeRecovery creates a new composite recovery
func NewCompositeRecovery(strategies []RecoveryStrategy, logger Logger, maxAttempts int) *CompositeRecovery {
	return &CompositeRecovery{
		strategies:  strategies,
		logger:      logger,
		maxAttempts: maxAttempts,
	}
}

// Attempt tries to recover from an error using available strategies
func (r *CompositeRecovery) Attempt(ctx context.Context, err error, execCtx *ExecutionContext) (*RecoveryAction, error) {
	if execCtx.RecoveryAttempts >= r.maxAttempts {
		return nil, fmt.Errorf("maximum recovery attempts reached")
	}

	execCtx.RecoveryAttempts++

	for _, strategy := range r.strategies {
		if strategy.CanRecover(err, execCtx) {
			action, recoverErr := strategy.Recover(ctx, err, execCtx)
			if recoverErr == nil {
				r.logger.Info("Recovery strategy succeeded",
					"strategy", fmt.Sprintf("%T", strategy),
					"action", action.Type)
				return action, nil
			}
			r.logger.Debug("Recovery strategy failed",
				"strategy", fmt.Sprintf("%T", strategy),
				"error", recoverErr)
		}
	}
	return nil, fmt.Errorf("no recovery strategy succeeded")
}

// AIFixStrategy uses AI to fix command errors
type AIFixStrategy struct {
	aiClient AIClient
	maxFixes int
}

// AIClient interface for AI operations
type AIClient interface {
	AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error)
}

// NewAIFixStrategy creates a new AI fix strategy
func NewAIFixStrategy(aiClient AIClient, maxFixes int) *AIFixStrategy {
	return &AIFixStrategy{
		aiClient: aiClient,
		maxFixes: maxFixes,
	}
}

func (s *AIFixStrategy) CanRecover(err error, ctx *ExecutionContext) bool {
	// Don't try AI fix if we've already tried too many times
	if ctx.AIFixAttempts >= s.maxFixes {
		return false
	}

	// Only for execution errors
	return errors.GetCategory(err) == errors.CategoryExecution
}

func (s *AIFixStrategy) Recover(ctx context.Context, err error, execCtx *ExecutionContext) (*RecoveryAction, error) {
	if s.aiClient == nil {
		return nil, fmt.Errorf("AI client not available")
	}

	execCtx.AIFixAttempts++

	fixedCmd, explanation, canFix, aiErr := s.aiClient.AutoFixError(
		execCtx.CurrentCommand.Command,
		execCtx.LastOutput,
		err.Error(),
		execCtx.WorkingDir,
	)

	if aiErr != nil || !canFix {
		return nil, fmt.Errorf("AI could not fix the error: %w", aiErr)
	}

	return &RecoveryAction{
		Type:        RecoveryReplace,
		Description: explanation,
		NewCommand: &executor.Command{
			Command:     fixedCmd,
			Description: execCtx.CurrentCommand.Description + " (AI-fixed)",
			WorkingDir:  execCtx.CurrentCommand.WorkingDir,
			Required:    execCtx.CurrentCommand.Required,
		},
	}, nil
}

// MissingToolStrategy handles missing command errors
type MissingToolStrategy struct {
	toolInstallers map[string]string
}

// NewMissingToolStrategy creates a new missing tool strategy
func NewMissingToolStrategy(toolInstallers map[string]string) *MissingToolStrategy {
	return &MissingToolStrategy{
		toolInstallers: toolInstallers,
	}
}

func (s *MissingToolStrategy) CanRecover(err error, ctx *ExecutionContext) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "not recognized") ||
		strings.Contains(errStr, "command not found")
}

func (s *MissingToolStrategy) Recover(ctx context.Context, err error, execCtx *ExecutionContext) (*RecoveryAction, error) {
	// Extract tool name
	tool := extractToolName(execCtx.CurrentCommand.Command)

	// Check if we know how to install it
	installer, ok := s.toolInstallers[tool]
	if !ok {
		return &RecoveryAction{
			Type:        RecoveryAbort,
			Description: fmt.Sprintf("Tool '%s' is not installed. Please install it and try again.", tool),
		}, nil
	}

	return &RecoveryAction{
		Type:        RecoveryReplace,
		Description: fmt.Sprintf("Installing missing tool: %s", tool),
		NewCommand: &executor.Command{
			Command:     installer,
			Description: fmt.Sprintf("Install %s", tool),
			WorkingDir:  execCtx.CurrentCommand.WorkingDir,
			Required:    true,
		},
	}, nil
}

func extractToolName(command string) string {
	parts := strings.Fields(command)
	if len(parts) > 0 {
		return parts[0]
	}
	return command
}
