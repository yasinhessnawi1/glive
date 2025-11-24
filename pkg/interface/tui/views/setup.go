package views

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/domain/values"
	"github.com/glive/interface/tui"
	"github.com/glive/interface/tui/components"
)

// SetupView represents the project setup wizard view
type SetupView struct {
	state               *tui.AppState
	step                int
	textInput           *components.TextInput
	errorMsg            string
	submitting          bool
	force               bool            // Force re-clone even if project exists
	validating          bool            // Whether validation is in progress
	lastInputTime       time.Time       // For debouncing validation
	checkedClipboard    map[string]bool // Cache of checked clipboard values
	clipboardCheckCount int             // Number of clipboard checks performed
}

// NewSetupView creates a new setup view
func NewSetupView(state *tui.AppState) *SetupView {
	input := components.NewTextInput("username/repo or https://github.com/username/repo", state.Styles)
	input.Width = state.Width - 20
	if input.Width < 40 {
		input.Width = 40
	}
	input.Focus()

	return &SetupView{
		state:               state,
		step:                1,
		textInput:           input,
		force:               false, // Default to not forcing
		validating:          false,
		lastInputTime:       time.Now(),
		checkedClipboard:    make(map[string]bool),
		clipboardCheckCount: 0,
	}
}

// isLikelyGitHubURL performs a lightweight check without full validation
func isLikelyGitHubURL(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	// Quick checks that don't require regex compilation
	lower := strings.ToLower(text)

	// Check for common GitHub patterns without heavy regex
	if strings.Contains(lower, "github.com") {
		return true
	}

	// Check for owner/repo format (simple check: contains exactly one slash)
	if strings.Count(text, "/") == 1 && !strings.Contains(text, "://") && !strings.HasPrefix(text, "git@") {
		parts := strings.Split(text, "/")
		if len(parts) == 2 {
			owner := strings.TrimSpace(parts[0])
			repo := strings.TrimSpace(parts[1])
			// Basic validation: non-empty, reasonable length
			if len(owner) > 0 && len(owner) < 100 && len(repo) > 0 && len(repo) < 100 {
				return true
			}
		}
	}

	return false
}

// Init initializes the setup view
func (s *SetupView) Init() tea.Cmd {
	// Check clipboard immediately, then schedule delayed checks using tea.Tick
	// This avoids blocking the UI thread
	return tea.Batch(
		s.readClipboardCmd(), // Immediate check
		tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return ClipboardTickMsg{}
		}),
		tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
			return ClipboardTickMsg{}
		}),
		tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
			return ClipboardTickMsg{}
		}),
		tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
			return ClipboardTickMsg{}
		}),
	)
}

// ClipboardTickMsg is a message to trigger clipboard check
type ClipboardTickMsg struct{}

