package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/glive/infrastructure/monitoring"
	"github.com/glive/infrastructure/security"
)

// RecoveryAgent generates and executes recovery plans
type RecoveryAgent struct {
	client        *Client
	validator     *security.CommandValidator
	riskAssessor  *RiskAssessor
	maxAttempts   int
	knowledgeBase *RecoveryKnowledgeBase
}

// NewRecoveryAgent creates a recovery agent
func NewRecoveryAgent(client *Client, validator *security.CommandValidator) *RecoveryAgent {
	if validator == nil {
		validator = security.NewCommandValidator()
	}

	return &RecoveryAgent{
		client:        client,
		validator:     validator,
		riskAssessor:  NewRiskAssessor(),
		maxAttempts:   3,
		knowledgeBase: NewRecoveryKnowledgeBase(),
	}
}

// AnalyzeFailure analyzes a command failure and generates recovery plan
func (ra *RecoveryAgent) AnalyzeFailure(ctx context.Context, result *monitoring.ExecutionResult, projectInfo *ProjectInfo) (*RecoveryPlan, error) {
	// Check knowledge base first
	if plan := ra.knowledgeBase.FindMatchingPlan(result); plan != nil {
		plan.Source = "knowledge_base"
		return plan, nil
	}

	// Generate AI recovery plan
	prompt := ra.buildRecoveryPrompt(result, projectInfo)

	messages := []Message{
		{
			Role:    "system",
			Content: recoverySystemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ra.client.Chat(messages)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse recovery plan
	var plan RecoveryPlan
	if err := ra.parseRecoveryPlan(response, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse recovery plan: %w", err)
	}

	// Validate and assess risk
	if err := ra.validateRecoveryPlan(&plan); err != nil {
		return nil, err
	}

	plan.Source = "ai_generated"
	return &plan, nil
}

// parseRecoveryPlan parses the AI response into a RecoveryPlan
func (ra *RecoveryAgent) parseRecoveryPlan(response string, plan *RecoveryPlan) error {
	// Try to extract JSON from response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end <= start {
		return fmt.Errorf("no JSON found in AI response")
	}

	jsonStr := response[start : end+1]
	if err := json.Unmarshal([]byte(jsonStr), plan); err != nil {
		return fmt.Errorf("failed to unmarshal recovery plan: %w", err)
	}

	return nil
}

// validateRecoveryPlan validates a recovery plan
func (ra *RecoveryAgent) validateRecoveryPlan(plan *RecoveryPlan) error {
	if plan.Confidence < 0 || plan.Confidence > 1 {
		return fmt.Errorf("invalid confidence: %f (must be 0-1)", plan.Confidence)
	}

	if len(plan.Steps) == 0 {
		return fmt.Errorf("recovery plan has no steps")
	}

	// Validate each step
	for i, step := range plan.Steps {
		if err := ra.validator.Validate(step.Command); err != nil {
			return fmt.Errorf("step %d failed validation: %w", i+1, err)
		}

		// Assess risk
		assessment := ra.riskAssessor.AssessRisk(&step)
		if assessment.Level == RiskCritical {
			plan.RequiresApproval = true
		}
	}

	return nil
}

// buildRecoveryPrompt creates the AI prompt for recovery
func (ra *RecoveryAgent) buildRecoveryPrompt(result *monitoring.ExecutionResult, projectInfo *ProjectInfo) string {
	return fmt.Sprintf(`You are a debugging expert helping fix a failed command.

Command: %s
Exit Code: %d
Duration: %v

Standard Output (last 50 lines):
%s

Standard Error (last 50 lines):
%s

Project Information:
- Type: %s
- Path: %s
- Detected Technologies: %v

Detected Issues:
%s

Chain of Thought - Answer these questions:
1. What type of error is this?
2. What is the root cause?
3. What steps would fix this?
4. Are these steps safe to execute automatically?
5. What's the confidence level (0-100)?
6. Can each step be rolled back if it fails?

Generate a recovery plan following these constraints:
- ONLY use safe, well-known commands from official package managers
- NEVER modify system files or global configurations
- PREFER project-local changes over system-wide changes
- PROVIDE rollback commands where possible
- ESTIMATE risk level accurately (low/medium/high/critical)
- EXPLAIN reasoning for each step

Output Format: JSON matching this structure:
{
  "error_type": "missing_dependency|configuration_error|build_failure|permission_error|network_error|environment_error|version_conflict|syntax_error|runtime_error|unknown_error",
  "root_cause": "Brief explanation of root cause",
  "confidence": 0.85,
  "recovery_steps": [
    {
      "command": "npm install missing-package",
      "reason": "Install missing dependency",
      "risk_level": "low",
      "can_rollback": true,
      "rollback_command": "npm uninstall missing-package",
      "timeout": "5m"
    }
  ],
  "manual_steps_if_failed": [
    "Check network connection",
    "Verify package name"
  ],
  "explanation": "Overall explanation of recovery strategy",
  "estimated_time": "2 minutes"
}`,
		result.Command,
		result.ExitCode,
		result.Duration,
		ra.formatOutputLines(result.StdoutLines, 50),
		ra.formatOutputLines(result.StderrLines, 50),
		projectInfo.Type,
		projectInfo.Path,
		projectInfo.Technologies,
		ra.formatDetectedIssues(result.DetectedIssues),
	)
}

const recoverySystemPrompt = `You are an expert DevOps engineer and debugger with 20 years of experience.

Your expertise:
- Diagnosing build failures, dependency issues, configuration errors
- Node.js, Python, Go, Java, Rust ecosystems
- Package managers: npm, pip, go mod, Maven, Cargo, apt, chocolatey, yarn, brew
- Common CI/CD failure patterns
- Safe recovery strategies

Your principles:
- Safety first: never suggest dangerous operations
- Prefer official solutions over hacks
- Explain reasoning clearly
- Provide rollback options
- Be honest about confidence levels
- When unsure, recommend manual investigation

Respond ONLY with valid JSON matching the RecoveryPlan structure.`

// formatOutputLines formats output lines for prompt
func (ra *RecoveryAgent) formatOutputLines(lines []monitoring.OutputLine, maxLines int) string {
	if len(lines) == 0 {
		return "(no output)"
	}

	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
	}

	var builder strings.Builder
	for i := start; i < len(lines); i++ {
		builder.WriteString(fmt.Sprintf("[%s] %s\n", lines[i].Stream, lines[i].Content))
	}

	return builder.String()
}

