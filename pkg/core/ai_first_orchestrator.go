package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/glive/core/ai"
	"github.com/glive/core/state"
	"github.com/glive/core/types"
	infraai "github.com/glive/infrastructure/ai"
	infraexecutor "github.com/glive/infrastructure/executor"
)

// AIFirstExecutionMode defines execution mode for AI-first orchestrator
type AIFirstExecutionMode string

const (
	ModeAIFirst     AIFirstExecutionMode = "ai_first"
	ModeTraditional AIFirstExecutionMode = "traditional"
	ModeHybrid      AIFirstExecutionMode = "hybrid"
)

// AIFirstConfig contains configuration for AI-first orchestrator
type AIFirstConfig struct {
	PrimaryMode         AIFirstExecutionMode `json:"primary_mode"`
	FallbackEnabled     bool                 `json:"fallback_enabled"`
	ConfidenceThreshold float64              `json:"confidence_threshold"`
	AIMonitoring        bool                 `json:"ai_monitoring"`
	AutoRecovery        bool                 `json:"auto_recovery"`
	LearningEnabled     bool                 `json:"learning_enabled"`
	Timeout             time.Duration        `json:"timeout"`
}

// DefaultAIFirstConfig returns default AI-first configuration
func DefaultAIFirstConfig() AIFirstConfig {
	return AIFirstConfig{
		PrimaryMode:         ModeAIFirst,
		FallbackEnabled:     true,
		ConfidenceThreshold: 0.7,
		AIMonitoring:        true,
		AutoRecovery:        true,
		LearningEnabled:     false, // Disabled by default initially
		Timeout:             30 * time.Second,
	}
}

// RunInput represents input for running a project
type RunInput struct {
	GitHubURL   string
	ProjectPath string
	Mode        types.ExecutionMode
	ProgressCB  ProgressCallback
}

// RunOutput represents output from running a project
type RunOutput struct {
	Project      *Project
	Success      bool
	UsedFallback bool
	Confidence   float64
}

// AIFirstOrchestrator orchestrates AI-first execution
type AIFirstOrchestrator struct {
	aiAnalyzer    *infraai.ComprehensiveAIAnalyzer
	aiMonitor     *infraai.AIExecutionMonitor
	recoveryAgent *RecoveryAgent
	fallbackOrch  *Orchestrator
	config        AIFirstConfig
	fallbackSel   *FallbackSelector
	output        io.Writer
	stateManager  *state.Manager
}

// RecoveryAgent handles autonomous recovery
type RecoveryAgent struct {
	aiClient *ai.Client
}

// NewRecoveryAgent creates a new recovery agent
func NewRecoveryAgent(aiClient *ai.Client) *RecoveryAgent {
	return &RecoveryAgent{
		aiClient: aiClient,
	}
}

// NewAIFirstOrchestrator creates a new AI-first orchestrator
func NewAIFirstOrchestrator(
	cfg *Config,
	aiClient *ai.Client,
	fallbackOrch *Orchestrator,
	output io.Writer,
) (*AIFirstOrchestrator, error) {
	if output == nil {
		output = os.Stdout
	}

	// Create infrastructure AI client
	infraAIClient := infraai.NewClient(cfg.APIKey, cfg.APIProvider, cfg.APIEndpoint)

	// Create comprehensive analyzer
	analyzer := infraai.NewComprehensiveAIAnalyzer(infraAIClient)

	// Create AI monitor (will be created when needed)
	var monitor *infraai.AIExecutionMonitor

	// Create recovery agent
	recoveryAgent := NewRecoveryAgent(aiClient)

	// Create fallback selector with cache
	// Cache provider will be initialized when cache is available
	var cacheProvider CacheProvider = nil
	fallbackSel := NewFallbackSelector(cacheProvider)

	// Create state manager
	stateDir := filepath.Join(cfg.WorkspaceDir, ".glive")
	stateMgr, err := state.New(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &AIFirstOrchestrator{
		aiAnalyzer:    analyzer,
		aiMonitor:     monitor,
		recoveryAgent: recoveryAgent,
		fallbackOrch:  fallbackOrch,
		config:        DefaultAIFirstConfig(),
		fallbackSel:   fallbackSel,
		output:        output,
		stateManager:  stateMgr,
	}, nil
}

// SetConfig sets the AI-first configuration
func (o *AIFirstOrchestrator) SetConfig(config AIFirstConfig) {
	o.config = config
}

// Run executes a project using AI-first approach
func (o *AIFirstOrchestrator) Run(ctx context.Context, input RunInput) (*RunOutput, error) {
	// Apply timeout to context
	if o.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.config.Timeout)
		defer cancel()
	}

	// Phase 1: Agent Analysis
	o.log("🤖 Agent analyzing project...\n")
	analysis, err := o.aiAnalyzer.Analyze(ctx, input.ProjectPath)
	if err != nil {
		o.log(fmt.Sprintf("   ⚠️  Agent analysis failed: %v\n", err))
		if o.config.FallbackEnabled {
			o.log("   ℹ️  Using traditional mode...\n")
			return o.runFallback(ctx, input, TriggerAIFailure)
		}
		return nil, fmt.Errorf("agent analysis failed and fallback disabled: %w", err)
	}

	// Check confidence threshold
	if analysis.Confidence < o.config.ConfidenceThreshold {
		o.log(fmt.Sprintf("   ⚠️  Low confidence (%.0f%%)\n", analysis.Confidence*100))
		if o.config.FallbackEnabled {
			o.log("   ℹ️  Using traditional mode...\n")
			return o.runFallback(ctx, input, TriggerLowConfidence)
		}
		return nil, fmt.Errorf("agent confidence too low (%.0f%%) and fallback disabled", analysis.Confidence*100)
	}

	o.log(fmt.Sprintf("   ✓ %s project (%s)\n", analysis.ProjectType, analysis.Framework))

	// Phase 2: Execute with Agent Supervision
	o.log("⚙️  Setting up project...\n")
	result, err := o.executeWithAISupervision(ctx, input, analysis)
	if err != nil {
		o.log(fmt.Sprintf("   ⚠️  Setup failed: %v\n", err))

		// Recovery attempts
		if o.config.AutoRecovery {
			o.log("   🔄 Attempting recovery...\n")
			recovered, recErr := o.attemptRecovery(ctx, result, analysis, err)
			if recErr == nil && recovered != nil {
				o.log("   ✓ Recovery successful!\n")
				return recovered, nil
			}
		}

		// Fallback if recovery failed
		if o.config.FallbackEnabled {
			o.log("   ℹ️  Using traditional mode...\n")
			return o.runFallback(ctx, input, TriggerAIFailure)
		}

		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Learn from outcome
	if o.config.LearningEnabled {
		o.recordSuccess(analysis, result)
	}

	o.log("   ✓ Setup complete\n")
	return result, nil
}

