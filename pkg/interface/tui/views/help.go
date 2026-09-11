package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/interface/tui"
)

// HelpView represents the help view
type HelpView struct {
	state           *tui.AppState
	selectedSection int
	viewportStart   int // Start index of visible sections
	sections        []HelpSection
}

// HelpSection represents a section of help content
type HelpSection struct {
	Title   string
	Content []string
}

// NewHelpView creates a new help view
func NewHelpView(state *tui.AppState) *HelpView {
	sections := []HelpSection{
		{
			Title: "Global Commands (Work Everywhere)",
			Content: []string{
				"[Esc] or [Q]   Return to dashboard or quit",
				"[?]            Show this help view from anywhere",
				"[↑↓] or [K/J]  Navigate up/down in lists",
				"[Enter]        Select/Execute/Confirm action",
			},
		},
		{
			Title: "Dashboard View Commands",
			Content: []string{
				"[N]            Create a new project (opens setup wizard)",
				"[L]            Open logs view to see project logs",
				"[S]            Open settings to configure GLive",
				"[R]            Refresh projects list and statistics",
				"[X]            Stop the last active/pending project",
				"[H] or [?]     Show help view",
				"[↑↓] or [K/J]  Navigate through project list",
				"[Enter]        Open selected project in execution view",
				"[Q]            Quit application",
				"",
				"The dashboard shows your recent projects, statistics,",
				"system health, and quick actions. Projects auto-refresh",
				"every 5 seconds to show latest status.",
			},
		},
		{
			Title: "Project Setup View Commands",
			Content: []string{
				"[Enter]        Submit GitHub URL and start setup",
				"[F]            Toggle Force Re-clone option",
				"[Ctrl+V]       Paste GitHub URL from clipboard",
				"[Esc]          Cancel and return to dashboard",
				"[Tab]          Normal tab behavior in input field",
				"",
				"Enter a GitHub repository URL in any format:",
				"• Full URL: https://github.com/user/repo",
				"• Short form: user/repo",
				"",
				"The setup wizard will automatically detect if you have",
				"a GitHub URL in your clipboard and pre-fill it.",
				"",
				"Force Re-clone: When enabled, will re-clone the repository",
				"even if it already exists in your workspace. Useful for",
				"getting the latest changes or resetting a project.",
			},
		},
		{
			Title: "Execution View Commands",
			Content: []string{
				"[↑↓] or [K/J]  Scroll up/down through execution output",
				"[F]            Toggle auto-scroll (follows latest output)",
				"[X]            Cancel running execution",
				"[Q] or [Esc]   Quit and return to dashboard",
				"",
				"This view shows real-time project setup progress with:",
				"• Execution timeline showing each step",
				"• Live output from commands",
				"• Progress bars and status indicators",
				"",
				"Steps include: URL parsing, cloning, security scanning,",
				"analysis, AI analysis, dependency installation, and completion.",
			},
		},
		{
			Title: "Settings View Commands",
			Content: []string{
				"[↑↓] or [K/J]  Navigate through settings list",
				"[Enter] or [E] Edit the selected setting",
				"[Esc]          Cancel editing or return to dashboard",
				"[Q]            Return to dashboard",
				"",
				"While editing:",
				"• [Enter]      Save the new value",
				"• [Esc]        Cancel editing",
				"",
				"For select-type settings (e.g., AI Provider, Mode):",
				"• [Enter]      Cycle through available options",
				"",
				"For boolean settings (e.g., Enable Sandbox):",
				"• [Enter]      Toggle between true/false",
				"",
				"For string/int settings (e.g., API Key, Workspace):",
				"• Type the new value and press [Enter] to save",
				"",
				"Settings are automatically saved when you confirm changes.",
			},
		},
		{
			Title: "Logs View Commands",
			Content: []string{
				"[↑↓] or [K/J]  Select different project from list",
				"[←→] or [H/L]  Scroll left/right through log entries",
				"[F]            Cycle filter: all → error → success → all",
				"[R]            Refresh project list and logs",
				"[Esc] or [Q]   Return to dashboard",
				"",
				"This view shows:",
				"• Project list on the left",
				"• Selected project's logs on the right",
				"• Filter options: all logs, errors only, success only",
				"",
				"Logs include project status changes, timestamps, and",
				"execution metadata. Useful for debugging failed setups.",
			},
		},
		{
			Title: "Recovery View Commands",
			Content: []string{
				"[↑↓] or [K/J]  Navigate through projects",
				"[R]            Refresh recovery information",
				"[Esc] or [Q]   Return to dashboard",
				"",
				"This view shows:",
				"• Recovery overview statistics",
				"• Available recovery strategies",
				"• Project recovery status for each project",
				"",
				"Recovery strategies include:",
				"• AI Auto-Fix: Automatically fixes command errors",
				"• Retry: Retries failed commands with backoff",
				"• Skip: Skips optional commands that fail",
				"• Rollback: Reverts changes if recovery fails",
				"• Missing Tool: Installs missing dependencies",
			},
		},
		{
			Title: "Help View Commands",
			Content: []string{
				"[↑↓] or [K/J]  Navigate through help sections",
				"[Home]         Jump to first section",
				"[End]          Jump to last section",
				"[Esc] or [Q]   Return to dashboard",
				"[?]            Toggle help (return to dashboard)",
				"",
				"Use arrow keys to browse through all available",
				"commands and their explanations for each view.",
			},
		},
		{
			Title: "Project Status Meanings",
			Content: []string{
				"pending        Project is queued, waiting to start",
				"cloning        Currently cloning repository from GitHub",
				"analyzing      Analyzing project structure and dependencies",
				"installing     Installing dependencies (npm, pip, etc.)",
				"running        Project is currently running",
				"ready          Project setup completed successfully",
				"failed         Setup failed (check logs for details)",
				"stopped        Project execution was manually stopped",
				"",
				"Status icons:",
				"○  Pending/Active",
				"✓  Ready/Success",
				"✗  Failed/Error",
				"⟳  Running/In Progress",
			},
		},
		{
			Title: "Tips & Best Practices",
			Content: []string{
				"• Projects are automatically saved to disk",
				"• Press [R] in dashboard to manually refresh",
				"• Completed projects show as 'ready' status",
				"• Failed projects can be retried by running them again",
				"• Settings are saved automatically when changed",
				"• API keys are stored securely in encrypted storage",
				"• Use [X] to stop stuck or unwanted project executions",
				"• Help is available from any view by pressing [?]",
				"• Dashboard auto-refreshes every 5 seconds",
				"• Execution view shows real-time progress with live output",
				"• Logs view helps debug setup issues",
				"• Settings view supports clipboard paste for API keys",
				"• Use [F] in project setup to toggle Force Re-clone option",
				"• Force Re-clone is useful for getting latest changes or",
				"  resetting a project that already exists",
			},
		},
	}

	return &HelpView{
		state:           state,
		selectedSection: 0,
		viewportStart:   0, // Start at the beginning
		sections:        sections,
	}
}

