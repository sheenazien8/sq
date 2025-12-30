package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/theme"
)

// Helper functions
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

	if m.CreateConnectionModal.Visible() {
		return m.CreateConnectionModal.View()
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

	// Show tabs if they exist, otherwise show placeholder
	if m.Tabs.HasTabs() {
		// For all tabs, use full height since filter is now inside tab for table tabs
		contentView := tableBorderStyle.
			Width(m.ContentWidth - 4).
			Height(contentHeight).
			Render(m.Tabs.View())
		mainArea = contentView
	} else {
		// Show placeholder when no tabs are open
		// Account for border (2 chars on each side = 4 total)
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

	// Debug: log if width exceeds terminal
	if middleSectionWidth > m.TerminalWidth {
		tea.Printf("WIDTH OVERFLOW: terminal=%d, sidebar=%d, mainArea=%d, total=%d",
			m.TerminalWidth, sidebarActualWidth, lipgloss.Width(mainArea), middleSectionWidth)
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.HeaderStyle, middleSection, m.FooterStyle)
}
