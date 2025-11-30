package views

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glive/domain/entities"
	"github.com/glive/domain/repository"
	"github.com/glive/interface/tui"
)

// DashboardView represents the dashboard view
type DashboardView struct {
	state         *tui.AppState
	projects      []*entities.Project
	stats         DashboardStats
	selectedIdx   int
	viewportStart int // Start index of visible items in projects list
}

// DashboardStats contains dashboard statistics
type DashboardStats struct {
	ProjectsToday   int
	SuccessRate     float64
	AIRecoveryCount int
	AIRecoveryTotal int
	ActiveProjects  int
	TotalProjects   int
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(state *tui.AppState) *DashboardView {
	return &DashboardView{
		state:        state,
		projects:     make([]*entities.Project, 0),
		selectedIdx:  0,
	}
}

// Init initializes the dashboard view
func (d *DashboardView) Init() tea.Cmd {
	return tea.Batch(
		d.loadProjects(),
		d.startRefreshTicker(),
	)
}

// Update handles messages
func (d *DashboardView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reset viewport to top when terminal is resized
		d.viewportStart = 0
		d.selectedIdx = 0
		return d, nil
	
	case tea.KeyMsg:
		switch msg.String() {
		case "n", "N":
			return d, tui.SwitchView(tui.ViewProjectSetup)
		case "l", "L":
			return d, tui.SwitchView(tui.ViewLogs)
		case "s", "S":
			return d, tui.SwitchView(tui.ViewSettings)
		case "up", "k":
			if d.selectedIdx > 0 {
				d.selectedIdx--
				d.adjustViewport()
			}
			return d, nil
		case "down", "j":
			if d.selectedIdx < len(d.projects)-1 {
				d.selectedIdx++
				d.adjustViewport()
			}
			return d, nil
		case "enter":
			if d.selectedIdx < len(d.projects) {
				// Get the selected project
				sortedProjects := make([]*entities.Project, len(d.projects))
				copy(sortedProjects, d.projects)
				sort.Slice(sortedProjects, func(i, j int) bool {
					return sortedProjects[i].UpdatedAt().After(sortedProjects[j].UpdatedAt())
				})
				
				maxProjects := 5
				if len(sortedProjects) > maxProjects {
					sortedProjects = sortedProjects[:maxProjects]
				}
				
				if d.selectedIdx < len(sortedProjects) {
					selectedProject := sortedProjects[d.selectedIdx]
					// Set the project URL in state and switch to execution view
					if selectedProject.URL() != nil {
						d.state.ProjectURL = selectedProject.URL().String()
						d.state.ForceSetup = false // Don't force re-clone by default
						return d, tui.SwitchView(tui.ViewExecution)
					}
				}
			}
			return d, nil
		case "r", "R":
			// Refresh projects and stats
			return d, tea.Batch(d.loadProjects(), d.startRefreshTicker())
		case "?", "h", "H":
			// Show help view
			return d, tui.SwitchView(tui.ViewHelp)
		case "x", "X":
			// Stop last pending/running project
			return d, d.stopLastActiveProject()
		}

	case RefreshProjectsMsg:
		d.projects = msg.Projects
		d.calculateStats()
		d.adjustViewport() // Adjust viewport after loading projects
		return d, nil

	case RefreshTickMsg:
		// Refresh stats periodically
		return d, tea.Batch(d.loadProjects(), d.startRefreshTicker())

	case StopProjectMsg:
		// Project stop result
		if msg.Success {
			// Refresh projects list to show updated status
			return d, d.loadProjects()
		}
		// Could show error message here if needed
		return d, nil
	}

	return d, nil
}

