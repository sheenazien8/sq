package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/logger"
	"github.com/sheenazien8/db-client-tui/storage"
	"github.com/sheenazien8/db-client-tui/ui/filter"
	"github.com/sheenazien8/db-client-tui/ui/modal"
	"github.com/sheenazien8/db-client-tui/ui/sidebar"
	"github.com/sheenazien8/db-client-tui/ui/table"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case sidebar.TableSelectedMsg:
		logger.Debug("Table selected", map[string]any{
			"connection": msg.ConnectionName,
			"table":      msg.TableName,
		})

		// Get connection from sidebar
		activeDB := m.Sidebar.ActiveDatabase()
		if activeDB == nil {
			logger.Error("No active database", map[string]any{})
			return m, nil
		}

		// TODO: Load actual table data from database
		// For now, use mock data
		m.columns, m.allRows = getTableData()
		m.columnNames = getColumnNames(m.columns)

		// Initialize filter if not already done
		if m.Filter.GetFilter() == nil && len(m.columnNames) == 0 {
			m.Filter = filter.New(m.columnNames)
			m.Filter.SetWidth(m.ContentWidth)
		}

		// Add tab with table data
		tabName := msg.ConnectionName + "." + msg.TableName
		m.Tabs.AddTableTab(tabName, m.columns, m.allRows)

		// Set tab dimensions (filter bar is always 3 lines with border)
		tableWidth := m.ContentWidth - 4
		tableHeight := m.ContentHeight - 3 - 2
		m.Tabs.SetSize(tableWidth, tableHeight)

		// Switch focus to main area
		m.Focus = FocusMain
		m.Sidebar.SetFocused(false)
		m.Tabs.SetFocused(true)

		return m, nil

	case tea.WindowSizeMsg:
		m.TerminalWidth = msg.Width
		m.TerminalHeight = msg.Height
		m.SidebarWidth = 32
		contentWidth := m.TerminalWidth - m.SidebarWidth

		t := theme.Current

		headerStyle := t.Header.Width(m.TerminalWidth)

		footerStyle := t.Footer.Width(m.TerminalWidth)

		m.HeaderStyle = headerStyle.Render("DB Client TUI [" + t.Name + "]")
		m.FooterStyle = footerStyle.Render("Tab: Switch | /: Filter | T: Theme | q: Quit")

		headerHeight := lipgloss.Height(m.HeaderStyle)
		footerHeight := lipgloss.Height(m.FooterStyle)

		contentHeight := m.TerminalHeight - headerHeight - footerHeight

		m.ContentWidth = contentWidth
		m.ContentHeight = contentHeight

		tableWidth := contentWidth - 4
		// Filter bar is always 3 lines (with border)
		tableHeight := contentHeight - 3 - 2

		if !m.initialized {
			logger.Debug("Initial window size", map[string]any{
				"width":  msg.Width,
				"height": msg.Height,
			})
			m.Filter = filter.New([]string{}) // Initialize with empty columns
			m.initialized = true
		}

		m.Filter.SetWidth(contentWidth)
		m.Tabs.SetSize(tableWidth, tableHeight)

		m.Sidebar.SetSize(m.SidebarWidth, contentHeight)

		m.ExitModal.SetSize(m.TerminalWidth, m.TerminalHeight)
		m.CreateConnectionModal.SetSize(m.TerminalWidth, m.TerminalHeight)

	case tea.KeyMsg:
		if m.ExitModal.Visible() {
			m.ExitModal, cmd = m.ExitModal.Update(msg)
			cmds = append(cmds, cmd)

			// Check modal result
			if !m.ExitModal.Visible() {
				if m.ExitModal.Confirmed() {
					return m, tea.Quit
				} else {
					m.Focus = FocusSidebar
					m.Sidebar.SetFocused(true)
				}
			}
			return m, tea.Batch(cmds...)
		}

		if m.Filter.Visible() {
			prevActive := m.Filter.Active()
			m.Filter, cmd = m.Filter.Update(msg)
			cmds = append(cmds, cmd)

			// Check if filter was closed (applied or escaped)
			if !m.Filter.Visible() {
				f := m.Filter.GetFilter()
				
				// If filter was applied (Enter pressed), add it to the tab
				if f != nil && (m.Filter.Active() || m.Filter.Active() != prevActive) {
					m.Tabs.AddActiveTabFilter(*f)
					m = m.applyFilterToActiveTab()
				}
				
				// Adjust table size now that filter is hidden
				m = m.updateTabSize()
				// Return focus to tabs
				m.Focus = FocusMain
				m.Sidebar.SetFocused(false)
				m.Tabs.SetFocused(true)
			}
			return m, tea.Batch(cmds...)
		}

		if m.CreateConnectionModal.Visible() {
			m.CreateConnectionModal, cmd = m.CreateConnectionModal.Update(msg)
			cmds = append(cmds, cmd)

			// Check if modal was closed
			if !m.CreateConnectionModal.Visible() {
				// Check if user submitted the form
				if m.CreateConnectionModal.Result() == modal.ResultSubmit {
					name := m.CreateConnectionModal.GetName()
					driver := m.CreateConnectionModal.GetDriver()
					url := m.CreateConnectionModal.GetURL()
					_, err := storage.CreateConnection(
						name,
						driver,
						url,
					)

					if err != nil {
						logger.Error(fmt.Sprintf("Failed for creating connection: %s", err), map[string]any{
							"name":   name,
							"driver": driver,
							"url":    url,
						})
						m.Focus = FocusCreateConnectionModal
						m.CreateConnectionModal.Show()
						return m, tea.Batch(cmds...)
					}

					// Refresh sidebar connections list after successful creation
					m.Sidebar.RefreshConnections()
				}
				m.Focus = FocusSidebar
				m.Sidebar.SetFocused(true)
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.Focus == FocusSidebar || m.Focus == FocusMain {
				m.ExitModal.Show()
				m.Focus = FocusExitModal
			}

		case "/", "f":
			if m.Focus == FocusMain && m.Tabs.HasTabs() {
				// Get active tab's column names
				_, _, columnNames := m.Tabs.GetActiveTabData()
				
				if columnNames != nil {
					// Update filter columns based on active tab
					m.Filter = filter.New(columnNames)
					m.Filter.SetWidth(m.ContentWidth)
					m.Filter.SetVisible(true)
					m.Focus = FocusFilter
					m = m.updateTabSize()
				}
			} else {
				m.Sidebar, cmd = m.Sidebar.Update(msg)
				cmds = append(cmds, cmd)
			}

		case "n":
			if m.Focus == FocusSidebar {
				m.CreateConnectionModal.Show()
				m.Focus = FocusCreateConnectionModal
			}
		case "tab":
			// Only allow switching to main table if tabs are open
			if m.Focus == FocusSidebar {
				if m.Tabs.HasTabs() {
					logger.Debug("Focus changed", map[string]any{
						"from": "sidebar",
						"to":   "main",
					})
					m.Focus = FocusMain
					m.Sidebar.SetFocused(false)
					m.Tabs.SetFocused(true)
				}
			} else {
				logger.Debug("Focus changed", map[string]any{
					"from": "main",
					"to":   "sidebar",
				})
				m.Focus = FocusSidebar
				m.Sidebar.SetFocused(true)
				m.Tabs.SetFocused(false)
			}

		case "T":
			themes := theme.GetAvailableThemes()
			m.themeIndex = (m.themeIndex + 1) % len(themes)
			newTheme := themes[m.themeIndex]
			logger.Info("Theme changed", map[string]any{"theme": newTheme})
			theme.SetTheme(theme.GetThemeByName(newTheme))
			if m.config != nil {
				m.config.SetTheme(newTheme)
				_ = m.config.Save()
			}
			m = m.updateStyles()

		case "C":
			m.Tabs.ClearActiveTabFilters()
			m = m.applyFilterToActiveTab()
			m = m.updateTabSize()

		default:
			if m.Focus == FocusSidebar {
				m.Sidebar, cmd = m.Sidebar.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.Tabs, cmd = m.Tabs.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// applyFilterToActiveTab filters the table rows in the active tab based on current filters
func (m Model) applyFilterToActiveTab() Model {
	activeTab := m.Tabs.ActiveTab()
	if activeTab == nil {
		return m
	}

	// Get the active tab's original data
	allRows, _, columnNames := m.Tabs.GetActiveTabData()
	if allRows == nil {
		return m
	}

	filters := m.Tabs.GetActiveTabFilters()
	
	if len(filters) == 0 {
		logger.Debug("Filter cleared", map[string]any{"total_rows": len(allRows)})
		// Reset to original data
		if tableModel, ok := activeTab.Content.(table.Model); ok {
			tableModel.SetRows(allRows)
			m.Tabs.UpdateActiveTabContent(tableModel)
		}
	} else {
		logger.Debug("Filters applied", map[string]any{
			"filter_count": len(filters),
		})
		
		// Convert to [][]string for filtering
		rows := make([][]string, len(allRows))
		for i, row := range allRows {
			rows[i] = []string(row)
		}
		
		// Apply all filters sequentially (AND logic)
		filtered := rows
		for _, f := range filters {
			filtered = filter.FilterRows(filtered, columnNames, &f)
		}
		
		tableRows := make([]table.Row, len(filtered))
		for i, row := range filtered {
			tableRows[i] = table.Row(row)
		}
		
		logger.Debug("Filter result", map[string]any{
			"original_rows": len(allRows),
			"filtered_rows": len(tableRows),
		})
		
		if tableModel, ok := activeTab.Content.(table.Model); ok {
			tableModel.SetRows(tableRows)
			m.Tabs.UpdateActiveTabContent(tableModel)
		}
	}
	return m
}

// updateStyles refreshes the header and footer styles after theme change
func (m Model) updateStyles() Model {
	t := theme.Current
	m.HeaderStyle = t.Header.Width(m.TerminalWidth).Render("DB Client TUI [" + t.Name + "]")
	m.FooterStyle = t.Footer.Width(m.TerminalWidth).Render("Tab: Switch | /: Filter | T: Theme | q: Quit")
	return m
}

// updateTabSize adjusts tab size based on filter visibility
func (m Model) updateTabSize() Model {
	tableWidth := m.ContentWidth - 4
	contentHeight := m.ContentHeight

	// Filter bar is always 3 lines (with border)
	filterBarHeight := 3

	tableHeight := contentHeight - filterBarHeight - 2
	m.Tabs.SetSize(tableWidth, tableHeight)
	return m
}

// getColumnNames extracts column names from columns
func getColumnNames(columns []table.Column) []string {
	names := make([]string, len(columns))
	for i, col := range columns {
		names[i] = col.Title
	}
	return names
}

// getTableData returns sample table columns and rows
func getTableData() ([]table.Column, []table.Row) {
	columns := []table.Column{
		{Title: "Rank", Width: 6},
		{Title: "City", Width: 20},
		{Title: "Country", Width: 20},
		{Title: "Population", Width: 15},
		{Title: "Continent", Width: 15},
		{Title: "Region", Width: 20},
		{Title: "GDP (Billion $)", Width: 18},
		{Title: "Area (km²)", Width: 15},
	}

	rows := []table.Row{
		{"1", "Tokyo", "Japan", "37,274,000", "Asia", "East Asia", "1,520", "2,194"},
		{"2", "Delhi", "India", "32,065,760", "Asia", "South Asia", "293", "1,484"},
		{"3", "Shanghai", "China", "28,516,904", "Asia", "East Asia", "516", "6,341"},
		{"4", "Dhaka", "Bangladesh", "22,478,116", "Asia", "South Asia", "110", "306"},
		{"5", "São Paulo", "Brazil", "22,429,800", "S. America", "South America", "430", "1,521"},
		{"6", "Mexico City", "Mexico", "22,085,140", "N. America", "Central America", "411", "1,485"},
		{"7", "Cairo", "Egypt", "21,750,020", "Africa", "North Africa", "135", "3,085"},
		{"8", "Beijing", "China", "21,333,332", "Asia", "East Asia", "578", "16,411"},
		{"9", "Mumbai", "India", "20,961,472", "Asia", "South Asia", "310", "603"},
		{"10", "Osaka", "Japan", "19,059,856", "Asia", "East Asia", "654", "13,229"},
		{"11", "Chongqing", "China", "16,874,740", "Asia", "East Asia", "215", "82,403"},
		{"12", "Karachi", "Pakistan", "16,839,950", "Asia", "South Asia", "78", "3,780"},
		{"13", "Istanbul", "Turkey", "15,636,243", "Europe", "Western Asia", "245", "5,461"},
		{"14", "Kinshasa", "DR Congo", "15,628,085", "Africa", "Central Africa", "17", "9,965"},
		{"15", "Lagos", "Nigeria", "15,387,639", "Africa", "West Africa", "136", "1,171"},
		{"16", "Buenos Aires", "Argentina", "15,369,919", "S. America", "South America", "316", "203"},
		{"17", "Kolkata", "India", "15,133,888", "Asia", "South Asia", "150", "1,887"},
		{"18", "Manila", "Philippines", "14,406,059", "Asia", "Southeast Asia", "123", "1,780"},
		{"19", "Tianjin", "China", "14,011,828", "Asia", "East Asia", "188", "11,917"},
		{"20", "Guangzhou", "China", "13,964,637", "Asia", "East Asia", "390", "7,434"},
		{"21", "Rio De Janeiro", "Brazil", "13,634,274", "S. America", "South America", "176", "1,182"},
		{"22", "Lahore", "Pakistan", "13,541,764", "Asia", "South Asia", "58", "1,772"},
		{"23", "Bangalore", "India", "13,193,035", "Asia", "South Asia", "110", "709"},
		{"24", "Shenzhen", "China", "12,831,330", "Asia", "East Asia", "422", "1,997"},
		{"25", "Moscow", "Russia", "12,640,818", "Europe", "Eastern Europe", "402", "2,561"},
		{"26", "Chennai", "India", "11,503,293", "Asia", "South Asia", "78", "426"},
		{"27", "Bogota", "Colombia", "11,344,312", "S. America", "South America", "103", "1,775"},
		{"28", "Paris", "France", "11,142,303", "Europe", "Western Europe", "669", "2,845"},
		{"29", "Jakarta", "Indonesia", "11,074,811", "Asia", "Southeast Asia", "175", "664"},
		{"30", "Lima", "Peru", "11,044,607", "S. America", "South America", "98", "2,672"},
		{"31", "Bangkok", "Thailand", "10,899,698", "Asia", "Southeast Asia", "247", "1,569"},
		{"32", "Hyderabad", "India", "10,534,418", "Asia", "South Asia", "74", "650"},
		{"33", "Seoul", "South Korea", "9,975,709", "Asia", "East Asia", "779", "605"},
		{"34", "Nagoya", "Japan", "9,571,596", "Asia", "East Asia", "363", "5,190"},
		{"35", "London", "United Kingdom", "9,540,576", "Europe", "Western Europe", "731", "1,572"},
		{"36", "Chengdu", "China", "9,478,521", "Asia", "East Asia", "187", "14,312"},
		{"37", "Nanjing", "China", "9,429,381", "Asia", "East Asia", "169", "6,587"},
		{"38", "Tehran", "Iran", "9,381,546", "Asia", "Western Asia", "150", "730"},
		{"39", "Ho Chi Minh City", "Vietnam", "9,077,158", "Asia", "Southeast Asia", "61", "2,061"},
		{"40", "Luanda", "Angola", "8,952,496", "Africa", "Central Africa", "65", "2,257"},
		{"41", "Wuhan", "China", "8,591,611", "Asia", "East Asia", "163", "8,494"},
		{"42", "Xi An Shaanxi", "China", "8,537,646", "Asia", "East Asia", "115", "10,752"},
		{"43", "Ahmedabad", "India", "8,450,228", "Asia", "South Asia", "68", "505"},
		{"44", "Kuala Lumpur", "Malaysia", "8,419,566", "Asia", "Southeast Asia", "150", "243"},
		{"45", "New York City", "United States", "8,177,020", "N. America", "North America", "1,210", "783"},
		{"46", "Hangzhou", "China", "8,044,878", "Asia", "East Asia", "152", "16,596"},
		{"47", "Surat", "India", "7,784,276", "Asia", "South Asia", "45", "326"},
		{"48", "Suzhou", "China", "7,764,499", "Asia", "East Asia", "185", "8,488"},
		{"49", "Hong Kong", "Hong Kong", "7,643,256", "Asia", "East Asia", "341", "1,106"},
		{"50", "Riyadh", "Saudi Arabia", "7,538,200", "Asia", "Western Asia", "215", "1,973"},
	}

	return columns, rows
}