// Init initializes the help view
func (h *HelpView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (h *HelpView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reset viewport to top when terminal is resized
		h.viewportStart = 0
		h.selectedSection = 0
		return h, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "?":
			return h, tui.SwitchView(tui.ViewDashboard)
		case "up", "k":
			if h.selectedSection > 0 {
				h.selectedSection--
				h.adjustViewport()
			}
			return h, nil
		case "down", "j":
			if h.selectedSection < len(h.sections)-1 {
				h.selectedSection++
				h.adjustViewport()
			}
			return h, nil
		case "home":
			h.selectedSection = 0
			h.viewportStart = 0
			return h, nil
		case "end":
			h.selectedSection = len(h.sections) - 1
			h.adjustViewport()
			return h, nil
		}
	}
	return h, nil
}

// View renders the help view
func (h *HelpView) View() string {
	var sections []string

	// Header
	header := tui.RenderHeader(h.state, "Help & Keyboard Shortcuts")
	sections = append(sections, header)

	// Help content
	content := h.renderContent()
	sections = append(sections, content)

	// Footer
	footer := tui.RenderFooter(h.state, "[↑↓] Navigate • [Esc] or [Q] Return to Dashboard • [?] Toggle Help")
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// adjustViewport adjusts viewport to keep selected section visible
func (h *HelpView) adjustViewport() {
	if len(h.sections) == 0 {
		h.viewportStart = 0
		return
	}

	// Calculate available height for content
	// Reserve space: header (3) + footer (3) + box borders/padding (4) = ~10 lines
	availableHeight := h.state.Height - 10
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}

	// Calculate how many lines each section takes
	sectionHeights := make([]int, len(h.sections))
	totalLines := 0
	for i, section := range h.sections {
		// Title (1) + blank line (1) + content lines + spacing (1)
		sectionHeight := 1 + 1 + len(section.Content) + 1
		sectionHeights[i] = sectionHeight
		totalLines += sectionHeight
	}

	// If all content fits, show everything from the start
	if totalLines <= availableHeight {
		h.viewportStart = 0
		return
	}

	// Calculate which sections to show to keep selected section visible
	// Try to center the selected section when possible
	selectedStart := 0
	for i := 0; i < h.selectedSection; i++ {
		selectedStart += sectionHeights[i]
	}

	// Determine viewport start section
	// Try to show selected section in the middle third of the screen
	targetStart := selectedStart - (availableHeight / 3)
	if targetStart < 0 {
		targetStart = 0
	}

	// Find which section corresponds to targetStart
	currentLine := 0
	newViewportStart := 0
	for i := 0; i < len(h.sections); i++ {
		if currentLine >= targetStart {
			newViewportStart = i
			break
		}
		currentLine += sectionHeights[i]
	}

	h.viewportStart = newViewportStart

	// Ensure we don't scroll past the end
	if h.viewportStart > 0 {
		// Calculate if we're showing too much empty space at the bottom
		visibleLines := 0
		for i := h.viewportStart; i < len(h.sections); i++ {
			visibleLines += sectionHeights[i]
		}
		if visibleLines < availableHeight && h.viewportStart > 0 {
			// Scroll back up to fill the screen
			h.viewportStart--
		}
	}
}

