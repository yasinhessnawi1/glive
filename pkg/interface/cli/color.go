package cli

import (
	"os"
)

// Color represents a terminal color
type Color int

const (
	Red Color = iota
	Green
	Yellow
	Blue
	Magenta
	Cyan
	Gray
)

// Colorize applies color to text if output is a terminal
func Colorize(text string, color Color) string {
	if !isTerminal(os.Stdout) || os.Getenv("NO_COLOR") != "" {
		return text
	}

	codes := map[Color]string{
		Red:     "\033[31m",
		Green:   "\033[32m",
		Yellow:  "\033[33m",
		Blue:    "\033[34m",
		Magenta: "\033[35m",
		Cyan:    "\033[36m",
		Gray:    "\033[90m",
	}

	code, ok := codes[color]
	if !ok {
		return text
	}

	return code + text + "\033[0m"
}

// Success formats text as a success message with green color and checkmark
func Success(text string) string {
	return "✓ " + Colorize(text, Green)
}

// Warning formats text as a warning message with yellow color and warning icon
func Warning(text string) string {
	return "⚠ " + Colorize(text, Yellow)
}

// Error formats text as an error message with red color and cross mark
func Error(text string) string {
	return "✗ " + Colorize(text, Red)
}

// Info formats text as an info message with blue color and info icon
func Info(text string) string {
	return "ℹ " + Colorize(text, Blue)
}

// Dim formats text with gray color for less important output
func Dim(text string) string {
	return Colorize(text, Gray)
}

// Bold makes text bold (for terminals that support it)
func Bold(text string) string {
	if !isTerminal(os.Stdout) || os.Getenv("NO_COLOR") != "" {
		return text
	}
	return "\033[1m" + text + "\033[0m"
}
