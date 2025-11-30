package views

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/domain/entities"
	"github.com/glive/domain/repository"
	"github.com/glive/infrastructure/executor"
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

// ExecutionSuggestion represents an AI-powered suggestion for the user
type ExecutionSuggestion struct {
	Type        string // info, warning, action, link
	Title       string
	Description string
	ActionKey   string // Keyboard shortcut
	Priority    string // low, medium, high
}

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

	// Post-completion state
	projectPath       string // Path to the cloned project
	showRunOption     bool   // Whether to show run option after completion
	runOptionSelected int    // 0=Run, 1=Open folder, 2=Back to dashboard

	// Project running state
	isRunningProject bool               // Whether we're running the project
	runningCmd       *exec.Cmd          // The running command
	runCtx           context.Context    // Context for the running project
	runCancel        context.CancelFunc // Cancel function for running project

	// Port management for running projects
	currentPort     int  // Current port being used
	portRetryCount  int  // Number of port retries attempted
	maxPortRetries  int  // Maximum port retry attempts
	detectedPortErr bool // Whether a port error was detected

	// Preview and summary state
	previewURL     string                 // Detected preview URL for web projects
	previewPort    int                    // Detected preview port
	projectType    string                 // Detected project type (nodejs, python, go, etc.)
	suggestions    []ExecutionSuggestion  // AI suggestions for the user
	isWebProject   bool                   // Whether this is a web project
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
		// Use mode from state if set, otherwise use default from config
		mode := executor.ExecutionMode(e.state.Container.Config().DefaultMode)
		if e.state.ExecutionMode != "" {
			mode = executor.ExecutionMode(e.state.ExecutionMode)
		}

		result, err := setupUC.Execute(ctx, project.SetupProjectInput{
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
			// Get project path from result
			projectPath := ""
			if result != nil && result.Project != nil && result.Project.LocalPath() != nil {
				projectPath = result.Project.LocalPath().String()
			}
			progressChan <- ExecutionCompleteMsg{
				Duration:    time.Since(e.startTime),
				ProjectPath: projectPath,
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

		// Handle run option selection when completed
		if e.showRunOption {
			switch msg.String() {
			case "up", "k":
				if e.runOptionSelected > 0 {
					e.runOptionSelected--
				}
				return e, nil
			case "down", "j":
				if e.runOptionSelected < 4 {
					e.runOptionSelected++
				}
				return e, nil
			case "enter":
				switch e.runOptionSelected {
				case 0: // Run project
					if e.projectPath != "" {
						return e, e.runProject()
					}
					return e, nil
				case 1: // Open folder
					if e.projectPath != "" {
						return e, e.openFolder()
					}
					return e, nil
				case 2: // Open in VS Code
					if e.projectPath != "" {
						return e, e.openInVSCode()
					}
					return e, nil
				case 3: // Show setup instructions
					e.showSetupInstructions()
					return e, nil
				case 4: // Back to dashboard
					return e, tui.SwitchView(tui.ViewDashboard)
				}
				return e, nil
			case "r", "R":
				e.runOptionSelected = 0
				if e.projectPath != "" {
					return e, e.runProject()
				}
				return e, nil
			case "o", "O":
				e.runOptionSelected = 1
				if e.projectPath != "" {
					return e, e.openFolder()
				}
				return e, nil
			case "v", "V":
				e.runOptionSelected = 2
				if e.projectPath != "" {
					return e, e.openInVSCode()
				}
				return e, nil
			case "i", "I":
				e.runOptionSelected = 3
				e.showSetupInstructions()
				return e, nil
			case "d", "D":
				return e, tui.SwitchView(tui.ViewDashboard)
			case "esc", "q":
				return e, tui.SwitchView(tui.ViewDashboard)
			}
		}

		switch msg.String() {
		case "esc", "q":
			// Allow quit at any time, just switch back to dashboard or exit
			return e, tea.Quit
		case "x", "X":
			// Stop running project
			if e.isRunningProject && e.runCancel != nil {
				e.runCancel() // Cancel the running project context
				e.outputLines = append(e.outputLines, "")
				e.outputLines = append(e.outputLines, "⚠️  Stopping project...")
				return e, e.waitForNextMessage()
			}
			// Cancel running execution (setup phase)
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
		// Check for preview URL in output
		if url, port := detectPreviewURL(msg.Line); url != "" {
			e.previewURL = url
			e.previewPort = port
			// Regenerate suggestions with new URL
			e.generateSuggestions()
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
		e.showRunOption = true
		duration := time.Since(e.startTime)

		// Store project path from message if available
		if msg.ProjectPath != "" {
			e.projectPath = msg.ProjectPath
			// Detect project type
			e.projectType = detectProjectType(e.projectPath)
		}

		// Mark all steps as complete (since execution succeeded, all steps are done)
		for i := range e.steps {
			if e.steps[i].Status == StepRunning || e.steps[i].Status == StepPending {
				e.steps[i].Status = StepComplete
				e.steps[i].Progress = 1.0
			}
		}
		e.currentStep = len(e.steps) - 1

		// Generate suggestions for post-completion
		e.generateSuggestions()

		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, fmt.Sprintf("✅ Project setup completed in %s", formatDuration(duration)))
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "What would you like to do next?")
		return e, nil

	case ExecutionErrorMsg:
		// Execution failed
		e.failed = true
		e.errorMsg = msg.Error.Error()
		if msg.Step >= 0 && msg.Step < len(e.steps) {
			e.steps[msg.Step].Status = StepFailed
		}
		// Generate failure suggestions
		e.generateSuggestions()
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

	case ProjectStoppedMsg:
		// Running project has stopped
		e.isRunningProject = false
		e.runningCmd = nil
		e.showRunOption = true // Show run options again
		// Reset port retry state
		e.portRetryCount = 0
		e.currentPort = 0
		e.detectedPortErr = false
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "─────────────────────────────────────────────────────")
		e.outputLines = append(e.outputLines, "")
		return e, nil

	case PortRetryMsg:
		// Port conflict detected, retry with a different port
		e.portRetryCount++
		e.currentPort = msg.NewPort
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, fmt.Sprintf("🔄 Attempting retry %d/%d with port %d...", e.portRetryCount, e.maxPortRetries, msg.NewPort))
		e.outputLines = append(e.outputLines, "")

		// Cancel current process if still running
		if e.runCancel != nil {
			e.runCancel()
		}

		// Small delay before retry
		return e, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return portRetryTickMsg{port: msg.NewPort}
		})

	case portRetryTickMsg:
		// Actually perform the retry
		return e, e.runProjectWithPort(msg.port)

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
	// Special view when running a project
	if e.isRunningProject {
		return e.renderRunningProjectView()
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
	} else if e.showRunOption {
		// Show run options after completion with summary
		runOptions := e.renderRunOptions()
		summary := e.renderSummary()
		sections = append(sections, "", output)
		if summary != "" {
			sections = append(sections, "", summary)
		}
		sections = append(sections, "", runOptions)
		runFooter := "[↑↓] Select  [Enter] Confirm  [R] Run  [O] Open Folder  [D] Dashboard"
		sections = append(sections, "", runFooter)
	} else if e.failed {
		// Show summary with suggestions after failure
		summary := e.renderSummary()
		sections = append(sections, "", output)
		if summary != "" {
			sections = append(sections, "", summary)
		}
		sections = append(sections, "", footer)
	} else {
		sections = append(sections, "", output, "", footer)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderRunningProjectView renders the view when a project is actively running
func (e *ExecutionView) renderRunningProjectView() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("2")). // Green
		Padding(1).
		Render("🚀 Project Running")

	// Regenerate suggestions to include preview URL if detected
	e.generateSuggestions()

	output := e.renderOutput()
	previewBanner := e.renderPreviewBanner()
	summary := e.renderSummary()

	autoScrollStatus := "OFF"
	if e.autoScroll {
		autoScrollStatus = "ON"
	}
	footer := fmt.Sprintf("[X] Stop Project  [Q] Quit  [↑↓] Scroll  [Home/End] Jump  [F] Auto-scroll: %s", autoScrollStatus)

	sections := []string{header, ""}

	// Show preview banner if URL detected
	if previewBanner != "" {
		sections = append(sections, previewBanner)
	}

	sections = append(sections, output)

	// Show summary with suggestions
	if summary != "" {
		sections = append(sections, "", summary)
	}

	sections = append(sections, "", footer)

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

