package monitoring

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// CommandMonitor captures and analyzes command execution in real-time
type CommandMonitor struct {
	cmd       *exec.Cmd
	stdout    *OutputCapture
	stderr    *OutputCapture
	exitCode  int
	startTime time.Time
	endTime   time.Time
	patterns  *PatternDetector
	observers []OutputObserver
	mu        sync.RWMutex
}

// ExecutionResult contains complete execution information
type ExecutionResult struct {
	Command        string
	ExitCode       int
	Duration       time.Duration
	StdoutLines    []OutputLine
	StderrLines    []OutputLine
	DetectedIssues []DetectedIssue
	StartTime      time.Time
	EndTime        time.Time
}

// DetectedIssue represents an issue found during execution
type DetectedIssue struct {
	Category   ErrorCategory
	Severity   Severity
	Message    string
	Line       int
	Stream     string
	Confidence float64
	Context    map[string]string
}

// OutputObserver observes command output in real-time
type OutputObserver interface {
	OnOutputLine(line OutputLine)
	OnPatternDetected(line OutputLine, match PatternMatch)
}

// NewCommandMonitor creates a monitor for a command
func NewCommandMonitor(cmd *exec.Cmd, patterns *PatternDetector) *CommandMonitor {
	if patterns == nil {
		patterns = NewPatternDetector()
	}

	mon := &CommandMonitor{
		cmd:       cmd,
		stdout:    NewOutputCapture(nil, 10*1024*1024), // 10MB max
		stderr:    NewOutputCapture(nil, 10*1024*1024),
		patterns:  patterns,
		observers: make([]OutputObserver, 0),
	}

	return mon
}

// AddObserver adds an output observer
func (m *CommandMonitor) AddObserver(observer OutputObserver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers = append(m.observers, observer)
}

// Start begins monitoring
func (m *CommandMonitor) Start(ctx context.Context) error {
	m.startTime = time.Now()

	// Get stdout and stderr pipes
	stdoutPipe, err := m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := m.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start output analyzers
	go m.analyzeOutputStream(ctx, stdoutPipe, m.stdout, "stdout")
	go m.analyzeOutputStream(ctx, stderrPipe, m.stderr, "stderr")

	// Start command
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for completion
	err = m.cmd.Wait()
	m.endTime = time.Now()

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		m.exitCode = exitErr.ExitCode()
	} else if err == nil {
		m.exitCode = 0
	} else {
		m.exitCode = -1
	}

	return err
}

// analyzeOutputStream analyzes output in real-time
func (m *CommandMonitor) analyzeOutputStream(ctx context.Context, pipe io.Reader, capture *OutputCapture, stream string) {
	scanner := bufio.NewScanner(pipe)

	lineNum := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			lineNum++
			outputLine := OutputLine{
				Timestamp: time.Now(),
				Content:   line,
				Stream:    stream,
			}

			// Pattern detection
			matches := m.patterns.Detect(line)
			outputLine.Detected = matches

			// Store line
			capture.AddLine(outputLine)

			// Notify observers
			m.mu.RLock()
			observers := make([]OutputObserver, len(m.observers))
			copy(observers, m.observers)
			m.mu.RUnlock()

			for _, obs := range observers {
				obs.OnOutputLine(outputLine)
			}

			// Notify on critical patterns
			for _, match := range matches {
				if match.Severity >= SeverityError {
					for _, obs := range observers {
						obs.OnPatternDetected(outputLine, match)
					}
				}
			}
		}
	}
}

// GetExecutionResult returns the complete execution result
func (m *CommandMonitor) GetExecutionResult() *ExecutionResult {
	return &ExecutionResult{
		Command:        m.cmd.String(),
		ExitCode:       m.exitCode,
		Duration:       m.endTime.Sub(m.startTime),
		StdoutLines:    m.stdout.GetLines(),
		StderrLines:    m.stderr.GetLines(),
		DetectedIssues: m.collectDetectedIssues(),
		StartTime:      m.startTime,
		EndTime:        m.endTime,
	}
}

// collectDetectedIssues collects all detected issues from output lines
func (m *CommandMonitor) collectDetectedIssues() []DetectedIssue {
	var issues []DetectedIssue
	seen := make(map[string]bool) // Deduplicate by message

	// Collect from stdout
	for i, line := range m.stdout.GetLines() {
		for _, match := range line.Detected {
			key := fmt.Sprintf("%s:%s:%d", match.Category, match.Pattern, i)
			if !seen[key] {
				issues = append(issues, DetectedIssue{
					Category:   match.Category,
					Severity:   match.Severity,
					Message:    line.Content,
					Line:       i + 1,
					Stream:     "stdout",
					Confidence: match.Confidence,
					Context:    match.Context,
				})
				seen[key] = true
			}
		}
	}

	// Collect from stderr
	for i, line := range m.stderr.GetLines() {
		for _, match := range line.Detected {
			key := fmt.Sprintf("%s:%s:%d", match.Category, match.Pattern, i)
			if !seen[key] {
				issues = append(issues, DetectedIssue{
					Category:   match.Category,
					Severity:   match.Severity,
					Message:    line.Content,
					Line:       i + 1,
					Stream:     "stderr",
					Confidence: match.Confidence,
					Context:    match.Context,
				})
				seen[key] = true
			}
		}
	}

	return issues
}
