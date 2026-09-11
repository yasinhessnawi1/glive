package recovery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/glive/domain/errors"
)

// Logger interface for panic logging
type Logger interface {
	Error(msg string, args ...interface{})
}

var defaultLogger Logger

// SetLogger sets the default logger for panic recovery
func SetLogger(logger Logger) {
	defaultLogger = logger
}

// Recover wraps a function to catch panics
func Recover(operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = &errors.GliveError{
				Code:     "INTERNAL_PANIC",
				Message:  fmt.Sprintf("Internal error in %s", operation),
				Category: errors.CategoryInternal,
				Severity: errors.SeverityCritical,
				Context: map[string]string{
					"panic":     fmt.Sprintf("%v", r),
					"operation": operation,
				},
			}

			// Log full stack trace
			if defaultLogger != nil {
				defaultLogger.Error("Panic recovered",
					"operation", operation,
					"panic", r,
					"stack", string(stack))
			}
		}
	}()

	return fn()
}

// RecoverMiddleware is HTTP middleware for panic recovery
func RecoverMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()

					if logger != nil {
						logger.Error("HTTP handler panic",
							"path", r.URL.Path,
							"method", r.Method,
							"panic", rec,
							"stack", string(stack))
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Internal server error",
						"code":  "INTERNAL_ERROR",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// SafeGo runs a goroutine with panic recovery
func SafeGo(name string, logger Logger, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				if logger != nil {
					logger.Error("Goroutine panic",
						"name", name,
						"panic", r,
						"stack", string(stack))
				}
			}
		}()

		fn()
	}()
}
