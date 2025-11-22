package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/interface/tui"
	"github.com/glive/usecase/project"
)

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ConflictResolution represents user's choice for handling conflicts
type ConflictResolution int

const (
	ConflictNone ConflictResolution = iota
	ConflictDelete
	ConflictRename
	ConflictCancel
)

// ExecutionView represents the live execution view
type ExecutionView struct {
	state        *tui.AppState
	projectName  string
	githubURL    string
	currentStep  int
	totalSteps   int
	steps        []ExecutionStep
	milestones   []string // Key milestones that persist above live output
	outputLines  []string
	scrollOffset int
	autoScroll   bool
	startTime    time.Time
	isSimulation bool // If true, run simulation; if false, run real execution
	completed    bool
	failed       bool
	cancelled    bool // Whether execution was cancelled
	errorMsg     string
	progressChan chan tea.Msg       // Channel for progress messages in real execution
	forceSetup   bool               // Force re-clone even if project exists
	ctx          context.Context    // Cancellable context
	cancel       context.CancelFunc // Cancel function

	// Conflict resolution state
	hasConflict      bool                    // Whether we're waiting for conflict resolution
	conflictPath     string                  // Path of the conflicting directory
	conflictSelected int                     // Currently selected option (0=Delete, 1=Rename, 2=Cancel)
	conflictChan     chan ConflictResolution // Channel to send user's choice back to execution
}

// ExecutionStep represents a single execution step
type ExecutionStep struct {
	Name      string
	Status    StepStatus
	Progress  float64
	Duration  time.Duration
	StartTime time.Time
}

// StepStatus represents the status of a step
type StepStatus string

const (
	StepPending  StepStatus = "pending"
	StepRunning  StepStatus = "running"
	StepComplete StepStatus = "complete"
	StepFailed   StepStatus = "failed"
)

// NewExecutionView creates a new execution view
func NewExecutionView(state *tui.AppState, projectName string) *ExecutionView {
	// Steps in actual execution order (AI runs in parallel after clone)
	steps := []ExecutionStep{
		{Name: "Parsing GitHub URL", Status: StepPending},
		{Name: "Cloning Repository", Status: StepPending},
		{Name: "AI-Powered Analysis", Status: StepPending}, // Runs in parallel after clone
		{Name: "Security Scanning", Status: StepPending},
		{Name: "Analyzing Project", Status: StepPending},
		{Name: "Installing Dependencies", Status: StepPending},
		{Name: "Project Ready", Status: StepPending},
	}

	// Check if we have a GitHub URL to run real execution
	githubURL := state.ProjectURL
	isSimulation := githubURL == ""

	// Create cancellable context for real execution
	var ctx context.Context
	var cancel context.CancelFunc
	if !isSimulation {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx = context.Background()
		cancel = func() {} // No-op for simulation
	}

	return &ExecutionView{
		state:        state,
		projectName:  projectName,
		githubURL:    githubURL,
		totalSteps:   len(steps),
		steps:        steps,
		milestones:   make([]string, 0),
		outputLines:  make([]string, 0),
		autoScroll:   true,
		startTime:    time.Now(),
		isSimulation: isSimulation,
		forceSetup:   state.ForceSetup, // Pass through the force flag
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Init initializes the execution view
func (e *ExecutionView) Init() tea.Cmd {
	// Refresh state from global app state
	e.githubURL = e.state.ProjectURL
	e.forceSetup = e.state.ForceSetup
	e.projectName = "" // Will be set during execution

	// Reset execution state
	e.currentStep = 0
	e.milestones = make([]string, 0)
	e.outputLines = make([]string, 0)
	e.completed = false
	e.failed = false
	e.cancelled = false
	e.errorMsg = ""
	e.startTime = time.Now()
	e.hasConflict = false
	e.conflictPath = ""

	// Reset steps
	e.steps = []ExecutionStep{
		{Name: "Parsing GitHub URL", Status: StepPending},
		{Name: "Cloning Repository", Status: StepPending},
		{Name: "AI-Powered Analysis", Status: StepPending},
		{Name: "Security Scanning", Status: StepPending},
		{Name: "Analyzing Project", Status: StepPending},
		{Name: "Installing Dependencies", Status: StepPending},
		{Name: "Project Ready", Status: StepPending},
	}

	// Re-create context
	if e.cancel != nil {
		e.cancel()
	}
	if !e.isSimulation {
		e.ctx, e.cancel = context.WithCancel(context.Background())
	} else {
		e.ctx = context.Background()
		e.cancel = func() {}
	}

	if e.isSimulation {
		// Start a ticker to simulate progress
		return tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})
	}

	// Start real execution in a goroutine
	return e.startRealExecution()
}

