package container

import (
	"io"
	"os"

	"github.com/glive/infrastructure/logging"
)

// SimpleLogger is a basic logger implementation that wraps the structured logger
// for backward compatibility
type SimpleLogger struct {
	logger logging.Logger
}

// NewSimpleLogger creates a new simple logger using the structured logging system
func NewSimpleLogger(writer io.Writer) Logger {
	if writer == nil {
		writer = os.Stdout
	}

	// Create structured logger with text format for backward compatibility
	logger := logging.NewTextLogger(writer, logging.LevelInfo)
	return &SimpleLogger{logger: logger}
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string) {
	l.logger.Info(msg)
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, err error) {
	if err != nil {
		l.logger.Error(msg, logging.Error(err))
	} else {
		l.logger.Error(msg)
	}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string) {
	l.logger.Debug(msg)
}