// renderSuggestions renders the AI-powered suggestions section
func (e *ExecutionView) renderSuggestions() string {
	if len(e.suggestions) == 0 {
		return ""
	}

	lines := []string{"💡 Suggestions:", ""}

	// Styles for different suggestion types
	linkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // Green
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // Cyan
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // Blue
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // Yellow

	for _, sug := range e.suggestions {
		var icon string
		var style lipgloss.Style
		switch sug.Type {
		case "link":
			icon = "🔗"
			style = linkStyle
		case "info":
			icon = "ℹ️ "
			style = infoStyle
		case "action":
			icon = "⚡"
			style = actionStyle
		case "warning":
			icon = "⚠️ "
			style = warnStyle
		default:
			icon = "•"
			style = infoStyle
		}

		line := fmt.Sprintf("  %s %s", icon, sug.Title)
		if sug.ActionKey != "" {
			line += fmt.Sprintf(" [%s]", sug.ActionKey)
		}
		lines = append(lines, style.Render(line))
		if sug.Description != "" {
			lines = append(lines, style.Render(fmt.Sprintf("     %s", sug.Description)))
		}
	}

	return strings.Join(lines, "\n")
}

// renderPreviewBanner renders a banner when a preview URL is detected
func (e *ExecutionView) renderPreviewBanner() string {
	if e.previewURL == "" || !e.isRunningProject {
		return ""
	}

	bannerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("2")).
		Padding(0, 2)

	urlStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	lines := []string{
		"",
		bannerStyle.Render("🚀 Application Running!"),
		"",
		fmt.Sprintf("   Your application is live at: %s", urlStyle.Render(e.previewURL)),
		"",
		"   📋 Open this URL in your browser to view the application",
		"   ⚠️  This is only accessible from your local machine",
		"",
	}

	return strings.Join(lines, "\n")
}

