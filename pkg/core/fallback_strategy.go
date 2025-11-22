package core

import (
	"fmt"
)

// FallbackTrigger represents when to trigger fallback
type FallbackTrigger string

const (
	TriggerLowConfidence   FallbackTrigger = "low_confidence"   // <0.7
	TriggerAIFailure       FallbackTrigger = "ai_failure"       // API error
	TriggerTimeout         FallbackTrigger = "timeout"          // AI too slow
	TriggerInvalidResponse FallbackTrigger = "invalid_response" // Bad JSON
)

// FallbackStrategy defines fallback behavior
type FallbackStrategy struct {
	Trigger    FallbackTrigger
	Strategy   ExecutionStrategy
	Confidence float64
}

// ExecutionStrategy represents how to execute
type ExecutionStrategy string

const (
	StrategyTraditional ExecutionStrategy = "traditional" // Pattern-based detection
	StrategyCached      ExecutionStrategy = "cached"     // Use cached analysis
	StrategyHeuristic   ExecutionStrategy = "heuristic"  // Quick heuristics
)

// FallbackSelector selects appropriate fallback strategy
type FallbackSelector struct {
	cache CacheProvider
}

// CacheProvider interface for accessing cached analysis
type CacheProvider interface {
	Get(projectSignature string) interface{}
}

// NewFallbackSelector creates a new fallback selector
func NewFallbackSelector(cache CacheProvider) *FallbackSelector {
	return &FallbackSelector{
		cache: cache,
	}
}

// SelectFallback selects the appropriate fallback strategy based on trigger
func (fs *FallbackSelector) SelectFallback(trigger FallbackTrigger, projectSignature string) ExecutionStrategy {
	switch trigger {
	case TriggerLowConfidence:
		// Use traditional pattern-based detection
		return StrategyTraditional
	case TriggerAIFailure:
		// Use cached analysis if available
		if fs.cache != nil {
			if cached := fs.cache.Get(projectSignature); cached != nil {
				return StrategyCached
			}
		}
		return StrategyTraditional
	case TriggerTimeout:
		// Use quick heuristics
		return StrategyHeuristic
	case TriggerInvalidResponse:
		// Try cached first, then traditional
		if fs.cache != nil {
			if cached := fs.cache.Get(projectSignature); cached != nil {
				return StrategyCached
			}
		}
		return StrategyTraditional
	default:
		return StrategyTraditional
	}
}

// GetFallbackStrategy returns a fallback strategy for a trigger
func (fs *FallbackSelector) GetFallbackStrategy(trigger FallbackTrigger, projectSignature string) *FallbackStrategy {
	strategy := fs.SelectFallback(trigger, projectSignature)
	
	confidence := 0.5 // Default fallback confidence
	if strategy == StrategyCached {
		confidence = 0.7 // Cached analysis has higher confidence
	}

	return &FallbackStrategy{
		Trigger:    trigger,
		Strategy:   strategy,
		Confidence: confidence,
	}
}

// String returns string representation of trigger
func (ft FallbackTrigger) String() string {
	return string(ft)
}

// String returns string representation of strategy
func (es ExecutionStrategy) String() string {
	return string(es)
}

// ValidateTrigger validates a fallback trigger
func ValidateTrigger(trigger string) (FallbackTrigger, error) {
	switch trigger {
	case "low_confidence":
		return TriggerLowConfidence, nil
	case "ai_failure":
		return TriggerAIFailure, nil
	case "timeout":
		return TriggerTimeout, nil
	case "invalid_response":
		return TriggerInvalidResponse, nil
	default:
		return "", fmt.Errorf("invalid fallback trigger: %s", trigger)
	}
}


