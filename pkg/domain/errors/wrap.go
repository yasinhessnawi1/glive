package errors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Wrap adds context to an error
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	// If it's already a GliveError, enhance it
	var gliveErr *GliveError
	if errors.As(err, &gliveErr) {
		return &GliveError{
			Code:       gliveErr.Code,
			Message:    message + ": " + gliveErr.Message,
			Category:   gliveErr.Category,
			Severity:   gliveErr.Severity,
			Cause:      gliveErr.Cause,
			Context:    gliveErr.Context,
			Retryable:  gliveErr.Retryable,
			UserAction: gliveErr.UserAction,
		}
	}

	// Wrap standard error
	return fmt.Errorf("%s: %w", message, err)
}

// WrapWithCode wraps an error with a specific error code
func WrapWithCode(err error, code, message string) *GliveError {
	return &GliveError{
		Code:     code,
		Message:  message,
		Category: CategoryInternal,
		Severity: SeverityError,
		Cause:    err,
	}
}

// WrapNetwork wraps a network-related error
func WrapNetwork(err error, operation string) *GliveError {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrNetworkTimeout.WithCause(err).WithContext("operation", operation)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrNetworkTimeout.WithCause(err).WithContext("operation", operation)
	}

	return NewNetworkError("NET_ERROR", fmt.Sprintf("Network error during %s", operation), err).
		WithContext("operation", operation)
}

// WrapGit wraps a git-related error
func WrapGit(err error, operation string, repoURL string) *GliveError {
	if err == nil {
		return nil
	}

	message := fmt.Sprintf("Git %s failed", operation)

	gliveErr := &GliveError{
		Code:     "GIT_" + strings.ToUpper(operation) + "_FAILED",
		Message:  message,
		Category: CategoryGit,
		Severity: SeverityError,
		Cause:    err,
		Context: map[string]string{
			"operation": operation,
			"repo_url":  repoURL,
		},
	}

	// Detect specific git errors
	errStr := err.Error()
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
		gliveErr.Code = "GIT_REPO_NOT_FOUND"
		gliveErr.Message = "Repository not found"
		gliveErr.UserAction = "Verify the repository URL is correct and accessible"
	} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "403") {
		gliveErr.Code = "GIT_AUTH_FAILED"
		gliveErr.Message = "Authentication failed"
		gliveErr.UserAction = "Check your credentials or try a public repository"
	} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "timed out") {
		gliveErr.Retryable = true
		gliveErr.UserAction = "Check your network connection and try again"
	}

	return gliveErr
}

// WrapExecution wraps a command execution error
func WrapExecution(err error, command string, exitCode int) *GliveError {
	if err == nil {
		return nil
	}

	return &GliveError{
		Code:     "EXEC_COMMAND_FAILED",
		Message:  fmt.Sprintf("Command failed with exit code %d", exitCode),
		Category: CategoryExecution,
		Severity: SeverityError,
		Cause:    err,
		Context: map[string]string{
			"command":   truncateCommand(command),
			"exit_code": strconv.Itoa(exitCode),
		},
	}
}

// WrapAI wraps an AI-related error
func WrapAI(err error, operation string) *GliveError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for specific AI errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "timed out") {
		return ErrNetworkTimeout.WithCause(err).WithContext("operation", operation)
	}

	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		return &GliveError{
			Code:       "AI_AUTH_FAILED",
			Message:    "AI API authentication failed",
			Category:   CategoryAI,
			Severity:   SeverityError,
			Cause:      err,
			UserAction: "Check your API key: glive config set api-key YOUR_KEY",
		}
	}

	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return &GliveError{
			Code:      "AI_RATE_LIMIT",
			Message:   "AI API rate limit exceeded",
			Category:  CategoryAI,
			Severity:  SeverityError,
			Cause:     err,
			Retryable: true,
			UserAction: "Wait a few minutes and try again",
		}
	}

	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") {
		return ErrAPIUnavailable.WithCause(err).WithContext("operation", operation)
	}

	return &GliveError{
		Code:     "AI_ERROR",
		Message:  fmt.Sprintf("AI service error during %s", operation),
		Category: CategoryAI,
		Severity: SeverityError,
		Cause:    err,
		Context: map[string]string{
			"operation": operation,
		},
		Retryable: true,
	}
}

// WrapSecurity wraps a security-related error
func WrapSecurity(err error, violation string) *GliveError {
	if err == nil {
		return nil
	}

	return &GliveError{
		Code:      "SEC_VIOLATION",
		Message:   fmt.Sprintf("Security violation: %s", violation),
		Category:  CategorySecurity,
		Severity:  SeverityCritical,
		Cause:     err,
		Retryable: false,
		Context: map[string]string{
			"violation": violation,
		},
	}
}

// WrapSystem wraps a system-related error
func WrapSystem(err error, operation string) *GliveError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for disk full
	if strings.Contains(errStr, "no space") || strings.Contains(errStr, "disk full") {
		return ErrDiskFull.WithCause(err).WithContext("operation", operation)
	}

	return &GliveError{
		Code:     "SYS_ERROR",
		Message:  fmt.Sprintf("System error during %s", operation),
		Category: CategorySystem,
		Severity: SeverityError,
		Cause:    err,
		Context: map[string]string{
			"operation": operation,
		},
	}
}

func truncateCommand(cmd string) string {
	if len(cmd) > 100 {
		return cmd[:100] + "..."
	}
	return cmd
}

