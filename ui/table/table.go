package table

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

// Column represents a table column with title and width
type Column struct {
	Title string
	Width int
}

// Row is a slice of strings representing a table row
type Row []string

// Model represents a scrollable table with both vertical and horizontal scrolling
type Model struct {
	columns []Column
	rows    []Row

	// Viewport dimensions
	width  int
	height int

	// Scroll offsets
	colOffset int
	rowOffset int
	cursorRow int
	cursorCol int

	focused bool
}

// New creates a new table model
func New(columns []Column, rows []Row) Model {
	return Model{
		columns:   columns,
		rows:      rows,
		colOffset: 0,
		rowOffset: 0,
		cursorRow: 0,
		cursorCol: 0,
		focused:   true,
	}
}

// SetSize sets the viewport dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets whether the table is focused
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// Focused returns whether the table is focused
func (m Model) Focused() bool {
	return m.focused
}

// Cursor returns the current cursor row
func (m Model) Cursor() int {
	return m.cursorRow
}

// SelectedRow returns the currently selected row
func (m Model) SelectedRow() Row {
	if m.cursorRow >= 0 && m.cursorRow < len(m.rows) {
		return m.rows[m.cursorRow]
	}
	return nil
}

// SelectedCell returns the content of the currently selected cell
func (m Model) SelectedCell() string {
	if m.cursorRow >= 0 && m.cursorRow < len(m.rows) {
		row := m.rows[m.cursorRow]
		if m.cursorCol >= 0 && m.cursorCol < len(row) {
			return row[m.cursorCol]
		}
	}
	return ""
}

// SetRows updates the table rows
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	if m.cursorRow >= len(rows) {
		m.cursorRow = max(0, len(rows)-1)
	}
	// Ensure cursorCol is valid
	if m.cursorCol >= len(m.columns) {
		m.cursorCol = max(0, len(m.columns)-1)
	}
}

// SetColumns updates the table columns
func (m *Model) SetColumns(columns []Column) {
	m.columns = columns
	// Ensure cursorCol is valid
	if m.cursorCol >= len(columns) {
		m.cursorCol = max(0, len(columns)-1)
	}
}

// visibleRows returns the number of rows that can be displayed
func (m Model) visibleRows() int {
	return max(0, m.height-3)
}

// visibleCols calculates how many columns fit in the current width
func (m Model) visibleCols() int {
	if len(m.columns) == 0 {
		return 0
	}

	usedWidth := 0
	count := 0

	for i := m.colOffset; i < len(m.columns); i++ {
		colWidth := m.columns[i].Width + 3
		if usedWidth+colWidth > m.width {
			break
		}
		usedWidth += colWidth
		count++
	}

	return max(1, count)
}

// maxRowOffset returns the maximum vertical scroll offset
func (m Model) maxRowOffset() int {
	visible := m.visibleRows()
	if len(m.rows) <= visible {
		return 0
	}
	return len(m.rows) - visible
}

// maxColOffset returns the maximum horizontal scroll offset
func (m Model) maxColOffset() int {
	return max(0, len(m.columns)-1)
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Vertical navigation
		case "up", "k":
			if m.cursorRow > 0 {
				m.cursorRow--
				if m.cursorRow < m.rowOffset {
					m.rowOffset = m.cursorRow
				}
			}
		case "down", "j":
			if m.cursorRow < len(m.rows)-1 {
				m.cursorRow++
				if m.cursorRow >= m.rowOffset+m.visibleRows() {
					m.rowOffset = m.cursorRow - m.visibleRows() + 1
				}
			}
		case "pgup", "K":
			m.cursorRow = max(0, m.cursorRow-m.visibleRows())
			m.rowOffset = max(0, m.rowOffset-m.visibleRows())
		case "pgdown", "J":
			m.cursorRow = min(len(m.rows)-1, m.cursorRow+m.visibleRows())
			m.rowOffset = min(m.maxRowOffset(), m.rowOffset+m.visibleRows())
		case "home":
			m.cursorRow = 0
			m.rowOffset = 0
		case "end":
			m.cursorRow = max(0, len(m.rows)-1)
			m.rowOffset = m.maxRowOffset()

		// Horizontal navigation (move cursor between columns)
		case "left", "h":
			if m.cursorCol > 0 {
				m.cursorCol--
				// Adjust column offset if cursor goes off screen
				if m.cursorCol < m.colOffset {
					m.colOffset = m.cursorCol
				}
			}
		case "right", "l":
			if m.cursorCol < len(m.columns)-1 {
				m.cursorCol++
				// Adjust column offset if cursor goes off screen
				visibleCols := m.visibleCols()
				if m.cursorCol >= m.colOffset+visibleCols {
					m.colOffset = m.cursorCol - visibleCols + 1
				}
			}
		case "H":
			m.cursorCol = 0
			m.colOffset = 0
		case "L":
			m.cursorCol = len(m.columns) - 1
			// Adjust column offset to show the last columns
			visibleCols := m.visibleCols()
			if len(m.columns) > visibleCols {
				m.colOffset = len(m.columns) - visibleCols
			} else {
				m.colOffset = 0
			}
		}
	}

	return m, nil
}

