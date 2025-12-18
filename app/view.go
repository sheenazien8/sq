package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

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

	t := theme.Current

	sidebarView := m.Sidebar.View()

	var tableBorderStyle lipgloss.Style
	if m.Focus == FocusMain {
		tableBorderStyle = t.BorderFocused
	} else {
		tableBorderStyle = t.BorderUnfocused
	}

	contentHeight := m.ContentHeight - 2
	filterBarHeight := 0

	if m.Filter.Visible() {
		filterBarHeight = 3
	} else if m.Filter.Active() {
		filterBarHeight = 1
	}

	tableHeight := contentHeight - filterBarHeight

	var mainArea string

	// Only show table content if a database has been selected
	if !m.Sidebar.HasActiveDatabase() {
		// Show placeholder when no database is selected
		placeholderStyle := lipgloss.NewStyle().
			Foreground(t.Colors.ForegroundDim).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.ContentWidth).
			Height(contentHeight)

		placeholder := placeholderStyle.Render("Select a database from the sidebar\n(Press Enter to select)")

		mainArea = tableBorderStyle.
			Width(m.ContentWidth).
			Height(contentHeight).
			Render(placeholder)
	} else {
		contentView := tableBorderStyle.
			Width(m.ContentWidth).
			Height(tableHeight).
			Render(m.Main.View())

		if m.Filter.Visible() {
			filterView := m.Filter.View()
			mainArea = lipgloss.JoinVertical(lipgloss.Left, filterView, contentView)
		} else if m.Filter.Active() {
			f := m.Filter.GetFilter()
			statusStyle := lipgloss.NewStyle().
				Foreground(t.Colors.Foreground).
				Background(t.Colors.Primary).
				Padding(0, 1)
			clearHint := lipgloss.NewStyle().
				Foreground(t.Colors.ForegroundDim).
				Render(" | C: clear | /: edit")
			filterStatus := statusStyle.Render("Active: "+f.Column+" "+string(f.Operator)+" \""+f.Value+"\"") + clearHint
			mainArea = lipgloss.JoinVertical(lipgloss.Left, filterStatus, contentView)
		} else {
			mainArea = contentView
		}
	}

	middleSection := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainArea)

	return lipgloss.JoinVertical(lipgloss.Left, m.HeaderStyle, middleSection, m.FooterStyle)
}