// readClipboardCmd returns a command that reads clipboard and checks for GitHub URL
// This runs in a goroutine to avoid blocking the UI
func (s *SetupView) readClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		// Run clipboard operations in a goroutine to avoid blocking
		resultChan := make(chan ClipboardReadMsg, 1)

		go func() {
			defer func() {
				// Recover from any panic in clipboard operations
				if r := recover(); r != nil {
					resultChan <- ClipboardReadMsg{URL: ""}
				}
			}()

			// Check if clipboard is available
			if !clipboardAvailable() {
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Initialize clipboard
			if err := initClipboard(); err != nil {
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Read clipboard content - only read text format
			content := readClipboard()
			if len(content) == 0 {
				// Clipboard might contain non-text content (image, file, etc.)
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Verify that the content is valid UTF-8 text
			// If clipboard contains binary data (image, file), this will fail
			if !utf8.Valid(content) {
				// Clipboard contains non-text content, skip it
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			text := string(content)

			// Additional check: ensure it's not just binary data masquerading as text
			// GitHub URLs should be reasonable length and contain printable characters
			if len(text) > 500 {
				// Too long to be a URL, likely binary data
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Check for control characters that shouldn't be in URLs
			hasControlChars := false
			for _, r := range text {
				if r < 32 && r != '\n' && r != '\r' && r != '\t' {
					hasControlChars = true
					break
				}
			}
			if hasControlChars {
				// Contains control characters, likely binary data
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Clean up the text
			text = strings.ReplaceAll(text, "\n", "")
			text = strings.ReplaceAll(text, "\r", "")
			text = strings.TrimSpace(text)

			if text == "" {
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Quick lightweight check first
			if !isLikelyGitHubURL(text) {
				resultChan <- ClipboardReadMsg{URL: ""}
				return
			}

			// Only do full validation if it looks promising
			repoURL, err := values.ParseRepoURL(text)
			if err == nil && repoURL != nil {
				// Only accept GitHub URLs
				if repoURL.Provider() == "github" {
					resultChan <- ClipboardReadMsg{URL: text, AlreadyChecked: false}
					return
				}
			}

			// Fallback: try NewURL
			url, err := values.NewURL(text)
			if err == nil {
				owner, name, err := url.ParseOwnerAndName()
				if err == nil && owner != "" && name != "" {
					urlStr := url.String()
					if strings.Contains(urlStr, "github.com") {
						resultChan <- ClipboardReadMsg{URL: text, AlreadyChecked: false}
						return
					}
				}
			}

			resultChan <- ClipboardReadMsg{URL: ""}
		}()

		// Wait for result with timeout to avoid blocking forever
		select {
		case result := <-resultChan:
			return result
		case <-time.After(50 * time.Millisecond):
			// Timeout - return empty to avoid blocking
			// The goroutine will continue but we won't wait for it
			return ClipboardReadMsg{URL: ""}
		}
	}
}

// Update handles messages
func (s *SetupView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	// if s.submitting {
	// 	// Don't process input while submitting
	// 	return s, nil
	// }

	switch msg := msg.(type) {
	case ClipboardTickMsg:
		// Trigger clipboard check on tick
		// Only check if we haven't found a URL yet and haven't checked too many times
		if s.clipboardCheckCount < 5 {
			s.clipboardCheckCount++
			return s, s.readClipboardCmd()
		}
		return s, nil

	case ClipboardReadMsg:
		// Handle clipboard read result
		// Only update if we don't already have a value or if the new URL is different
		if msg.URL != "" && !msg.AlreadyChecked {
			s.submitting = false
			// Check if we've already seen this URL
			if s.checkedClipboard[msg.URL] {
				// Already checked, skip
				return s, nil
			}

			// Mark as checked
			s.checkedClipboard[msg.URL] = true

			currentValue := strings.TrimSpace(s.textInput.GetValue())
			// Only set if input is empty or different from what we're setting
			if currentValue == "" || currentValue != msg.URL {
				s.textInput.SetValue(msg.URL)
				// Clear any error message when clipboard URL is loaded
				s.errorMsg = ""
				// Stop checking once we found a valid URL
				s.clipboardCheckCount = 10
			}
		}
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			s.submitting = false
			return s, tui.SwitchView(tui.ViewDashboard)
		case "f", "F":
			// Toggle force option
			s.force = !s.force
			return s, nil
		case "ctrl+v":
			// Check clipboard when user presses Ctrl+V
			// Reset check count to allow checking again
			s.clipboardCheckCount = 0
			s.submitting = false
			return s, s.readClipboardCmd()
		case "enter":
			// Reset submitting state to allow re-submission if previous attempt failed or stalled
			s.submitting = false

			// Validate and submit
			repoInput := strings.TrimSpace(s.textInput.GetValue())
			if repoInput == "" {
				s.errorMsg = "Please enter a repository URL or username/repo"
				return s, nil
			}

			// Set validating state to prevent input blocking
			s.validating = true
			s.errorMsg = ""

			// Validate URL format (this is quick, but we still mark as validating)
			url, err := values.NewURL(repoInput)
			if err != nil {
				s.validating = false
				s.errorMsg = fmt.Sprintf("Invalid repository URL: %v", err)
				return s, nil
			}

			// Clear error and set submitting state
			s.errorMsg = ""
			s.validating = false
			s.submitting = true

			// Store the URL and force flag in state and switch to execution view
			s.state.ProjectURL = url.String()
			s.state.ForceSetup = s.force

			// Switch to execution view
			return s, tui.SwitchView(tui.ViewExecution)
		case "tab":
			// Allow tab to work normally in input
			updatedInput, cmd := s.textInput.Update(msg)
			s.textInput = updatedInput
			return s, cmd
		default:
			// Reset submitting state on any other key press
			s.submitting = false
			// Pass all other keys to text input immediately - never block
			updatedInput, cmd := s.textInput.Update(msg)
			s.textInput = updatedInput
			// Clear error message when user starts typing again
			if s.errorMsg != "" {
				s.errorMsg = ""
			}
			return s, cmd
		}

	case components.PasteMsg:
		s.submitting = false
		updatedInput, cmd := s.textInput.Update(msg)
		s.textInput = updatedInput
		return s, cmd
	}

	// Pass messages to text input (including paste messages) - never block
	updatedInput, cmd := s.textInput.Update(msg)
	s.textInput = updatedInput
	if cmd != nil {
		return s, cmd
	}
	return s, nil
}

// View renders the setup view
func (s *SetupView) View() string {
	if s.state.Width < 80 || s.state.Height < 16 {
		return "Terminal too small. Please resize."
	}

	var sections []string

	// Header
	header := tui.RenderHeader(s.state, "New Project Setup")
	sections = append(sections, header)

	// Force option indicator
	forceStatus := "OFF"
	forceStyle := s.state.Styles.TextDim
	if s.force {
		forceStatus = "ON"
		forceStyle = s.state.Styles.Warning
	}
	forceIndicator := fmt.Sprintf("\nForce Re-clone: %s", forceStyle.Render(forceStatus))
	if s.force {
		forceIndicator += s.state.Styles.TextDim.Render(" (will re-clone even if project exists)")
	}

	// Instructions
	instructions := tui.RenderBox(
		s.state,
		"Enter Repository",
		fmt.Sprintf("Enter a GitHub repository URL or username/repo format:\n\n%s\n\n%s%s",
			s.textInput.Render(),
			func() string {
				if s.errorMsg != "" {
					return s.state.Styles.Error.Render("⚠ "+s.errorMsg) + "\n"
				}
				return s.state.Styles.TextDim.Render("Examples: vercel/next.js or https://github.com/vercel/next.js") + "\n"
			}(),
			forceIndicator,
		),
		s.state.Width-4,
	)
	sections = append(sections, instructions)

	// Footer
	footer := tui.RenderFooter(s.state, "[Enter] Submit • [F] Toggle Force • [Ctrl+V] Paste • [Esc] Cancel")
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// ClipboardReadMsg is a message containing clipboard read result
type ClipboardReadMsg struct {
	URL            string
	AlreadyChecked bool // Whether this URL was already checked
}
