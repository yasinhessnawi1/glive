package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// JSONLogger implements Logger with JSON output format
type JSONLogger struct {
	writer  io.Writer
	level   LogLevel
	fields  map[string]interface{}
	mu      sync.Mutex
	context context.Context
}

// JSONLogEntry represents a structured log entry in JSON format
type JSONLogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewJSONLogger creates a new JSON logger
func NewJSONLogger(writer io.Writer, level LogLevel) *JSONLogger {
	if writer == nil {
		writer = os.Stdout
	}
	return &JSONLogger{
		writer: writer,
		level:  level,
		fields: make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *JSONLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *JSONLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *JSONLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *JSONLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

// WithFields returns a new logger with additional fields
func (l *JSONLogger) WithFields(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}

	return &JSONLogger{
		writer:  l.writer,
		level:   l.level,
		fields:  newFields,
		context: l.context,
	}
}

// WithContext returns a new logger with context
func (l *JSONLogger) WithContext(ctx context.Context) Logger {
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

	return &JSONLogger{
		writer:  l.writer,
		level:   l.level,
		fields:  newFields,
		context: ctx,
	}
}

// log writes a log entry
func (l *JSONLogger) log(level LogLevel, msg string, fields ...Field) {
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

	entry := JSONLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Message:   msg,
		Fields:    allFields,
	}

	// Remove empty fields map for cleaner output
	if len(entry.Fields) == 0 {
		entry.Fields = nil
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple output if JSON marshaling fails
		l.writer.Write([]byte(`{"error":"failed to marshal log entry"}` + "\n"))
		return
	}

	jsonData = append(jsonData, '\n')
	l.writer.Write(jsonData)
}
