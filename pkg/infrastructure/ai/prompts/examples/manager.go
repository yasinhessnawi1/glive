package examples

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Example represents a few-shot learning example
type Example struct {
	Input     map[string]interface{} `json:"input"`
	Reasoning string                 `json:"reasoning"`
	Output    interface{}            `json:"output"`
	Tags      []string               `json:"tags,omitempty"`
	Score     float64                `json:"score,omitempty"` // Relevance score
}

// ExampleSet represents a collection of examples
type ExampleSet struct {
	Category string    `json:"category"`
	Examples []Example `json:"examples"`
}

// ExampleManager manages few-shot learning examples
type ExampleManager struct {
	examples map[string]*ExampleSet
	basePath string
}

// NewExampleManager creates a new example manager
func NewExampleManager(basePath string) (*ExampleManager, error) {
	em := &ExampleManager{
		examples: make(map[string]*ExampleSet),
		basePath: basePath,
	}

	// Load examples
	if err := em.loadExamples(); err != nil {
		return nil, fmt.Errorf("failed to load examples: %w", err)
	}

	return em, nil
}

// loadExamples loads examples from JSON files
func (em *ExampleManager) loadExamples() error {
	if em.basePath == "" {
		em.basePath = filepath.Join("pkg", "infrastructure", "ai", "prompts", "examples")
	}

	exampleFiles := []string{
		"project_analysis_examples.json",
		"error_recovery_examples.json",
		"security_scan_examples.json",
		"command_generation_examples.json",
	}

	for _, filename := range exampleFiles {
		filePath := filepath.Join(em.basePath, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			// File doesn't exist, that's okay
			continue
		}

		var exampleSet ExampleSet
		if err := json.Unmarshal(data, &exampleSet); err != nil {
			continue
		}

		// Extract category from filename if not set
		if exampleSet.Category == "" {
			exampleSet.Category = strings.TrimSuffix(filename, "_examples.json")
		}

		em.examples[exampleSet.Category] = &exampleSet
	}

	return nil
}

// SelectExamples selects the most relevant examples based on input
func (em *ExampleManager) SelectExamples(category string, input map[string]interface{}, maxExamples int, maxTokens int) ([]Example, error) {
	exampleSet, ok := em.examples[category]
	if !ok {
		return []Example{}, nil
	}

	// Score examples based on relevance
	scored := make([]Example, len(exampleSet.Examples))
	for i, ex := range exampleSet.Examples {
		score := em.scoreExample(ex, input)
		ex.Score = score
		scored[i] = ex
	}

	// Sort by score (highest first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Select top examples within token budget
	selected := make([]Example, 0, maxExamples)
	tokenCount := 0

	for _, ex := range scored {
		exTokens := em.estimateTokens(ex)
		if len(selected) >= maxExamples || tokenCount+exTokens > maxTokens {
			break
		}
		selected = append(selected, ex)
		tokenCount += exTokens
	}

	return selected, nil
}

// scoreExample scores an example based on relevance to input
func (em *ExampleManager) scoreExample(ex Example, input map[string]interface{}) float64 {
	score := 0.0

	// Match input keys
	for key, value := range input {
		if exValue, ok := ex.Input[key]; ok {
			if exValue == value {
				score += 1.0 // Exact match
			} else if strings.Contains(fmt.Sprintf("%v", exValue), fmt.Sprintf("%v", value)) {
				score += 0.5 // Partial match
			}
		}
	}

	// Boost score for tag matches
	for _, tag := range ex.Tags {
		for _, value := range input {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", value)), strings.ToLower(tag)) {
				score += 0.3
			}
		}
	}

	return score
}

// estimateTokens estimates token count for an example
func (em *ExampleManager) estimateTokens(ex Example) int {
	// Simple estimation: ~4 characters per token
	jsonBytes, _ := json.Marshal(ex)
	return len(jsonBytes) / 4
}

// FormatExamples formats examples for inclusion in prompts
func FormatExamples(examples []Example) string {
	if len(examples) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Here are some examples:\n\n")

	for i, ex := range examples {
		builder.WriteString(fmt.Sprintf("Example %d:\n", i+1))
		builder.WriteString(fmt.Sprintf("Input: %v\n", ex.Input))
		if ex.Reasoning != "" {
			builder.WriteString(fmt.Sprintf("Reasoning: %s\n", ex.Reasoning))
		}
		builder.WriteString(fmt.Sprintf("Output: %v\n\n", ex.Output))
	}

	return builder.String()
}


