package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/glive/interface/tui"
)

// Table represents a sortable table
type Table struct {
	Headers    []string
	Rows       [][]string
	Selected   int
	SortColumn int
	SortAsc    bool
	Width      int
	Styles     *tui.Styles
}

// NewTable creates a new table
func NewTable(headers []string, styles *tui.Styles) *Table {
	return &Table{
		Headers:    headers,
		Rows:       make([][]string, 0),
		Selected:   -1,
		SortColumn: -1,
		SortAsc:    true,
		Styles:     styles,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(row []string) {
	t.Rows = append(t.Rows, row)
}

// SetRows sets all rows at once
func (t *Table) SetRows(rows [][]string) {
	t.Rows = rows
}

// Sort sorts the table by the specified column
func (t *Table) Sort(column int, ascending bool) {
	if column < 0 || column >= len(t.Headers) {
		return
	}

	t.SortColumn = column
	t.SortAsc = ascending

	// Simple string-based sorting
	// In a real implementation, you might want type-aware sorting
	for i := 0; i < len(t.Rows)-1; i++ {
		for j := i + 1; j < len(t.Rows); j++ {
			valI := ""
			valJ := ""
			if column < len(t.Rows[i]) {
				valI = t.Rows[i][column]
			}
			if column < len(t.Rows[j]) {
				valJ = t.Rows[j][column]
			}

			shouldSwap := false
			if t.SortAsc {
				shouldSwap = valI > valJ
			} else {
				shouldSwap = valI < valJ
			}

			if shouldSwap {
				t.Rows[i], t.Rows[j] = t.Rows[j], t.Rows[i]
			}
		}
	}
}

// MoveSelection moves the selection up or down
func (t *Table) MoveSelection(delta int) {
	t.Selected += delta
	if t.Selected < 0 {
		t.Selected = 0
	}
	if t.Selected >= len(t.Rows) {
		t.Selected = len(t.Rows) - 1
	}
}

// GetSelectedRow returns the currently selected row
func (t *Table) GetSelectedRow() []string {
	if t.Selected < 0 || t.Selected >= len(t.Rows) {
		return nil
	}
	return t.Rows[t.Selected]
}

// Render renders the table
func (t *Table) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	if t.Width <= 0 {
		t.Width = 80
	}

	// Calculate column widths
	colWidths := t.calculateColumnWidths()

	// Render header
	header := t.renderHeader(colWidths)

	// Render rows
	rows := make([]string, 0, len(t.Rows))
	for i, row := range t.Rows {
		rowStr := t.renderRow(row, colWidths, i == t.Selected)
		rows = append(rows, rowStr)
	}

	// Combine
	result := []string{header}
	result = append(result, rows...)

	return strings.Join(result, "\n")
}

// calculateColumnWidths calculates optimal column widths
func (t *Table) calculateColumnWidths() []int {
	colCount := len(t.Headers)
	colWidths := make([]int, colCount)

	// Calculate max width for each column
	for i := 0; i < colCount; i++ {
		maxWidth := len(t.Headers[i])
		for _, row := range t.Rows {
			if i < len(row) && len(row[i]) > maxWidth {
				maxWidth = len(row[i])
			}
		}
		colWidths[i] = maxWidth + 2 // Add padding
	}

	// Distribute available width
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w
	}

	availableWidth := t.Width - (colCount + 1) // Account for borders
	if totalWidth > availableWidth {
		// Scale down proportionally
		scale := float64(availableWidth) / float64(totalWidth)
		for i := range colWidths {
			colWidths[i] = int(float64(colWidths[i]) * scale)
			if colWidths[i] < 3 {
				colWidths[i] = 3
			}
		}
	}

	return colWidths
}

// renderHeader renders the table header
func (t *Table) renderHeader(colWidths []int) string {
	parts := make([]string, len(t.Headers))
	for i, header := range t.Headers {
		width := colWidths[i]
		if i < len(colWidths) {
			width = colWidths[i]
		}
		parts[i] = lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(header)
	}

	headerRow := strings.Join(parts, " │ ")

	if t.Styles != nil {
		headerRow = t.Styles.TableHeader.Render(headerRow)
	}

	// Add border
	border := strings.Repeat("─", t.Width-2)
	topBorder := "╔" + border + "╗"
	bottomBorder := "╠" + strings.Repeat("═", t.Width-2) + "╣"

	// Use ASCII borders if Unicode not supported (check via styles capabilities if available)
	// For now, assume Unicode support

	return topBorder + "\n│ " + headerRow + " │\n" + bottomBorder
}

// renderRow renders a table row
func (t *Table) renderRow(row []string, colWidths []int, selected bool) string {
	parts := make([]string, len(t.Headers))
	for i := range t.Headers {
		width := colWidths[i]
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		parts[i] = lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(cell)
	}

	rowStr := strings.Join(parts, " │ ")

	// Apply styling
	if selected && t.Styles != nil {
		rowStr = t.Styles.ButtonActive.Render(rowStr)
	} else if t.Styles != nil {
		rowStr = t.Styles.TableRow.Render(rowStr)
	}

	return "│ " + rowStr + " │"
}

// GetHeight returns the height of the table
func (t *Table) GetHeight() int {
	return len(t.Rows) + 3 // Header + border + rows
}
