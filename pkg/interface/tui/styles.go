package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents the color theme
type Theme string

const (
	ThemeAuto  Theme = "auto"
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

var (
	// Brand Colors (Dark Mode)
	colorPrimary   = lipgloss.Color("#60A5FA")  // Blue
	colorSuccess   = lipgloss.Color("#34D399")  // Green
	colorWarning   = lipgloss.Color("#FBBF24")  // Yellow
	colorError     = lipgloss.Color("#F87171")  // Red
	colorRecovery  = lipgloss.Color("#A78BFA")  // Purple (AI)
	colorInfo      = lipgloss.Color("#60A5FA")  // Blue

	// UI Colors (Dark Mode)
	colorBg        = lipgloss.Color("#1E293B")  // Dark bg
	colorBgAlt     = lipgloss.Color("#0F172A")  // Darker bg
	colorFg        = lipgloss.Color("#E2E8F0")  // Text
	colorFgDim     = lipgloss.Color("#94A3B8")  // Dimmed text
	colorBorder    = lipgloss.Color("#475569")  // Borders

	// Syntax Highlighting
	colorKeyword   = lipgloss.Color("#C792EA")
	colorString    = lipgloss.Color("#C3E88D")
	colorNumber    = lipgloss.Color("#F78C6C")
	colorComment   = lipgloss.Color("#697098")
)

// Styles contains all UI styles
type Styles struct {
	// Base styles
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Box        lipgloss.Style
	BoxAlt     lipgloss.Style
	Border     lipgloss.Style

	// Status styles
	Success    lipgloss.Style
	Warning    lipgloss.Style
	Error      lipgloss.Style
	Info       lipgloss.Style
	Recovery   lipgloss.Style

	// Text styles
	Text       lipgloss.Style
	TextDim    lipgloss.Style
	TextBold   lipgloss.Style

	// Component styles
	ProgressBar lipgloss.Style
	Button      lipgloss.Style
	ButtonActive lipgloss.Style
	Input       lipgloss.Style
	InputFocus  lipgloss.Style

	// Table styles
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	TableRowAlt lipgloss.Style

	// Capabilities (exported for component access)
	Caps *TerminalCapabilities
}

// NewStyles creates a new styles instance with terminal capabilities
func NewStyles(caps *TerminalCapabilities) *Styles {
	s := &Styles{
		Caps: caps,
	}

	// Adapt colors based on capabilities
	if !caps.SupportsColor() {
		s.initNoColor()
	} else if caps.ColorSupport < Color256 {
		s.initBasicColors()
	} else {
		s.initFullColors()
	}

	return s
}

// initFullColors initializes styles with full color support
func (s *Styles) initFullColors() {
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		MarginBottom(1)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(colorFgDim).
		MarginBottom(1)

	s.Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Background(colorBgAlt)

	s.BoxAlt = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Background(colorBg)

	s.Border = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorBorder)

	s.Success = lipgloss.NewStyle().
		Foreground(colorSuccess).
		Bold(true)

	s.Warning = lipgloss.NewStyle().
		Foreground(colorWarning).
		Bold(true)

	s.Error = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true)

	s.Info = lipgloss.NewStyle().
		Foreground(colorInfo).
		Bold(true)

	s.Recovery = lipgloss.NewStyle().
		Foreground(colorRecovery).
		Bold(true)

	s.Text = lipgloss.NewStyle().
		Foreground(colorFg)

	s.TextDim = lipgloss.NewStyle().
		Foreground(colorFgDim)

	s.TextBold = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorFg)

	s.ProgressBar = lipgloss.NewStyle().
		Foreground(colorPrimary)

	s.Button = lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Foreground(colorFg)

	s.ButtonActive = lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Foreground(colorPrimary).
		Bold(true)

	s.Input = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Foreground(colorFg)

	s.InputFocus = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Foreground(colorFg).
		Bold(true)

	s.TableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Padding(0, 1)

	s.TableRow = lipgloss.NewStyle().
		Foreground(colorFg).
		Padding(0, 1)

	s.TableRowAlt = lipgloss.NewStyle().
		Foreground(colorFg).
		Background(colorBgAlt).
		Padding(0, 1)
}

// initBasicColors initializes styles with basic 16-color support
func (s *Styles) initBasicColors() {
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")). // Blue
		MarginBottom(1)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		MarginBottom(1)

	s.Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)

	s.BoxAlt = s.Box.Copy()

	s.Border = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8"))

	s.Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Green
		Bold(true)

	s.Warning = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")). // Yellow
		Bold(true)

	s.Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")). // Red
		Bold(true)

	s.Info = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")). // Blue
		Bold(true)

	s.Recovery = lipgloss.NewStyle().
		Foreground(lipgloss.Color("13")). // Magenta
		Bold(true)

	s.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")) // White

	s.TextDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")) // Gray

	s.TextBold = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	s.ProgressBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12"))

	s.Button = lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Foreground(lipgloss.Color("15"))

	s.ButtonActive = lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Foreground(lipgloss.Color("12")).
		Bold(true)

	s.Input = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		Foreground(lipgloss.Color("15"))

	s.InputFocus = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1).
		Foreground(lipgloss.Color("15")).
		Bold(true)

	s.TableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1)

	s.TableRow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	s.TableRowAlt = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)
}

// initNoColor initializes styles without color support
func (s *Styles) initNoColor() {
	// Simple styles without color
	s.Title = lipgloss.NewStyle().
		Bold(true).
		MarginBottom(1)

	s.Subtitle = lipgloss.NewStyle().
		MarginBottom(1)

	s.Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	s.BoxAlt = s.Box.Copy()

	s.Border = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder())

	s.Success = lipgloss.NewStyle().Bold(true)
	s.Warning = lipgloss.NewStyle().Bold(true)
	s.Error = lipgloss.NewStyle().Bold(true)
	s.Info = lipgloss.NewStyle().Bold(true)
	s.Recovery = lipgloss.NewStyle().Bold(true)

	s.Text = lipgloss.NewStyle()
	s.TextDim = lipgloss.NewStyle()
	s.TextBold = lipgloss.NewStyle().Bold(true)

	s.ProgressBar = lipgloss.NewStyle()
	s.Button = lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder())
	s.ButtonActive = lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder()).Bold(true)
	s.Input = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	s.InputFocus = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Bold(true)

	s.TableHeader = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	s.TableRow = lipgloss.NewStyle().Padding(0, 1)
	s.TableRowAlt = lipgloss.NewStyle().Padding(0, 1)
}

// StatusIcon returns the appropriate icon for a status
func (s *Styles) StatusIcon(status string) string {
	if s.Caps == nil || !s.Caps.SupportsUnicode() {
		// ASCII fallback
		switch status {
		case "success", "ready", "completed":
			return "[OK]"
		case "running", "pending":
			return "[...]"
		case "warning", "recovered":
			return "[!]"
		case "failed", "error":
			return "[X]"
		default:
			return "[ ]"
		}
	}

	// Unicode icons
	switch status {
	case "success", "ready", "completed":
		return "✓"
	case "running", "pending":
		return "⟳"
	case "warning", "recovered":
		return "⚠"
	case "failed", "error":
		return "✗"
	default:
		return "○"
	}
}

// StatusColor returns the styled status text
func (s *Styles) StatusColor(status string, text string) string {
	switch status {
	case "success", "ready", "completed":
		return s.Success.Render(text)
	case "running", "pending":
		return s.Info.Render(text)
	case "warning", "recovered":
		return s.Warning.Render(text)
	case "failed", "error":
		return s.Error.Render(text)
	default:
		return s.Text.Render(text)
	}
}

