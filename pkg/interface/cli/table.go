package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/glive/domain/entities"
)

// Table represents a table for formatted output
type Table struct {
	headers []string
	rows    [][]string
}

// NewTable creates a new table with headers
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
}

// Render renders the table as a formatted string
func (t *Table) Render() string {
	if len(t.headers) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(t.headers))
	for i, header := range t.headers {
		colWidths[i] = len(header)
	}

	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(colWidths) {
				if len(cell) > colWidths[i] {
					colWidths[i] = len(cell)
				}
			}
		}
	}

	// Limit column widths to terminal width
	maxWidth := getTerminalWidth()
	if maxWidth > 0 {
		totalWidth := 0
		for i := range colWidths {
			totalWidth += colWidths[i] + 3 // +3 for padding and separators
		}
		if totalWidth > maxWidth {
			// Proportionally reduce column widths
			scale := float64(maxWidth-3*len(colWidths)) / float64(totalWidth-3*len(colWidths))
			for i := range colWidths {
				colWidths[i] = int(float64(colWidths[i]) * scale)
			}
		}
	}

	var sb strings.Builder

	// Render header
	sb.WriteString("  ")
	for i, header := range t.headers {
		if i < len(colWidths) {
			fmt.Fprintf(&sb, "%-*s", colWidths[i], truncate(header, colWidths[i]))
			if i < len(t.headers)-1 {
				sb.WriteString("  ")
			}
		}
	}
	sb.WriteString("\n")

	// Render separator
	sb.WriteString("  ")
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("-", width))
		if i < len(colWidths)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	// Render rows
	for _, row := range t.rows {
		sb.WriteString("  ")
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Fprintf(&sb, "%-*s", colWidths[i], truncate(cell, colWidths[i]))
				if i < len(row)-1 {
					sb.WriteString("  ")
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// FormatCommandStatus formats a command status with icon and color
func FormatCommandStatus(cmd *entities.Command) string {
	icon := "○" // pending
	statusColor := Gray

	switch cmd.Status() {
	case entities.CommandRunning:
		icon = "◐"
		statusColor = Blue
	case entities.CommandCompleted:
		icon = "●"
		statusColor = Green
	case entities.CommandFailed:
		icon = "✗"
		statusColor = Red
	case entities.CommandSkipped:
		icon = "⊘"
		statusColor = Yellow
	}

	statusText := string(cmd.Status())
	if isTerminal(os.Stdout) {
		statusText = Colorize(statusText, statusColor)
	}

	return fmt.Sprintf("%s %s  %s", icon, statusText, cmd.Description())
}

// FormatCommandStatusSimple formats command status without the command object
func FormatCommandStatusSimple(status entities.CommandStatus, description string) string {
	icon := "○" // pending
	statusColor := Gray

	switch status {
	case entities.CommandRunning:
		icon = "◐"
		statusColor = Blue
	case entities.CommandCompleted:
		icon = "●"
		statusColor = Green
	case entities.CommandFailed:
		icon = "✗"
		statusColor = Red
	case entities.CommandSkipped:
		icon = "⊘"
		statusColor = Yellow
	}

	statusText := string(status)
	if isTerminal(os.Stdout) {
		statusText = Colorize(statusText, statusColor)
	}

	return fmt.Sprintf("%s %s  %s", icon, statusText, description)
}