// formatDetectedIssues formats detected issues for prompt
func (ra *RecoveryAgent) formatDetectedIssues(issues []monitoring.DetectedIssue) string {
	if len(issues) == 0 {
		return "(no issues detected)"
	}

	var builder strings.Builder
	for _, issue := range issues {
		builder.WriteString(fmt.Sprintf("- [%s] %s: %s (confidence: %.2f)\n",
			issue.Severity.String(),
			issue.Category,
			issue.Message,
			issue.Confidence))
	}

	return builder.String()
}

// ExecuteRecovery executes a recovery plan
func (ra *RecoveryAgent) ExecuteRecovery(ctx context.Context, plan *RecoveryPlan, workDir string, observer RecoveryObserver) (*RecoveryResult, error) {
	result := &RecoveryResult{
		Plan:      plan,
		StartTime: time.Now(),
		Steps:     make([]StepResult, 0, len(plan.Steps)),
	}

	// Check if approval needed
	if plan.RequiresApproval {
		if observer != nil {
			if !observer.RequestApproval(plan) {
				result.Status = StatusCancelled
				result.Message = "User declined recovery plan"
				result.EndTime = time.Now()
				return result, nil
			}
		}
	}

	// Execute each step
	for i, step := range plan.Steps {
		stepResult := ra.executeStep(ctx, &step, workDir, observer, i+1, len(plan.Steps))
		result.Steps = append(result.Steps, stepResult)

		if !stepResult.Success {
			// Step failed - attempt rollback if possible
			if step.CanRollback && step.RollbackCmd != "" {
				ra.rollbackStep(ctx, &step, workDir, observer)
			}

			result.Status = StatusFailed
			result.Message = fmt.Sprintf("Step %d failed: %s", i+1, stepResult.Error)
			result.EndTime = time.Now()
			return result, nil
		}

		// Notify observer of progress
		if observer != nil {
			observer.OnStepCompleted(i+1, len(plan.Steps), &stepResult)
		}
	}

	result.Status = StatusSuccess
	result.Message = "Recovery completed successfully"
	result.EndTime = time.Now()

	// Store successful recovery in knowledge base
	ra.knowledgeBase.RecordSuccess(plan, result)

	return result, nil
}

// executeStep executes a single recovery step
func (ra *RecoveryAgent) executeStep(ctx context.Context, step *RecoveryStep, workDir string, observer RecoveryObserver, stepNum, totalSteps int) StepResult {
	startTime := time.Now()
	result := StepResult{
		StepIndex: stepNum,
		Command:   step.Command,
		Success:   false,
	}

	if observer != nil {
		observer.OnStepStarted(stepNum, totalSteps, step.Command)
	}

	// Parse command
	parts := strings.Fields(step.Command)
	if len(parts) == 0 {
		result.Error = "empty command"
		result.Duration = time.Since(startTime)
		return result
	}

	// Create command
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = workDir

	// Set environment if provided
	if len(step.Environment) > 0 {
		env := os.Environ()
		for k, v := range step.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Error = fmt.Sprintf("exit code %d: %s", exitErr.ExitCode(), string(output))
		} else {
			result.Error = err.Error()
		}
		result.Output = string(output)
		return result
	}

	result.Success = true
	result.Output = string(output)
	return result
}

// rollbackStep executes a rollback command
func (ra *RecoveryAgent) rollbackStep(ctx context.Context, step *RecoveryStep, workDir string, observer RecoveryObserver) {
	if step.RollbackCmd == "" {
		return
	}

	parts := strings.Fields(step.RollbackCmd)
	if len(parts) == 0 {
		return
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = workDir
	cmd.Run() // Best effort rollback
}