// startRealExecution launches the actual project setup in a goroutine
func (e *ExecutionView) startRealExecution() tea.Cmd {
	return func() tea.Msg {
		return ExecutionStartMsg{
			ProjectName: e.projectName,
			GitHubURL:   e.githubURL,
		}
	}
}

// executeProject runs the actual project setup in a goroutine
func (e *ExecutionView) executeProject() tea.Cmd {
	// Create a progress callback that sends Bubble Tea messages
	progressChan := make(chan tea.Msg, 100)

	// Store the channel immediately for message listening
	e.progressChan = progressChan

	// Map stage names to step indices (matches actual execution order)
	stageToStep := map[string]int{
		"parsing":     0,
		"cloning":     1,
		"ai_analysis": 2, // Runs in parallel after clone
		"scanning":    3,
		"analyzing":   4,
		"installing":  5,
		"ready":       6,
	}

	// Start execution in background goroutine
	go func() {
		defer close(progressChan)

		// Use the cancellable context from ExecutionView
		ctx := e.ctx

		// Create use case with progress callback
		projectRepo := e.state.Container.ProjectRepository()

		// Create a custom output writer that sends output as messages
		outputWriter := &messageWriter{ch: progressChan}

		setupUC := project.NewSetupProjectUseCase(projectRepo, e.state.Container, outputWriter)

		// Set up conflict resolution callback
		conflictResponseChan := make(chan ConflictResolution, 1)
		setupUC.SetConflictCallback(func(path string) project.ConflictAction {
			// Send message to UI to show conflict dialog
			progressChan <- ConflictDetectedMsg{
				Path:         path,
				ResponseChan: conflictResponseChan,
			}
			// Wait for user response
			resolution := <-conflictResponseChan
			switch resolution {
			case ConflictDelete:
				return project.ConflictActionDelete
			case ConflictRename:
				return project.ConflictActionRename
			case ConflictCancel:
				return project.ConflictActionCancel
			default:
				return project.ConflictActionCancel
			}
		})

		// Track the last stage to detect stage transitions
		lastStage := ""
		lastMessage := ""
		lastMessageTime := time.Time{}
		lastProgressPercentage := -1

		// Set up progress callback with debouncing
		setupUC.SetProgressCallback(func(projectID string, stage string, message string, percentage int) {
			stepIndex, ok := stageToStep[stage]
			if !ok {
				return
			}

			// When stage changes, mark the previous stage as complete
			if lastStage != "" && lastStage != stage {
				if prevIndex, ok := stageToStep[lastStage]; ok {
					progressChan <- ExecutionStepMsg{
						StepIndex: prevIndex,
						Status:    StepComplete,
						Message:   "",
					}
				}
				// Reset message tracking on stage change
				lastMessage = ""
				lastProgressPercentage = -1
			}
			lastStage = stage

			// Debounce messages: only send if message changed or enough time passed
			now := time.Now()
			messageChanged := message != "" && message != lastMessage
			timeSinceLastMessage := now.Sub(lastMessageTime)
			shouldSendMessage := messageChanged && (timeSinceLastMessage >= 500*time.Millisecond || lastMessageTime.IsZero())

			// Debounce progress: only send if percentage changed significantly (5% or more)
			progressChanged := lastProgressPercentage < 0 || abs(percentage-lastProgressPercentage) >= 5
			shouldSendProgress := progressChanged || percentage >= 100

			// Determine status based on percentage
			status := StepRunning
			if percentage >= 100 {
				status = StepComplete
			}

			// Send step update only if message should be sent
			if shouldSendMessage {
				progressChan <- ExecutionStepMsg{
					StepIndex: stepIndex,
					Status:    status,
					Message:   message,
				}
				lastMessage = message
				lastMessageTime = now
			} else if status != StepRunning {
				// Still send status update even without message if status changed
				progressChan <- ExecutionStepMsg{
					StepIndex: stepIndex,
					Status:    status,
					Message:   "",
				}
			}

			// Send progress update only if significant change
			if shouldSendProgress {
				progressChan <- ExecutionProgressMsg{
					StepIndex: stepIndex,
					Progress:  float64(percentage) / 100.0,
					Message:   "",
				}
				lastProgressPercentage = percentage
			}
		})

		// Execute the setup
		workspaceDir := e.state.Container.Config().WorkspaceDir
		mode := e.state.Container.Config().DefaultMode

		_, err := setupUC.Execute(ctx, project.SetupProjectInput{
			GitHubURL:    e.githubURL,
			Mode:         mode,
			WorkspaceDir: workspaceDir,
			Force:        e.forceSetup, // Use the force flag from CLI
		})

		// Check if cancellation was requested
		if ctx.Err() == context.Canceled {
			progressChan <- ExecutionCancelledMsg{
				Step: e.currentStep,
			}
			return
		}

		if err != nil {
			// Mark the last running stage as failed
			if lastStage != "" {
				if lastIndex, ok := stageToStep[lastStage]; ok {
					progressChan <- ExecutionStepMsg{
						StepIndex: lastIndex,
						Status:    StepFailed,
						Message:   "",
					}
				}
			}
			progressChan <- ExecutionErrorMsg{
				Error: err,
				Step:  e.currentStep,
			}
		} else {
			// Mark the last running stage as complete before sending completion
			if lastStage != "" {
				if lastIndex, ok := stageToStep[lastStage]; ok {
					progressChan <- ExecutionStepMsg{
						StepIndex: lastIndex,
						Status:    StepComplete,
						Message:   "",
					}
				}
			}
			progressChan <- ExecutionCompleteMsg{
				Duration: time.Since(e.startTime),
			}
		}
	}()

	// Start listening to progress updates
	return e.waitForNextMessage()
}