// View renders the table
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var lines []string

	// Calculate visible columns
	visibleColCount := m.visibleCols()
	endCol := min(m.colOffset+visibleColCount, len(m.columns))

	// Render header
	headerLine := m.renderHeaderLine(m.colOffset, endCol)
	lines = append(lines, headerLine)

	// Render separator
	separatorLine := m.renderSeparator(m.colOffset, endCol)
	lines = append(lines, separatorLine)

	// Render data rows
	visibleRowCount := m.visibleRows()
	endRow := min(m.rowOffset+visibleRowCount, len(m.rows))

	for i := m.rowOffset; i < endRow; i++ {
		rowLine := m.renderDataRow(i, m.colOffset, endCol)
		lines = append(lines, rowLine)
	}

	// Fill empty rows if needed
	for i := endRow - m.rowOffset; i < visibleRowCount; i++ {
		emptyLine := m.renderEmptyRow(m.colOffset, endCol)
		lines = append(lines, emptyLine)
	}

	// Add status bar
	statusBar := m.renderStatusBar()
	lines = append(lines, statusBar)

	return strings.Join(lines, "\n")
}

// renderHeaderLine renders the header row
func (m Model) renderHeaderLine(startCol, endCol int) string {
	t := theme.Current
	var cells []string

	for i := startCol; i < endCol; i++ {
		col := m.columns[i]
		cellText := truncateOrPad(col.Title, col.Width)
		cell := t.TableHeader.Render(" " + cellText + " ")
		cells = append(cells, cell)
	}

	separatorStyle := lipgloss.NewStyle().Foreground(t.Colors.BorderUnfocused)
	return strings.Join(cells, separatorStyle.Render("│"))
}

// renderSeparator renders the separator between header and data
func (m Model) renderSeparator(startCol, endCol int) string {
	t := theme.Current
	separatorStyle := lipgloss.NewStyle().Foreground(t.Colors.BorderUnfocused)

	var parts []string
	for i := startCol; i < endCol; i++ {
		col := m.columns[i]
		parts = append(parts, strings.Repeat("─", col.Width+2))
	}

	return separatorStyle.Render(strings.Join(parts, "┼"))
}

// renderDataRow renders a single data row
func (m Model) renderDataRow(rowIdx, startCol, endCol int) string {
	t := theme.Current
	var cells []string
	row := m.rows[rowIdx]
	isSelectedRow := rowIdx == m.cursorRow

	for i := startCol; i < endCol; i++ {
		col := m.columns[i]
		cellContent := ""
		if i < len(row) {
			cellContent = row[i]
		}

		cellText := truncateOrPad(cellContent, col.Width)

		var cell string
		isSelectedCell := isSelectedRow && i == m.cursorCol
		if isSelectedCell && m.focused {
			cell = t.TableSelected.Render(" " + cellText + " ")
		} else {
			cell = t.TableCell.Render(" " + cellText + " ")
		}
		cells = append(cells, cell)
	}

	separatorStyle := lipgloss.NewStyle().Foreground(t.Colors.BorderUnfocused)
	return strings.Join(cells, separatorStyle.Render("│"))
}

// renderEmptyRow renders an empty row for padding
func (m Model) renderEmptyRow(startCol, endCol int) string {
	t := theme.Current
	var cells []string

	for i := startCol; i < endCol; i++ {
		col := m.columns[i]
		cell := t.TableCell.Render(" " + strings.Repeat(" ", col.Width) + " ")
		cells = append(cells, cell)
	}

	separatorStyle := lipgloss.NewStyle().Foreground(t.Colors.BorderUnfocused)
	return strings.Join(cells, separatorStyle.Render("│"))
}

// renderStatusBar renders the status bar with navigation info
func (m Model) renderStatusBar() string {
	t := theme.Current

	leftInfo := t.StatusBar.Render("Row " + intToStr(m.cursorRow+1) + "/" + intToStr(len(m.rows)) + ", Col " + intToStr(m.cursorCol+1) + "/" + intToStr(len(m.columns)))
	rightInfo := t.StatusBar.Render("Cols " + intToStr(m.colOffset+1) + "-" + intToStr(min(m.colOffset+m.visibleCols(), len(m.columns))) + "/" + intToStr(len(m.columns)) + " | h/l:cell ←→")

	// Calculate spacing
	spacing := max(m.width-lipgloss.Width(leftInfo)-lipgloss.Width(rightInfo), 1)

	return leftInfo + strings.Repeat(" ", spacing) + rightInfo
}

// Helper functions

func truncateOrPad(s string, width int) string {
	// Use lipgloss for proper width calculation
	currentWidth := lipgloss.Width(s)

	if currentWidth > width {
		// Truncate
		runes := []rune(s)
		if width > 3 {
			truncated := ""
			w := 0
			for _, r := range runes {
				rw := lipgloss.Width(string(r))
				if w+rw > width-3 {
					break
				}
				truncated += string(r)
				w += rw
			}
			return truncated + "..."
		}
		return string(runes[:width])
	}

	// Pad with spaces
	return s + strings.Repeat(" ", width-currentWidth)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte(n%10) + '0'}, digits...)
		n /= 10
	}
	return string(digits)
}
