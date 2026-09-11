package prompts

import (
	"fmt"
	"strings"
)

// ChainOfThoughtBuilder builds chain-of-thought reasoning prompts
type ChainOfThoughtBuilder struct {
	steps []CoTStep
}

// CoTStep represents a step in chain-of-thought reasoning
type CoTStep struct {
	Number       int
	Title        string
	Instructions []string
	Question     string
	Answer       string // For examples
}

// NewChainOfThoughtBuilder creates a new CoT builder
func NewChainOfThoughtBuilder() *ChainOfThoughtBuilder {
	return &ChainOfThoughtBuilder{
		steps: make([]CoTStep, 0),
	}
}

// AddStep adds a reasoning step
func (cot *ChainOfThoughtBuilder) AddStep(number int, title string, instructions []string, question string) *ChainOfThoughtBuilder {
	cot.steps = append(cot.steps, CoTStep{
		Number:       number,
		Title:        title,
		Instructions: instructions,
		Question:     question,
	})
	return cot
}

// Build builds the chain-of-thought prompt
func (cot *ChainOfThoughtBuilder) Build(finalAnswerFormat string) string {
	var builder strings.Builder

	builder.WriteString("You are analyzing a problem. Think step by step:\n\n")

	for _, step := range cot.steps {
		fmt.Fprintf(&builder, "Step %d: %s\n", step.Number, step.Title)
		for _, instruction := range step.Instructions {
			fmt.Fprintf(&builder, "- %s\n", instruction)
		}
		if step.Question != "" {
			fmt.Fprintf(&builder, "%s [answer]\n", step.Question)
		}
		builder.WriteString("\n")
	}

	if finalAnswerFormat != "" {
		fmt.Fprintf(&builder, "Final Answer (%s):\n", finalAnswerFormat)
	}

	return builder.String()
}

// ErrorDiagnosisCoT returns a chain-of-thought template for error diagnosis
func ErrorDiagnosisCoT() string {
	cot := NewChainOfThoughtBuilder()
	cot.AddStep(1, "Identify the Error Type", []string{
		"Look at the exit code",
		"Scan error messages",
		"Check stack traces",
	}, "What type of error is this?")

	cot.AddStep(2, "Determine Root Cause", []string{
		"What dependency is missing?",
		"What configuration is wrong?",
		"What prerequisite wasn't met?",
	}, "Root cause:")

	cot.AddStep(3, "Assess Severity", []string{
		"Can this be fixed automatically?",
		"What's the risk level?",
		"Will it require user intervention?",
	}, "Severity assessment:")

	cot.AddStep(4, "Generate Solution", []string{
		"What commands will fix this?",
		"What's the order of operations?",
		"What are the rollback steps?",
	}, "Solution:")

	return cot.Build("JSON")
}

// ProjectAnalysisCoT returns a chain-of-thought template for project analysis
func ProjectAnalysisCoT() string {
	cot := NewChainOfThoughtBuilder()
	cot.AddStep(1, "Detect Project Type", []string{
		"Examine file structure",
		"Look for configuration files (package.json, requirements.txt, go.mod, etc.)",
		"Check for language-specific patterns",
	}, "What type of project is this?")

	cot.AddStep(2, "Identify Dependencies", []string{
		"List all dependencies",
		"Check package manager files",
		"Identify system requirements",
	}, "What dependencies are needed?")

	cot.AddStep(3, "Determine Setup Steps", []string{
		"What commands install dependencies?",
		"What commands build the project?",
		"What commands run the project?",
	}, "What are the setup steps?")

	return cot.Build("JSON")
}

// SecurityScanCoT returns a chain-of-thought template for security scanning
func SecurityScanCoT() string {
	cot := NewChainOfThoughtBuilder()
	cot.AddStep(1, "Scan for Malware", []string{
		"Check for suspicious file patterns",
		"Look for obfuscated code",
		"Examine executable files",
	}, "Are there any malware indicators?")

	cot.AddStep(2, "Check for Credentials", []string{
		"Scan for API keys",
		"Look for passwords",
		"Check for tokens",
	}, "Are credentials exposed?")

	cot.AddStep(3, "Audit Dependencies", []string{
		"Check for known vulnerabilities",
		"Review dependency versions",
		"Assess security risks",
	}, "Are there security vulnerabilities?")

	return cot.Build("JSON")
}

// BuildRecoveryPlanCoT builds a CoT prompt for recovery planning
func BuildRecoveryPlanCoT(errorMessage, command, output string) string {
	cot := NewChainOfThoughtBuilder()
	cot.AddStep(1, "Identify the Error Type", []string{
		"Look at the exit code",
		"Scan error messages",
		"Check stack traces",
	}, fmt.Sprintf("What type of error is this?\nError: %s", errorMessage))

	cot.AddStep(2, "Determine Root Cause", []string{
		"What dependency is missing?",
		"What configuration is wrong?",
		"What prerequisite wasn't met?",
	}, fmt.Sprintf("Root cause:\nCommand: %s\nOutput: %s", command, output))

	cot.AddStep(3, "Assess Severity", []string{
		"Can this be fixed automatically?",
		"What's the risk level?",
		"Will it require user intervention?",
	}, "Severity assessment:")

	cot.AddStep(4, "Generate Solution", []string{
		"What commands will fix this?",
		"What's the order of operations?",
		"What are the rollback steps?",
	}, "Solution:")

	prompt := cot.Build("JSON")
	prompt += `{
  "error_type": "one of the allowed enum values",
  "root_cause": "clear explanation here",
  "confidence": 0.85,
  "recovery_steps": [
    {
      "command": "command to execute",
      "reason": "why this fixes it",
      "risk_level": "low|medium|high|critical",
      "can_rollback": true,
      "rollback_command": "command to undo"
    }
  ],
  "manual_steps_if_failed": ["step 1", "step 2"],
  "explanation": "Clear explanation for user"
}`

	return prompt
}