// renderSummary renders the execution summary section
func (e *ExecutionView) renderSummary() string {
	if !e.completed && !e.failed && !e.isRunningProject {
		return ""
	}

	var lines []string

	// Summary header
	summaryStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2)

	var statusLine string
	if e.isRunningProject {
		statusLine = "🚀 Project Status: Running"
	} else if e.completed {
		statusLine = "✅ Project Status: Setup Complete"
	} else if e.failed {
		statusLine = "❌ Project Status: Failed"
	}

	lines = append(lines, statusLine)
	lines = append(lines, "")

	// Project info
	if e.projectType != "" {
		typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
		lines = append(lines, fmt.Sprintf("   Project Type: %s", typeStyle.Render(e.projectType)))
	}

	if e.projectPath != "" {
		pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		lines = append(lines, fmt.Sprintf("   Location: %s", pathStyle.Render(e.projectPath)))
	}

	if e.previewURL != "" {
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		lines = append(lines, fmt.Sprintf("   Preview: %s", urlStyle.Render(e.previewURL)))
	}

	duration := time.Since(e.startTime)
	lines = append(lines, fmt.Sprintf("   Duration: %s", formatDuration(duration)))

	// Milestone summary
	completedSteps := 0
	for _, step := range e.steps {
		if step.Status == StepComplete {
			completedSteps++
		}
	}
	lines = append(lines, fmt.Sprintf("   Milestones: %d/%d completed", completedSteps, len(e.steps)))

	// Suggestions section
	if len(e.suggestions) > 0 {
		lines = append(lines, "")
		lines = append(lines, "─────────────────────────────────")
		for _, sug := range e.suggestions {
			icon := "•"
			switch sug.Type {
			case "link":
				icon = "🔗"
			case "info":
				icon = "ℹ️ "
			case "action":
				icon = "⚡"
			case "warning":
				icon = "⚠️ "
			}
			if sug.ActionKey != "" {
				lines = append(lines, fmt.Sprintf("   %s %s [%s]", icon, sug.Title, sug.ActionKey))
			} else {
				lines = append(lines, fmt.Sprintf("   %s %s", icon, sug.Title))
			}
		}
	}

	// Next steps for running projects
	if e.isRunningProject && e.previewURL != "" {
		lines = append(lines, "")
		lines = append(lines, "─────────────────────────────────")
		lines = append(lines, "   Next Steps:")
		lines = append(lines, fmt.Sprintf("   1. Open %s in your browser", e.previewURL))
		lines = append(lines, "   2. Test your application")
		lines = append(lines, "   3. Press [X] to stop when done")
	}

	return summaryStyle.Render(strings.Join(lines, "\n"))
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

// detectPreviewURL extracts localhost/127.0.0.1 URLs from a log line
func detectPreviewURL(line string) (string, int) {
	// Patterns to match localhost URLs
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`https?://localhost[:\d]*/?\S*`),
		regexp.MustCompile(`https?://127\.0\.0\.1[:\d]*/?\S*`),
		regexp.MustCompile(`https?://0\.0\.0\.0[:\d]*/?\S*`),
	}

	for _, pattern := range patterns {
		if match := pattern.FindString(line); match != "" {
			// Extract port from URL
			portMatch := regexp.MustCompile(`:(\d+)`).FindStringSubmatch(match)
			port := 0
			if len(portMatch) > 1 {
				fmt.Sscanf(portMatch[1], "%d", &port)
			}
			return match, port
		}
	}
	return "", 0
}

