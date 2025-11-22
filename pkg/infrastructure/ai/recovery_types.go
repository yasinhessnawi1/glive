package ai

import (
	"time"

	"github.com/glive/infrastructure/monitoring"
)

// RecoveryPlan contains steps to recover from failure
type RecoveryPlan struct {
	ErrorType       monitoring.ErrorCategory `json:"error_type"`
	RootCause       string                   `json:"root_cause"`
	Confidence      float64                  `json:"confidence"`
	Steps           []RecoveryStep          `json:"recovery_steps"`
	ManualSteps     []string                 `json:"manual_steps_if_failed"`
	Explanation     string                   `json:"explanation"`
	EstimatedTime   string                   `json:"estimated_time"`
	Source          string                   // "ai_generated" or "knowledge_base"
	RequiresApproval bool                    // true for high-risk operations
}

// RecoveryStep is a single step in recovery
type RecoveryStep struct {
	Command     string            `json:"command"`
	Reason      string            `json:"reason"`
	RiskLevel   RiskLevel         `json:"risk_level"`
	CanRollback bool              `json:"can_rollback"`
	RollbackCmd string            `json:"rollback_command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     string            `json:"timeout,omitempty"`
}

// RiskLevel indicates risk of a recovery step
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// RecoveryResult contains the result of recovery execution
type RecoveryResult struct {
	Plan      *RecoveryPlan
	Status    RecoveryStatus
	Message   string
	Steps     []StepResult
	StartTime time.Time
	EndTime   time.Time
}

// RecoveryStatus indicates the status of recovery execution
type RecoveryStatus string

const (
	StatusSuccess   RecoveryStatus = "success"
	StatusFailed    RecoveryStatus = "failed"
	StatusCancelled RecoveryStatus = "cancelled"
)

// StepResult contains the result of a single recovery step
type StepResult struct {
	StepIndex int
	Command   string
	Success   bool
	Output    string
	Error     string
	Duration  time.Duration
}

// RecoveryObserver observes recovery execution
type RecoveryObserver interface {
	RequestApproval(plan *RecoveryPlan) bool
	OnStepStarted(step int, total int, command string)
	OnStepCompleted(step int, total int, result *StepResult)
	OnStepFailed(step int, total int, result *StepResult)
}

// ProjectInfo contains project information for recovery context
type ProjectInfo struct {
	Type        string
	Path        string
	Technologies []string
}


