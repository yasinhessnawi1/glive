package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LearningSystem learns from execution outcomes
type LearningSystem struct {
	successPatterns *PatternDatabase
	failurePatterns *PatternDatabase
	feedbackStore   *FeedbackStore
	storagePath     string
}

// PatternDatabase stores patterns for matching
type PatternDatabase struct {
	patterns map[string]*PatternEntry
}

// PatternEntry represents a learned pattern
type PatternEntry struct {
	Pattern     string    `json:"pattern"`
	Outcome     string    `json:"outcome"` // success, failure
	Count       int       `json:"count"`
	LastSeen    time.Time `json:"last_seen"`
	Confidence  float64   `json:"confidence"`
}

// FeedbackStore stores feedback from executions
type FeedbackStore struct {
	storagePath string
	feedbacks   []*Feedback
}

// Feedback represents feedback from an execution
type Feedback struct {
	ProjectSignature string                   `json:"project_signature"`
	Analysis         *ComprehensiveAnalysis   `json:"analysis"`
	Result           *ExecutionResult         `json:"result"`
	Recovery         *LearningRecoveryResult  `json:"recovery,omitempty"`
	Outcome          Outcome                  `json:"outcome"`
	Timestamp        time.Time                `json:"timestamp"`
}

// ExecutionResult represents execution result
type ExecutionResult struct {
	Success      bool          `json:"success"`
	CommandsRun  int           `json:"commands_run"`
	CommandsFailed int         `json:"commands_failed"`
	Duration     time.Duration `json:"duration"`
	Error        string        `json:"error,omitempty"`
}

// LearningRecoveryResult represents recovery attempt result for learning
type LearningRecoveryResult struct {
	Attempted   bool   `json:"attempted"`
	Success     bool   `json:"success"`
	Action      string `json:"action"`
	Explanation string `json:"explanation"`
}

// Outcome represents execution outcome
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomePartial Outcome = "partial"
)

// NewLearningSystem creates a new learning system
func NewLearningSystem(storagePath string) (*LearningSystem, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	ls := &LearningSystem{
		successPatterns: &PatternDatabase{
			patterns: make(map[string]*PatternEntry),
		},
		failurePatterns: &PatternDatabase{
			patterns: make(map[string]*PatternEntry),
		},
		feedbackStore: &FeedbackStore{
			storagePath: storagePath,
			feedbacks:   make([]*Feedback, 0),
		},
		storagePath: storagePath,
	}

	// Load existing feedback
	if err := ls.loadFeedback(); err != nil {
		// Log error but continue
		_ = err
	}

	return ls, nil
}

// RecordSuccess records a successful execution
func (ls *LearningSystem) RecordSuccess(analysis *ComprehensiveAnalysis, result *ExecutionResult) {
	feedback := &Feedback{
		ProjectSignature: generateSignature(analysis),
		Analysis:         analysis,
		Result:           result,
		Outcome:          OutcomeSuccess,
		Timestamp:        time.Now(),
	}

	ls.feedbackStore.Save(feedback)
	ls.successPatterns.Add(analysis.ExecutionPlan, result)
}

// RecordFailure records a failed execution
func (ls *LearningSystem) RecordFailure(analysis *ComprehensiveAnalysis, result *ExecutionResult, recovery *LearningRecoveryResult) {
	feedback := &Feedback{
		ProjectSignature: generateSignature(analysis),
		Analysis:         analysis,
		Result:           result,
		Recovery:         recovery,
		Outcome:          OutcomeFailure,
		Timestamp:        time.Now(),
	}

	ls.feedbackStore.Save(feedback)
	ls.failurePatterns.Add(analysis.ExecutionPlan, result)
}

// GetSimilarCases finds similar cases from past executions
func (ls *LearningSystem) GetSimilarCases(projectSig string, limit int) []*Feedback {
	return ls.feedbackStore.FindSimilar(projectSig, limit)
}