// detectProjectType detects the project type from the project path
func detectProjectType(projectPath string) string {
	if fileExists(projectPath + "/package.json") {
		return "nodejs"
	}
	if fileExists(projectPath + "/go.mod") {
		return "go"
	}
	if fileExists(projectPath+"/requirements.txt") || fileExists(projectPath+"/setup.py") || fileExists(projectPath+"/pyproject.toml") {
		return "python"
	}
	if fileExists(projectPath + "/Cargo.toml") {
		return "rust"
	}
	if fileExists(projectPath + "/pom.xml") {
		return "java"
	}
	if fileExists(projectPath+"/docker-compose.yml") || fileExists(projectPath+"/docker-compose.yaml") {
		return "docker"
	}
	return "unknown"
}

// generateSuggestions generates AI-powered suggestions based on current state
func (e *ExecutionView) generateSuggestions() {
	e.suggestions = nil

	// Detect project type if not set
	if e.projectType == "" && e.projectPath != "" {
		e.projectType = detectProjectType(e.projectPath)
	}

	// Check if it's a web project
	webTypes := map[string]bool{"nodejs": true, "python": true, "go": true, "rust": true, "java": true}
	e.isWebProject = webTypes[e.projectType]

	if e.isRunningProject {
		// Suggestions while running
		if e.previewURL != "" {
			e.suggestions = append(e.suggestions, ExecutionSuggestion{
				Type:        "link",
				Title:       "View Your Application",
				Description: fmt.Sprintf("Open %s in your browser", e.previewURL),
				ActionKey:   "",
				Priority:    "high",
			})
		}
		e.suggestions = append(e.suggestions, ExecutionSuggestion{
			Type:        "info",
			Title:       "Monitor Output",
			Description: "Watch for errors or success messages in the terminal",
			Priority:    "medium",
		})
		e.suggestions = append(e.suggestions, ExecutionSuggestion{
			Type:        "action",
			Title:       "Stop Project",
			Description: "Press [X] to stop the running application",
			ActionKey:   "X",
			Priority:    "low",
		})
	} else if e.completed {
		// Suggestions after completion
		e.suggestions = append(e.suggestions, ExecutionSuggestion{
			Type:        "action",
			Title:       "Run Project",
			Description: "Start the application with detected command",
			ActionKey:   "R",
			Priority:    "high",
		})
		e.suggestions = append(e.suggestions, ExecutionSuggestion{
			Type:        "action",
			Title:       "Open Folder",
			Description: "Open project in file manager",
			ActionKey:   "O",
			Priority:    "medium",
		})
		if e.isWebProject {
			e.suggestions = append(e.suggestions, ExecutionSuggestion{
				Type:        "info",
				Title:       "Web Project Detected",
				Description: "After running, look for the localhost URL in the output",
				Priority:    "low",
			})
		}
	} else if e.failed {
		// Suggestions after failure
		e.suggestions = append(e.suggestions, ExecutionSuggestion{
			Type:        "warning",
			Title:       "Check Errors",
			Description: "Review the error messages above to understand what went wrong",
			Priority:    "high",
		})
		e.suggestions = append(e.suggestions, ExecutionSuggestion{
			Type:        "action",
			Title:       "Back to Dashboard",
			Description: "Return to try a different project",
			ActionKey:   "D",
			Priority:    "medium",
		})
	}
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
	Project     interface{} // *entities.Project
	Duration    time.Duration
	ProjectPath string // Path to the cloned project directory
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

// PortRetryMsg signals that we should retry with a different port
type PortRetryMsg struct {
	NewPort int
	Command string
}

// portRetryTickMsg is used internally for delayed port retry
type portRetryTickMsg struct {
	port int
}

// loadProjectAnalysis loads the AI analysis for the current project
func (e *ExecutionView) loadProjectAnalysis() *entities.Analysis {
	if e.projectPath == "" {
		return nil
	}

	// Load project from repository by matching path
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	projects, err := e.state.Container.ProjectRepository().List(ctx, repository.ProjectFilter{})
	if err != nil {
		return nil
	}

	// Find project matching this path
	for _, project := range projects {
		if project.LocalPath() != nil && project.LocalPath().String() == e.projectPath {
			return project.Analysis()
		}
	}
	return nil
}

// runProject runs the project using detected start command within the TUI
func (e *ExecutionView) runProject() tea.Cmd {
	return e.runProjectWithPort(0) // Start with default port
}

// runProjectWithPort runs the project with a specific port (0 means default)
func (e *ExecutionView) runProjectWithPort(port int) tea.Cmd {
	if e.projectPath == "" {
		e.outputLines = append(e.outputLines, "❌ No project path available")
		return nil
	}

	// Detect the start command
	startCmd := detectProjectStartCommand(e.projectPath)
	if startCmd == "" {
		e.outputLines = append(e.outputLines, "❌ Could not detect start command. Check the project README.")
		return nil
	}

	// Modify command if we have a specific port
	if port > 0 {
		startCmd = modifyCommandWithPort(startCmd, port)
		e.currentPort = port
	}

	// Initialize port retry settings
	if e.maxPortRetries == 0 {
		e.maxPortRetries = 5
	}

	// Hide run options and show running state
	e.showRunOption = false
	e.isRunningProject = true
	e.detectedPortErr = false

	// Clear previous output only on first attempt
	if e.portRetryCount == 0 {
		e.outputLines = []string{}
		e.outputLines = append(e.outputLines, "")
	}

	e.outputLines = append(e.outputLines, fmt.Sprintf("🚀 Running project: %s", e.projectPath))
	e.outputLines = append(e.outputLines, fmt.Sprintf("   Command: %s", startCmd))
	if port > 0 {
		e.outputLines = append(e.outputLines, fmt.Sprintf("   Port: %d", port))
	}
	e.outputLines = append(e.outputLines, "")
	e.outputLines = append(e.outputLines, "─────────────────────────────────────────────────────")
	e.outputLines = append(e.outputLines, "")

	// Create cancellable context
	e.runCtx, e.runCancel = context.WithCancel(context.Background())

	// Create progress channel for output
	outputChan := make(chan tea.Msg, 100)
	e.progressChan = outputChan

	// Parse the command - handle environment variables properly
	var cmdParts []string
	var envVars []string

	// Split command and detect env vars
	rawParts := strings.Fields(startCmd)
	for _, part := range rawParts {
		if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
			envVars = append(envVars, part)
		} else {
			cmdParts = append(cmdParts, part)
		}
	}

	if len(cmdParts) == 0 {
		e.outputLines = append(e.outputLines, "❌ Invalid command")
		return nil
	}

	parts := cmdParts

	// Track port errors in output
	portErrorDetected := false
	currentPort := e.currentPort

	// Start the command in a goroutine
	go func() {
		defer close(outputChan)

		cmd := exec.CommandContext(e.runCtx, parts[0], parts[1:]...)
		cmd.Dir = e.projectPath

		// Add environment variables
		cmd.Env = os.Environ()
		for _, envVar := range envVars {
			cmd.Env = append(cmd.Env, envVar)
		}

		e.runningCmd = cmd

		// Create pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			outputChan <- ExecutionOutputMsg{Line: fmt.Sprintf("❌ Error creating stdout pipe: %v", err)}
			outputChan <- ProjectStoppedMsg{}
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			outputChan <- ExecutionOutputMsg{Line: fmt.Sprintf("❌ Error creating stderr pipe: %v", err)}
			outputChan <- ProjectStoppedMsg{}
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			// Check if the error itself is a port issue
			if isPortError(err.Error()) {
				portErrorDetected = true
			}
			outputChan <- ExecutionOutputMsg{Line: fmt.Sprintf("❌ Error starting command: %v", err)}
			if portErrorDetected && e.portRetryCount < e.maxPortRetries {
				nextPort := getNextPort(currentPort)
				outputChan <- PortRetryMsg{NewPort: nextPort}
			} else {
				outputChan <- ProjectStoppedMsg{}
			}
			return
		}

		// Read stdout in a goroutine with port error detection
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				select {
				case <-e.runCtx.Done():
					return
				default:
					// Check for port errors
					if isPortError(line) && !portErrorDetected {
						portErrorDetected = true
						e.detectedPortErr = true
					}
					outputChan <- ExecutionOutputMsg{Line: line}
				}
			}
		}()

		// Read stderr in a goroutine with port error detection
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				select {
				case <-e.runCtx.Done():
					return
				default:
					// Check for port errors
					if isPortError(line) && !portErrorDetected {
						portErrorDetected = true
						e.detectedPortErr = true
					}
					outputChan <- ExecutionOutputMsg{Line: line}
				}
			}
		}()

		// Wait for command to finish
		err = cmd.Wait()
		if e.runCtx.Err() == context.Canceled {
			outputChan <- ExecutionOutputMsg{Line: ""}
			outputChan <- ExecutionOutputMsg{Line: "🛑 Project stopped by user"}
			outputChan <- ProjectStoppedMsg{}
		} else if err != nil {
			outputChan <- ExecutionOutputMsg{Line: ""}

			// If we detected a port error and haven't exceeded retries, suggest retry
			if (portErrorDetected || e.detectedPortErr) && e.portRetryCount < e.maxPortRetries {
				nextPort := getNextPort(currentPort)
				outputChan <- ExecutionOutputMsg{Line: fmt.Sprintf("⚠️ Port conflict detected! Retrying on port %d...", nextPort)}
				outputChan <- PortRetryMsg{NewPort: nextPort}
			} else {
				outputChan <- ExecutionOutputMsg{Line: fmt.Sprintf("❌ Process exited with error: %v", err)}
				outputChan <- ProjectStoppedMsg{}
			}
		} else {
			outputChan <- ExecutionOutputMsg{Line: ""}
			outputChan <- ExecutionOutputMsg{Line: "✅ Process completed successfully"}
			outputChan <- ProjectStoppedMsg{}
		}
	}()

	// Start listening to output
	return e.waitForNextMessage()
}

