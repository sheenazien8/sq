package app

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/drivers"
	"github.com/sheenazien8/db-client-tui/logger"
	"github.com/sheenazien8/db-client-tui/storage"
	"github.com/sheenazien8/db-client-tui/ui/filter"
	"github.com/sheenazien8/db-client-tui/ui/modal"
	"github.com/sheenazien8/db-client-tui/ui/sidebar"
	"github.com/sheenazien8/db-client-tui/ui/tab"
	"github.com/sheenazien8/db-client-tui/ui/table"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case sidebar.ConnectionSelectedMsg:
		logger.Debug("Connection selected", map[string]any{
			"name": msg.ConnectionName,
			"type": msg.ConnectionType,
			"url":  msg.ConnectionURL,
		})

		// Connect to database and load tables
		err := m.connectToDatabase(msg.ConnectionName, msg.ConnectionType, msg.ConnectionURL)
		if err != nil {
			logger.Error("Failed to connect to database", map[string]any{
				"connection": msg.ConnectionName,
				"error":      err.Error(),
			})
			// TODO: Show error message to user
			return m, nil
		}

		// Sidebar is updated via updateSidebarConnection

		return m, nil

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

		// Load actual table data from database
		err := m.loadTableData(msg.ConnectionName, msg.TableName)
		if err != nil {
			logger.Error("Failed to load table data", map[string]any{
				"connection": msg.ConnectionName,
				"table":      msg.TableName,
				"error":      err.Error(),
			})
			// TODO: Show error message to user
			return m, nil
		}

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
		m = m.updateFooter()

		return m, nil

	case tea.WindowSizeMsg:
		m.TerminalWidth = msg.Width
		m.TerminalHeight = msg.Height
		m.SidebarWidth = 32
		contentWidth := m.TerminalWidth
		if !m.sidebarCollapsed {
			contentWidth -= m.SidebarWidth
		}

		t := theme.Current

		headerStyle := t.Header.Width(m.TerminalWidth)

		footerStyle := t.Footer.Width(m.TerminalWidth)

		m.HeaderStyle = headerStyle.Render("DB Client TUI [" + t.Name + "]")
		m.FooterStyle = footerStyle.Render(m.getFooterHelp())

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
		m.CellPreviewModal.SetSize(m.TerminalWidth, m.TerminalHeight)

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
					m = m.updateFooter()
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
				m = m.updateFooter()
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
				m = m.updateFooter()
			}
			return m, tea.Batch(cmds...)
		}

		if m.CellPreviewModal.Visible() {
			m.CellPreviewModal, cmd = m.CellPreviewModal.Update(msg)
			cmds = append(cmds, cmd)

			// Check if modal was closed
			if !m.CellPreviewModal.Visible() {
				m.Focus = FocusMain
				m.Sidebar.SetFocused(false)
				m.Tabs.SetFocused(true)
				m = m.updateFooter()
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.Focus == FocusSidebar || m.Focus == FocusMain {
				m.ExitModal.Show()
				m.Focus = FocusExitModal
				m = m.updateFooter()
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
					m = m.updateFooter()
				}
			} else {
				m.Sidebar, cmd = m.Sidebar.Update(msg)
				cmds = append(cmds, cmd)
			}

		case "n":
			if m.Focus == FocusSidebar {
				m.CreateConnectionModal.Show()
				m.Focus = FocusCreateConnectionModal
				m = m.updateFooter()
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
					m = m.updateFooter()
				}
			} else {
				// Only switch to sidebar if it's not collapsed
				if !m.sidebarCollapsed {
					logger.Debug("Focus changed", map[string]any{
						"from": "main",
						"to":   "sidebar",
					})
					m.Focus = FocusSidebar
					m.Sidebar.SetFocused(true)
					m.Tabs.SetFocused(false)
					m = m.updateFooter()
				}
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

		case "p":
			if m.Focus == FocusMain && m.Tabs.HasTabs() {
				// Get the selected cell content
				activeTab := m.Tabs.ActiveTab()
				if tableModel, ok := activeTab.Content.(table.Model); ok {
					cellContent := tableModel.SelectedCell()
					if cellContent != "" {
						m.CellPreviewModal.Show(cellContent)
						m.Focus = FocusCellPreviewModal
						m = m.updateFooter()
					}
				}
			}

		case "y":
			if m.Focus == FocusMain && m.Tabs.HasTabs() {
				// Yank (copy) the selected cell content to clipboard
				activeTab := m.Tabs.ActiveTab()
				if tableModel, ok := activeTab.Content.(table.Model); ok {
					cellContent := tableModel.SelectedCell()
					if cellContent != "" {
						err := clipboard.WriteAll(cellContent)
						if err != nil {
							logger.Error("Failed to copy to clipboard", map[string]any{"error": err.Error()})
						} else {
							logger.Info("Cell content copied to clipboard", map[string]any{"length": len(cellContent)})
						}
					}
				}
			}

		case "d":
			// Show table structure in a new tab
			if m.Focus == FocusMain && m.Tabs.HasTabs() {
				err := m.loadTableStructure()
				if err != nil {
					logger.Error("Failed to load table structure", map[string]any{"error": err.Error()})
				}
				return m, nil
			} else if m.Focus == FocusSidebar {
				// Load structure for selected table in sidebar
				activeDB := m.Sidebar.ActiveDatabase()
				if activeDB != nil && activeDB.Connected {
					selectedTable := m.Sidebar.SelectedTable()
					if selectedTable != "" {
						m.currentConnection = activeDB.Name
						connections := m.Sidebar.GetConnections()
						for _, conn := range connections {
							if conn.Name == activeDB.Name {
								m.currentDatabase = extractDatabaseName(conn.Host, conn.Type)
								break
							}
						}
						m.currentTable = selectedTable
						err := m.loadTableStructure()
						if err != nil {
							logger.Error("Failed to load table structure", map[string]any{"error": err.Error()})
						} else {
							// Switch focus to main area
							m.Focus = FocusMain
							m.Sidebar.SetFocused(false)
							m.Tabs.SetFocused(true)
							m = m.updateFooter()
						}
						return m, nil
					}
				}
			}

		case "s", "S":
			m.sidebarCollapsed = !m.sidebarCollapsed
			// Recalculate layout after toggling sidebar
			contentWidth := m.TerminalWidth
			if !m.sidebarCollapsed {
				contentWidth -= m.SidebarWidth
			}
			m.ContentWidth = contentWidth
			m.Filter.SetWidth(contentWidth)
			m.Tabs.SetSize(contentWidth-4, m.ContentHeight-3-2)
			m = m.updateFooter()

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

// connectToDatabase creates a driver instance and connects to the database
func (m *Model) connectToDatabase(name, connType, url string) error {
	var driver drivers.Driver

	switch connType {
	case "mysql":
		driver = &drivers.MySQL{}
	default:
		return fmt.Errorf("unsupported database type: %s", connType)
	}

	err := driver.Connect(url)
	if err != nil {
		return err
	}

	// Extract database name from URL for MySQL
	dbName := extractDatabaseName(url, connType)

	// Get tables from database
	tables, err := driver.GetTables(dbName)
	if err != nil {
		return err
	}

	// Store the driver connection
	m.dbConnections[name] = driver

	// Update sidebar with real tables and connected status
	m.Sidebar.UpdateConnection(name, tables[dbName], true)

	return nil
}

// extractDatabaseName extracts the database name from connection URL
func extractDatabaseName(url, connType string) string {
	switch connType {
	case "mysql":
		// For MySQL URLs like "user:pass@tcp(host:port)/database"
		parts := strings.Split(url, "/")
		if len(parts) > 1 {
			// Remove query parameters if any
			dbPart := strings.Split(parts[len(parts)-1], "?")[0]
			return dbPart
		}
	}
	return ""
}

// loadTableData loads table data from the database connection
func (m *Model) loadTableData(connectionName, tableName string) error {
	driver, exists := m.dbConnections[connectionName]
	if !exists {
		return fmt.Errorf("no active connection for %s", connectionName)
	}

	// Extract database name from connection
	connections := m.Sidebar.GetConnections()
	var dbName string
	for _, conn := range connections {
		if conn.Name == connectionName {
			dbName = extractDatabaseName(conn.Host, conn.Type)
			break
		}
	}

	if dbName == "" {
		return fmt.Errorf("could not extract database name from connection")
	}

	// Store current context for filter reloading
	m.currentConnection = connectionName
	m.currentDatabase = dbName
	m.currentTable = tableName

	// Get table columns
	columnsData, err := driver.GetTableColumns(dbName, tableName)
	if err != nil {
		return err
	}

	// Convert columns to table.Column format
	m.columns = make([]table.Column, len(columnsData))
	m.columnNames = make([]string, len(columnsData))
	for i, col := range columnsData {
		m.columns[i] = table.Column{
			Title: col[0], // column name
			Width: max(10, len(col[0])+2),
		}
		m.columnNames[i] = col[0]
	}

	// Get table data
	data, err := driver.GetTableData(dbName, tableName)
	if err != nil {
		return err
	}

	// Convert data to table.Row format (skip header row since we have columns)
	m.allRows = make([]table.Row, len(data)-1)
	for i := 1; i < len(data); i++ {
		m.allRows[i-1] = table.Row(data[i])
	}

	return nil
}

// applyFilterToActiveTab reloads table data from database with filters
func (m Model) applyFilterToActiveTab() Model {
	activeTab := m.Tabs.ActiveTab()
	if activeTab == nil {
		return m
	}

	filters := m.Tabs.GetActiveTabFilters()

	// Get connection and table info from tab name (format: "connection.table")
	tabName := m.Tabs.GetActiveTabName()
	parts := strings.Split(tabName, ".")
	if len(parts) != 2 {
		logger.Error("Invalid tab name format", map[string]any{"tab": tabName})
		return m
	}

	connectionName := parts[0]
	tableName := parts[1]

	driver, exists := m.dbConnections[connectionName]
	if !exists {
		logger.Error("No active connection", map[string]any{"connection": connectionName})
		return m
	}

	// Extract database name
	connections := m.Sidebar.GetConnections()
	var dbName string
	for _, conn := range connections {
		if conn.Name == connectionName {
			dbName = extractDatabaseName(conn.Host, conn.Type)
			break
		}
	}

	if dbName == "" {
		logger.Error("Could not extract database name", map[string]any{})
		return m
	}

	var data [][]string
	var err error

	if len(filters) == 0 {
		logger.Debug("Loading data without filters", map[string]any{})
		// No filters - use regular query
		data, err = driver.GetTableData(dbName, tableName)
	} else {
		logger.Debug("Loading data with filters", map[string]any{
			"filter_count": len(filters),
		})

		// Convert filter.Filter to drivers.FilterCondition
		driverFilters := make([]drivers.FilterCondition, len(filters))
		for i, f := range filters {
			driverFilters[i] = drivers.FilterCondition{
				Column:   f.Column,
				Operator: string(f.Operator),
				Value:    f.Value,
			}
		}

		// Load data with filters
		data, err = driver.GetTableDataWithFilter(dbName, tableName, driverFilters)
	}

	if err != nil {
		logger.Error("Failed to load filtered data", map[string]any{
			"error": err.Error(),
		})
		return m
	}

	// Convert data to table.Row format (skip header row)
	tableRows := make([]table.Row, len(data)-1)
	for i := 1; i < len(data); i++ {
		tableRows[i-1] = table.Row(data[i])
	}

	logger.Debug("Filter result", map[string]any{
		"filtered_rows": len(tableRows),
	})

	// Update tab with filtered data
	if tableModel, ok := activeTab.Content.(table.Model); ok {
		tableModel.SetRows(tableRows)
		m.Tabs.UpdateActiveTabContent(tableModel)
	}

	return m
}

// updateStyles refreshes the header and footer styles after theme change
func (m Model) updateStyles() Model {
	t := theme.Current
	m.HeaderStyle = t.Header.Width(m.TerminalWidth).Render("DB Client TUI [" + t.Name + "]")
	m.FooterStyle = t.Footer.Width(m.TerminalWidth).Render(m.getFooterHelp())
	return m
}

// updateFooter refreshes just the footer with current help text
func (m Model) updateFooter() Model {
	t := theme.Current
	m.FooterStyle = t.Footer.Width(m.TerminalWidth).Render(m.getFooterHelp())
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

// getFooterHelp returns context-sensitive help text based on current focus
func (m Model) getFooterHelp() string {
	switch m.Focus {
	case FocusSidebar:
		return "j/k: Navigate | Enter: Select/Connect | d: Structure | n: New Connection | s: Toggle Sidebar | Tab: Switch | T: Theme | q: Quit"
	case FocusMain:
		if m.Tabs.HasTabs() {
			tabType := m.Tabs.GetActiveTabType()
			if tabType == tab.TabTypeStructure {
				return "j/k/h/l: Navigate | 1-4: Sections | Tab: Next Section | []: Switch Tab | Ctrl+W: Close | s: Toggle Sidebar | q: Quit"
			}
			if !m.sidebarCollapsed {
				return "j/k/h/l: Navigate | d: Structure | y: Yank | p: Preview | /: Filter | C: Clear | []: Switch Tab | s: Toggle Sidebar | q: Quit"
			}
			return "j/k/h/l: Navigate | d: Structure | y: Yank | p: Preview | /: Filter | C: Clear | []: Switch Tab | s: Toggle Sidebar | q: Quit"
		}
		if !m.sidebarCollapsed {
			return "s: Toggle Sidebar | Tab: Switch | T: Theme | q: Quit"
		}
		return "s: Toggle Sidebar | T: Theme | q: Quit"
	case FocusFilter:
		return "Tab/h/l: Switch Field | j/k: Navigate Options | Enter: Apply | Esc: Cancel | Ctrl+C: Clear"
	case FocusExitModal:
		return "y: Yes | n/Esc: No | h/l: Switch Button"
	case FocusCreateConnectionModal:
		return "Tab: Next Field | Shift+Tab: Previous Field | Enter: Submit | Esc: Cancel"
	default:
		return "Tab: Switch | T: Theme | q: Quit"
	}
}

// getColumnNames extracts column names from columns
func getColumnNames(columns []table.Column) []string {
	names := make([]string, len(columns))
	for i, col := range columns {
		names[i] = col.Title
	}
	return names
}

// loadTableStructure loads the table structure and opens it in a new tab
func (m *Model) loadTableStructure() error {
	// Get connection and table info from current context or active tab
	connectionName := m.currentConnection
	tableName := m.currentTable
	dbName := m.currentDatabase

	// If we have an active tab, try to extract info from it
	if m.Tabs.HasTabs() {
		tabName := m.Tabs.GetActiveTabName()
		parts := strings.Split(tabName, ".")
		if len(parts) >= 2 {
			connectionName = parts[0]
			tableName = parts[1]
			// Remove [S] prefix if present (structure tab)
			if strings.HasPrefix(tableName, "[S] ") {
				tableName = tableName[4:]
			}
		}
	}

	if connectionName == "" || tableName == "" {
		return fmt.Errorf("no table selected")
	}

	driver, exists := m.dbConnections[connectionName]
	if !exists {
		return fmt.Errorf("no active connection for %s", connectionName)
	}

	// Get database name if not set
	if dbName == "" {
		connections := m.Sidebar.GetConnections()
		for _, conn := range connections {
			if conn.Name == connectionName {
				dbName = extractDatabaseName(conn.Host, conn.Type)
				break
			}
		}
	}

	if dbName == "" {
		return fmt.Errorf("could not extract database name from connection")
	}

	// Get table structure
	structure, err := driver.GetTableStructure(dbName, tableName)
	if err != nil {
		return err
	}

	// Add structure tab
	tabName := connectionName + "." + tableName
	m.Tabs.AddStructureTab(tabName, structure)

	// Set tab dimensions
	tableWidth := m.ContentWidth - 4
	tableHeight := m.ContentHeight - 3 - 2
	m.Tabs.SetSize(tableWidth, tableHeight)

	return nil
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
