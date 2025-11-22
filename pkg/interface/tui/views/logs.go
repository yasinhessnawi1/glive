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

// LogsView represents the logs view
type LogsView struct {
	state           *tui.AppState
	projects        []*entities.Project
	selectedIdx     int
	projectsViewportStart int // Viewport for projects list
	selectedLogs    []LogEntry
	scrollOffset    int
	filter          string // "all", "error", "success"
}

// LogEntry represents a log entry
type LogEntry struct {
	ProjectID   string
	ProjectName string
	Timestamp   time.Time
	Type        string // "info", "error", "success", "warning"
	Message     string
}

// NewLogsView creates a new logs view
func NewLogsView(state *tui.AppState) *LogsView {
	return &LogsView{
		state:                state,
		selectedIdx:          0,
		projectsViewportStart: 0,
		selectedLogs:         make([]LogEntry, 0),
		scrollOffset:         0,
		filter:               "all",
	}
}

// Init initializes the logs view
func (l *LogsView) Init() tea.Cmd {
	return l.loadProjects()
}

// Update handles messages
func (l *LogsView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return l, tui.SwitchView(tui.ViewDashboard)
		case "up", "k":
			if l.selectedIdx > 0 {
				l.selectedIdx--
				l.adjustProjectsViewport()
				l.loadLogsForProject()
				l.scrollOffset = 0
			}
			return l, nil
		case "down", "j":
			if l.selectedIdx < len(l.projects)-1 {
				l.selectedIdx++
				l.adjustProjectsViewport()
				l.loadLogsForProject()
				l.scrollOffset = 0
			}
			return l, nil
		case "left", "h":
			// Scroll logs up
			if l.scrollOffset > 0 {
				l.scrollOffset--
			}
			return l, nil
		case "right", "l":
			// Scroll logs down
			visibleLogHeight := 10
			if l.state.Height > 0 {
				visibleLogHeight = l.state.Height - 22
				if visibleLogHeight < 5 {
					visibleLogHeight = 5
				}
			}
			maxScroll := len(l.selectedLogs) - visibleLogHeight
			if maxScroll > 0 && l.scrollOffset < maxScroll {
				l.scrollOffset++
			}
			return l, nil
		case "f":
			// Toggle filter
			switch l.filter {
			case "all":
				l.filter = "error"
			case "error":
				l.filter = "success"
			case "success":
				l.filter = "all"
			}
			l.loadLogsForProject()
			return l, nil
		case "r":
			// Refresh
			return l, tea.Batch(l.loadProjects(), func() tea.Msg {
				return RefreshLogsMsg{}
			})
		}

	case RefreshProjectsMsg:
		l.projects = msg.Projects
		if len(l.projects) > 0 && l.selectedIdx >= len(l.projects) {
			l.selectedIdx = len(l.projects) - 1
		}
		l.adjustProjectsViewport()
		l.loadLogsForProject()
		return l, nil

	case RefreshLogsMsg:
		l.loadLogsForProject()
		return l, nil
	}
	return l, nil
}

// loadProjects loads projects from repository
func (l *LogsView) loadProjects() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		projects, err := l.state.Container.ProjectRepository().List(ctx, repository.ProjectFilter{})
		if err != nil {
			return RefreshProjectsMsg{Projects: []*entities.Project{}}
		}
		return RefreshProjectsMsg{Projects: projects}
	}
}

