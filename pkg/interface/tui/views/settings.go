package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/core/config"
	"github.com/glive/interface/tui"
	"github.com/glive/interface/tui/components"
)

// SettingsView represents the settings view
type SettingsView struct {
	state         *tui.AppState
	selectedIdx   int
	viewportStart int // Start index of visible items
	configManager *config.Manager
	settings      []SettingItem
	editingIdx    int
	textInput     *components.TextInput
	errorMsg      string
}

// SettingItem represents a configurable setting
type SettingItem struct {
	Key         string
	Label       string
	Value       string
	Type        string // "string", "int", "bool", "select"
	Options     []string // For select type
	Description string
}

// NewSettingsView creates a new settings view
func NewSettingsView(state *tui.AppState) *SettingsView {
	cfgManager, _ := config.New()
	if cfgManager == nil {
		cfgManager, _ = config.New()
	}

	view := &SettingsView{
		state:         state,
		selectedIdx:   0,
		configManager: cfgManager,
		editingIdx:    -1,
	}

	view.loadSettings()
	return view
}

// loadSettings loads current settings from config
func (s *SettingsView) loadSettings() {
	if s.configManager == nil {
		return
	}

	cfg := s.configManager.Get()

	// Mask API key for display
	apiKeyDisplay := "(not set)"
	if cfg.APIKey != "" {
		maskedKey := cfg.APIKey
		if len(maskedKey) > 8 {
			maskedKey = maskedKey[:4] + "..." + maskedKey[len(maskedKey)-4:]
		}
		apiKeyDisplay = maskedKey
	}

	apiEndpointDisplay := "(default)"
	if cfg.APIEndpoint != "" {
		apiEndpointDisplay = cfg.APIEndpoint
	}

	s.settings = []SettingItem{
		{
			Key:         "api-key",
			Label:       "API Key",
			Value:       apiKeyDisplay,
			Type:        "string",
			Description: "DeepSeek/OpenAI/Claude API key (stored securely)",
		},
		{
			Key:         "api-provider",
			Label:       "AI Provider",
			Value:       cfg.APIProvider,
			Type:        "select",
			Options:     []string{"deepseek", "openai", "claude"},
			Description: "AI provider to use",
		},
		{
			Key:         "api-endpoint",
			Label:       "API Endpoint",
			Value:       apiEndpointDisplay,
			Type:        "string",
			Description: "Custom API endpoint (optional)",
		},
		{
			Key:         "default-mode",
			Label:       "Default Mode",
			Value:       string(cfg.DefaultMode),
			Type:        "select",
			Options:     []string{"auto", "assisted", "manual"},
			Description: "Default execution mode",
		},
		{
			Key:         "workspace-dir",
			Label:       "Workspace Directory",
			Value:       cfg.WorkspaceDir,
			Type:        "string",
			Description: "Directory for cloned projects",
		},
		{
			Key:         "max-concurrent",
			Label:       "Max Concurrent",
			Value:       fmt.Sprintf("%d", cfg.MaxConcurrent),
			Type:        "int",
			Description: "Maximum concurrent projects",
		},
		{
			Key:         "enable-sandbox",
			Label:       "Enable Sandbox",
			Value:       fmt.Sprintf("%t", cfg.EnableSandbox),
			Type:        "bool",
			Description: "Enable sandbox mode for execution",
		},
		{
			Key:         "agent-port",
			Label:       "Agent Port",
			Value:       fmt.Sprintf("%d", cfg.AgentPort),
			Type:        "int",
			Description: "Port for local agent server",
		},
	}
}

// Init initializes the settings view
func (s *SettingsView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (s *SettingsView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	// If editing, handle input
	if s.editingIdx >= 0 && s.textInput != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Save the value
				value := strings.TrimSpace(s.textInput.GetValue())
				if err := s.saveSetting(s.editingIdx, value); err != nil {
					s.errorMsg = err.Error()
				} else {
					s.errorMsg = ""
					s.editingIdx = -1
					s.textInput = nil
					s.loadSettings() // Reload to refresh display
				}
				return s, nil
			case "esc":
				s.editingIdx = -1
				s.textInput = nil
				s.errorMsg = ""
				return s, nil
			default:
				updatedInput, cmd := s.textInput.Update(msg)
				s.textInput = updatedInput
				return s, cmd
			}
		default:
			updatedInput, cmd := s.textInput.Update(msg)
			s.textInput = updatedInput
			return s, cmd
		}
	}

	// Normal navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return s, tui.SwitchView(tui.ViewDashboard)
		case "up", "k":
			if s.selectedIdx > 0 {
				s.selectedIdx--
				s.adjustViewport()
			}
			return s, nil
		case "down", "j":
			if s.selectedIdx < len(s.settings)-1 {
				s.selectedIdx++
				s.adjustViewport()
			}
			return s, nil
		case "enter", "e":
			// Start editing selected setting
			if s.selectedIdx < len(s.settings) {
				setting := s.settings[s.selectedIdx]
				
				// Handle select type differently
				if setting.Type == "select" {
					// Cycle through options
					currentIdx := -1
					for i, opt := range setting.Options {
						if opt == setting.Value {
							currentIdx = i
							break
						}
					}
					nextIdx := (currentIdx + 1) % len(setting.Options)
					if err := s.saveSetting(s.selectedIdx, setting.Options[nextIdx]); err != nil {
						s.errorMsg = err.Error()
					} else {
						s.errorMsg = ""
						s.loadSettings()
					}
					return s, nil
				} else if setting.Type == "bool" {
					// Toggle boolean
					newValue := "false"
					if setting.Value == "false" {
						newValue = "true"
					}
					if err := s.saveSetting(s.selectedIdx, newValue); err != nil {
						s.errorMsg = err.Error()
					} else {
						s.errorMsg = ""
						s.loadSettings()
					}
					return s, nil
				} else {
					// Start text input for string/int
					s.editingIdx = s.selectedIdx
					currentValue := setting.Value
					if setting.Key == "api-key" && currentValue == "(not set)" {
						currentValue = ""
					}
					if setting.Key == "api-endpoint" && currentValue == "(default)" {
						currentValue = ""
					}
					s.textInput = components.NewTextInput("", s.state.Styles)
					s.textInput.SetValue(currentValue)
					s.textInput.Width = s.state.Width - 20
					if s.textInput.Width < 40 {
						s.textInput.Width = 40
					}
					s.textInput.Focus()
					return s, nil
				}
			}
			return s, nil
		}
	}
	return s, nil
}

