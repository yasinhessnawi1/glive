package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// AIExecutionMonitor monitors command execution with AI
type AIExecutionMonitor struct {
	outputAnalyzer  *MonitorOutputAnalyzer
	patternDetector *MonitorPatternDetector
	recoveryAgent   *MonitorRecoveryAgent
	monitoringPlan  *MonitoringPlan
	client          *Client
}

// MonitorOutputAnalyzer analyzes command output with AI
type MonitorOutputAnalyzer struct {
	client *Client
}

// MonitorPatternDetector detects error patterns quickly
type MonitorPatternDetector struct {
	patterns []*regexp.Regexp
}

// MonitorRecoveryAgent triggers recovery actions
type MonitorRecoveryAgent struct {
	client *Client
}

// MonitorExecution represents a command execution being monitored
type MonitorExecution struct {
	CommandID     string
	Command       string
	OutputChannel chan MonitorOutputLine
	Status        MonitorExecutionStatus
}

// MonitorOutputLine represents a line of output
type MonitorOutputLine struct {
	Line      string
	Timestamp int64
	Source    string // stdout, stderr
}

// MonitorExecutionStatus represents execution status
type MonitorExecutionStatus string

const (
	MonitorStatusRunning    MonitorExecutionStatus = "running"
	MonitorStatusSuccess    MonitorExecutionStatus = "success"
	MonitorStatusFailed     MonitorExecutionStatus = "failed"
	MonitorStatusRecovering MonitorExecutionStatus = "recovering"
)

// AnalysisResult contains AI analysis of output
type AnalysisResult struct {
	RequiresIntervention bool     `json:"requires_intervention"`
	Severity             string   `json:"severity"` // high, medium, low
	Issue                string   `json:"issue"`
	Recommendation       string   `json:"recommendation"`
	CanAutoRecover       bool     `json:"can_auto_recover"`
	RecoveryAction       string   `json:"recovery_action"`
}

// NewAIExecutionMonitor creates a new AI execution monitor
func NewAIExecutionMonitor(client *Client, monitoringPlan *MonitoringPlan) *AIExecutionMonitor {
	return &AIExecutionMonitor{
		outputAnalyzer:  NewMonitorOutputAnalyzer(client),
		patternDetector: NewMonitorPatternDetector(monitoringPlan),
		recoveryAgent:   NewMonitorRecoveryAgent(client),
		monitoringPlan:  monitoringPlan,
		client:          client,
	}
}

// NewMonitorOutputAnalyzer creates a new output analyzer
func NewMonitorOutputAnalyzer(client *Client) *MonitorOutputAnalyzer {
	return &MonitorOutputAnalyzer{
		client: client,
	}
}

// NewMonitorPatternDetector creates a new pattern detector
func NewMonitorPatternDetector(plan *MonitoringPlan) *MonitorPatternDetector {
	patterns := make([]*regexp.Regexp, 0)

	if plan != nil {
		for _, pattern := range plan.WatchFor {
			re, err := regexp.Compile(pattern)
			if err == nil {
				patterns = append(patterns, re)
			}
		}
	}

	return &MonitorPatternDetector{
		patterns: patterns,
	}
}

// NewMonitorRecoveryAgent creates a new monitor recovery agent
func NewMonitorRecoveryAgent(client *Client) *MonitorRecoveryAgent {
	return &MonitorRecoveryAgent{
		client: client,
	}
}

// Monitor monitors command execution with AI
func (m *AIExecutionMonitor) Monitor(ctx context.Context, execution *MonitorExecution) error {
	// Stream output to AI for real-time analysis
	for output := range execution.OutputChannel {
		// Pattern matching (fast path)
		if patterns := m.patternDetector.Detect(output.Line); len(patterns) > 0 {
			m.handleDetectedPatterns(ctx, patterns, output, execution)
		}

		// AI analysis (for critical decisions)
		if m.shouldInvokeAI(output, execution) {
			analysis, err := m.outputAnalyzer.Analyze(ctx, output, m.monitoringPlan)
			if err == nil && analysis.RequiresIntervention {
				m.triggerRecovery(ctx, analysis, execution)
			}
		}
	}

	return nil
}

