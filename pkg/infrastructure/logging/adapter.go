package logging

import (
	"context"
)

// Adapter adapts the new structured Logger to a simple logger interface
// This avoids circular dependencies
type Adapter struct {
	logger Logger
}

// SimpleLogger interface matches container.Logger but avoids import cycle
type SimpleLogger interface {
	Info(msg string)
	Error(msg string, err error)
	Debug(msg string)
}

// NewAdapter creates a new logger adapter
func NewAdapter(logger Logger) SimpleLogger {
	return &Adapter{logger: logger}
}

// Info logs an info message
func (a *Adapter) Info(msg string) {
	a.logger.Info(msg)
}

// Error logs an error message
func (a *Adapter) Error(msg string, err error) {
	if err != nil {
		a.logger.Error(msg, Error(err))
	} else {
		a.logger.Error(msg)
	}
}

// Debug logs a debug message
func (a *Adapter) Debug(msg string) {
	a.logger.Debug(msg)
}

// GetLogger returns the underlying structured logger
func (a *Adapter) GetLogger() Logger {
	return a.logger
}

// WithFields returns a new adapter with additional fields
func (a *Adapter) WithFields(fields ...Field) *Adapter {
	return &Adapter{logger: a.logger.WithFields(fields...)}
}

// WithContext returns a new adapter with context
func (a *Adapter) WithContext(ctx context.Context) *Adapter {
	return &Adapter{logger: a.logger.WithContext(ctx)}
}