// Add adds a pattern to the database
func (pd *PatternDatabase) Add(plan *ExecutionPlan, result *ExecutionResult) {
	if plan == nil {
		return
	}

	// Generate pattern signature from execution plan
	sig := generatePlanSignature(plan)
	
	entry, exists := pd.patterns[sig]
	if !exists {
		entry = &PatternEntry{
			Pattern:    sig,
			Outcome:    "success",
			Count:      0,
			LastSeen:   time.Now(),
			Confidence: 0.5,
		}
		pd.patterns[sig] = entry
	}

	entry.Count++
	entry.LastSeen = time.Now()
	
	// Update confidence based on success rate
	if result.Success {
		entry.Confidence = min(1.0, entry.Confidence+0.1)
	} else {
		entry.Confidence = max(0.0, entry.Confidence-0.1)
	}
}

// Save saves feedback to disk
func (fs *FeedbackStore) Save(feedback *Feedback) {
	fs.feedbacks = append(fs.feedbacks, feedback)
	
	// Persist to disk
	fs.persist()
}

// FindSimilar finds similar feedback entries
func (fs *FeedbackStore) FindSimilar(projectSig string, limit int) []*Feedback {
	similar := make([]*Feedback, 0)
	
	for _, fb := range fs.feedbacks {
		similarity := calculateSimilarity(projectSig, fb.ProjectSignature)
		if similarity > 0.7 { // 70% similarity threshold
			similar = append(similar, fb)
		}
	}
	
	// Sort by similarity and limit
	if len(similar) > limit {
		similar = similar[:limit]
	}
	
	return similar
}

// persist saves feedbacks to disk
func (fs *FeedbackStore) persist() error {
	filePath := filepath.Join(fs.storagePath, "feedback.json")
	
	data, err := json.MarshalIndent(fs.feedbacks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal feedback: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write feedback: %w", err)
	}
	
	return nil
}

// loadFeedback loads feedback from disk
func (ls *LearningSystem) loadFeedback() error {
	filePath := filepath.Join(ls.storagePath, "feedback.json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's okay
		}
		return fmt.Errorf("failed to read feedback: %w", err)
	}
	
	if err := json.Unmarshal(data, &ls.feedbackStore.feedbacks); err != nil {
		return fmt.Errorf("failed to unmarshal feedback: %w", err)
	}
	
	return nil
}

// generateSignature generates a signature for a project
func generateSignature(analysis *ComprehensiveAnalysis) string {
	hash := sha256.New()
	
	hash.Write([]byte(analysis.ProjectType))
	hash.Write([]byte(analysis.Framework))
	hash.Write([]byte(analysis.BuildSystem))
	hash.Write([]byte(analysis.PackageManager))
	
	for _, tech := range analysis.Technologies {
		hash.Write([]byte(tech.Name))
		hash.Write([]byte(tech.Version))
	}
	
	return hex.EncodeToString(hash.Sum(nil))
}

// generatePlanSignature generates a signature for an execution plan
func generatePlanSignature(plan *ExecutionPlan) string {
	hash := sha256.New()
	
	for _, phase := range plan.Phases {
		hash.Write([]byte(phase.Name))
		for _, cmd := range phase.Commands {
			hash.Write([]byte(cmd.Command))
		}
	}
	
	return hex.EncodeToString(hash.Sum(nil))
}

// calculateSimilarity calculates similarity between two signatures
func calculateSimilarity(sig1, sig2 string) float64 {
	if sig1 == sig2 {
		return 1.0
	}
	
	// Simple similarity based on common prefix
	// In a real implementation, use more sophisticated algorithms
	common := 0
	minLen := len(sig1)
	if len(sig2) < minLen {
		minLen = len(sig2)
	}
	
	for i := 0; i < minLen; i++ {
		if sig1[i] == sig2[i] {
			common++
		} else {
			break
		}
	}
	
	if minLen == 0 {
		return 0.0
	}
	
	return float64(common) / float64(minLen)
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}


