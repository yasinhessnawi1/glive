package logging

import (
	"io"
	"os"
	"strings"
)

// Format represents the log output format
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Config holds logger configuration
type Config struct {
	Level  LogLevel
	Format Format
	Output io.Writer
}

// NewLogger creates a new logger based on the configuration
func NewLogger(config Config) Logger {
	if config.Output == nil {
		config.Output = os.Stdout
	}

	if config.Format == "" {
		config.Format = FormatText
	}

	switch strings.ToLower(string(config.Format)) {
	case "json":
		return NewJSONLogger(config.Output, config.Level)
	default:
		return NewTextLogger(config.Output, config.Level)
	}
}

// ParseLevel parses a log level string
func ParseLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}
