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

// RecoveryView represents the AI recovery view
type RecoveryView struct {
	state        *tui.AppState
	projects     []*entities.Project
	selectedIdx  int
	viewportStart int // Start index of visible items
	recoveryInfo []RecoveryInfo
}

// RecoveryInfo represents recovery information for a project
type RecoveryInfo struct {
	ProjectID      string
	ProjectName    string
	Status         string
	HasRecovery    bool
	RecoveryType   string
	LastRecovery   time.Time
	RecoveryCount  int
	SuccessRate    float64
	Description    string
}

// NewRecoveryView creates a new recovery view
func NewRecoveryView(state *tui.AppState) *RecoveryView {
	return &RecoveryView{
		state:        state,
		recoveryInfo: make([]RecoveryInfo, 0),
	}
}

// Init initializes the recovery view
func (r *RecoveryView) Init() tea.Cmd {
	return r.loadProjects()
}

// Update handles messages
func (r *RecoveryView) Update(msg tea.Msg) (tui.View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reset viewport to top when terminal is resized
		r.viewportStart = 0
		r.selectedIdx = 0
		return r, nil
	
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return r, tui.SwitchView(tui.ViewDashboard)
		case "up", "k":
			if r.selectedIdx > 0 {
				r.selectedIdx--
				r.adjustViewport()
			}
			return r, nil
		case "down", "j":
			if r.selectedIdx < len(r.recoveryInfo)-1 {
				r.selectedIdx++
				r.adjustViewport()
			}
			return r, nil
		case "r":
			// Refresh
			return r, r.loadProjects()
		}

	case RefreshProjectsMsg:
		r.projects = msg.Projects
		r.analyzeRecovery()
		r.adjustViewport()
		return r, nil
	}
	return r, nil
}

// loadProjects loads projects from repository
func (r *RecoveryView) loadProjects() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		projects, err := r.state.Container.ProjectRepository().List(ctx, repository.ProjectFilter{})
		if err != nil {
			return RefreshProjectsMsg{Projects: []*entities.Project{}}
		}
		return RefreshProjectsMsg{Projects: projects}
	}
}

// analyzeRecovery analyzes recovery information from projects
func (r *RecoveryView) analyzeRecovery() {
	recoveryInfo := make([]RecoveryInfo, 0)

	for _, project := range r.projects {
		projectName := project.Name()
		if project.URL() != nil {
			projectName = project.URL().String()
		}

		info := RecoveryInfo{
			ProjectID:   project.ID().Value(),
			ProjectName: projectName,
			Status:      string(project.Status()),
		}

		// Determine recovery status based on project state
		status := project.Status()
		if status == entities.StatusFailed {
			info.HasRecovery = true
			info.RecoveryType = "Manual Recovery Available"
			info.Description = "Project failed. You can retry setup or use recovery strategies."
		} else if status == entities.StatusReady {
			info.HasRecovery = false
			info.RecoveryType = "No Recovery Needed"
			info.Description = "Project is ready and running successfully."
		} else if status == entities.StatusRunning || status == entities.StatusInstalling {
			info.HasRecovery = true
			info.RecoveryType = "Auto-Recovery Active"
			info.Description = "Project is running with automatic error recovery enabled."
		} else {
			info.HasRecovery = false
			info.RecoveryType = "Pending"
			info.Description = "Project setup is pending."
		}

		info.LastRecovery = project.UpdatedAt()
		info.RecoveryCount = 0 // Would need additional tracking
		info.SuccessRate = 100.0
		if status == entities.StatusFailed {
			info.SuccessRate = 0.0
		}

		recoveryInfo = append(recoveryInfo, info)
	}

	// Sort by last recovery time (newest first)
	sort.Slice(recoveryInfo, func(i, j int) bool {
		return recoveryInfo[i].LastRecovery.After(recoveryInfo[j].LastRecovery)
	})

	r.recoveryInfo = recoveryInfo
}