// saveSetting saves a setting value
func (s *SettingsView) saveSetting(idx int, value string) error {
	if s.configManager == nil || idx < 0 || idx >= len(s.settings) {
		return fmt.Errorf("invalid setting index")
	}

	setting := s.settings[idx]
	
	// Convert value based on type
	var valueToSet interface{} = value
	switch setting.Type {
	case "int":
		var err error
		valueToSet, err = parseInt(value)
		if err != nil {
			return fmt.Errorf("invalid integer: %v", err)
		}
	case "bool":
		var err error
		valueToSet, err = parseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %v", err)
		}
	}

	return s.configManager.Set(setting.Key, valueToSet)
}

// View renders the settings view
func (s *SettingsView) View() string {
	if s.state.Width < 80 || s.state.Height < 16 {
		return "Terminal too small. Please resize."
	}

	var sections []string

	// Header
	header := tui.RenderHeader(s.state, "Settings")
	sections = append(sections, header)

	// Settings list
	settingsContent := s.renderSettings()
	sections = append(sections, settingsContent)

	// Error message if any
	if s.errorMsg != "" {
		errorBox := tui.RenderBox(s.state, "Error", s.state.Styles.Error.Render(s.errorMsg), s.state.Width-4)
		sections = append(sections, errorBox)
	}

	// Footer
	helpText := "[↑↓] Navigate • [Enter] Edit/Toggle • [Esc] Cancel/Return"
	if s.editingIdx >= 0 {
		helpText = "[Enter] Save • [Esc] Cancel"
	}
	footer := tui.RenderFooter(s.state, helpText)
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// adjustViewport adjusts the viewport to keep selected item visible
func (s *SettingsView) adjustViewport() {
	if len(s.settings) == 0 {
		s.viewportStart = 0
		return
	}

	// Calculate visible height for settings list
	visibleHeight := 15
	if s.state.Height > 0 {
		visibleHeight = s.state.Height - 15 // Reserve space for header, error, footer
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Adjust viewport to keep selected item visible
	if s.selectedIdx < s.viewportStart {
		s.viewportStart = s.selectedIdx
	} else if s.selectedIdx >= s.viewportStart+visibleHeight {
		s.viewportStart = s.selectedIdx - visibleHeight + 1
	}

	// Ensure viewportStart is valid
	if s.viewportStart < 0 {
		s.viewportStart = 0
	}
	maxStart := len(s.settings) - visibleHeight
	if s.viewportStart > maxStart {
		s.viewportStart = maxStart
		if s.viewportStart < 0 {
			s.viewportStart = 0
		}
	}
}

// renderSettings renders the settings list
func (s *SettingsView) renderSettings() string {
	if len(s.settings) == 0 {
		return tui.RenderBox(s.state, "Settings", "No settings available", s.state.Width-4)
	}

	// Adjust viewport
	s.adjustViewport()

	// Calculate visible height
	visibleHeight := 15
	if s.state.Height > 0 {
		visibleHeight = s.state.Height - 15
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Calculate visible range
	start := s.viewportStart
	end := start + visibleHeight
	if end > len(s.settings) {
		end = len(s.settings)
	}

	// Only render visible items
	visibleSettings := s.settings[start:end]
	lines := make([]string, 0, len(visibleSettings))
	for i, setting := range visibleSettings {
		actualIdx := start + i
		// Format value display
		valueDisplay := setting.Value
		if setting.Type == "string" && valueDisplay == "" {
			valueDisplay = "(empty)"
		}

		// Format line
		line := fmt.Sprintf("%s: %s", setting.Label, valueDisplay)
		
		// Add description
		if setting.Description != "" {
			line += fmt.Sprintf("  %s", s.state.Styles.TextDim.Render("("+setting.Description+")"))
		}

		// Highlight selected
		if actualIdx == s.selectedIdx {
			if s.editingIdx == actualIdx {
				// Show input field
				line = fmt.Sprintf("%s: %s", setting.Label, s.textInput.Render())
			} else {
				line = s.state.Styles.ButtonActive.Render("> " + line)
			}
		} else {
			line = "  " + line
		}

		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	
	// Add scroll indicator if needed
	if len(s.settings) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n(Showing %d-%d of %d settings)", start+1, end, len(s.settings))
		content += s.state.Styles.TextDim.Render(scrollInfo)
	}

	return tui.RenderBox(s.state, "Configuration", content, s.state.Width-4)
}

// Helper functions

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}

