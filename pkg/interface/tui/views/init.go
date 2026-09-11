package views

import (
	"github.com/glive/domain/values"
	"github.com/glive/interface/tui"
)

func init() {
	// Register view initializer to break import cycle
	tui.SetViewInitializer(initView)
}

// initView initializes a view based on type
func initView(viewType tui.ViewType, state *tui.AppState) tui.View {
	switch viewType {
	case tui.ViewDashboard:
		return NewDashboardView(state)
	case tui.ViewProjectSetup:
		return NewSetupView(state)
	case tui.ViewExecution:
		// Extract project name from URL if available
		projectName := "Project"
		if state.ProjectURL != "" {
			if url, err := values.NewURL(state.ProjectURL); err == nil {
				if owner, name, err := url.ParseOwnerAndName(); err == nil {
					projectName = owner + "/" + name
				} else {
					projectName = state.ProjectURL
				}
			} else {
				projectName = state.ProjectURL
			}
		}
		return NewExecutionView(state, projectName)
	case tui.ViewRecovery:
		return NewRecoveryView(state)
	case tui.ViewLogs:
		return NewLogsView(state)
	case tui.ViewSettings:
		return NewSettingsView(state)
	case tui.ViewHelp:
		return NewHelpView(state)
	default:
		return nil
	}
}
