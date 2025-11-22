package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// TerminalUI manages the terminal UI display with proper rendering
type TerminalUI struct {
	writer           io.Writer
	mu               sync.Mutex
	currentStep      int
	totalSteps       int
	spinner          *Spinner
	progressBar      *ProgressBar
	lastLine         string
	stopSpinner      chan struct{}
	spinnerDone      chan struct{}
	lastProgress     int      // Last progress percentage shown
	lastProgressTime time.Time // Last time progress was updated
}

// NewTerminalUI creates a new terminal UI manager
func NewTerminalUI(writer io.Writer, totalSteps int) *TerminalUI {
	if writer == nil {
		writer = os.Stdout
	}
	return &TerminalUI{
		writer:     writer,
		totalSteps: totalSteps,
	}
}

// StartStep starts a new step with a spinner
func (ui *TerminalUI) StartStep(stepNum int, emoji string, title string, message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.currentStep = stepNum

	// Clear previous line
	ui.clearLineLocked()

	// Print step header
	header := fmt.Sprintf("\n%s Step %d/%d: %s", emoji, stepNum, ui.totalSteps, title)
	fmt.Fprintln(ui.writer, header)

	// Start spinner on a new line with proper indentation
	ui.spinner = NewSpinner("   " + message)
	ui.startSpinnerLoop()
}

// UpdateStepMessage updates the current step's message
func (ui *TerminalUI) UpdateStepMessage(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.spinner != nil {
		ui.spinner.label = "   " + message
	}
}

// CompleteStep marks the current step as complete
func (ui *TerminalUI) CompleteStep(items ...string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Stop spinner
	ui.stopSpinnerLoop()
	ui.clearLineLocked()

	// Print success items
	if len(items) > 0 {
		for _, item := range items {
			if item != "" {
				fmt.Fprintf(ui.writer, "   %s %s\n", Success("✓"), item)
			}
		}
	}
}

// FailStep marks the current step as failed
func (ui *TerminalUI) FailStep(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Stop spinner
	ui.stopSpinnerLoop()
	ui.clearLineLocked()

	// Print error
	fmt.Fprintf(ui.writer, "   %s %s\n", Error("✗"), message)
}

// Warning prints a warning message
func (ui *TerminalUI) Warning(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	wasSpinning := ui.spinner != nil
	if wasSpinning {
		ui.stopSpinnerLoop()
		ui.clearLineLocked()
	}

	fmt.Fprintf(ui.writer, "   %s %s\n", Warning("⚠️"), message)

	if wasSpinning {
		ui.startSpinnerLoop()
	}
}

// Info prints an info message
func (ui *TerminalUI) Info(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	wasSpinning := ui.spinner != nil
	if wasSpinning {
		ui.stopSpinnerLoop()
		ui.clearLineLocked()
	}

	fmt.Fprintf(ui.writer, "   %s %s\n", Info("ℹ️"), message)

	if wasSpinning {
		ui.startSpinnerLoop()
	}
}

// ShowProgress shows a progress bar for determinate tasks
func (ui *TerminalUI) ShowProgress(current, total int, label string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Aggressive throttling to prevent flickering
	// Only update if:
	// 1. Progress changed by at least 5% OR
	// 2. At least 500ms has passed since last update OR
	// 3. Progress is complete (100%) OR
	// 4. This is the first update (progressBar is nil)
	now := time.Now()
	percentChange := 0
	if total > 0 && ui.lastProgress >= 0 {
		oldPercent := (ui.lastProgress * 100) / total
		newPercent := (current * 100) / total
		if oldPercent < newPercent {
			percentChange = newPercent - oldPercent
		} else {
			percentChange = oldPercent - newPercent
		}
	}
	
	timeSinceLastUpdate := now.Sub(ui.lastProgressTime)
	isFirstUpdate := ui.progressBar == nil
	shouldUpdate := isFirstUpdate || percentChange >= 5 || timeSinceLastUpdate >= 500*time.Millisecond || current >= total
	
	if !shouldUpdate {
		// Skip this update to avoid flickering
		return
	}

	// Stop spinner if running
	if ui.spinner != nil {
		ui.stopSpinnerLoop()
		ui.clearLineLocked()
		ui.spinner = nil
	}

	// Create or update progress bar
	if ui.progressBar == nil || ui.progressBar.total != total {
		ui.progressBar = NewProgressBar(total, "   "+label)
	}
	ui.progressBar.Update(current)

	// Render progress bar
	ui.clearLineLocked()
	fmt.Fprint(ui.writer, ui.progressBar.Render())
	
	// Update tracking
	ui.lastProgress = current
	ui.lastProgressTime = now
	
	if current >= total {
		fmt.Fprintln(ui.writer)
		ui.progressBar = nil
		ui.lastProgress = 0
	}
}

// PrintCommandOutput prints command output with proper indentation
func (ui *TerminalUI) PrintCommandOutput(line string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	wasSpinning := ui.spinner != nil
	if wasSpinning {
		ui.stopSpinnerLoop()
		ui.clearLineLocked()
	}

	// Print with indentation
	if line != "" {
		fmt.Fprintf(ui.writer, "   %s\n", Dim(line))
	}

	if wasSpinning {
		ui.startSpinnerLoop()
	}
}

// Close cleans up the terminal UI
func (ui *TerminalUI) Close() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.stopSpinnerLoop()
	ui.clearLineLocked()
}

// Private methods

func (ui *TerminalUI) startSpinnerLoop() {
	if ui.spinner == nil || ui.stopSpinner != nil {
		return
	}

	ui.stopSpinner = make(chan struct{})
	ui.spinnerDone = make(chan struct{})

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		defer close(ui.spinnerDone)

		for {
			select {
			case <-ui.stopSpinner:
				return
			case <-ticker.C:
				ui.mu.Lock()
				if ui.spinner != nil {
					ui.clearLineLocked()
					frame := ui.spinner.Tick()
					fmt.Fprint(ui.writer, frame)
					_, _ = ui.writer.(*os.File)
					if f, ok := ui.writer.(*os.File); ok {
						f.Sync()
					}
				}
				ui.mu.Unlock()
			}
		}
	}()
}

func (ui *TerminalUI) stopSpinnerLoop() {
	if ui.stopSpinner != nil {
		close(ui.stopSpinner)
		<-ui.spinnerDone
		ui.stopSpinner = nil
		ui.spinnerDone = nil
	}
	ui.spinner = nil
}

func (ui *TerminalUI) clearLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.clearLineLocked()
}

func (ui *TerminalUI) clearLineLocked() {
	// Use ANSI escape codes to properly clear the line
	// \033[2K clears the entire line
	// \r moves cursor to beginning of line
	if isTerminalFile(ui.writer) {
		// Use ANSI escape sequence to clear line
		fmt.Fprint(ui.writer, "\r\033[2K")
	} else {
		// Fallback for non-terminal: just move to beginning and clear with spaces
		// Use a wider clear to handle long lines
		fmt.Fprint(ui.writer, "\r"+strings.Repeat(" ", 150)+"\r")
	}
}

// isTerminalFile checks if the writer is a terminal file
func isTerminalFile(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isTerminal(f)
	}
	return false
}
