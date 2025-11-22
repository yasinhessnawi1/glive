package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/infrastructure/container"
)

// ViewType represents different views in the application
type ViewType string

const (
	ViewDashboard    ViewType = "dashboard"
	ViewProjectSetup ViewType = "project_setup"
	ViewExecution    ViewType = "execution"
	ViewRecovery     ViewType = "recovery"
	ViewLogs         ViewType = "logs"
	ViewSettings     ViewType = "settings"
	ViewHelp         ViewType = "help"
)

// View is the interface that all views must implement
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
}

// AppState represents the global application state
type AppState struct {
	CurrentView ViewType
	Views       map[ViewType]View
	Width       int
	Height      int
	Styles      *Styles
	Caps        *TerminalCapabilities
	Container   *container.Container
	Quitting    bool
	Error       error
	ProjectURL  string // GitHub URL for project execution
	ForceSetup  bool   // Force re-clone even if project exists
}

// App is the main TUI application
type App struct {
	state AppState
}

// NewApp creates a new TUI application
func NewApp(cont *container.Container) (*App, error) {
	if cont == nil {
		return nil, fmt.Errorf("container cannot be nil")
	}

	caps := DetectCapabilities()
	styles := NewStyles(caps)

	app := &App{
		state: AppState{
			CurrentView: ViewDashboard,
			Views:       make(map[ViewType]View),
			Width:       caps.Width,  // Initialize with detected size
			Height:      caps.Height, // Initialize with detected size
			Styles:      styles,
			Caps:        caps,
			Container:   cont,
		},
	}

	// Initialize views - use lazy initialization to avoid import cycle
	// Views will be initialized on first access

	return app, nil
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	// Don't check size here - Bubble Tea hasn't sent WindowSizeMsg yet
	// Size check will happen in Update() after we receive WindowSizeMsg

	// Initialize current view
	view := a.getView(a.state.CurrentView)
	if view != nil {
		return view.Init()
	}

	return nil
}

// Update handles messages and updates the application state
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Only update if we get valid dimensions from Bubble Tea
		// On Windows PowerShell, this might be 0x0, so keep our detected values
		if msg.Width > 0 && msg.Height > 0 {
			a.state.Width = msg.Width
			a.state.Height = msg.Height
		}
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if a.state.CurrentView == ViewDashboard {
				return a, tea.Quit
			}
			// Return to dashboard
			a.state.CurrentView = ViewDashboard
			if view, ok := a.state.Views[a.state.CurrentView]; ok {
				return a, view.Init()
			}
			return a, nil

		case "?":
			// Show help from any view
			return a, SwitchView(ViewHelp)
		}

	case SwitchViewMsg:
		a.state.CurrentView = msg.View
		view := a.getView(a.state.CurrentView)
		if view != nil {
			return a, view.Init()
		}
		return a, nil

	case ErrorMsg:
		a.state.Error = msg.Error
		return a, nil
	}

	// Delegate to current view
	view := a.getView(a.state.CurrentView)
	if view != nil {
		updatedView, cmd := view.Update(msg)
		a.state.Views[a.state.CurrentView] = updatedView
		return a, cmd
	}

	return a, nil
}

// View renders the application
func (a *App) View() string {
	if a.state.Quitting {
		return "Goodbye!\n"
	}

	if a.state.Error != nil {
		return a.renderError()
	}

	// Check minimum size (only if we have received dimensions)
	if a.state.Width > 0 && a.state.Height > 0 {
		if a.state.Width < 80 || a.state.Height < 16 {
			return a.renderSizeWarning()
		}
	} else {
		// Still waiting for WindowSizeMsg, show loading with debug info
		return fmt.Sprintf("Initializing... (size: %dx%d, view: %s)\n", a.state.Width, a.state.Height, a.state.CurrentView)
	}

	// Render current view
	view := a.getView(a.state.CurrentView)
	if view != nil {
		return view.View()
	}

	return "Unknown view\n"
}

