package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"

	glerrors "github.com/glive/domain/errors"
)

// Config configures retry behavior
type Config struct {
	MaxAttempts     int           // Maximum number of attempts (including first)
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	Multiplier      float64       // Delay multiplier for exponential backoff
	Jitter          float64       // Random jitter factor (0-1)
	RetryableErrors []string      // Error codes that should be retried
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// NetworkConfig returns config optimized for network operations
func NetworkConfig() Config {
	return Config{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Second,
		MaxDelay:     1 * time.Minute,
		Multiplier:   2.0,
		Jitter:       0.2,
	}
}

// Logger interface for retry logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// Retrier executes operations with retry logic
type Retrier struct {
	config Config
	logger Logger
}

// New creates a new retrier
func New(config Config, logger Logger) *Retrier {
	return &Retrier{
		config: config,
		logger: logger,
	}
}

// Do executes the operation with retries
func (r *Retrier) Do(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute operation
		err := fn()
		if err == nil {
			if attempt > 1 {
				r.logger.Debug("Operation succeeded after retry",
					"operation", operation,
					"attempt", attempt)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.shouldRetry(err) {
			r.logger.Debug("Non-retryable error encountered",
				"operation", operation,
				"error", err)
			return err
		}

		// Don't sleep after last attempt
		if attempt == r.config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt)

		r.logger.Warn("Operation failed, retrying",
			"operation", operation,
			"attempt", attempt,
			"max_attempts", r.config.MaxAttempts,
			"delay", delay,
			"error", err)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return glerrors.Wrap(lastErr, fmt.Sprintf("failed after %d attempts", r.config.MaxAttempts))
}

func (r *Retrier) shouldRetry(err error) bool {
	// Check if error explicitly marked as retryable
	if glerrors.IsRetryable(err) {
		return true
	}

	// Check specific error codes
	var gliveErr *glerrors.GliveError
	if errors.As(err, &gliveErr) {
		for _, code := range r.config.RetryableErrors {
			if gliveErr.Code == code {
				return true
			}
		}
	}

	// Check for common transient errors
	return isTransientError(err)
}

func (r *Retrier) calculateDelay(attempt int) time.Duration {
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt-1))

	// Add jitter
	if r.config.Jitter > 0 {
		jitter := delay * r.config.Jitter * (rand.Float64()*2 - 1)
		delay += jitter
	}

	// Cap at max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

func isTransientError(err error) bool {
	// Network timeouts
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Specific HTTP status codes
	errStr := err.Error()
	transientPatterns := []string{
		"429", // Too Many Requests
		"500", // Internal Server Error
		"502", // Bad Gateway
		"503", // Service Unavailable
		"504", // Gateway Timeout
		"connection refused",
		"connection reset",
		"temporary failure",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

