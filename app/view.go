package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/ui/tab"
	"github.com/sheenazien8/db-client-tui/ui/theme"
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

	// Filter bar height depends on whether we're editing or just showing status
	var filterBarHeight int
	if m.Filter.Visible() {
		filterBarHeight = 3 // Editing mode needs 3 lines
	} else {
		filterBarHeight = 3 // Status line with border needs 3 lines
	}

	tableHeight := contentHeight - filterBarHeight

	var mainArea string

	// Show tabs if they exist, otherwise show placeholder
	if m.Tabs.HasTabs() {
		tabType := m.Tabs.GetActiveTabType()

		// For query editor tabs, don't show filter bar - use full height
		if tabType == tab.TabTypeQuery {
			contentView := tableBorderStyle.
				Width(m.ContentWidth - 4).
				Height(contentHeight).
				Render(m.Tabs.View())
			mainArea = contentView
		} else {
			// Show tabbed interface with filter bar
			// Account for border (2 chars on each side = 4 total)
			contentView := tableBorderStyle.
				Width(m.ContentWidth - 4).
				Height(tableHeight).
				Render(m.Tabs.View())

			// Always show filter bar for table/structure tabs
			var filterView string
			if m.Filter.Visible() {
				// Show filter input (3 lines)
				filterView = m.Filter.View()
			} else {
				// Show filter status or empty filter bar (1 line with border)
				activeTabFilters := m.Tabs.GetActiveTabFilters()

				var message string
				if len(activeTabFilters) > 0 {
					// Show all active filters with AND
					var filterStrings []string
					for _, f := range activeTabFilters {
						filterStrings = append(filterStrings, f.Column+" "+string(f.Operator)+" \""+f.Value+"\"")
					}
					message = "Filters (" + intToStr(len(activeTabFilters)) + "): " + joinStrings(filterStrings, " AND ") + " | C: clear | /: add"
				} else {
					message = "No filter | Press / to filter"
				}

				filterBarStyle := lipgloss.NewStyle().
					Foreground(t.Colors.Foreground).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(t.Colors.BorderUnfocused).
					Width(m.ContentWidth-4).
					Padding(0, 1)
				filterView = filterBarStyle.Render(message)
			}

			mainArea = lipgloss.JoinVertical(lipgloss.Left, filterView, contentView)
		}
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
			Height(contentHeight - 2).
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