// ProjectStoppedMsg signals that the running project has stopped
type ProjectStoppedMsg struct{}

// openFolder opens the project folder in the system file manager
func (e *ExecutionView) openFolder() tea.Cmd {
	return func() tea.Msg {
		if e.projectPath == "" {
			return ExecutionOutputMsg{Line: "No project path available"}
		}
		// Use the tui package's message type
		return tui.OpenFolderMsg{Path: e.projectPath}
	}
}

// detectProjectStartCommand detects the start command for a project
func detectProjectStartCommand(projectPath string) string {
	// Check for package.json (Node.js)
	if fileExists(projectPath + "/package.json") {
		return detectNodeStart(projectPath)
	}

	// Check for go.mod (Go)
	if fileExists(projectPath + "/go.mod") {
		return detectGoStart(projectPath)
	}

	// Check for Python
	if fileExists(projectPath+"/requirements.txt") || fileExists(projectPath+"/setup.py") || fileExists(projectPath+"/pyproject.toml") {
		return detectPythonStart(projectPath)
	}

	// Check for Cargo.toml (Rust)
	if fileExists(projectPath + "/Cargo.toml") {
		return "cargo run"
	}

	// Check for pom.xml (Java/Maven)
	if fileExists(projectPath + "/pom.xml") {
		return "mvn spring-boot:run"
	}

	// Check for docker-compose
	if fileExists(projectPath+"/docker-compose.yml") || fileExists(projectPath+"/docker-compose.yaml") {
		return "docker-compose up"
	}

	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func detectNodeStart(projectPath string) string {
	content, err := os.ReadFile(projectPath + "/package.json")
	if err != nil {
		return "npm start"
	}

	contentStr := string(content)

	// Detect package manager
	pm := "npm"
	if fileExists(projectPath + "/pnpm-lock.yaml") {
		pm = "pnpm"
	} else if fileExists(projectPath + "/yarn.lock") {
		pm = "yarn"
	}

	if strings.Contains(contentStr, `"dev"`) {
		return pm + " run dev"
	}
	if strings.Contains(contentStr, `"start"`) {
		return pm + " start"
	}
	return pm + " start"
}

func detectGoStart(projectPath string) string {
	if fileExists(projectPath + "/main.go") {
		return "go run main.go"
	}
	// Check cmd directory
	if entries, err := os.ReadDir(projectPath + "/cmd"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				return "go run ./cmd/" + entry.Name()
			}
		}
	}
	return "go run ."
}

