package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glive/infrastructure/monitoring"
)

// RecoveryKnowledgeBase stores successful recovery patterns
type RecoveryKnowledgeBase struct {
	mu       sync.RWMutex
	patterns map[string]*RecoveryPattern
	storagePath string
}

// RecoveryPattern represents a learned recovery pattern
type RecoveryPattern struct {
	Signature    string
	ErrorSignature string
	Plan         *RecoveryPlan
	SuccessCount int
	FailureCount int
	SuccessRate  float64
	LastUsed     time.Time
	ProjectTypes []string
	AverageTime  time.Duration
}

// NewRecoveryKnowledgeBase creates a knowledge base
func NewRecoveryKnowledgeBase() *RecoveryKnowledgeBase {
	// Default storage path: .glive/recovery-kb.json in current directory or home
	homeDir, _ := os.UserHomeDir()
	storagePath := filepath.Join(homeDir, ".glive", "recovery-kb.json")
	
	// Try current directory first
	if cwd, err := os.Getwd(); err == nil {
		localPath := filepath.Join(cwd, ".glive", "recovery-kb.json")
		if _, err := os.Stat(filepath.Dir(localPath)); err == nil {
			storagePath = localPath
		}
	}

	kb := &RecoveryKnowledgeBase{
		patterns:    make(map[string]*RecoveryPattern),
		storagePath: storagePath,
	}

	// Load existing patterns
	kb.load()

	return kb
}

// FindMatchingPlan finds a matching recovery plan
func (kb *RecoveryKnowledgeBase) FindMatchingPlan(result *monitoring.ExecutionResult) *RecoveryPlan {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	signature := kb.generateSignature(result)

	if pattern, exists := kb.patterns[signature]; exists {
		// Check success rate threshold
		if pattern.SuccessRate >= 0.7 {
			pattern.LastUsed = time.Now()
			return pattern.Plan
		}
	}

	return nil
}

// RecordSuccess records a successful recovery
func (kb *RecoveryKnowledgeBase) RecordSuccess(plan *RecoveryPlan, result *RecoveryResult) {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	signature := kb.generatePlanSignature(plan)

	pattern, exists := kb.patterns[signature]
	if !exists {
		pattern = &RecoveryPattern{
			Signature: signature,
			Plan:      plan,
		}
		kb.patterns[signature] = pattern
	}

	pattern.SuccessCount++
	pattern.LastUsed = time.Now()
	
	// Update average time
	if result.EndTime.After(result.StartTime) {
		duration := result.EndTime.Sub(result.StartTime)
		if pattern.AverageTime == 0 {
			pattern.AverageTime = duration
		} else {
			// Simple moving average
			pattern.AverageTime = (pattern.AverageTime + duration) / 2
		}
	}

	pattern.SuccessRate = float64(pattern.SuccessCount) / float64(pattern.SuccessCount+pattern.FailureCount)

	// Save to disk
	kb.save()
}

// RecordFailure records a failed recovery attempt
func (kb *RecoveryKnowledgeBase) RecordFailure(plan *RecoveryPlan) {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	signature := kb.generatePlanSignature(plan)

	pattern, exists := kb.patterns[signature]
	if !exists {
		return // Don't create pattern for failures
	}

	pattern.FailureCount++
	pattern.SuccessRate = float64(pattern.SuccessCount) / float64(pattern.SuccessCount+pattern.FailureCount)

	// Save to disk
	kb.save()
}

// generateSignature creates a signature from execution result
func (kb *RecoveryKnowledgeBase) generateSignature(result *monitoring.ExecutionResult) string {
	hash := sha256.New()

	// Hash command
	hash.Write([]byte(result.Command))

	// Hash exit code
	hash.Write([]byte(fmt.Sprintf("%d", result.ExitCode)))

	// Hash detected issue categories
	for _, issue := range result.DetectedIssues {
		hash.Write([]byte(string(issue.Category)))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// generatePlanSignature creates a signature from recovery plan
func (kb *RecoveryKnowledgeBase) generatePlanSignature(plan *RecoveryPlan) string {
	hash := sha256.New()

	// Hash error type
	hash.Write([]byte(string(plan.ErrorType)))

	// Hash root cause
	hash.Write([]byte(plan.RootCause))

	// Hash step commands
	for _, step := range plan.Steps {
		hash.Write([]byte(step.Command))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// load loads patterns from disk
func (kb *RecoveryKnowledgeBase) load() {
	data, err := os.ReadFile(kb.storagePath)
	if err != nil {
		return // No existing data
	}

	var patterns map[string]*RecoveryPattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		return // Invalid data, start fresh
	}

	kb.patterns = patterns
}

// save saves patterns to disk
func (kb *RecoveryKnowledgeBase) save() {
	// Ensure directory exists
	dir := filepath.Dir(kb.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return // Can't create directory, skip save
	}

	data, err := json.MarshalIndent(kb.patterns, "", "  ")
	if err != nil {
		return // Can't marshal, skip save
	}

	// Write to temp file first, then rename (atomic write)
	tmpPath := kb.storagePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}

	os.Rename(tmpPath, kb.storagePath)
}


