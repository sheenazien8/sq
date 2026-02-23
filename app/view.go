package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/storage"
	"github.com/sheenazien8/sq/ui/theme"
)

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// View renders the main application view
func (m Model) View() string {
	if m.TerminalWidth == 0 || m.TerminalHeight == 0 {
		return "Loading..."
	}

	if m.ExitModal.Visible() {
		return m.ExitModal.View()
	}

	if m.AlertModal.Visible() {
		return m.AlertModal.View()
	}

	if m.CreateConnectionModal.Visible() {
		return m.CreateConnectionModal.View()
	}

	if m.EditConnectionModal.Visible() {
		return m.EditConnectionModal.View()
	}

	if m.DeleteConnectionModal.Visible() {
		return m.DeleteConnectionModal.View()
	}

	if m.CellPreviewModal.Visible() {
		return m.CellPreviewModal.View()
	}

	if m.ActionModal.Visible() {
		return m.ActionModal.View()
	}

	if m.EditCellModal.Visible() {
		return m.EditCellModal.View()
	}

	if m.ConfirmModal.Visible() {
		return m.ConfirmModal.View()
	}

	if m.HelpModal.Visible() {
		return m.HelpModal.View()
	}

	if m.ColumnVisibilityModal.Visible() {
		return m.ColumnVisibilityModal.View()
	}

	if m.QueryHistoryModal.Visible() {
		return m.QueryHistoryModal.View()
	}

	switch m.CurrentPage {
	case PageConnectionManager:
		return m.viewConnectionManager()
	case PageDatabaseOperations:
		return m.viewDatabaseOperations()
	default:
		return "Unknown page"
	}
}

func (m Model) viewConnectionManager() string {
	t := theme.Current

	title := t.Header.Render("Connection Manager")

	connections, err := storage.GetAllConnections()
	if err != nil {
		content := t.Header.Render("Connection Manager") + "\n\n"
		content += "Error loading connections: " + err.Error()
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.HeaderStyle,
			lipgloss.Place(
				m.TerminalWidth,
				m.TerminalHeight-lipgloss.Height(m.HeaderStyle)-lipgloss.Height(m.FooterStyle),
				lipgloss.Center,
				lipgloss.Center,
				content,
			),
			m.FooterStyle,
		)
	}

	var content string
	if len(connections) == 0 {
		content = title + "\n\n"
		content += "No saved connections found.\n\n"
		content += "Press 'n' to create a new connection\n"
		content += "Press '?' for help, 'q' to quit"
	} else {
		content = title + "\n\n"

		visibleHeight := m.TerminalHeight - lipgloss.Height(m.HeaderStyle) - lipgloss.Height(m.FooterStyle) - 8
		startIdx := m.connectionManagerOffset
		endIdx := min(startIdx+visibleHeight, len(connections))

		for i := startIdx; i < endIdx; i++ {
			conn := connections[i]
			line := ""

			if i == m.connectionManagerCursor {
				line += t.SidebarSelected.Render("> ")
			} else {
				line += "  "
			}

			line += conn.Name + " (" + conn.Driver + ")"

			if i == m.connectionManagerCursor {
				content += t.SidebarSelected.Render(line) + "\n"
			} else {
				content += t.SidebarItem.Render(line) + "\n"
			}
		}

		content += "\n"
		content += "j/k: Navigate | Enter: Connect | n: New | w: Edit | x: Delete | q: Quit"
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.HeaderStyle,
		lipgloss.Place(
			m.TerminalWidth,
			m.TerminalHeight-lipgloss.Height(m.HeaderStyle)-lipgloss.Height(m.FooterStyle),
			lipgloss.Center,
			lipgloss.Center,
			content,
		),
		m.FooterStyle,
	)
}

func (m Model) viewDatabaseOperations() string {
	t := theme.Current

	var sidebarView string
	var sidebarActualWidth int
	if !m.sidebarCollapsed {
		sidebarView = m.Sidebar.View()
		sidebarActualWidth = lipgloss.Width(sidebarView)
	} else {
		sidebarActualWidth = 0
	}

	var tableBorderStyle lipgloss.Style
	if m.Focus == FocusMain {
		tableBorderStyle = t.BorderFocused
	} else {
		tableBorderStyle = t.BorderUnfocused
	}

	contentHeight := m.ContentHeight - 2

	var mainArea string

	if m.Tabs.HasTabs() {
		contentView := tableBorderStyle.
			Width(m.ContentWidth - 4).
			Height(contentHeight).
			Render(m.Tabs.View())
		mainArea = contentView
	} else {
		placeholderStyle := lipgloss.NewStyle().
			Foreground(t.Colors.ForegroundDim).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.ContentWidth - 4).
			Height(contentHeight - 2)

		placeholder := placeholderStyle.Render("Select a table from the sidebar to open it in a tab\n(Press Enter on a table to open)")

		mainArea = tableBorderStyle.
			Width(m.ContentWidth - 4).
			Height(contentHeight).
			Render(placeholder)
	}

	var middleSection string
	if !m.sidebarCollapsed {
		middleSection = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainArea)
	} else {
		middleSection = mainArea
	}
	middleSectionWidth := lipgloss.Width(middleSection)

	if middleSectionWidth > m.TerminalWidth {
		tea.Printf("WIDTH OVERFLOW: terminal=%d, sidebar=%d, mainArea=%d, total=%d",
			m.TerminalWidth, sidebarActualWidth, lipgloss.Width(mainArea), middleSectionWidth)
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.HeaderStyle, middleSection, m.FooterStyle)
}
