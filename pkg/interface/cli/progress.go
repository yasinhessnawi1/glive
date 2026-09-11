package cli

import (
	"fmt"
	"os"
	"strings"
)

// ProgressBar displays a progress bar
type ProgressBar struct {
	total   int
	current int
	width   int
	label   string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, label string) *ProgressBar {
	width := 40 // Default width
	if isTerminal(os.Stdout) {
		// Try to get terminal width, fallback to default
		if w := getTerminalWidth(); w > 0 {
			// Reserve space for label and percentage
			width = w - 30
			if width < 20 {
				width = 20
			}
		}
	}
	return &ProgressBar{
		total:   total,
		current: 0,
		width:   width,
		label:   label,
	}
}

// Update updates the current progress
func (p *ProgressBar) Update(current int) {
	if current < 0 {
		current = 0
	}
	if current > p.total {
		current = p.total
	}
	p.current = current
}

// Render renders the progress bar as a string
func (p *ProgressBar) Render() string {
	if p.total == 0 {
		return fmt.Sprintf("\r%s [%s] %3.0f%%", p.label, strings.Repeat("░", p.width), 0.0)
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	if filled > p.width {
		filled = p.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	return fmt.Sprintf("\r%s [%s] %3.0f%%", p.label, bar, percent*100)
}

// Spinner displays an animated spinner
type Spinner struct {
	frames  []string
	current int
	label   string
}

// DefaultFrames contains the default spinner frames (braille patterns)
var DefaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner
func NewSpinner(label string) *Spinner {
	return &Spinner{
		frames:  DefaultFrames,
		current: 0,
		label:   label,
	}
}

// Tick advances the spinner and returns the current frame
func (s *Spinner) Tick() string {
	if !isTerminal(os.Stdout) {
		// For non-terminals, just return the label without animation
		return fmt.Sprintf("\r%s", s.label)
	}

	frame := s.frames[s.current%len(s.frames)]
	s.current++
	return fmt.Sprintf("\r%s %s", frame, s.label)
}

// Reset resets the spinner to the first frame
func (s *Spinner) Reset() {
	s.current = 0
}

// isTerminal checks if the given file descriptor is a terminal
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Use file descriptor to check if it's a terminal
	fileInfo, err := f.Stat()
	if err != nil {
		return false
	}
	// Check if it's a character device (terminal)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// getTerminalWidth gets the terminal width, returns 0 if unable to determine
func getTerminalWidth() int {
	// Try to get terminal width from environment
	// This is a simple fallback - in production you might want to use golang.org/x/term
	if width := os.Getenv("COLUMNS"); width != "" {
		var w int
		if _, err := fmt.Sscanf(width, "%d", &w); err == nil && w > 0 {
			return w
		}
	}
	return 0
}