// View renders the recovery view
func (r *RecoveryView) View() string {
	var sections []string

	// Header
	header := tui.RenderHeader(r.state, "AI Recovery & Error Handling")
	sections = append(sections, header)

	// Recovery overview
	overviewBox := r.renderOverview()
	sections = append(sections, overviewBox)

	// Recovery strategies info
	strategiesBox := r.renderStrategies()
	sections = append(sections, strategiesBox)

	// Project recovery status
	projectsBox := r.renderProjects()
	sections = append(sections, projectsBox)

	// Footer
	footer := tui.RenderFooter(r.state, "[↑↓] Navigate • [R] Refresh • [Esc] Return")
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderOverview renders recovery overview statistics
func (r *RecoveryView) renderOverview() string {
	totalProjects := len(r.recoveryInfo)
	failedProjects := 0
	recoveryAvailable := 0

	for _, info := range r.recoveryInfo {
		if info.Status == "failed" {
			failedProjects++
		}
		if info.HasRecovery {
			recoveryAvailable++
		}
	}

	overviewText := fmt.Sprintf(
		"Total Projects: %d │ Failed: %d │ Recovery Available: %d",
		totalProjects,
		failedProjects,
		recoveryAvailable,
	)

	return tui.RenderBox(r.state, "Recovery Overview", overviewText, r.state.Width-4)
}

// renderStrategies renders information about recovery strategies
func (r *RecoveryView) renderStrategies() string {
	strategies := []string{
		"🤖 AI Auto-Fix: Automatically fixes command errors using AI",
		"🔄 Retry: Retries failed commands with exponential backoff",
		"⏭️  Skip: Skips optional commands that fail",
		"🔙 Rollback: Reverts changes if recovery fails",
		"🛠️  Missing Tool: Installs missing dependencies automatically",
	}

	content := strings.Join(strategies, "\n")
	return tui.RenderBox(r.state, "Recovery Strategies", content, r.state.Width-4)
}

// adjustViewport adjusts viewport to keep selected item visible
func (r *RecoveryView) adjustViewport() {
	if len(r.recoveryInfo) == 0 {
		r.viewportStart = 0
		return
	}

	// Calculate visible height (accounting for description lines)
	visibleHeight := 15
	if r.state.Height > 0 {
		visibleHeight = r.state.Height - 15 // Reserve space for header, overview, strategies, footer
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Adjust viewport to keep selected item visible
	if r.selectedIdx < r.viewportStart {
		r.viewportStart = r.selectedIdx
	} else if r.selectedIdx >= r.viewportStart+visibleHeight {
		r.viewportStart = r.selectedIdx - visibleHeight + 1
	}

	// Ensure viewportStart is valid
	if r.viewportStart < 0 {
		r.viewportStart = 0
	}
	maxStart := len(r.recoveryInfo) - visibleHeight
	if r.viewportStart > maxStart {
		r.viewportStart = maxStart
		if r.viewportStart < 0 {
			r.viewportStart = 0
		}
	}
}

// renderProjects renders project recovery status
func (r *RecoveryView) renderProjects() string {
	if len(r.recoveryInfo) == 0 {
		return tui.RenderBox(r.state, "Project Recovery Status", "No projects found", r.state.Width-4)
	}

	// Adjust viewport
	r.adjustViewport()

	// Calculate visible height
	visibleHeight := 15
	if r.state.Height > 0 {
		visibleHeight = r.state.Height - 15
		if visibleHeight < 5 {
			visibleHeight = 5
		}
	}

	// Calculate visible range
	start := r.viewportStart
	end := start + visibleHeight
	if end > len(r.recoveryInfo) {
		end = len(r.recoveryInfo)
	}

	// Only render visible items
	visibleInfo := r.recoveryInfo[start:end]
	lines := make([]string, 0, len(visibleInfo)*2) // *2 for description lines
	for i, info := range visibleInfo {
		actualIdx := start + i
		// Format status icon
		statusIcon := "○"
		if info.Status == "failed" {
			statusIcon = "✗"
		} else if info.Status == "ready" {
			statusIcon = "✓"
		} else if info.Status == "running" || info.Status == "installing" {
			statusIcon = "⟳"
		}

		// Format recovery type
		recoveryType := info.RecoveryType
		if info.HasRecovery {
			recoveryType = r.state.Styles.Success.Render(recoveryType)
		} else {
			recoveryType = r.state.Styles.TextDim.Render(recoveryType)
		}

		// Format time
		timeAgo := formatTimeAgo(info.LastRecovery)

		line := fmt.Sprintf("%s %s │ %s │ %s",
			statusIcon,
			info.ProjectName,
			recoveryType,
			timeAgo,
		)

		if actualIdx == r.selectedIdx {
			line = r.state.Styles.ButtonActive.Render("> " + line)
			// Add description for selected item
			lines = append(lines, line)
			lines = append(lines, r.state.Styles.TextDim.Render("  "+info.Description))
		} else {
			lines = append(lines, "  "+line)
		}
	}

	content := strings.Join(lines, "\n")
	
	// Add scroll indicator if needed
	if len(r.recoveryInfo) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n(Showing %d-%d of %d)", start+1, end, len(r.recoveryInfo))
		content += r.state.Styles.TextDim.Render(scrollInfo)
	}

	return tui.RenderBox(r.state, "Project Recovery Status", content, r.state.Width-4)
}