// Detect detects error patterns in output
func (pd *MonitorPatternDetector) Detect(line string) []string {
	detected := make([]string, 0)

	for _, pattern := range pd.patterns {
		if pattern.MatchString(line) {
			detected = append(detected, pattern.String())
		}
	}

	return detected
}

// Analyze analyzes output with AI
func (oa *MonitorOutputAnalyzer) Analyze(ctx context.Context, output MonitorOutputLine, plan *MonitoringPlan) (*AnalysisResult, error) {
	if oa.client == nil {
		return nil, fmt.Errorf("AI client not available")
	}

	// Build prompt
	prompt := oa.buildAnalysisPrompt(output, plan)

	messages := []Message{
		{
			Role:    "system",
			Content: "You are monitoring command execution output. Analyze errors and determine if intervention is needed.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := oa.client.Chat(messages)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse response
	result, err := oa.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	return result, nil
}

// buildAnalysisPrompt creates the analysis prompt
func (oa *MonitorOutputAnalyzer) buildAnalysisPrompt(output MonitorOutputLine, plan *MonitoringPlan) string {
	watchFor := "None"
	if plan != nil && len(plan.WatchFor) > 0 {
		watchFor = strings.Join(plan.WatchFor, ", ")
	}

	return fmt.Sprintf(`Analyze this command output line and determine if intervention is needed.

Output Line: %s
Source: %s

Watch For Patterns: %s

Respond with JSON:
{
  "requires_intervention": true/false,
  "severity": "high|medium|low",
  "issue": "Description of the issue",
  "recommendation": "What should be done",
  "can_auto_recover": true/false,
  "recovery_action": "Action to take if auto-recovery is possible"
}

Rules:
- Only require intervention for critical errors
- Set can_auto_recover to true only if you're confident in the fix
- Be conservative - don't over-react to warnings

Provide ONLY valid JSON.`, output.Line, output.Source, watchFor)
}

// parseAnalysisResponse parses AI response
func (oa *MonitorOutputAnalyzer) parseAnalysisResponse(response string) (*AnalysisResult, error) {
	// Extract JSON
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no valid JSON found")
	}

	jsonStr := response[start : end+1]

	var result AnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

// shouldInvokeAI determines if AI should be invoked for this output
func (m *AIExecutionMonitor) shouldInvokeAI(output MonitorOutputLine, execution *MonitorExecution) bool {
	// Invoke AI for:
	// 1. Error output (stderr)
	// 2. Lines matching critical patterns
	// 3. Every 10th line for general monitoring (if enabled)

	if output.Source == "stderr" {
		return true
	}

	// Check if line matches critical patterns
	if m.monitoringPlan != nil {
		for _, pattern := range m.monitoringPlan.WatchFor {
			if strings.Contains(strings.ToLower(output.Line), strings.ToLower(pattern)) {
				return true
			}
		}
	}

	return false
}

// handleDetectedPatterns handles detected error patterns
func (m *AIExecutionMonitor) handleDetectedPatterns(ctx context.Context, patterns []string, output MonitorOutputLine, execution *MonitorExecution) {
	// Log detected patterns
	// Check recovery rules
	if m.monitoringPlan != nil {
		for _, rule := range m.monitoringPlan.RecoveryRules {
			for _, pattern := range patterns {
				if strings.Contains(pattern, rule.Pattern) {
					// Trigger recovery based on rule
					if rule.Action == "retry" && rule.Severity == "high" {
						execution.Status = MonitorStatusRecovering
					}
				}
			}
		}
	}
}

// triggerRecovery triggers recovery action
func (m *AIExecutionMonitor) triggerRecovery(ctx context.Context, analysis *AnalysisResult, execution *MonitorExecution) {
	if analysis.CanAutoRecover && m.recoveryAgent != nil {
		// Execute recovery action
		execution.Status = MonitorStatusRecovering
		// TODO: Implement actual recovery logic
	}
}

