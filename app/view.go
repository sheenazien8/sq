package app

import (
	"github.com/charmbracelet/lipgloss"
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

	t := theme.Current

	sidebarView := m.Sidebar.View()

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
		filterBarHeight = 3 // Status line with border needs 3 lines (1 content + 2 border)
	}
	
	tableHeight := contentHeight - filterBarHeight

	var mainArea string

	// Show tabs if they exist, otherwise show placeholder
	if m.Tabs.HasTabs() {
		// Show tabbed interface
		contentView := tableBorderStyle.
			Width(m.ContentWidth).
			Height(tableHeight).
			Render(m.Tabs.View())

		// Always show filter bar
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
				Padding(0, 1)
			filterView = filterBarStyle.Render(message)
		}

		mainArea = lipgloss.JoinVertical(lipgloss.Left, filterView, contentView)
	} else {
		// Show placeholder when no tabs are open
		placeholderStyle := lipgloss.NewStyle().
			Foreground(t.Colors.ForegroundDim).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.ContentWidth).
			Height(contentHeight)

		placeholder := placeholderStyle.Render("Select a table from the sidebar to open it in a tab\n(Press Enter on a table to open)")

		mainArea = tableBorderStyle.
			Width(m.ContentWidth).
			Height(contentHeight).
			Render(placeholder)
	}

	middleSection := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainArea)

	return lipgloss.JoinVertical(lipgloss.Left, m.HeaderStyle, middleSection, m.FooterStyle)
}