// executeWithAISupervision executes commands with AI monitoring
func (o *AIFirstOrchestrator) executeWithAISupervision(ctx context.Context, input RunInput, analysis *infraai.ComprehensiveAnalysis) (*RunOutput, error) {
	if analysis.ExecutionPlan == nil || len(analysis.ExecutionPlan.Phases) == 0 {
		return nil, fmt.Errorf("no execution plan provided")
	}

	// Create executor
	execMode := infraexecutor.ExecutionMode(string(input.Mode))
	if execMode == "" {
		execMode = infraexecutor.ModeAuto
	}
	exec := infraexecutor.New(input.ProjectPath, execMode, nil)

	// Execute each phase
	for _, phase := range analysis.ExecutionPlan.Phases {
		o.log(fmt.Sprintf("   📦 Phase: %s\n", phase.Name))
		o.log(fmt.Sprintf("   📝 Reason: %s\n", phase.Reason))

		// Convert commands to executor format
		commands := make([]*infraexecutor.Command, 0, len(phase.Commands))
		for _, cmd := range phase.Commands {
			execCmd := &infraexecutor.Command{
				ID:          cmd.ID,
				Description: cmd.Description,
				Command:     cmd.Command,
				WorkingDir:  cmd.WorkingDir,
				Stage:       cmd.Stage,
				Required:    cmd.Required,
				Status:      infraexecutor.CommandPending,
			}
			commands = append(commands, execCmd)
		}

		// Execute commands
		outputHandler := infraexecutor.OutputHandler(func(line string) {
			o.log(fmt.Sprintf("      %s\n", line))
		})

		// Apply timeout if specified
		phaseCtx := ctx
		if phase.Timeout > 0 {
			var cancel context.CancelFunc
			phaseCtx, cancel = context.WithTimeout(ctx, phase.Timeout)
			defer cancel()
		}

		err := exec.ExecuteMultiple(phaseCtx, commands, outputHandler)
		if err != nil {
			if phase.CanFail {
				o.log(fmt.Sprintf("   ⚠️  Phase failed but can continue: %v\n", err))
				continue
			}
			return nil, fmt.Errorf("phase %s failed: %w", phase.Name, err)
		}

		o.log("   ✓ Phase completed\n\n")
	}

	// Create project result
	project := &Project{
		ID:        generateProjectID(),
		Name:      filepath.Base(input.ProjectPath),
		LocalPath: input.ProjectPath,
		Status:    StatusReady,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return &RunOutput{
		Project:      project,
		Success:      true,
		UsedFallback: false,
		Confidence:   analysis.Confidence,
	}, nil
}

// attemptRecovery attempts to recover from execution failure
func (o *AIFirstOrchestrator) attemptRecovery(ctx context.Context, result *RunOutput, analysis *infraai.ComprehensiveAnalysis, err error) (*RunOutput, error) {
	if o.recoveryAgent == nil {
		return nil, fmt.Errorf("recovery agent not available")
	}

	// TODO: Implement recovery logic using AI
	// For now, return error to trigger fallback
	return nil, fmt.Errorf("recovery not yet implemented")
}

// runFallback runs using traditional orchestrator
func (o *AIFirstOrchestrator) runFallback(ctx context.Context, input RunInput, trigger FallbackTrigger) (*RunOutput, error) {
	if o.fallbackOrch == nil {
		return nil, fmt.Errorf("fallback orchestrator not available")
	}

	// Convert RunInput to orchestrator format
	execMode := types.ExecutionMode(input.Mode)
	if execMode == "" {
		execMode = types.ModeAuto
	}

	// Convert ProgressCallback
	var progressCB ProgressCallback
	if input.ProgressCB != nil {
		progressCB = func(update ProgressUpdate) {
			// Convert to expected format - just pass through
		}
	}

	project, err := o.fallbackOrch.RunProject(ctx, "", input.GitHubURL, execMode, progressCB)
	if err != nil {
		return nil, fmt.Errorf("fallback execution failed: %w", err)
	}

	return &RunOutput{
		Project:      project,
		Success:      true,
		UsedFallback: true,
		Confidence:   0.5, // Lower confidence for fallback
	}, nil
}

// recordSuccess records successful execution for learning
func (o *AIFirstOrchestrator) recordSuccess(analysis *infraai.ComprehensiveAnalysis, result *RunOutput) {
	// TODO: Implement learning system integration
	// For now, just log
	o.log("   📚 Recording success for future learning...\n")
}

// log writes to the output writer
func (o *AIFirstOrchestrator) log(message string) {
	fmt.Fprint(o.output, message)
}
