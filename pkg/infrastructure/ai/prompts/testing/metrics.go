package testing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// QualityScorer scores the quality of AI responses
type QualityScorer struct {
}

// NewQualityScorer creates a new quality scorer
func NewQualityScorer() *QualityScorer {
	return &QualityScorer{}
}

// Score scores a response against expected output
func (qs *QualityScorer) Score(actual, expected interface{}) float64 {
	score := 0.0

	// Correctness (0-50 points)
	if qs.correctnessMatch(actual, expected) {
		score += 50.0
	} else {
		// Partial correctness
		score += qs.partialCorrectness(actual, expected) * 50.0
	}

	// Completeness (0-20 points)
	score += qs.completenessScore(actual, expected) * 20.0

	// Format compliance (0-15 points)
	if qs.validJSON(actual) && qs.matchesSchema(actual, expected) {
		score += 15.0
	}

	// Confidence calibration (0-15 points)
	score += qs.confidenceCalibrationScore(actual, expected) * 15.0

	return score / 100.0 // Normalize to 0-1
}

// correctnessMatch checks if actual matches expected
func (qs *QualityScorer) correctnessMatch(actual, expected interface{}) bool {
	// Try JSON comparison first
	actualJSON, err1 := json.Marshal(actual)
	expectedJSON, err2 := json.Marshal(expected)

	if err1 == nil && err2 == nil {
		var actualMap, expectedMap map[string]interface{}
		if json.Unmarshal(actualJSON, &actualMap) == nil &&
			json.Unmarshal(expectedJSON, &expectedMap) == nil {
			return reflect.DeepEqual(actualMap, expectedMap)
		}
	}

	// Fallback to string comparison
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

// partialCorrectness calculates partial correctness score
func (qs *QualityScorer) partialCorrectness(actual, expected interface{}) float64 {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	// Simple string similarity
	matches := 0
	expectedWords := strings.Fields(expectedStr)
	for _, word := range expectedWords {
		if strings.Contains(actualStr, word) {
			matches++
		}
	}

	if len(expectedWords) == 0 {
		return 0.0
	}

	return float64(matches) / float64(len(expectedWords))
}

// completenessScore scores completeness
func (qs *QualityScorer) completenessScore(actual, expected interface{}) float64 {
	// Try to extract fields from JSON
	actualMap, ok1 := qs.toMap(actual)
	expectedMap, ok2 := qs.toMap(expected)

	if !ok1 || !ok2 {
		return 0.5 // Default score if can't compare
	}

	// Count how many expected fields are present
	present := 0
	total := 0

	for key := range expectedMap {
		total++
		if _, exists := actualMap[key]; exists {
			present++
		}
	}

	if total == 0 {
		return 1.0
	}

	return float64(present) / float64(total)
}

// validJSON checks if response is valid JSON
func (qs *QualityScorer) validJSON(response interface{}) bool {
	var test interface{}
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return false
	}
	return json.Unmarshal(jsonBytes, &test) == nil
}

// matchesSchema checks if response matches expected schema
func (qs *QualityScorer) matchesSchema(actual, expected interface{}) bool {
	actualMap, ok1 := qs.toMap(actual)
	expectedMap, ok2 := qs.toMap(expected)

	if !ok1 || !ok2 {
		return false
	}

	// Check if actual has same structure as expected
	for key, expectedValue := range expectedMap {
		actualValue, exists := actualMap[key]
		if !exists {
			return false
		}

		// Check type compatibility
		if reflect.TypeOf(actualValue) != reflect.TypeOf(expectedValue) {
			// Allow some flexibility (e.g., int vs float)
			if !qs.typeCompatible(actualValue, expectedValue) {
				return false
			}
		}
	}

	return true
}

// confidenceCalibrationScore scores confidence calibration
func (qs *QualityScorer) confidenceCalibrationScore(actual, expected interface{}) float64 {
	actualMap, ok1 := qs.toMap(actual)
	_, ok2 := qs.toMap(expected)

	if !ok1 || !ok2 {
		return 0.5
	}

	// Check if confidence field exists and is reasonable
	if conf, exists := actualMap["confidence"]; exists {
		confFloat, ok := conf.(float64)
		if ok {
			// If actual matches expected, confidence should be high
			matches := qs.correctnessMatch(actual, expected)
			if matches && confFloat >= 0.8 {
				return 1.0
			} else if !matches && confFloat < 0.5 {
				return 1.0
			} else {
				return 0.5
			}
		}
	}

	return 0.5
}

// toMap converts interface{} to map[string]interface{}
func (qs *QualityScorer) toMap(v interface{}) (map[string]interface{}, bool) {
	switch val := v.(type) {
	case map[string]interface{}:
		return val, true
	case string:
		var m map[string]interface{}
		if json.Unmarshal([]byte(val), &m) == nil {
			return m, true
		}
		return nil, false
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var m map[string]interface{}
		if json.Unmarshal(jsonBytes, &m) == nil {
			return m, true
		}
		return nil, false
	}
}

// typeCompatible checks if two types are compatible
func (qs *QualityScorer) typeCompatible(a, b interface{}) bool {
	typeA := reflect.TypeOf(a)
	typeB := reflect.TypeOf(b)

	if typeA == typeB {
		return true
	}

	// Allow int/float compatibility
	if (typeA.Kind() == reflect.Int || typeA.Kind() == reflect.Float64) &&
		(typeB.Kind() == reflect.Int || typeB.Kind() == reflect.Float64) {
		return true
	}

	return false
}
