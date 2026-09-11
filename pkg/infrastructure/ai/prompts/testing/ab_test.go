package testing

import (
	"context"
	"fmt"
	"math"

	prompts "github.com/glive/infrastructure/ai/prompts"
)

// ABTest represents an A/B test configuration
type ABTest struct {
	ID         string
	VariantA   string // Template name/ID for variant A
	VariantB   string // Template name/ID for variant B
	TestCases  []TestCase
	Confidence float64 // Statistical confidence threshold (e.g., 0.95)
	MinSamples int     // Minimum samples per variant
}

// ABTestResult represents the result of an A/B test
type ABTestResult struct {
	TestID         string
	VariantA       string
	VariantB       string
	MetricsA       TestMetrics
	MetricsB       TestMetrics
	Winner         string
	Confidence     float64
	StatisticalSig bool
	Improvement    ABTestImprovement
}

// ABTestImprovement shows improvement metrics
type ABTestImprovement struct {
	AccuracyDelta  float64
	TokenReduction float64
	LatencyDelta   float64
	CostReduction  float64
}

// RunABTest runs an A/B test
func RunABTest(ctx context.Context, test *ABTest, runner *TestRunner, templateA, templateB *prompts.PromptTemplate) (*ABTestResult, error) {
	// Create prompt test
	promptTest := &PromptTest{
		ID:        test.ID,
		VariantA:  templateA,
		VariantB:  templateB,
		TestCases: test.TestCases,
	}

	// Run test
	result, err := runner.RunTest(ctx, promptTest)
	if err != nil {
		return nil, fmt.Errorf("failed to run A/B test: %w", err)
	}

	// Check statistical significance
	significant := checkStatisticalSignificance(result.Metrics, test.Confidence)

	// Calculate improvement
	improvement := ABTestImprovement{
		AccuracyDelta:  result.Metrics.AccuracyB - result.Metrics.AccuracyA,
		TokenReduction: float64(result.Metrics.AvgTokensA-result.Metrics.AvgTokensB) / float64(result.Metrics.AvgTokensA) * 100,
		LatencyDelta:   result.Metrics.AvgLatencyB.Seconds() - result.Metrics.AvgLatencyA.Seconds(),
		CostReduction:  (result.Metrics.CostPerRequestA - result.Metrics.CostPerRequestB) / result.Metrics.CostPerRequestA * 100,
	}

	return &ABTestResult{
		TestID:         test.ID,
		VariantA:       test.VariantA,
		VariantB:       test.VariantB,
		MetricsA:       result.Metrics,
		MetricsB:       result.Metrics, // Same metrics, different interpretation
		Winner:         result.Winner,
		Confidence:     test.Confidence,
		StatisticalSig: significant,
		Improvement:    improvement,
	}, nil
}

// checkStatisticalSignificance checks if results are statistically significant
func checkStatisticalSignificance(metrics TestMetrics, confidence float64) bool {
	// Simple t-test approximation
	// For more accuracy, use proper statistical library

	n := float64(metrics.TotalTests)
	if n < 30 {
		return false // Need at least 30 samples
	}

	// Calculate standard error
	seA := math.Sqrt(metrics.AccuracyA * (1 - metrics.AccuracyA) / n)
	seB := math.Sqrt(metrics.AccuracyB * (1 - metrics.AccuracyB) / n)
	seDiff := math.Sqrt(seA*seA + seB*seB)

	// Calculate t-statistic
	tStat := math.Abs(metrics.AccuracyB-metrics.AccuracyA) / seDiff

	// Critical value for 95% confidence (two-tailed)
	criticalValue := 1.96 // Approximate for large n

	return tStat > criticalValue
}

// RecommendVariant recommends which variant to use
func RecommendVariant(result *ABTestResult) string {
	if !result.StatisticalSig {
		return "insufficient_data"
	}

	switch result.Winner {
	case "A":
		return result.VariantA
	case "B":
		return result.VariantB
	}

	// If tie, choose based on other factors
	if result.Improvement.TokenReduction > 10 && result.Improvement.AccuracyDelta > -0.05 {
		return result.VariantB // B uses fewer tokens with similar accuracy
	}

	return result.VariantA // Default to A
}