// waitForNextMessage creates a command that waits for the next progress message
func (e *ExecutionView) waitForNextMessage() tea.Cmd {
	if e.progressChan == nil {
		return nil
	}

	return func() tea.Msg {
		msg, ok := <-e.progressChan
		if !ok {
			// Channel closed, execution finished
			return nil
		}
		return msg
	}
}

// messageWriter writes output as Bubble Tea messages
type messageWriter struct {
	ch chan<- tea.Msg
}

func (w *messageWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			w.ch <- ExecutionOutputMsg{Line: line}
		}
	}
	return len(p), nil
}

// Update handles messages
func (e *ExecutionView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle conflict resolution keys first
		if e.hasConflict {
			switch msg.String() {
			case "up", "k":
				if e.conflictSelected > 0 {
					e.conflictSelected--
				}
				return e, nil
			case "down", "j":
				if e.conflictSelected < 2 {
					e.conflictSelected++
				}
				return e, nil
			case "enter":
				// Send the selected resolution
				var resolution ConflictResolution
				switch e.conflictSelected {
				case 0:
					resolution = ConflictDelete
					e.outputLines = append(e.outputLines, "   🗑️  Deleting existing directory...")
				case 1:
					resolution = ConflictRename
					e.outputLines = append(e.outputLines, "   📁 Renaming existing directory...")
				case 2:
					resolution = ConflictCancel
					e.outputLines = append(e.outputLines, "   ❌ Cancelled by user")
				}
				e.hasConflict = false
				if e.conflictChan != nil {
					e.conflictChan <- resolution
				}
				return e, e.waitForNextMessage()
			case "1":
				e.conflictSelected = 0
				return e, nil
			case "2":
				e.conflictSelected = 1
				return e, nil
			case "3":
				e.conflictSelected = 2
				return e, nil
			case "esc", "q":
				// Cancel on escape
				e.hasConflict = false
				if e.conflictChan != nil {
					e.conflictChan <- ConflictCancel
				}
				e.outputLines = append(e.outputLines, "   ❌ Cancelled by user")
				return e, e.waitForNextMessage()
			}
			return e, nil
		}

		switch msg.String() {
		case "esc", "q":
			// Allow quit at any time, just switch back to dashboard or exit
			return e, tea.Quit
		case "x", "X":
			// Cancel running execution
			if !e.completed && !e.failed && !e.cancelled && !e.isSimulation {
				e.cancel() // Cancel the context
				e.cancelled = true
				e.outputLines = append(e.outputLines, "")
				e.outputLines = append(e.outputLines, "⚠️  Cancellation requested...")

				// Mark current running step as cancelled
				for i := range e.steps {
					if e.steps[i].Status == StepRunning {
						e.steps[i].Status = StepFailed
						e.steps[i].Progress = 0
					}
				}

				// Close progress channel to stop listening
				if e.progressChan != nil {
					close(e.progressChan)
					e.progressChan = nil
				}

				return e, nil
			}
			return e, nil
		case "up", "k":
			if e.scrollOffset > 0 {
				e.scrollOffset--
				e.autoScroll = false
			}
			return e, nil
		case "down", "j":
			// Calculate dynamic visible height
			visibleHeight := e.calculateVisibleOutputHeight()
			maxScroll := len(e.outputLines) - visibleHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			if e.scrollOffset < maxScroll {
				e.scrollOffset++
				e.autoScroll = false
			}
			return e, nil
		case "home", "g":
			// Jump to top
			e.scrollOffset = 0
			e.autoScroll = false
			return e, nil
		case "end", "G":
			// Jump to bottom
			visibleHeight := e.calculateVisibleOutputHeight()
			maxScroll := len(e.outputLines) - visibleHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			e.scrollOffset = maxScroll
			e.autoScroll = true
			return e, nil
		case "f":
			e.autoScroll = !e.autoScroll
			return e, nil
		}

	case ExecutionStartMsg:
		// Real execution is starting
		e.projectName = msg.ProjectName
		e.githubURL = msg.GitHubURL
		e.outputLines = append(e.outputLines, fmt.Sprintf("Starting setup for: %s", msg.GitHubURL))
		return e, e.executeProject()

	case ExecutionStepMsg:
		// Update step status
		if msg.StepIndex >= 0 && msg.StepIndex < len(e.steps) {
			step := &e.steps[msg.StepIndex]

			// If transitioning to running, record start time
			if step.Status != StepRunning && msg.Status == StepRunning {
				step.StartTime = time.Now()
				step.Progress = 0
			}

			// If transitioning to complete, record duration
			if msg.Status == StepComplete {
				step.Progress = 1.0
				if !step.StartTime.IsZero() {
					step.Duration = time.Since(step.StartTime)
				}
			}

			step.Status = msg.Status

			// Only add message to output if it's not a duplicate
			if msg.Message != "" {
				// Check if this message was recently added (deduplication)
				isDuplicate := false
				if len(e.outputLines) > 0 {
					lastLine := e.outputLines[len(e.outputLines)-1]
					// Check if it's the same message (allowing for slight variations)
					if strings.TrimSpace(lastLine) == strings.TrimSpace(msg.Message) {
						isDuplicate = true
					}
				}
				if !isDuplicate {
					e.outputLines = append(e.outputLines, msg.Message)
				}
			}

			// Update current step to the highest running/complete step
			if msg.StepIndex >= e.currentStep {
				e.currentStep = msg.StepIndex
			}
		}
		// Keep listening for more messages
		return e, e.waitForNextMessage()

	case ExecutionProgressMsg:
		// Update progress within a step (don't add to output lines - progress is visual only)
		if msg.StepIndex >= 0 && msg.StepIndex < len(e.steps) {
			e.steps[msg.StepIndex].Progress = msg.Progress
			// Don't add progress messages to output lines to avoid spam
		}
		// Keep listening for more messages
		return e, e.waitForNextMessage()

	case ExecutionOutputMsg:
		// Add output line
		e.outputLines = append(e.outputLines, msg.Line)
		// Check if this is a milestone
		if e.isMilestone(msg.Line) {
			e.addMilestone(msg.Line)
		}
		// Auto-scroll to bottom if enabled
		if e.autoScroll {
			visibleHeight := e.calculateVisibleOutputHeight()
			maxScroll := len(e.outputLines) - visibleHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			e.scrollOffset = maxScroll
		}
		// Keep listening for more messages
		return e, e.waitForNextMessage()

	case ExecutionCompleteMsg:
		// Execution completed successfully
		e.completed = true
		duration := time.Since(e.startTime)

		// Mark all steps as complete (since execution succeeded, all steps are done)
		for i := range e.steps {
			if e.steps[i].Status == StepRunning || e.steps[i].Status == StepPending {
				e.steps[i].Status = StepComplete
				e.steps[i].Progress = 1.0
			}
		}
		e.currentStep = len(e.steps) - 1

		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, fmt.Sprintf("✅ Project setup completed in %s", formatDuration(duration)))
		return e, nil

	case ExecutionErrorMsg:
		// Execution failed
		e.failed = true
		e.errorMsg = msg.Error.Error()
		if msg.Step >= 0 && msg.Step < len(e.steps) {
			e.steps[msg.Step].Status = StepFailed
		}
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, fmt.Sprintf("❌ Error: %s", msg.Error.Error()))
		return e, nil

	case ExecutionCancelledMsg:
		// Execution was cancelled
		e.cancelled = true
		if msg.Step >= 0 && msg.Step < len(e.steps) {
			e.steps[msg.Step].Status = StepFailed
		}
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "🛑 Execution cancelled by user")
		return e, nil

	case ConflictDetectedMsg:
		// Directory conflict detected - show resolution options
		e.hasConflict = true
		e.conflictPath = msg.Path
		e.conflictSelected = 0 // Default to first option (Delete)
		e.conflictChan = msg.ResponseChan
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "⚠️  Directory conflict detected!")
		e.outputLines = append(e.outputLines, fmt.Sprintf("   Path: %s", msg.Path))
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "   Choose an action:")
		// Don't wait for next message - we're in interactive mode now
		return e, nil

	case TickMsg:
		// Simulation mode - advance current step
		if e.isSimulation && e.currentStep < len(e.steps) {
			currentStep := &e.steps[e.currentStep]

			if currentStep.Status == StepPending {
				// Start this step
				currentStep.Status = StepRunning
				currentStep.StartTime = time.Now()
				currentStep.Progress = 0
				e.outputLines = append(e.outputLines, fmt.Sprintf("Starting: %s", currentStep.Name))
			} else if currentStep.Status == StepRunning {
				// Progress this step
				currentStep.Progress += 0.2
				if currentStep.Progress >= 1.0 {
					currentStep.Progress = 1.0
					currentStep.Status = StepComplete
					currentStep.Duration = time.Since(currentStep.StartTime)
					e.outputLines = append(e.outputLines, fmt.Sprintf("Completed: %s (%s)", currentStep.Name, formatDuration(currentStep.Duration)))
					e.currentStep++
				}
			}
		}

		// Keep ticking in simulation mode
		if e.isSimulation {
			return e, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return TickMsg(t)
			})
		}
		return e, nil
	}

	return e, nil
}

