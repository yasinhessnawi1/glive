package project

import (
	"context"
	"fmt"

	"github.com/glive/domain/errors"
	"github.com/glive/infrastructure/executor"
)

// SetupInput is the input for setup
type SetupInput struct {
	GitHubURL string
	Mode      string
}

// SetupOutput is the output from setup
type SetupOutput struct {
	ProjectID      string
	Degraded       bool
	DegradedReason string
}

// SetupWithFallback wraps a setup service with fallback behavior
type SetupWithFallback struct {
	primary  SetupProjectUseCase
	fallback BasicSetupService
	logger   Logger
}

// Logger interface for logging
type Logger interface {
	Warn(msg string, args ...interface{})
	Info(msg string, args ...interface{})
}

// BasicSetupService provides basic setup without AI
type BasicSetupService interface {
	SetupBasic(ctx context.Context, input SetupInput) (*SetupOutput, error)
}

// NewSetupWithFallback creates a new setup service with fallback
func NewSetupWithFallback(primary SetupProjectUseCase, fallback BasicSetupService, logger Logger) *SetupWithFallback {
	return &SetupWithFallback{
		primary:  primary,
		fallback: fallback,
		logger:   logger,
	}
}

// Setup attempts setup with primary service, falls back if needed
func (s *SetupWithFallback) Setup(ctx context.Context, input SetupInput) (*SetupOutput, error) {
	// Convert input to SetupProjectInput
	setupInput := SetupProjectInput{
		GitHubURL:    input.GitHubURL,
		Mode:         executor.ExecutionMode(input.Mode),
		WorkspaceDir: "",
		Force:        false,
	}

	// Try primary path (with AI)
	output, err := s.primary.Execute(ctx, setupInput)
	if err == nil {
		projectID := ""
		if output.Project != nil {
			projectID = output.Project.ID().Value()
		}
		return &SetupOutput{
			ProjectID: projectID,
			Degraded:  false,
		}, nil
	}

	// Log the failure
	s.logger.Warn("Primary setup failed, attempting fallback",
		"error", err,
		"category", errors.GetCategory(err))

	// Determine if fallback is appropriate
	if !s.shouldFallback(err) {
		return nil, err
	}

	// Notify user about degraded mode
	s.notifyDegradedMode(input, err)

	// Try fallback (without AI)
	fallbackOutput, fallbackErr := s.fallback.SetupBasic(ctx, input)
	if fallbackErr != nil {
		// Both failed - return original error with fallback attempt info
		return nil, errors.Wrap(err, "primary and fallback both failed")
	}

	// Mark output as degraded
	fallbackOutput.Degraded = true
	fallbackOutput.DegradedReason = err.Error()

	s.logger.Info("Fallback setup succeeded",
		"project_id", fallbackOutput.ProjectID)

	return fallbackOutput, nil
}

func (s *SetupWithFallback) shouldFallback(err error) bool {
	category := errors.GetCategory(err)

	// Fallback for AI and network errors only
	switch category {
	case errors.CategoryAI, errors.CategoryNetwork:
		return true
	default:
		return false
	}
}

func (s *SetupWithFallback) notifyDegradedMode(input SetupInput, err error) {
	fmt.Printf("\n⚠️  AI-enhanced analysis unavailable: %s\n", err.Error())
	fmt.Printf("   Continuing with basic analysis mode...\n\n")
}