// renderContent renders the help content
func (h *HelpView) renderContent() string {
	// Always start at 0 if we're at the first section
	if h.selectedSection == 0 {
		h.viewportStart = 0
	}

	// Adjust viewport
	h.adjustViewport()

	// Calculate available height for content
	availableHeight := h.state.Height - 10
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate how many lines each section takes and determine visible sections
	visibleSections := []int{}
	currentLines := 0
	for i := h.viewportStart; i < len(h.sections); i++ {
		sectionHeight := 1 + 1 + len(h.sections[i].Content) + 1
		if currentLines+sectionHeight <= availableHeight || len(visibleSections) == 0 {
			visibleSections = append(visibleSections, i)
			currentLines += sectionHeight
		} else {
			break
		}
	}

	// Render only visible sections
	start := h.viewportStart
	end := start + len(visibleSections)
	if end > len(h.sections) {
		end = len(h.sections)
	}

	visibleSectionsList := h.sections[start:end]
	lines := make([]string, 0)

	for i, section := range visibleSectionsList {
		actualIdx := start + i
		// Section title
		titleStyle := h.state.Styles.Title.Copy()
		if actualIdx == h.selectedSection {
			titleStyle = h.state.Styles.ButtonActive.Copy()
		}

		title := titleStyle.Render("▶ " + section.Title)
		lines = append(lines, title)
		lines = append(lines, "")

		// Section content
		for _, line := range section.Content {
			// Indent content
			indentedLine := "  " + line
			if actualIdx == h.selectedSection {
				// Highlight selected section content
				indentedLine = h.state.Styles.Text.Render(indentedLine)
			} else {
				indentedLine = h.state.Styles.TextDim.Render(indentedLine)
			}
			lines = append(lines, indentedLine)
		}

		// Add spacing between sections (except for last visible section)
		if i < len(visibleSectionsList)-1 {
			lines = append(lines, "")
		}
	}

	// Join all lines
	content := strings.Join(lines, "\n")

	// Add scroll indicator if needed
	if len(visibleSections) < len(h.sections) {
		scrollInfo := fmt.Sprintf("\n(Showing sections %d-%d of %d)", start+1, end, len(h.sections))
		content += "\n" + h.state.Styles.TextDim.Render(scrollInfo)
	}

	// Wrap in a box
	return tui.RenderBox(h.state, "Keyboard Shortcuts & Help", content, h.state.Width-4)
}
