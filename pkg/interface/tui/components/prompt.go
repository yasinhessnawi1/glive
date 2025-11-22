package components

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/interface/tui"
	"golang.design/x/clipboard"
)

// TextInput represents a text input field
type TextInput struct {
	Value       string
	Placeholder string
	Focused     bool
	Styles      *tui.Styles
	Width       int
}

// NewTextInput creates a new text input
func NewTextInput(placeholder string, styles *tui.Styles) *TextInput {
	return &TextInput{
		Placeholder: placeholder,
		Focused:     false,
		Styles:      styles,
		Width:       50,
	}
}

// Focus focuses the input
func (t *TextInput) Focus() {
	t.Focused = true
}

// Blur unfocuses the input
func (t *TextInput) Blur() {
	t.Focused = false
}

// SetValue sets the input value
func (t *TextInput) SetValue(value string) {
	t.Value = value
}

// GetValue returns the input value
func (t *TextInput) GetValue() string {
	return t.Value
}

// Update handles input updates
func (t *TextInput) Update(msg tea.Msg) (*TextInput, tea.Cmd) {
	if !t.Focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyBackspace, tea.KeyCtrlH:
			if len(t.Value) > 0 {
				t.Value = t.Value[:len(t.Value)-1]
			}
			return t, nil

		case tea.KeyCtrlV:
			// Handle paste (Ctrl+V) via clipboard
			return t, t.pasteFromClipboard()

		case tea.KeyRunes:
			// Append characters respecting width limit
			if len(t.Value)+len(string(msg.Runes)) <= t.Width-4 {
				t.Value += string(msg.Runes)
			}
			return t, nil
		}
	case PasteMsg:
		// Append pasted text, respecting width limit
		remainingWidth := t.Width - 4 - len(t.Value)
		if remainingWidth > 0 {
			pasteText := msg.Text
			if len(pasteText) > remainingWidth {
				pasteText = pasteText[:remainingWidth]
			}
			t.Value += pasteText
		}
		// Return a no-op command to trigger a redraw
		return t, func() tea.Msg { return nil }
	}

	return t, nil
}

// pasteFromClipboard reads clipboard and returns a command to update the input
func (t *TextInput) pasteFromClipboard() tea.Cmd {
	return func() tea.Msg {
		// Initialize clipboard
		if err := clipboard.Init(); err != nil {
			return PasteMsg{Text: ""}
		}

		// Read clipboard content - only read text format
		content := clipboard.Read(clipboard.FmtText)
		if len(content) == 0 {
			// Clipboard might contain non-text content (image, file, etc.)
			return PasteMsg{Text: ""}
		}

		// Verify that the content is valid UTF-8 text
		// If clipboard contains binary data (image, file), this will fail
		if !utf8.Valid(content) {
			// Clipboard contains non-text content, skip it
			return PasteMsg{Text: ""}
		}

		text := string(content)

		// Additional check: ensure it's reasonable text (not binary data)
		// Check for excessive control characters that shouldn't be in normal text
		controlCharCount := 0
		for _, r := range text {
			if r < 32 && r != '\n' && r != '\r' && r != '\t' {
				controlCharCount++
			}
		}
		// If more than 10% are control characters, likely binary data
		if len(text) > 0 && controlCharCount*10 > len(text) {
			return PasteMsg{Text: ""}
		}

		return PasteMsg{Text: text}
	}
}

// PasteMsg is a message containing pasted text
type PasteMsg struct {
	Text string
}

// Render renders the input
func (t *TextInput) Render() string {
	displayValue := t.Value
	if displayValue == "" {
		displayValue = t.Placeholder
	}

	// Truncate if too long
	if len(displayValue) > t.Width-4 {
		displayValue = displayValue[:t.Width-7] + "..."
	}

	var style lipgloss.Style
	if t.Focused {
		if t.Styles != nil {
			style = t.Styles.InputFocus
		} else {
			style = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
		}
	} else {
		if t.Styles != nil {
			style = t.Styles.Input
		} else {
			style = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
		}
	}

	// Add cursor if focused
	if t.Focused && t.Value == "" {
		displayValue = t.Placeholder
	} else if t.Focused {
		displayValue = t.Value + "▊"
	}

	return style.Width(t.Width).Render(displayValue)
}

// SelectOption represents a selectable option
type SelectOption struct {
	Label string
	Value string
}

// Select represents a select/menu component
type Select struct {
	Options  []SelectOption
	Selected int
	Focused  bool
	Styles   *tui.Styles
	Width    int
}

// NewSelect creates a new select component
func NewSelect(options []SelectOption, styles *tui.Styles) *Select {
	return &Select{
		Options:  options,
		Selected: 0,
		Focused:  false,
		Styles:   styles,
		Width:    50,
	}
}

// Focus focuses the select
func (s *Select) Focus() {
	s.Focused = true
}

// Blur unfocuses the select
func (s *Select) Blur() {
	s.Focused = false
}

// MoveUp moves selection up
func (s *Select) MoveUp() {
	if s.Selected > 0 {
		s.Selected--
	}
}

// MoveDown moves selection down
func (s *Select) MoveDown() {
	if s.Selected < len(s.Options)-1 {
		s.Selected++
	}
}

// GetSelected returns the selected option
func (s *Select) GetSelected() *SelectOption {
	if s.Selected < 0 || s.Selected >= len(s.Options) {
		return nil
	}
	return &s.Options[s.Selected]
}

// Update handles select updates
func (s *Select) Update(msg tea.Msg) (*Select, tea.Cmd) {
	if !s.Focused {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			s.MoveUp()
			return s, nil
		case "down", "j":
			s.MoveDown()
			return s, nil
		case "enter", " ":
			return s, nil
		}
	}

	return s, nil
}

// Render renders the select
func (s *Select) Render() string {
	if len(s.Options) == 0 {
		return ""
	}

	lines := make([]string, len(s.Options))
	for i, opt := range s.Options {
		prefix := "  "
		if i == s.Selected {
			if s.Focused {
				prefix = "● "
			} else {
				prefix = "○ "
			}
		}

		line := prefix + opt.Label
		if i == s.Selected && s.Focused && s.Styles != nil {
			line = s.Styles.ButtonActive.Render(line)
		} else if i == s.Selected && s.Styles != nil {
			line = s.Styles.Button.Render(line)
		}

		lines[i] = line
	}

	return strings.Join(lines, "\n")
}