func detectPythonStart(projectPath string) string {
	entryPoints := []string{"main.py", "app.py", "run.py", "server.py", "manage.py"}
	for _, ep := range entryPoints {
		if fileExists(projectPath + "/" + ep) {
			if ep == "manage.py" {
				return "python manage.py runserver"
			}
			return "python " + ep
		}
	}
	return "python main.py"
}

// Port error patterns to detect
var portErrorPatterns = []string{
	"EADDRINUSE",
	"address already in use",
	"port is already in use",
	"Address already in use",
	"bind: address already in use",
	"listen EADDRINUSE",
	"port already allocated",
	"Only one usage of each socket address",
	"Failed to listen on",
	"unable to create listener",
	"Error: listen EADDRINUSE",
}

// isPortError checks if a line indicates a port conflict
func isPortError(line string) bool {
	lineLower := strings.ToLower(line)
	for _, pattern := range portErrorPatterns {
		if strings.Contains(lineLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// getNextPort returns the next available port to try
func getNextPort(currentPort int) int {
	if currentPort == 0 {
		return 3001 // Start with 3001 if no port specified
	}
	return currentPort + 1
}

// modifyCommandWithPort modifies a command to use a specific port
func modifyCommandWithPort(cmd string, port int) string {
	portStr := fmt.Sprintf("%d", port)

	// Node.js / npm / pnpm / yarn patterns
	if strings.Contains(cmd, "npm") || strings.Contains(cmd, "pnpm") || strings.Contains(cmd, "yarn") {
		// Check if it already has PORT env var
		if !strings.Contains(cmd, "PORT=") {
			// For dev servers, set PORT env var
			if strings.Contains(cmd, "dev") || strings.Contains(cmd, "start") {
				return fmt.Sprintf("PORT=%s %s", portStr, cmd)
			}
		}
		return cmd
	}

	// Next.js specific - add -p flag
	if strings.Contains(cmd, "next") {
		if !strings.Contains(cmd, "-p ") && !strings.Contains(cmd, "--port") {
			return cmd + " -p " + portStr
		}
		return cmd
	}

	// Vite - add --port flag
	if strings.Contains(cmd, "vite") {
		if !strings.Contains(cmd, "--port") {
			return cmd + " --port " + portStr
		}
		return cmd
	}

	// Python/Django - modify runserver port
	if strings.Contains(cmd, "runserver") {
		// Replace default port or add port
		if strings.Contains(cmd, "runserver 0.0.0.0:") || strings.Contains(cmd, "runserver localhost:") {
			// Already has host:port, replace port
			return cmd // Keep as-is for now
		}
		if strings.HasSuffix(cmd, "runserver") {
			return cmd + " 0.0.0.0:" + portStr
		}
		return cmd
	}

	// Python/Flask - set port env var
	if strings.Contains(cmd, "flask") {
		if !strings.Contains(cmd, "FLASK_RUN_PORT") {
			return fmt.Sprintf("FLASK_RUN_PORT=%s %s", portStr, cmd)
		}
		return cmd
	}

	// Python/FastAPI/Uvicorn - add --port flag
	if strings.Contains(cmd, "uvicorn") {
		if !strings.Contains(cmd, "--port") {
			return cmd + " --port " + portStr
		}
		return cmd
	}

	// Go - typically needs code change, but some accept -port flag
	if strings.Contains(cmd, "go run") {
		// Can't easily modify Go port, suggest manual change
		return cmd
	}

	// Generic fallback - try PORT env var
	if !strings.Contains(cmd, "PORT=") {
		return fmt.Sprintf("PORT=%s %s", portStr, cmd)
	}

	return cmd
}

// renderRunOptions renders the post-completion run options menu
func (e *ExecutionView) renderRunOptions() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")). // Green
		Padding(1, 2)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("2")).
		Bold(true)
	normalStyle := lipgloss.NewStyle()

	options := []string{
		"[R] Run project",
		"[O] Open folder in file manager",
		"[V] Open in VS Code",
		"[I] Show setup instructions",
		"[D] Back to dashboard",
	}

	var lines []string
	lines = append(lines, "What would you like to do next?")
	lines = append(lines, "")

	if e.projectPath != "" {
		lines = append(lines, fmt.Sprintf("Project: %s", e.projectPath))
		startCmd := detectProjectStartCommand(e.projectPath)
		if startCmd != "" {
			lines = append(lines, fmt.Sprintf("Start command: %s", startCmd))
		}
		lines = append(lines, "")
	}

	for i, opt := range options {
		if i == e.runOptionSelected {
			lines = append(lines, selectedStyle.Render(" → "+opt+" "))
		} else {
			lines = append(lines, normalStyle.Render("   "+opt))
		}
	}

	return boxStyle.Render(strings.Join(lines, "\n"))
}

// openInVSCode attempts to open the project in VS Code
func (e *ExecutionView) openInVSCode() tea.Cmd {
	return func() tea.Msg {
		if e.projectPath == "" {
			return ExecutionOutputMsg{Line: "No project path available"}
		}

		// Try to open VS Code using the 'code' command
		var cmd *exec.Cmd
		if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
			cmd = exec.Command("cmd", "/C", "code", e.projectPath)
		} else {
			cmd = exec.Command("code", e.projectPath)
		}

		if err := cmd.Start(); err != nil {
			return ExecutionOutputMsg{Line: fmt.Sprintf("⚠️  Could not open VS Code: %v\n   Try running: code %s", err, e.projectPath)}
		}

		return ExecutionOutputMsg{Line: "✅ Opening project in VS Code..."}
	}
}