// loadLogsForProject loads logs for the selected project
func (l *LogsView) loadLogsForProject() {
	if l.selectedIdx < 0 || l.selectedIdx >= len(l.projects) {
		l.selectedLogs = make([]LogEntry, 0)
		return
	}

	project := l.projects[l.selectedIdx]
	logs := make([]LogEntry, 0)

	// Create log entries from project status and metadata
	projectName := project.Name()
	if project.URL() != nil {
		projectName = project.URL().String()
	}

	// Status log
	statusLog := LogEntry{
		ProjectID:   project.ID().Value(),
		ProjectName: projectName,
		Timestamp:   project.UpdatedAt(),
		Type:        "info",
		Message:     fmt.Sprintf("Status: %s", project.Status()),
	}
	logs = append(logs, statusLog)

	// Type log
	if project.Type() != "" {
		typeLog := LogEntry{
			ProjectID:   project.ID().Value(),
			ProjectName: projectName,
			Timestamp:   project.UpdatedAt(),
			Type:        "info",
			Message:     fmt.Sprintf("Type: %s", project.Type()),
		}
		logs = append(logs, typeLog)
	}

	// Created log
	createdLog := LogEntry{
		ProjectID:   project.ID().Value(),
		ProjectName: projectName,
		Timestamp:   project.CreatedAt(),
		Type:        "info",
		Message:     "Project created",
	}
	logs = append(logs, createdLog)

	// Updated log
	if !project.UpdatedAt().Equal(project.CreatedAt()) {
		updatedLog := LogEntry{
			ProjectID:   project.ID().Value(),
			ProjectName: projectName,
			Timestamp:   project.UpdatedAt(),
			Type:        "info",
			Message:     "Last updated",
		}
		logs = append(logs, updatedLog)
	}

	// Local path log
	if project.LocalPath() != nil {
		pathLog := LogEntry{
			ProjectID:   project.ID().Value(),
			ProjectName: projectName,
			Timestamp:   project.UpdatedAt(),
			Type:        "info",
			Message:     fmt.Sprintf("Local path: %s", project.LocalPath().String()),
		}
		logs = append(logs, pathLog)
	}

	// Filter logs
	filteredLogs := make([]LogEntry, 0)
	for _, log := range logs {
		switch l.filter {
		case "all":
			filteredLogs = append(filteredLogs, log)
		case "error":
			if log.Type == "error" {
				filteredLogs = append(filteredLogs, log)
			}
		case "success":
			if log.Type == "success" {
				filteredLogs = append(filteredLogs, log)
			}
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(filteredLogs, func(i, j int) bool {
		return filteredLogs[i].Timestamp.After(filteredLogs[j].Timestamp)
	})

	l.selectedLogs = filteredLogs
}

// View renders the logs view
func (l *LogsView) View() string {
	if l.state.Width < 80 || l.state.Height < 16 {
		return "Terminal too small. Please resize."
	}

	var sections []string

	// Header
	header := tui.RenderHeader(l.state, "Project Logs")
	sections = append(sections, header)

	// Projects list (left side)
	projectsBox := l.renderProjects()
	sections = append(sections, projectsBox)

	// Logs (right side)
	logsBox := l.renderLogs()
	sections = append(sections, logsBox)

	// Footer
	filterText := fmt.Sprintf("Filter: %s", l.filter)
	helpText := fmt.Sprintf("[↑↓] Select Project • [←→] Scroll Logs • [F] Filter (%s) • [R] Refresh • [Esc] Return", filterText)
	footer := tui.RenderFooter(l.state, helpText)
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// adjustProjectsViewport adjusts viewport to keep selected project visible
func (l *LogsView) adjustProjectsViewport() {
	if len(l.projects) == 0 {
		l.projectsViewportStart = 0
		return
	}

	// Calculate visible height for projects list
	visibleHeight := 15
	if l.state.Height > 0 {
		visibleHeight = l.state.Height - 10 // Reserve space for header, logs, footer
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Adjust viewport to keep selected item visible
	if l.selectedIdx < l.projectsViewportStart {
		l.projectsViewportStart = l.selectedIdx
	} else if l.selectedIdx >= l.projectsViewportStart+visibleHeight {
		l.projectsViewportStart = l.selectedIdx - visibleHeight + 1
	}

	// Ensure viewportStart is valid
	if l.projectsViewportStart < 0 {
		l.projectsViewportStart = 0
	}
	maxStart := len(l.projects) - visibleHeight
	if l.projectsViewportStart > maxStart {
		l.projectsViewportStart = maxStart
		if l.projectsViewportStart < 0 {
			l.projectsViewportStart = 0
		}
	}
}

// renderProjects renders the project list
func (l *LogsView) renderProjects() string {
	if len(l.projects) == 0 {
		return tui.RenderBox(l.state, "Projects", "No projects found", l.state.Width-4)
	}

	// Adjust viewport
	l.adjustProjectsViewport()

	// Calculate visible height
	visibleHeight := 15
	if l.state.Height > 0 {
		visibleHeight = l.state.Height - 10
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Calculate visible range
	start := l.projectsViewportStart
	end := start + visibleHeight
	if end > len(l.projects) {
		end = len(l.projects)
	}

	// Only render visible items
	visibleProjects := l.projects[start:end]
	lines := make([]string, len(visibleProjects))
	for i, project := range visibleProjects {
		actualIdx := start + i
		projectName := project.Name()
		if project.URL() != nil {
			projectName = project.URL().String()
		}

		status := string(project.Status())
		icon := l.state.Styles.StatusIcon(status)

		line := fmt.Sprintf("%s %s", icon, projectName)

		if actualIdx == l.selectedIdx {
			line = l.state.Styles.ButtonActive.Render("> " + line)
		} else {
			line = "  " + line
		}

		lines[i] = line
	}

	content := strings.Join(lines, "\n")
	
	// Add scroll indicator if needed
	if len(l.projects) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n(Showing %d-%d of %d)", start+1, end, len(l.projects))
		content += l.state.Styles.TextDim.Render(scrollInfo)
	}

	return tui.RenderBox(l.state, "Projects", content, l.state.Width-4)
}

// renderLogs renders the log entries
func (l *LogsView) renderLogs() string {
	if len(l.selectedLogs) == 0 {
		if l.selectedIdx < len(l.projects) {
			return tui.RenderBox(l.state, "Logs", "No logs available for selected project", l.state.Width-4)
		}
		return tui.RenderBox(l.state, "Logs", "Select a project to view logs", l.state.Width-4)
	}

	lines := make([]string, 0)
	lines = append(lines, fmt.Sprintf("Showing %d log entries (filter: %s)", len(l.selectedLogs), l.filter))
	lines = append(lines, "")

	// Calculate visible height for logs (accounting for header, projects box, footer)
	visibleLogHeight := 10
	if l.state.Height > 0 {
		// Reserve space: header (3) + projects box (~15) + footer (2) + spacing (2) = ~22
		visibleLogHeight = l.state.Height - 22
		if visibleLogHeight < 5 {
			visibleLogHeight = 5
		}
	}

	// Ensure scrollOffset is valid
	maxScroll := len(l.selectedLogs) - visibleLogHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if l.scrollOffset > maxScroll {
		l.scrollOffset = maxScroll
	}
	if l.scrollOffset < 0 {
		l.scrollOffset = 0
	}

	// Show logs with scroll offset
	start := l.scrollOffset
	end := start + visibleLogHeight
	if end > len(l.selectedLogs) {
		end = len(l.selectedLogs)
	}

	for i := start; i < end; i++ {
		log := l.selectedLogs[i]
		timestamp := log.Timestamp.Format("15:04:05")
		
		// Format log type icon
		typeIcon := "ℹ"
		typeColor := l.state.Styles.TextDim
		switch log.Type {
		case "error":
			typeIcon = "✗"
			typeColor = l.state.Styles.Error
		case "success":
			typeIcon = "✓"
			typeColor = l.state.Styles.Success
		case "warning":
			typeIcon = "⚠"
			typeColor = l.state.Styles.Warning
		}

		line := fmt.Sprintf("[%s] %s %s", timestamp, typeColor.Render(typeIcon), log.Message)
		lines = append(lines, line)
	}

	if len(l.selectedLogs) > visibleLogHeight {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("(Showing %d-%d of %d log entries)", start+1, end, len(l.selectedLogs)))
	}

	content := strings.Join(lines, "\n")
	return tui.RenderBox(l.state, "Logs", content, l.state.Width-4)
}

// RefreshLogsMsg is a message to refresh logs
type RefreshLogsMsg struct{}

