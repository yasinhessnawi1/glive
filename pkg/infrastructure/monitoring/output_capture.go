package monitoring

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// OutputCapture captures output while streaming it
type OutputCapture struct {
	buffer    *bytes.Buffer
	writer    io.Writer
	mu        sync.RWMutex
	lines     []OutputLine
	maxBuffer int // bytes
}

// OutputLine represents a single line of output with metadata
type OutputLine struct {
	Timestamp time.Time
	Content   string
	Stream    string // "stdout" or "stderr"
	Detected  []PatternMatch
}

// PatternMatch represents a detected pattern in output
type PatternMatch struct {
	Pattern    string
	Confidence float64
	Category   ErrorCategory
	Severity   Severity
	Context    map[string]string
}

// NewOutputCapture creates a new output capture
func NewOutputCapture(writer io.Writer, maxBuffer int) *OutputCapture {
	if maxBuffer <= 0 {
		maxBuffer = 10 * 1024 * 1024 // Default 10MB
	}

	return &OutputCapture{
		buffer:    &bytes.Buffer{},
		writer:    writer,
		lines:     make([]OutputLine, 0),
		maxBuffer: maxBuffer,
	}
}

// Write implements io.Writer interface
func (oc *OutputCapture) Write(p []byte) (n int, err error) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Write to underlying writer if provided
	if oc.writer != nil {
		oc.writer.Write(p)
	}

	// Check buffer size and trim if needed
	if oc.buffer.Len()+len(p) > oc.maxBuffer {
		// Remove oldest lines until we have space
		oc.trimBuffer(len(p))
	}

	// Write to buffer
	n, err = oc.buffer.Write(p)
	return n, err
}

// trimBuffer removes oldest lines to make space
func (oc *OutputCapture) trimBuffer(needed int) {
	// Remove lines until we have enough space
	for oc.buffer.Len()+needed > oc.maxBuffer && len(oc.lines) > 0 {
		// Remove first line
		oc.lines = oc.lines[1:]

		// Rebuild buffer without first line
		oc.buffer.Reset()
		for _, line := range oc.lines {
			oc.buffer.WriteString(line.Content + "\n")
		}
	}
}

// AddLine adds a line to the capture
func (oc *OutputCapture) AddLine(line OutputLine) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	oc.lines = append(oc.lines, line)

	// Check buffer size
	if oc.buffer.Len() > oc.maxBuffer {
		oc.trimBuffer(0)
	}
}

// GetLines returns all captured lines
func (oc *OutputCapture) GetLines() []OutputLine {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	// Return a copy
	result := make([]OutputLine, len(oc.lines))
	copy(result, oc.lines)
	return result
}

// GetLastLines returns the last N lines
func (oc *OutputCapture) GetLastLines(n int) []OutputLine {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	if n <= 0 || n >= len(oc.lines) {
		result := make([]OutputLine, len(oc.lines))
		copy(result, oc.lines)
		return result
	}

	start := len(oc.lines) - n
	result := make([]OutputLine, n)
	copy(result, oc.lines[start:])
	return result
}

// GetContent returns the full buffer content as string
func (oc *OutputCapture) GetContent() string {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	return oc.buffer.String()
}