// renderError renders an error message
func (a *App) renderError() string {
	return a.state.Styles.Error.Render(fmt.Sprintf("Error: %v\n", a.state.Error)) +
		"\nPress 'q' to quit."
}

// renderSizeWarning renders a warning for small terminal size
func (a *App) renderSizeWarning() string {
	return fmt.Sprintf(
		"Terminal too small!\n\n"+
			"Current size: %dx%d\n"+
			"Minimum size: 80x16\n\n"+
			"Please resize your terminal and try again.",
		a.state.Width, a.state.Height,
	)
}

// getView gets a view, initializing it if needed (lazy initialization to avoid import cycle)
func (a *App) getView(viewType ViewType) View {
	if view, ok := a.state.Views[viewType]; ok {
		return view
	}

	// Lazy initialization - use function variable that gets set at runtime
	if initViewFunc != nil {
		view := initViewFunc(viewType, &a.state)
		if view != nil {
			a.state.Views[viewType] = view
			return view
		}
	}

	return nil
}

// initViewFunc is set by views package to avoid import cycle
var initViewFunc func(ViewType, *AppState) View

// SetViewInitializer sets the view initializer function (called by views package)
func SetViewInitializer(fn func(ViewType, *AppState) View) {
	initViewFunc = fn
}

// Run starts the TUI application
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// RunWithProject starts the TUI application directly in execution view for a project
func (a *App) RunWithProject(githubURL string) error {
	return a.RunWithProjectOptions(githubURL, false)
}

// RunWithProjectOptions starts the TUI application directly in execution view for a project with options
func (a *App) RunWithProjectOptions(githubURL string, force bool) error {
	// Store the GitHub URL and force flag in state for the execution view to use
	a.state.ProjectURL = githubURL
	a.state.ForceSetup = force

	// Switch to execution view
	a.state.CurrentView = ViewExecution

	// Run the TUI
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// Messages for view switching

// SwitchViewMsg is a message to switch views
type SwitchViewMsg struct {
	View ViewType
}

// ErrorMsg is a message containing an error
type ErrorMsg struct {
	Error error
}

// Helper functions

// SwitchView switches to a different view
func SwitchView(view ViewType) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: view}
	}
}

// ShowError shows an error message
func ShowError(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Error: err}
	}
}

// RenderHeader renders the application header
func RenderHeader(state *AppState, title string) string {
	now := time.Now().Format("15:04:05")
	version := "v2.0.0" // TODO: Get from build info

	headerStyle := state.Styles.Title.Copy().
		Width(state.Width-2).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("#475569")).
		PaddingBottom(1)

	left := fmt.Sprintf("🚀 %s", title)
	right := fmt.Sprintf("%s | %s", version, now)

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	availableWidth := state.Width - leftWidth - rightWidth - 4

	return headerStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", availableWidth), right),
	)
}

// RenderFooter renders the application footer
func RenderFooter(state *AppState, help string) string {
	footerStyle := state.Styles.TextDim.Copy().
		Width(state.Width-2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("#475569")).
		PaddingTop(1)

	return footerStyle.Render(help)
}

// RenderBox renders a box with content
func RenderBox(state *AppState, title string, content string, width int) string {
	boxStyle := state.Styles.Box.Copy().Width(width - 2)

	titleStyle := state.Styles.TextBold.Copy().
		Foreground(state.Styles.Title.GetForeground())

	// Create border with title
	border := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	if state.Caps.SupportsUnicode() {
		border = lipgloss.RoundedBorder()
	}

	boxStyle = boxStyle.Border(border)

	// If we have a title, we need to render it separately
	if title != "" {
		titleText := fmt.Sprintf("─ %s ", title)
		titleWidth := lipgloss.Width(titleText)
		remainingWidth := width - titleWidth - 2
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		titleLine := lipgloss.JoinHorizontal(
			lipgloss.Left,
			"┌",
			titleStyle.Render(titleText),
			strings.Repeat("─", remainingWidth),
			"┐",
		)
		body := boxStyle.Copy().
			BorderTop(false).
			Render(content)
		return lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
	}

	return boxStyle.Render(content)
}
