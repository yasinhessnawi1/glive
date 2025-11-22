package tui

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// ColorDepth represents the color support level of the terminal
type ColorDepth int

const (
	ColorNone ColorDepth = iota // No color support
	Color16                      // Basic ANSI 16 colors
	Color256                     // Extended 256 colors
	ColorTrue                    // 24-bit true color
)

// TerminalCapabilities represents detected terminal capabilities
type TerminalCapabilities struct {
	Width        int
	Height       int
	ColorSupport ColorDepth
	UnicodeSupport bool
	MouseSupport   bool
	IsTerminal     bool
}

// DetectCapabilities detects terminal capabilities
func DetectCapabilities() *TerminalCapabilities {
	cap := &TerminalCapabilities{
		IsTerminal:     isTerminal(),
		UnicodeSupport: true, // Assume Unicode support by default
		MouseSupport:   false, // Will be enabled explicitly if needed
	}

	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		cap.ColorSupport = ColorNone
		cap.IsTerminal = false
		return cap
	}

	if !cap.IsTerminal {
		cap.ColorSupport = ColorNone
		return cap
	}

	// Try to get terminal size
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		width, height, err := term.GetSize(fd)
		if err == nil && width > 0 && height > 0 {
			cap.Width = width
			cap.Height = height
		} else {
			// Windows PowerShell fallback - use reasonable defaults
			cap.Width = 120
			cap.Height = 30
		}
	} else {
		// Not a terminal, use safe defaults
		cap.Width = 120
		cap.Height = 30
	}

	// Detect color support
	cap.ColorSupport = detectColorSupport()

	// Check Unicode support (basic check)
	cap.UnicodeSupport = checkUnicodeSupport()

	return cap
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// detectColorSupport detects the color depth supported by the terminal
func detectColorSupport() ColorDepth {
	// Check TERM environment variable for hints
	termEnv := os.Getenv("TERM")
	if termEnv == "" {
		return Color16 // Default to basic colors
	}

	termLower := strings.ToLower(termEnv)

	// Check for true color support
	if strings.Contains(termLower, "truecolor") || strings.Contains(termLower, "24bit") {
		return ColorTrue
	}

	// Check for 256 color support
	if strings.Contains(termLower, "256") || strings.Contains(termLower, "xterm") {
		return Color256
	}

	// Check COLORTERM for additional hints
	colorTerm := os.Getenv("COLORTERM")
	if colorTerm != "" {
		if strings.Contains(strings.ToLower(colorTerm), "truecolor") ||
			strings.Contains(strings.ToLower(colorTerm), "24bit") {
			return ColorTrue
		}
	}

	// Default to 16 colors for most terminals
	return Color16
}

// checkUnicodeSupport checks if terminal supports Unicode
func checkUnicodeSupport() bool {
	// Most modern terminals support Unicode
	// Check for known problematic terminals
	termEnv := strings.ToLower(os.Getenv("TERM"))
	
	// Some old terminals don't support Unicode well
	noUnicodeTerms := []string{"linux", "dumb", "vt100", "vt102", "vt220"}
	for _, term := range noUnicodeTerms {
		if strings.Contains(termEnv, term) {
			return false
		}
	}

	return true
}

// SupportsColor returns true if terminal supports colors
func (c *TerminalCapabilities) SupportsColor() bool {
	return c.ColorSupport > ColorNone
}

// SupportsUnicode returns true if terminal supports Unicode
func (c *TerminalCapabilities) SupportsUnicode() bool {
	return c.UnicodeSupport
}

// MinSize returns true if terminal meets minimum size requirements
func (c *TerminalCapabilities) MinSize(minWidth, minHeight int) bool {
	return c.Width >= minWidth && c.Height >= minHeight
}

