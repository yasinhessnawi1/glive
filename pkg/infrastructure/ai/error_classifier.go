package ai

import (
	"regexp"
	"strings"
)

// ErrorCategory represents the type of error encountered
type ErrorCategory string

const (
	ErrorCategoryLinting      ErrorCategory = "linting"
	ErrorCategoryMissingDep   ErrorCategory = "missing_dep"
	ErrorCategoryPortConflict ErrorCategory = "port_conflict"
	ErrorCategoryPermission   ErrorCategory = "permission"
	ErrorCategoryNetwork      ErrorCategory = "network"
	ErrorCategorySyntax       ErrorCategory = "syntax"
	ErrorCategoryBuildFailure ErrorCategory = "build_failure"
	ErrorCategoryRuntime      ErrorCategory = "runtime"
	ErrorCategoryUnknown      ErrorCategory = "unknown"
)

// ErrorContext provides detailed information about an error
type ErrorContext struct {
	Category      ErrorCategory
	Severity      string // "critical", "warning", "info"
	IsAutoFixable bool
	RequiresUser  bool
	Confidence    float64
	Patterns      []string // Matched patterns
}

// ErrorPattern represents a known error pattern
type ErrorPattern struct {
	Category     ErrorCategory
	Regex        *regexp.Regexp
	Keywords     []string
	AutoFixable  bool
	RequiresUser bool
	Severity     string
}

// ErrorClassifier classifies errors into categories
type ErrorClassifier struct {
	patterns []ErrorPattern
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		patterns: buildErrorPatterns(),
	}
}

// buildErrorPatterns creates the list of known error patterns
func buildErrorPatterns() []ErrorPattern {
	return []ErrorPattern{
		// Linting errors
		{
			Category:     ErrorCategoryLinting,
			Regex:        regexp.MustCompile(`(?i)(eslint|tslint|lint error|type error.*@typescript-eslint)`),
			Keywords:     []string{"eslint", "tslint", "lint", "@typescript-eslint"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "warning",
		},
		{
			Category:     ErrorCategoryLinting,
			Regex:        regexp.MustCompile(`(?i)(Unexpected any|no-explicit-any|no-unused-vars)`),
			Keywords:     []string{"Unexpected any", "no-explicit-any", "no-unused-vars"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "warning",
		},

		// Port conflicts
		{
			Category:     ErrorCategoryPortConflict,
			Regex:        regexp.MustCompile(`(?i)(EADDRINUSE|port.*already in use|address already in use|bind.*failed)`),
			Keywords:     []string{"EADDRINUSE", "port", "already in use", "bind"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "warning",
		},

		// Missing dependencies
		{
			Category:     ErrorCategoryMissingDep,
			Regex:        regexp.MustCompile(`(?i)(cannot find module|modulenotfounderror|no module named|package.*not found)`),
			Keywords:     []string{"cannot find module", "ModuleNotFoundError", "no module named", "not found"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "critical",
		},
		{
			Category:     ErrorCategoryMissingDep,
			Regex:        regexp.MustCompile(`(?i)(command not found|is not recognized|not found in PATH)`),
			Keywords:     []string{"command not found", "not recognized", "not found in PATH"},
			AutoFixable:  false,
			RequiresUser: true,
			Severity:     "critical",
		},

		// Permission errors
		{
			Category:     ErrorCategoryPermission,
			Regex:        regexp.MustCompile(`(?i)(EACCES|permission denied|access denied|insufficient permissions)`),
			Keywords:     []string{"EACCES", "permission denied", "access denied"},
			AutoFixable:  false,
			RequiresUser: true,
			Severity:     "critical",
		},

		// Network errors
		{
			Category:     ErrorCategoryNetwork,
			Regex:        regexp.MustCompile(`(?i)(ETIMEDOUT|ECONNREFUSED|ENOTFOUND|network error|connection.*failed)`),
			Keywords:     []string{"ETIMEDOUT", "ECONNREFUSED", "ENOTFOUND", "network error"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "warning",
		},

		// Build failures
		{
			Category:     ErrorCategoryBuildFailure,
			Regex:        regexp.MustCompile(`(?i)(build failed|compilation error|Failed to compile|webpack.*error)`),
			Keywords:     []string{"build failed", "compilation error", "Failed to compile"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "critical",
		},
		{
			Category:     ErrorCategoryBuildFailure,
			Regex:        regexp.MustCompile(`(?i)(Could not find a production build|no.*build.*directory)`),
			Keywords:     []string{"production build", "build directory"},
			AutoFixable:  true,
			RequiresUser: false,
			Severity:     "warning",
		},

		// Syntax errors
		{
			Category:     ErrorCategorySyntax,
			Regex:        regexp.MustCompile(`(?i)(SyntaxError|unexpected token|invalid syntax|parse error)`),
			Keywords:     []string{"SyntaxError", "unexpected token", "invalid syntax"},
			AutoFixable:  false,
			RequiresUser: true,
			Severity:     "critical",
		},

		// Runtime errors
		{
			Category:     ErrorCategoryRuntime,
			Regex:        regexp.MustCompile(`(?i)(RuntimeError|NullPointerException|undefined is not|cannot read property)`),
			Keywords:     []string{"RuntimeError", "NullPointerException", "undefined is not"},
			AutoFixable:  false,
			RequiresUser: true,
			Severity:     "critical",
		},
	}
}

// Classify analyzes error output and classifies it
func (ec *ErrorClassifier) Classify(command, output, errorMsg string) *ErrorContext {
	combinedText := output + "\n" + errorMsg

	var matchedPatterns []ErrorPattern
	var matchedKeywords []string

	// Check each pattern
	for _, pattern := range ec.patterns {
		// Check regex match
		if pattern.Regex.MatchString(combinedText) {
			matchedPatterns = append(matchedPatterns, pattern)
			matchedKeywords = append(matchedKeywords, pattern.Keywords...)
			continue
		}

		// Check keyword match
		for _, keyword := range pattern.Keywords {
			if strings.Contains(strings.ToLower(combinedText), strings.ToLower(keyword)) {
				matchedPatterns = append(matchedPatterns, pattern)
				matchedKeywords = append(matchedKeywords, keyword)
				break
			}
		}
	}

	// No patterns matched
	if len(matchedPatterns) == 0 {
		return &ErrorContext{
			Category:      ErrorCategoryUnknown,
			Severity:      "warning",
			IsAutoFixable: false,
			RequiresUser:  true,
			Confidence:    0.3,
			Patterns:      []string{},
		}
	}

	// Use the first matched pattern (most specific)
	primary := matchedPatterns[0]

	// Calculate confidence based on number of matches
	confidence := 0.6
	if len(matchedPatterns) > 1 {
		confidence = 0.8
	}
	if len(matchedPatterns) > 2 {
		confidence = 0.9
	}

	return &ErrorContext{
		Category:      primary.Category,
		Severity:      primary.Severity,
		IsAutoFixable: primary.AutoFixable,
		RequiresUser:  primary.RequiresUser,
		Confidence:    confidence,
		Patterns:      matchedKeywords,
	}
}

// IsLintingError checks if the error is purely linting-related
func (ec *ErrorContext) IsLintingError() bool {
	return ec.Category == ErrorCategoryLinting
}

// IsCritical checks if the error is critical
func (ec *ErrorContext) IsCritical() bool {
	return ec.Severity == "critical"
}

// CanAutoFix checks if the error can be automatically fixed
func (ec *ErrorContext) CanAutoFix() bool {
	return ec.IsAutoFixable && !ec.RequiresUser && ec.Confidence >= 0.6
}
