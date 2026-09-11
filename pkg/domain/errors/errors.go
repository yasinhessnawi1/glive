package errors

import (
	"errors"
	"fmt"
)

// Severity indicates how serious an error is
type Severity int

const (
	SeverityWarning Severity = iota // Recoverable, operation may continue
	SeverityError                   // Operation failed, but system is stable
	SeverityCritical                // System may be in unstable state
	SeverityFatal                  // System cannot continue
)

// Category helps route errors to appropriate handlers
type Category int

const (
	CategoryUser Category = iota // User input/action error
	CategoryNetwork              // Network-related error
	CategorySystem               // OS/filesystem error
	CategorySecurity             // Security violation
	CategoryAI                   // AI service error
	CategoryGit                  // Git operation error
	CategoryExecution            // Command execution error
	CategoryConfig               // Configuration error
	CategoryInternal             // Internal bug
)

// GliveError is the base error type for all application errors
type GliveError struct {
	Code       string            // Machine-readable error code
	Message    string            // Human-readable message
	Category   Category          // Error category
	Severity   Severity          // Error severity
	Cause      error             // Underlying cause
	Context    map[string]string // Additional context
	Retryable  bool              // Can this operation be retried?
	UserAction string            // What the user can do
}

func (e *GliveError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *GliveError) Unwrap() error {
	return e.Cause
}

// Is implements error matching
func (e *GliveError) Is(target error) bool {
	if t, ok := target.(*GliveError); ok {
		return e.Code == t.Code
	}
	return false
}

// clone returns a copy of e with its own Context map, so the copy shares no
// mutable state with the original.
//
// This exists because the package-level sentinels below (ErrInvalidURL and its
// siblings) are shared *GliveError values. WithContext and WithCause used to
// mutate the receiver in place, which turned every sentinel into shared mutable
// global state — a data race under concurrency, and a violation of the
// "no package-level mutable state" rule.
func (e *GliveError) clone() *GliveError {
	c := *e
	if e.Context != nil {
		c.Context = make(map[string]string, len(e.Context))
		for k, v := range e.Context {
			c.Context[k] = v
		}
	}
	return &c
}

// WithContext returns a copy of the error with key=value added to its context.
// The receiver is never modified, so this is safe to call on a shared sentinel.
func (e *GliveError) WithContext(key, value string) *GliveError {
	c := e.clone()
	if c.Context == nil {
		c.Context = make(map[string]string, 1)
	}
	c.Context[key] = value
	return c
}

// WithCause returns a copy of the error wrapping err as its cause.
// The receiver is never modified, so this is safe to call on a shared sentinel.
func (e *GliveError) WithCause(err error) *GliveError {
	c := e.clone()
	c.Cause = err
	return c
}

// Builder functions for common error types
func NewUserError(code, message string) *GliveError {
	return &GliveError{
		Code:     code,
		Message:  message,
		Category: CategoryUser,
		Severity: SeverityError,
	}
}

func NewNetworkError(code, message string, cause error) *GliveError {
	return &GliveError{
		Code:      code,
		Message:   message,
		Category:  CategoryNetwork,
		Severity:  SeverityError,
		Cause:     cause,
		Retryable: true,
		UserAction: "Check your internet connection and try again",
	}
}

func NewSecurityError(code, message string) *GliveError {
	return &GliveError{
		Code:      code,
		Message:   message,
		Category:  CategorySecurity,
		Severity:  SeverityCritical,
		Retryable: false,
	}
}

