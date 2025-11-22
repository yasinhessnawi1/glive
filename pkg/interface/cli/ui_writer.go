package cli

import (
	"bytes"
	"strings"
)

// UIWriter is a writer that routes output through the TerminalUI
type UIWriter struct {
	ui     *TerminalUI
	buffer bytes.Buffer
}

// NewUIWriter creates a new UI-aware writer
func NewUIWriter(ui *TerminalUI) *UIWriter {
	return &UIWriter{
		ui: ui,
	}
}

// Write implements io.Writer
func (w *UIWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.buffer.Write(p)

	// Process complete lines
	for {
		line, err := w.buffer.ReadString('\n')
		if err != nil {
			// Put incomplete line back in buffer
			if line != "" {
				w.buffer.WriteString(line)
			}
			break
		}

		// Trim newline and process the line
		line = strings.TrimSuffix(line, "\n")
		w.processLine(line)
	}

	return n, nil
}

// processLine processes a single line of output
func (w *UIWriter) processLine(line string) {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return
	}

	// Skip step headers (they're handled by the UI)
	if strings.Contains(line, "Step ") && strings.Contains(line, ":") {
		return
	}

	// Route different types of output
	trimmed := strings.TrimSpace(line)

	// Success messages
	if strings.HasPrefix(trimmed, "✓") {
		msg := strings.TrimPrefix(trimmed, "✓")
		msg = strings.TrimSpace(msg)
		// Don't duplicate - let UI handle checkmarks
		return
	}

	// Warning messages
	if strings.HasPrefix(trimmed, "⚠️") || strings.HasPrefix(trimmed, "⚠") {
		msg := strings.TrimPrefix(trimmed, "⚠️")
		msg = strings.TrimPrefix(msg, "⚠")
		msg = strings.TrimSpace(msg)
		w.ui.Warning(msg)
		return
	}

	// Info messages
	if strings.HasPrefix(trimmed, "ℹ️") || strings.HasPrefix(trimmed, "ℹ") {
		msg := strings.TrimPrefix(trimmed, "ℹ️")
		msg = strings.TrimPrefix(msg, "ℹ")
		msg = strings.TrimSpace(msg)
		w.ui.Info(msg)
		return
	}

	// Error messages
	if strings.HasPrefix(trimmed, "✗") {
		msg := strings.TrimPrefix(trimmed, "✗")
		msg = strings.TrimSpace(msg)
		w.ui.FailStep(msg)
		return
	}

	// Command output or other messages - show dimmed
	if trimmed != "" {
		w.ui.PrintCommandOutput(trimmed)
	}
}

// Flush flushes any remaining buffered data
func (w *UIWriter) Flush() {
	if w.buffer.Len() > 0 {
		remaining := w.buffer.String()
		lines := strings.Split(remaining, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				w.processLine(line)
			}
		}
		w.buffer.Reset()
	}
}