// View renders the execution view
func (e *ExecutionView) View() string {
	// Allow rendering even with 0 size during initialization
	if e.state.Width > 0 && e.state.Height > 0 {
		if e.state.Width < 80 || e.state.Height < 24 {
			return "Terminal too small. Please resize to at least 80x24."
		}
	}

	elapsed := time.Since(e.startTime)
	progress := fmt.Sprintf("Progress: %d/%d | %s", e.currentStep+1, e.totalSteps, formatDuration(elapsed))

	header := lipgloss.NewStyle().
		Bold(true).
		Padding(1).
		Render(fmt.Sprintf("▶ Running: %s    %s", e.projectName, progress))

	timeline := e.renderTimeline()
	milestones := e.renderMilestones()
	output := e.renderOutput()
	autoScrollStatus := "OFF"
	if e.autoScroll {
		autoScrollStatus = "ON"
	}
	cancelHint := ""
	if !e.completed && !e.failed && !e.cancelled && !e.isSimulation {
		cancelHint = "  [X] Cancel"
	}
	footer := fmt.Sprintf("[Q] Quit%s  [↑↓] Scroll  [Home/End] Jump  [F] Auto-scroll: %s", cancelHint, autoScrollStatus)

	// Build view with milestones section between timeline and output
	sections := []string{header, "", timeline}
	if milestones != "" {
		sections = append(sections, "", milestones)
	}

	// Add conflict resolution dialog if active
	if e.hasConflict {
		conflict := e.renderConflictDialog()
		sections = append(sections, "", conflict)
		conflictFooter := "[↑↓/1-3] Select  [Enter] Confirm  [Esc] Cancel"
		sections = append(sections, "", output, "", conflictFooter)
	} else {
		sections = append(sections, "", output, "", footer)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (e *ExecutionView) renderTimeline() string {
	lines := []string{"Execution Timeline:", ""}

	for i, step := range e.steps {
		icon := "○"
		progressBar := strings.Repeat("░", 16)

		switch step.Status {
		case StepComplete:
			icon = "✓"
			progressBar = strings.Repeat("█", 16)
		case StepRunning:
			icon = "⟳"
			filled := int(step.Progress * 16)
			progressBar = strings.Repeat("█", filled) + strings.Repeat("░", 16-filled)
		case StepFailed:
			icon = "✗"
		}

		status := ""
		if step.Duration > 0 {
			status = formatDuration(step.Duration)
		} else if step.Status == StepComplete {
			status = "done"
		} else if step.Status == StepRunning {
			status = "running"
		} else if step.Status == StepFailed {
			status = "failed"
		} else {
			status = "pending"
		}

		current := ""
		if i == e.currentStep && step.Status == StepRunning {
			current = " ← Current"
		}

		line := fmt.Sprintf("%s %-25s [%s] %s%s",
			icon, step.Name, progressBar, status, current)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// calculateVisibleOutputHeight calculates how many output lines can be displayed
func (e *ExecutionView) calculateVisibleOutputHeight() int {
	if e.state.Height <= 0 {
		return 8 // Default fallback
	}

	// Reserve space for:
	// - Header: ~3 lines
	// - Timeline: ~9 lines (7 steps + title + spacing)
	// - Milestones: 2 + number of milestones (title + blank + milestones)
	// - "Live Output:" title: 1 line
	// - Footer: 1 line
	// - Spacing: ~3 lines
	// Total reserved: ~17 lines + milestones
	reservedHeight := 17
	if len(e.milestones) > 0 {
		reservedHeight += 2 + len(e.milestones) // title + blank + milestone lines
	}
	visibleHeight := e.state.Height - reservedHeight

	if visibleHeight < 5 {
		visibleHeight = 5 // Minimum visible lines
	}

	return visibleHeight
}

func (e *ExecutionView) renderOutput() string {
	lines := []string{"Live Output:", ""}

	if len(e.outputLines) == 0 {
		lines = append(lines, "Waiting for output...")
		return strings.Join(lines, "\n")
	}

	// Calculate dynamic visible height
	visibleHeight := e.calculateVisibleOutputHeight()

	// Validate and adjust scroll offset
	maxScroll := len(e.outputLines) - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scrollOffset > maxScroll {
		e.scrollOffset = maxScroll
	}
	if e.scrollOffset < 0 {
		e.scrollOffset = 0
	}

	// Render visible lines
	start := e.scrollOffset
	end := start + visibleHeight
	if end > len(e.outputLines) {
		end = len(e.outputLines)
	}

	for i := start; i < end; i++ {
		lines = append(lines, e.outputLines[i])
	}

	// Add scroll indicator if there's more content
	if len(e.outputLines) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n(Showing %d-%d of %d lines)", start+1, end, len(e.outputLines))
		if e.autoScroll {
			scrollInfo += " [Auto-scroll ON]"
		}
		lines = append(lines, scrollInfo)
	} else if e.autoScroll {
		lines = append(lines, "\n[Auto-scroll ON]")
	}

	return strings.Join(lines, "\n")
}

// isMilestone checks if a log line is a key milestone worth persisting
func (e *ExecutionView) isMilestone(line string) bool {
	// Key milestone patterns - important info that should persist
	milestonePatterns := []string{
		"✓ Cloned",
		"✓ Project type:",
		"✓ Detected:",
		"✓ Found",
		"✓ AI analysis",
		"✓ Security scan",
		"✓ Dependencies installed",
		"✓ Ready to run",
		"📍 Location:",
		"📦 Package manager:",
		"🔧 Entry point:",
		"⚠️ Warning:",
		"Cloned to:",
		"Project saved:",
	}

	lineLower := strings.ToLower(line)
	for _, pattern := range milestonePatterns {
		if strings.Contains(lineLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// addMilestone adds a milestone if it's not already present
func (e *ExecutionView) addMilestone(line string) {
	// Avoid duplicates
	for _, m := range e.milestones {
		if m == line {
			return
		}
	}
	// Limit to most recent 8 milestones
	if len(e.milestones) >= 8 {
		e.milestones = e.milestones[1:]
	}
	e.milestones = append(e.milestones, line)
}

// renderMilestones renders the persistent milestones section
func (e *ExecutionView) renderMilestones() string {
	if len(e.milestones) == 0 {
		return ""
	}

	lines := []string{"Key Milestones:", ""}
	milestoneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // Cyan
	for _, m := range e.milestones {
		lines = append(lines, milestoneStyle.Render("  "+m))
	}
	return strings.Join(lines, "\n")
}

// renderConflictDialog renders the conflict resolution dialog
func (e *ExecutionView) renderConflictDialog() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")). // Yellow
		Padding(1, 2)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("3")).
		Bold(true)
	normalStyle := lipgloss.NewStyle()

	options := []string{
		"[1] Delete existing directory and re-clone",
		"[2] Rename existing to backup and clone fresh",
		"[3] Cancel operation",
	}

	var lines []string
	lines = append(lines, "⚠️  Directory Conflict")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Path: %s", e.conflictPath))
	lines = append(lines, "")
	lines = append(lines, "Select an action:")
	lines = append(lines, "")

	for i, opt := range options {
		if i == e.conflictSelected {
			lines = append(lines, selectedStyle.Render(" → "+opt+" "))
		} else {
			lines = append(lines, normalStyle.Render("   "+opt))
		}
	}

	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (e *ExecutionView) isComplete() bool {
	for _, step := range e.steps {
		if step.Status == StepRunning || step.Status == StepPending {
			return false
		}
	}
	return true
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", d.Seconds()*1000)
	} else if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), d.Seconds()-d.Minutes()*60)
}

// Message types
type TickMsg time.Time

// ExecutionStartMsg signals that execution has started
type ExecutionStartMsg struct {
	ProjectName string
	GitHubURL   string
}

// ExecutionStepMsg updates the status of a step
type ExecutionStepMsg struct {
	StepIndex int
	Status    StepStatus
	Message   string
}

// ExecutionProgressMsg updates progress within a step
type ExecutionProgressMsg struct {
	StepIndex int
	Progress  float64
	Message   string
}

// ExecutionOutputMsg adds a line of output
type ExecutionOutputMsg struct {
	Line string
}

// ExecutionCompleteMsg signals successful completion
type ExecutionCompleteMsg struct {
	Project  interface{} // *entities.Project
	Duration time.Duration
}

// ExecutionErrorMsg signals an error
type ExecutionErrorMsg struct {
	Error error
	Step  int
}

// ExecutionCancelledMsg signals that execution was cancelled
type ExecutionCancelledMsg struct {
	Step int
}

// ConflictDetectedMsg signals that a directory conflict was detected
type ConflictDetectedMsg struct {
	Path         string
	ResponseChan chan ConflictResolution
}