// View renders the dashboard
func (d *DashboardView) View() string {
	var sections []string

	// Header
	header := tui.RenderHeader(d.state, "GLive - GitHub to Live")
	sections = append(sections, header)

	// Quick Stats
	statsBox := d.renderStats()
	sections = append(sections, statsBox)

	// Recent Projects
	projectsBox := d.renderProjects()
	sections = append(sections, projectsBox)

	// System Health
	healthBox := d.renderSystemHealth()
	sections = append(sections, healthBox)

	// Quick Actions
	actionsBox := d.renderQuickActions()
	sections = append(sections, actionsBox)

		// Footer with help
	helpText := "Use arrow keys to navigate • Press ? for help • [N] New Project • [L] Logs • [S] Settings • [R] Refresh • [X] Stop Last • [Q] Quit"
	footer := tui.RenderFooter(d.state, helpText)
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderStats renders the quick stats section
func (d *DashboardView) renderStats() string {
	statsText := fmt.Sprintf(
		"Projects Today: %d │ Success Rate: %.1f%% │ AI Recovery: %d/%d │ Active: %d",
		d.stats.ProjectsToday,
		d.stats.SuccessRate,
		d.stats.AIRecoveryCount,
		d.stats.AIRecoveryTotal,
		d.stats.ActiveProjects,
	)

	return tui.RenderBox(d.state, "Quick Stats", statsText, d.state.Width-4)
}

// adjustViewport adjusts the viewport to keep selected item visible
func (d *DashboardView) adjustViewport() {
	// Calculate visible height for projects list (accounting for header, stats, health, actions, footer)
	// Rough estimate: about 10-12 lines available for projects
	visibleHeight := 12
	if d.state.Height > 0 {
		// More dynamic calculation based on actual terminal height
		visibleHeight = d.state.Height - 20 // Reserve space for other UI elements
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	sortedProjects := make([]*entities.Project, len(d.projects))
	copy(sortedProjects, d.projects)
	sort.Slice(sortedProjects, func(i, j int) bool {
		return sortedProjects[i].UpdatedAt().After(sortedProjects[j].UpdatedAt())
	})

	maxProjects := len(sortedProjects)
	if maxProjects == 0 {
		d.viewportStart = 0
		return
	}

	// Adjust viewport to keep selected item visible
	if d.selectedIdx < d.viewportStart {
		d.viewportStart = d.selectedIdx
	} else if d.selectedIdx >= d.viewportStart+visibleHeight {
		d.viewportStart = d.selectedIdx - visibleHeight + 1
	}

	// Ensure viewportStart is valid
	if d.viewportStart < 0 {
		d.viewportStart = 0
	}
	if d.viewportStart > maxProjects-visibleHeight {
		d.viewportStart = maxProjects - visibleHeight
		if d.viewportStart < 0 {
			d.viewportStart = 0
		}
	}
}

// renderProjects renders the recent projects list
func (d *DashboardView) renderProjects() string {
	if len(d.projects) == 0 {
		return tui.RenderBox(d.state, "Recent Projects", "No projects yet. Press [N] to create a new project.", d.state.Width-4)
	}

	// Sort by updated time (most recent first)
	sortedProjects := make([]*entities.Project, len(d.projects))
	copy(sortedProjects, d.projects)
	sort.Slice(sortedProjects, func(i, j int) bool {
		return sortedProjects[i].UpdatedAt().After(sortedProjects[j].UpdatedAt())
	})

	// Calculate visible height for projects list
	visibleHeight := 12
	if d.state.Height > 0 {
		visibleHeight = d.state.Height - 20
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Adjust viewport if needed
	d.adjustViewport()

	// Calculate visible range
	start := d.viewportStart
	end := start + visibleHeight
	if end > len(sortedProjects) {
		end = len(sortedProjects)
	}

	// Only render visible items
	visibleProjects := sortedProjects[start:end]
	lines := make([]string, len(visibleProjects))
	for i, project := range visibleProjects {
		actualIdx := start + i
		status := string(project.Status())
		icon := d.state.Styles.StatusIcon(status)
		statusColor := d.state.Styles.StatusColor(status, status)

		// Format time ago
		timeAgo := formatTimeAgo(project.UpdatedAt())

		// Project type
		projectType := string(project.Type())
		if projectType == "" {
			projectType = "Unknown"
		}

		// Project name (owner/repo format)
		projectName := project.Name()
		if project.URL() != nil {
			projectName = project.URL().String()
		}

		line := fmt.Sprintf("%s %s  %s  %s  %s",
			icon,
			projectName,
			timeAgo,
			projectType,
			statusColor,
		)

		// Highlight selected
		if actualIdx == d.selectedIdx {
			line = d.state.Styles.ButtonActive.Render(line)
		}

		lines[i] = line
	}

	content := strings.Join(lines, "\n")
	
	// Add scroll indicator if needed
	if len(sortedProjects) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n(Showing %d-%d of %d projects)", start+1, end, len(sortedProjects))
		content += d.state.Styles.TextDim.Render(scrollInfo)
	}

	return tui.RenderBox(d.state, "Recent Projects", content, d.state.Width-4)
}

// renderSystemHealth renders system health indicators
func (d *DashboardView) renderSystemHealth() string {
	config := d.state.Container.Config()
	
	// API Status
	apiStatus := "● Disconnected"
	if config.APIKey != "" {
		apiStatus = "● Connected"
	}
	
	// Sandbox Status
	sandboxStatus := "● Disabled"
	if config.EnableSandbox {
		sandboxStatus = "● Enabled"
	}
	
	// AI Provider Status
	aiStatus := "● None"
	if config.APIProvider != "" {
		aiStatus = fmt.Sprintf("● %s", config.APIProvider)
	} else if config.APIKey != "" {
		aiStatus = "● Unknown"
	}
	
	// Workspace Status
	workspaceStatus := "● Ready"
	if config.WorkspaceDir == "" {
		workspaceStatus = "● Not Set"
	}

	healthText := fmt.Sprintf(
		"API: %s  │ Sandbox: %s  │ AI: %s  │ Workspace: %s",
		apiStatus,
		sandboxStatus,
		aiStatus,
		workspaceStatus,
	)

	return tui.RenderBox(d.state, "System Health", healthText, d.state.Width-4)
}

// renderQuickActions renders quick action buttons
func (d *DashboardView) renderQuickActions() string {
	actions := []string{
		"[N] New Project",
		"[L] Logs",
		"[S] Settings",
		"[R] Refresh",
		"[X] Stop Last Project",
		"[H] Help",
		"[Q] Quit",
	}

	actionsText := strings.Join(actions, "    ")
	return tui.RenderBox(d.state, "Quick Actions", actionsText, d.state.Width-4)
}

// loadProjects loads projects from repository
func (d *DashboardView) loadProjects() tea.Cmd {
	return func() tea.Msg {
		// Use a timeout context to prevent blocking forever
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		projects, err := d.state.Container.ProjectRepository().List(ctx, repository.ProjectFilter{})
		if err != nil {
			// Don't fail, just return empty list
			return RefreshProjectsMsg{Projects: []*entities.Project{}}
		}
		return RefreshProjectsMsg{Projects: projects}
	}
}

// calculateStats calculates dashboard statistics
func (d *DashboardView) calculateStats() {
	today := time.Now().Truncate(24 * time.Hour)
	successCount := 0
	totalCount := 0
	aiRecoveryCount := 0
	aiRecoveryTotal := 0
	activeCount := 0

	for _, project := range d.projects {
		// Projects today
		if project.CreatedAt().After(today) {
			totalCount++
		}

		// Success rate (for today's projects)
		if project.CreatedAt().After(today) {
			if project.Status() == entities.StatusReady {
				successCount++
			}
		}

		// Active projects
		if project.Status() == entities.StatusRunning || project.Status() == entities.StatusInstalling {
			activeCount++
		}

		// TODO: Track AI recovery stats (would need additional tracking)
	}

	d.stats.TotalProjects = len(d.projects)
	d.stats.ProjectsToday = totalCount
	d.stats.ActiveProjects = activeCount

	if totalCount > 0 {
		d.stats.SuccessRate = float64(successCount) / float64(totalCount) * 100
	} else {
		d.stats.SuccessRate = 0
	}

	d.stats.AIRecoveryCount = aiRecoveryCount
	d.stats.AIRecoveryTotal = aiRecoveryTotal
}

// startRefreshTicker starts a ticker for periodic refresh
func (d *DashboardView) startRefreshTicker() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}

// stopLastActiveProject stops the last pending/running project
func (d *DashboardView) stopLastActiveProject() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Get all projects
		projects, err := d.state.Container.ProjectRepository().List(ctx, repository.ProjectFilter{})
		if err != nil {
			return StopProjectMsg{Success: false, Message: fmt.Sprintf("Failed to load projects: %v", err)}
		}

		// Find the last active project (most recently updated)
		var lastActiveProject *entities.Project
		var lastActiveTime time.Time

		activeStatuses := []entities.ProjectStatus{
			entities.StatusPending,
			entities.StatusCloning,
			entities.StatusAnalyzing,
			entities.StatusInstalling,
			entities.StatusRunning,
		}

		for _, project := range projects {
			status := project.Status()
			for _, activeStatus := range activeStatuses {
				if status == activeStatus {
					updatedAt := project.UpdatedAt()
					if updatedAt.After(lastActiveTime) {
						lastActiveTime = updatedAt
						lastActiveProject = project
					}
					break
				}
			}
		}

		if lastActiveProject == nil {
			return StopProjectMsg{Success: false, Message: "No active project found to stop"}
		}

		// Stop the project
		if err := lastActiveProject.SetStatus(entities.StatusStopped); err != nil {
			return StopProjectMsg{Success: false, Message: fmt.Sprintf("Failed to stop project: %v", err)}
		}

		// Save the updated project
		if err := d.state.Container.ProjectRepository().Save(ctx, lastActiveProject); err != nil {
			return StopProjectMsg{Success: false, Message: fmt.Sprintf("Failed to save project: %v", err)}
		}

		projectName := lastActiveProject.Name()
		if lastActiveProject.URL() != nil {
			projectName = lastActiveProject.URL().String()
		}

		return StopProjectMsg{
			Success:     true,
			Message:     fmt.Sprintf("Stopped project: %s", projectName),
			ProjectID:   lastActiveProject.ID().Value(),
		}
	}
}

// Messages

// RefreshProjectsMsg is a message containing refreshed projects
type RefreshProjectsMsg struct {
	Projects []*entities.Project
}

// RefreshTickMsg is a tick message for refresh
type RefreshTickMsg struct{}

// StopProjectMsg is a message containing stop project result
type StopProjectMsg struct {
	Success   bool
	Message   string
	ProjectID string
}

// ErrorMsg is an error message
type ErrorMsg struct {
	Error error
}

// Helper functions

// formatTimeAgo formats a time as "X ago"
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}
	days := int(duration.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%dd ago", days)
	}
	return t.Format("Jan 2")
}

