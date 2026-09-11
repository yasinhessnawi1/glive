package testing

import (
	"context"
	"fmt"
	"time"

	prompts "github.com/glive/infrastructure/ai/prompts"
)

// TestCase represents a test case for prompt testing
type TestCase struct {
	ID          string
	Input       map[string]interface{}
	Expected    interface{}
	Description string
}

// PromptTest represents a prompt test configuration
type PromptTest struct {
	ID        string
	VariantA  *prompts.PromptTemplate
	VariantB  *prompts.PromptTemplate
	TestCases []TestCase
	Metrics   TestMetrics
}

// TestMetrics contains metrics for a test
type TestMetrics struct {
	AccuracyA       float64
	AccuracyB       float64
	AvgTokensA      int
	AvgTokensB      int
	AvgLatencyA     time.Duration
	AvgLatencyB     time.Duration
	CostPerRequestA float64
	CostPerRequestB float64
	TotalTests      int
	PassedA         int
	PassedB         int
}

// TestRunner runs prompt tests
type TestRunner struct {
	executor TestExecutor
	scorer   *QualityScorer
}

// TestExecutor executes prompts and returns responses
type TestExecutor interface {
	Execute(ctx context.Context, template *prompts.PromptTemplate, vars map[string]interface{}) (string, time.Duration, int, error)
}

// NewTestRunner creates a new test runner
func NewTestRunner(executor TestExecutor) *TestRunner {
	return &TestRunner{
		executor: executor,
		scorer:   NewQualityScorer(),
	}
}

// RunTest runs a prompt test
func (tr *TestRunner) RunTest(ctx context.Context, test *PromptTest) (*TestResult, error) {
	resultsA := make([]TestResultItem, 0, len(test.TestCases))
	resultsB := make([]TestResultItem, 0, len(test.TestCases))

	// Run variant A
	for _, tc := range test.TestCases {
		result, err := tr.runTestCase(ctx, test.VariantA, tc, "A")
		if err != nil {
			return nil, fmt.Errorf("variant A failed: %w", err)
		}
		resultsA = append(resultsA, result)
	}

	// Run variant B
	for _, tc := range test.TestCases {
		result, err := tr.runTestCase(ctx, test.VariantB, tc, "B")
		if err != nil {
			return nil, fmt.Errorf("variant B failed: %w", err)
		}
		resultsB = append(resultsB, result)
	}

	// Calculate metrics
	metrics := tr.calculateMetrics(resultsA, resultsB, test)

	// Determine winner
	winner := tr.determineWinner(metrics)

	return &TestResult{
		TestID:   test.ID,
		VariantA: resultsA,
		VariantB: resultsB,
		Metrics:  metrics,
		Winner:   winner,
	}, nil
}

// runTestCase runs a single test case
func (tr *TestRunner) runTestCase(ctx context.Context, template *prompts.PromptTemplate, tc TestCase, variant string) (TestResultItem, error) {
	response, latency, tokens, err := tr.executor.Execute(ctx, template, tc.Input)
	if err != nil {
		return TestResultItem{}, err
	}

	score := tr.scorer.Score(response, tc.Expected)

	return TestResultItem{
		TestCaseID: tc.ID,
		Variant:    variant,
		Input:      tc.Input,
		Expected:   tc.Expected,
		Actual:     response,
		Score:      score,
		Latency:    latency,
		Tokens:     tokens,
	}, nil
}

// calculateMetrics calculates test metrics
func (tr *TestRunner) calculateMetrics(resultsA, resultsB []TestResultItem, test *PromptTest) TestMetrics {
	metrics := TestMetrics{
		TotalTests: len(test.TestCases),
	}

	// Calculate accuracy
	totalScoreA := 0.0
	totalScoreB := 0.0
	totalTokensA := 0
	totalTokensB := 0
	totalLatencyA := time.Duration(0)
	totalLatencyB := time.Duration(0)
	passedA := 0
	passedB := 0

	for i, resultA := range resultsA {
		resultB := resultsB[i]

		totalScoreA += resultA.Score
		totalScoreB += resultB.Score
		totalTokensA += resultA.Tokens
		totalTokensB += resultB.Tokens
		totalLatencyA += resultA.Latency
		totalLatencyB += resultB.Latency

		if resultA.Score >= 0.8 {
			passedA++
		}
		if resultB.Score >= 0.8 {
			passedB++
		}
	}

	metrics.AccuracyA = totalScoreA / float64(len(resultsA))
	metrics.AccuracyB = totalScoreB / float64(len(resultsB))
	metrics.AvgTokensA = totalTokensA / len(resultsA)
	metrics.AvgTokensB = totalTokensB / len(resultsB)
	metrics.AvgLatencyA = totalLatencyA / time.Duration(len(resultsA))
	metrics.AvgLatencyB = totalLatencyB / time.Duration(len(resultsB))
	metrics.PassedA = passedA
	metrics.PassedB = passedB

	// Estimate cost (example: $0.002 per 1K tokens)
	metrics.CostPerRequestA = float64(metrics.AvgTokensA) / 1000.0 * 0.002
	metrics.CostPerRequestB = float64(metrics.AvgTokensB) / 1000.0 * 0.002

	return metrics
}

// determineWinner determines which variant won
func (tr *TestRunner) determineWinner(metrics TestMetrics) string {
	// Consider accuracy, tokens, and latency
	scoreA := metrics.AccuracyA*0.6 - float64(metrics.AvgTokensA)/10000.0*0.2 - metrics.AvgLatencyA.Seconds()*0.2
	scoreB := metrics.AccuracyB*0.6 - float64(metrics.AvgTokensB)/10000.0*0.2 - metrics.AvgLatencyB.Seconds()*0.2

	if scoreA > scoreB {
		return "A"
	} else if scoreB > scoreA {
		return "B"
	}
	return "tie"
}

// TestResult represents the result of a test
type TestResult struct {
	TestID   string
	VariantA []TestResultItem
	VariantB []TestResultItem
	Metrics  TestMetrics
	Winner   string
}

// TestResultItem represents a single test result
type TestResultItem struct {
	TestCaseID string
	Variant    string
	Input      map[string]interface{}
	Expected   interface{}
	Actual     interface{}
	Score      float64
	Latency    time.Duration
	Tokens     int
}
