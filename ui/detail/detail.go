package detail

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/keys"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/ui/table"
	"github.com/sheenazien8/sq/ui/theme"
)

// Model represents the detail pane for the selected row
type Model struct {
	visible bool
	width   int
	height  int
	columns []table.Column
	row     table.Row
	offset  int
}

// New creates a new detail model
func New() Model {
	return Model{
		visible: false,
		offset:  0,
	}
}

func (m Model) calculateLines() []string {
    if len(m.columns) == 0 || m.row == nil || len(m.row) == 0 {
		return []string{}
	}

	t := theme.Current
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Colors.Primary).
		MarginBottom(1)

	content.WriteString(titleStyle.Render("Record Details") + "\n")

	keyStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Secondary).
		Bold(true)

	valStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground)

	innerW := m.width - 2
	if innerW < 10 {
		innerW = 10
	}

	for i, col := range m.columns {
		if i >= len(m.row) {
			break
		}

		val := m.row[i]

		keyStr := keyStyle.Render(col.Title + ": ")

		content.WriteString(keyStr)
		if lipgloss.Width(keyStr)+lipgloss.Width(val) > innerW {
			wrappedVal := wordWrap(val, innerW-2)
			lines := strings.Split(wrappedVal, "\n")
			for _, line := range lines {
				content.WriteString("\n  " + valStyle.Render(line))
			}
			content.WriteString("\n")
		} else {
			content.WriteString(valStyle.Render(val) + "\n")
		}
	}

	return strings.Split(content.String(), "\n")
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keys.AppKeys.DetailScrollUp) {
			logger.Debug("Detail scroll up key pressed", map[string]any{"current_offset": m.offset})
			if m.offset > 0 {
				m.offset--
			}
		} else if key.Matches(msg, keys.AppKeys.DetailScrollDown) {
            lines := m.calculateLines()
            visibleHeight := m.height - 2
            if visibleHeight <= 0 {
                // Not enough height to scroll; keep offset unchanged.
                break
            }
            maxOffset := len(lines) - visibleHeight
            if maxOffset < 0 {
                maxOffset = 0
            }
			logger.Debug("Detail scroll down key pressed", map[string]any{"current_offset": m.offset, "total_lines": len(lines), "height": m.height, "max_offset": maxOffset})
			if m.offset < maxOffset {
				m.offset++
			}
		}
	}
	return m, nil
}

// SetSize updates the dimensions
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetVisible sets whether the detail pane is visible
func (m *Model) SetVisible(visible bool) {
	m.visible = visible
}

// ToggleVisibility toggles whether the detail pane is visible
func (m *Model) ToggleVisibility() {
	m.visible = !m.visible
}

// Visible returns whether the detail pane is visible
func (m Model) Visible() bool {
	return m.visible
}

// SetData updates the selected row data
func (m *Model) SetData(columns []table.Column, row table.Row) {
	same := true
	if len(m.row) != len(row) {
		same = false
	} else {
		for i := range row {
			if m.row[i] != row[i] {
				same = false
				break
			}
		}
	}

	m.columns = columns
	m.row = row
	if !same {
		m.offset = 0
	}
}

func truncateOrPad(s string, width int) string {
	currentWidth := lipgloss.Width(s)
	if currentWidth > width {
		runes := []rune(s)
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
	return s + strings.Repeat(" ", width-currentWidth)
}

// View renders the detail pane
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	t := theme.Current

	borderStyle := t.BorderUnfocused.Copy().
		Width(m.width).
		Height(m.height)

	if len(m.columns) == 0 || m.row == nil || len(m.row) == 0 {
		emptyMsg := "No row selected"
		emptyStyle := lipgloss.NewStyle().
			Width(m.width - 2).
			Height(m.height - 2).
			Align(lipgloss.Center).
			Foreground(t.Colors.ForegroundDim)
		return borderStyle.Render(emptyStyle.Render(emptyMsg))
	}

    lines := m.calculateLines()
	visibleHeight := m.height - 2

	if visibleHeight <= 0 {
		return borderStyle.Render("")
	}

	if m.offset > len(lines)-visibleHeight {
		m.offset = max(0, len(lines)-visibleHeight)
	}

	endIdx := min(m.offset+visibleHeight, len(lines))
	visibleLines := lines[m.offset:endIdx]

	renderedContent := lipgloss.NewStyle().
		Width(m.width - 2).
		Height(m.height - 2).
		Render(strings.Join(visibleLines, "\n"))

	return borderStyle.Render(renderedContent)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	var currentLine strings.Builder

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else {
			if currentLine.Len()+1+len(word) > width {
				result.WriteString(currentLine.String() + "\n")
				currentLine.Reset()
				currentLine.WriteString(word)
			} else {
				currentLine.WriteString(" " + word)
			}
		}
	}

	if currentLine.Len() > 0 {
		result.WriteString(currentLine.String())
	}

	return result.String()
}
