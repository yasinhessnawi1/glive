package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/glive/interface/tui"
)

// ProgressBar represents an animated progress bar
type ProgressBar struct {
	Current     float64
	Total       float64
	Width       int
	Label       string
	Animation   *Animation
	Style       *tui.Styles
	ShowPercent bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total float64, label string, width int, styles *tui.Styles) *ProgressBar {
	if width <= 0 {
		width = 40
	}
	return &ProgressBar{
		Current:     0,
		Total:       total,
		Width:       width,
		Label:       label,
		Style:       styles,
		ShowPercent: true,
	}
}

// SetProgress sets the progress value and animates if needed
func (p *ProgressBar) SetProgress(current float64, animate bool) {
	if animate && p.Animation == nil {
		from := p.Current
		to := current
		if to < 0 {
			to = 0
		}
		if to > p.Total {
			to = p.Total
		}
		p.Animation = NewAnimation(from, to, 300*time.Millisecond, EaseInOutCubic)
	} else if animate && p.Animation != nil {
		p.Animation.Reset(p.Current, current, 300*time.Millisecond)
	} else {
		p.Current = current
		if p.Current < 0 {
			p.Current = 0
		}
		if p.Current > p.Total {
			p.Current = p.Total
		}
	}
}

// Update updates the animation if active
func (p *ProgressBar) Update() {
	if p.Animation != nil && !p.Animation.IsDone() {
		p.Current = p.Animation.Value()
	}
}

// Render renders the progress bar
func (p *ProgressBar) Render() string {
	p.Update()

	percent := float64(0)
	if p.Total > 0 {
		percent = p.Current / p.Total
	}

	filled := int(percent * float64(p.Width))
	if filled > p.Width {
		filled = p.Width
	}

	// Choose characters based on Unicode support
	var filledChar, emptyChar string
	if p.Style != nil {
		// Assume Unicode support for modern terminals
		filledChar = "█"
		emptyChar = "░"
	} else {
		filledChar = "#"
		emptyChar = "-"
	}

	bar := strings.Repeat(filledChar, filled) + strings.Repeat(emptyChar, p.Width-filled)

	result := fmt.Sprintf("%s [%s]", p.Label, bar)
	if p.ShowPercent {
		result += fmt.Sprintf(" %.0f%%", percent*100)
	}

	return result
}

// RenderStyled renders the progress bar with styling
func (p *ProgressBar) RenderStyled() string {
	rendered := p.Render()
	if p.Style != nil {
		return p.Style.ProgressBar.Render(rendered)
	}
	return rendered
}

// IsComplete returns true if progress is complete
func (p *ProgressBar) IsComplete() bool {
	return p.Current >= p.Total
}

// ProgressStep represents a step in an execution timeline
type ProgressStep struct {
	ID          string
	Label       string
	Status      string  // pending, running, completed, failed
	Progress    float64 // 0-100
	Duration    time.Duration
	StartTime   time.Time
	EndTime     time.Time
	ProgressBar *ProgressBar
}

// NewProgressStep creates a new progress step
func NewProgressStep(id, label string, styles *tui.Styles) *ProgressStep {
	return &ProgressStep{
		ID:          id,
		Label:       label,
		Status:      "pending",
		Progress:    0,
		ProgressBar: NewProgressBar(100, label, 30, styles),
	}
}

// Start marks the step as running
func (s *ProgressStep) Start() {
	s.Status = "running"
	s.StartTime = time.Now()
	s.ProgressBar.SetProgress(0, false)
}

// Complete marks the step as completed
func (s *ProgressStep) Complete() {
	s.Status = "completed"
	s.Progress = 100
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	s.ProgressBar.SetProgress(100, true)
}

// Fail marks the step as failed
func (s *ProgressStep) Fail() {
	s.Status = "failed"
	s.EndTime = time.Now()
	if !s.StartTime.IsZero() {
		s.Duration = s.EndTime.Sub(s.StartTime)
	}
}

// UpdateProgress updates the progress percentage
func (s *ProgressStep) UpdateProgress(percent float64) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	s.Progress = percent
	s.ProgressBar.SetProgress(percent, true)
}

// Render renders the step
func (s *ProgressStep) Render(styles *tui.Styles) string {
	s.ProgressBar.Update()

	icon := styles.StatusIcon(s.Status)

	var durationStr string
	if s.Duration > 0 {
		durationStr = formatDuration(s.Duration)
	} else if s.Status == "running" && !s.StartTime.IsZero() {
		durationStr = formatDuration(time.Since(s.StartTime))
	} else {
		durationStr = "pending"
	}

	progressBar := s.ProgressBar.Render()

	// Add current indicator
	currentIndicator := ""
	if s.Status == "running" {
		currentIndicator = " ← Current"
	}

	return fmt.Sprintf("%s %s %s %s%s", icon, s.Label, progressBar, durationStr, currentIndicator)
}

// formatDuration formats a duration nicely
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