// showSetupInstructions displays step-by-step setup instructions
func (e *ExecutionView) showSetupInstructions() {
	e.outputLines = append(e.outputLines, "")
	e.outputLines = append(e.outputLines, "════════════════════════════════════════════════════════════")
	e.outputLines = append(e.outputLines, "📋 SETUP INSTRUCTIONS - How to Run This Project")
	e.outputLines = append(e.outputLines, "════════════════════════════════════════════════════════════")
	e.outputLines = append(e.outputLines, "")

	// Get project type
	projectType := detectProjectType(e.projectPath)

	e.outputLines = append(e.outputLines, fmt.Sprintf("📁 Project Path: %s", e.projectPath))
	e.outputLines = append(e.outputLines, fmt.Sprintf("🔧 Project Type: %s", projectType))
	e.outputLines = append(e.outputLines, "")

	e.outputLines = append(e.outputLines, "📝 Steps to Run Manually:")
	e.outputLines = append(e.outputLines, "")

	e.outputLines = append(e.outputLines, fmt.Sprintf("1. Navigate to project directory:"))
	e.outputLines = append(e.outputLines, fmt.Sprintf("   cd %s", e.projectPath))
	e.outputLines = append(e.outputLines, "")

	switch projectType {
	case "nodejs":
		e.outputLines = append(e.outputLines, "2. Install dependencies (if not already done):")
		e.outputLines = append(e.outputLines, "   npm install  # or: yarn install / pnpm install")
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "3. Run the project:")
		startCmd := detectProjectStartCommand(e.projectPath)
		if startCmd != "" {
			e.outputLines = append(e.outputLines, fmt.Sprintf("   %s", startCmd))
		} else {
			e.outputLines = append(e.outputLines, "   npm start  # or: npm run dev")
		}

	case "python":
		e.outputLines = append(e.outputLines, "2. Create virtual environment (recommended):")
		e.outputLines = append(e.outputLines, "   python -m venv venv")
		e.outputLines = append(e.outputLines, "   source venv/bin/activate  # Linux/Mac")
		e.outputLines = append(e.outputLines, "   # or: venv\\Scripts\\activate  # Windows")
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "3. Install dependencies:")
		e.outputLines = append(e.outputLines, "   pip install -r requirements.txt")
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "4. Run the project:")
		e.outputLines = append(e.outputLines, "   python main.py  # or: python app.py")

	case "go":
		e.outputLines = append(e.outputLines, "2. Download dependencies:")
		e.outputLines = append(e.outputLines, "   go mod download")
		e.outputLines = append(e.outputLines, "")
		e.outputLines = append(e.outputLines, "3. Run the project:")
		e.outputLines = append(e.outputLines, "   go run .  # or: go run main.go")

	case "rust":
		e.outputLines = append(e.outputLines, "2. Build and run:")
		e.outputLines = append(e.outputLines, "   cargo run")

	case "java":
		e.outputLines = append(e.outputLines, "2. Build and run:")
		e.outputLines = append(e.outputLines, "   mvn spring-boot:run  # or: mvn package && java -jar target/*.jar")

	case "docker":
		e.outputLines = append(e.outputLines, "2. Start with Docker Compose:")
		e.outputLines = append(e.outputLines, "   docker-compose up")

	default:
		e.outputLines = append(e.outputLines, "2. Check the README.md file for specific instructions.")
	}

	e.outputLines = append(e.outputLines, "")
	e.outputLines = append(e.outputLines, "💡 Tips:")
	e.outputLines = append(e.outputLines, "   • Press [V] to open in VS Code")
	e.outputLines = append(e.outputLines, "   • Press [O] to open folder in file manager")
	e.outputLines = append(e.outputLines, "   • Press [R] to run the project automatically")
	e.outputLines = append(e.outputLines, "")
	e.outputLines = append(e.outputLines, "════════════════════════════════════════════════════════════")
	e.outputLines = append(e.outputLines, "")
}