// Predefined errors
var (
	// User errors
	ErrInvalidURL = &GliveError{
		Code:       "USER_INVALID_URL",
		Message:    "Invalid repository URL",
		Category:   CategoryUser,
		Severity:   SeverityError,
		UserAction: "Please provide a valid GitHub URL (e.g., https://github.com/user/repo)",
	}

	ErrMissingAPIKey = &GliveError{
		Code:       "USER_MISSING_API_KEY",
		Message:    "API key not configured",
		Category:   CategoryConfig,
		Severity:   SeverityError,
		UserAction: "Run: glive config set api-key YOUR_API_KEY",
	}

	// Network errors
	ErrNetworkTimeout = &GliveError{
		Code:       "NET_TIMEOUT",
		Message:    "Request timed out",
		Category:   CategoryNetwork,
		Severity:   SeverityError,
		Retryable:  true,
		UserAction: "Check your internet connection and try again",
	}

	ErrAPIUnavailable = &GliveError{
		Code:       "NET_API_UNAVAILABLE",
		Message:    "AI service is temporarily unavailable",
		Category:   CategoryNetwork,
		Severity:   SeverityError,
		Retryable:  true,
		UserAction: "The AI service may be experiencing issues. Try again in a few minutes",
	}

	// Git errors
	ErrCloneFailed = &GliveError{
		Code:       "GIT_CLONE_FAILED",
		Message:    "Failed to clone repository",
		Category:   CategoryGit,
		Severity:   SeverityError,
		UserAction: "Verify the repository URL and your network connection",
	}

	ErrRepoNotFound = &GliveError{
		Code:       "GIT_REPO_NOT_FOUND",
		Message:    "Repository not found",
		Category:   CategoryGit,
		Severity:   SeverityError,
		UserAction: "Check if the repository exists and is accessible",
	}

	// Execution errors
	ErrCommandFailed = &GliveError{
		Code:     "EXEC_COMMAND_FAILED",
		Message:  "Command execution failed",
		Category: CategoryExecution,
		Severity: SeverityError,
	}

	ErrCommandTimeout = &GliveError{
		Code:      "EXEC_TIMEOUT",
		Message:   "Command timed out",
		Category:  CategoryExecution,
		Severity:  SeverityError,
		Retryable: true,
	}

	// Security errors
	ErrSecurityViolation = &GliveError{
		Code:      "SEC_VIOLATION",
		Message:   "Security policy violation",
		Category:  CategorySecurity,
		Severity:  SeverityCritical,
		Retryable: false,
	}

	ErrSuspiciousRepository = &GliveError{
		Code:       "SEC_SUSPICIOUS_REPO",
		Message:    "Repository contains suspicious content",
		Category:   CategorySecurity,
		Severity:   SeverityCritical,
		Retryable:  false,
		UserAction: "Review the security findings before proceeding",
	}

	// AI errors
	ErrAIResponseInvalid = &GliveError{
		Code:      "AI_INVALID_RESPONSE",
		Message:   "AI returned an invalid response",
		Category:  CategoryAI,
		Severity:  SeverityError,
		Retryable: true,
	}

	// System errors
	ErrDiskFull = &GliveError{
		Code:       "SYS_DISK_FULL",
		Message:    "Insufficient disk space",
		Category:   CategorySystem,
		Severity:   SeverityCritical,
		UserAction: "Free up disk space and try again",
	}

	// Internal errors
	ErrInternalPanic = &GliveError{
		Code:      "INTERNAL_PANIC",
		Message:   "Internal error occurred",
		Category:  CategoryInternal,
		Severity:  SeverityCritical,
		Retryable: false,
	}
)

// Error codes for domain validation
const (
	ErrCodeInvalidInput = "DOMAIN_INVALID_INPUT"
	ErrCodeNotFound     = "DOMAIN_NOT_FOUND"
	ErrCodeConflict     = "DOMAIN_CONFLICT"
	ErrCodeInvalidState = "DOMAIN_INVALID_STATE"
)

// NewDomainError creates a domain-level validation error
func NewDomainError(code string, message string) error {
	return &GliveError{
		Code:     code,
		Message:  message,
		Category: CategoryUser,
		Severity: SeverityError,
	}
}

// Helper functions
func IsRetryable(err error) bool {
	var gliveErr *GliveError
	if errors.As(err, &gliveErr) {
		return gliveErr.Retryable
	}
	return false
}

func GetCategory(err error) Category {
	var gliveErr *GliveError
	if errors.As(err, &gliveErr) {
		return gliveErr.Category
	}
	return CategoryInternal
}

func GetUserAction(err error) string {
	var gliveErr *GliveError
	if errors.As(err, &gliveErr) && gliveErr.UserAction != "" {
		return gliveErr.UserAction
	}
	return "Please try again or check the logs for more details"
}

func GetSeverity(err error) Severity {
	var gliveErr *GliveError
	if errors.As(err, &gliveErr) {
		return gliveErr.Severity
	}
	return SeverityError
}
