package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// TextLogger implements Logger with human-readable text output format
type TextLogger struct {
	writer  io.Writer
	level   LogLevel
	fields  map[string]interface{}
	mu      sync.Mutex
	context context.Context
}

// NewTextLogger creates a new text logger
func NewTextLogger(writer io.Writer, level LogLevel) *TextLogger {
	if writer == nil {
		writer = os.Stdout
	}
	return &TextLogger{
		writer: writer,
		level:  level,
		fields: make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *TextLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *TextLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *TextLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *TextLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

// WithFields returns a new logger with additional fields
func (l *TextLogger) WithFields(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}

	return &TextLogger{
		writer:  l.writer,
		level:   l.level,
		fields:  newFields,
		context: l.context,
	}
}

// WithContext returns a new logger with context
func (l *TextLogger) WithContext(ctx context.Context) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Extract trace context if available
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}

	// Try to extract trace ID from context
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		newFields["trace_id"] = traceID
	}
	if spanID := getSpanIDFromContext(ctx); spanID != "" {
		newFields["span_id"] = spanID
	}

	return &TextLogger{
		writer:  l.writer,
		level:   l.level,
		fields:  newFields,
		context: ctx,
	}
}

// log writes a log entry in human-readable format
func (l *TextLogger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Merge all fields
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	for _, field := range fields {
		allFields[field.Key] = field.Value
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	levelStr := level.String()

	// Build log line
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", timestamp))
	parts = append(parts, fmt.Sprintf("[%s]", levelStr))
	parts = append(parts, msg)

	// Add fields if present
	if len(allFields) > 0 {
		var fieldParts []string
		for k, v := range allFields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		if len(fieldParts) > 0 {
			parts = append(parts, strings.Join(fieldParts, " "))
		}
	}

	line := strings.Join(parts, " ") + "\n"
	l.writer.Write([]byte(line))
}

