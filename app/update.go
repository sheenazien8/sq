package app

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/drivers"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/storage"
	"github.com/sheenazien8/sq/ui/filter"
	"github.com/sheenazien8/sq/ui/modal"
	queryeditor "github.com/sheenazien8/sq/ui/query-editor"
	"github.com/sheenazien8/sq/ui/sidebar"
	"github.com/sheenazien8/sq/ui/tab"
	"github.com/sheenazien8/sq/ui/table"
	"github.com/sheenazien8/sq/ui/theme"
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

	case queryeditor.CellPreviewMsg:
		// Show cell preview modal for query editor results
		if msg.Content != "" {
			m.CellPreviewModal.Show(msg.Content)
			m.Focus = FocusCellPreviewModal
			m = m.updateFooter()
		}
		return m, nil

	case queryeditor.YankCellMsg:
		// Copy cell content to clipboard from query editor results
		if msg.Content != "" {
			err := clipboard.WriteAll(msg.Content)
			if err != nil {
				logger.Error("Failed to copy to clipboard", map[string]any{"error": err.Error()})
			} else {
				logger.Info("Cell content copied to clipboard", map[string]any{"length": len(msg.Content)})
			}
		}
		return m, nil

	case queryeditor.YankQueryMsg:
		// Copy entire query to system clipboard
		if msg.Content != "" {
			err := clipboard.WriteAll(msg.Content)
			if err != nil {
				logger.Error("Failed to copy query to clipboard", map[string]any{"error": err.Error()})
			} else {
				logger.Info("Query copied to clipboard", map[string]any{"length": len(msg.Content)})
			}
		}
		return m, nil

	case table.NextPageMsg:
		// Load next page of data
		m = m.loadNextPage()
		return m, nil

	case table.PrevPageMsg:
		// Load previous page of data
		m = m.loadPrevPage()
		return m, nil

	case tab.TabSwitchedMsg:
		// Update filter UI to show the new tab's filter
		m = m.updateFilterForActiveTab()
		m = m.updateFooter()
		return m, nil

	case queryeditor.QueryExecuteMsg:
		// Execute the query
		logger.Debug("Query execute requested", map[string]any{
			"query":      msg.Query,
			"connection": msg.ConnectionName,
			"database":   msg.DatabaseName,
		})

		driver, exists := m.dbConnections[msg.ConnectionName]
		if !exists {
			logger.Error("No active connection for query", map[string]any{
				"connection": msg.ConnectionName,
			})
			m.Tabs.SetQueryError("No active connection: " + msg.ConnectionName)
			return m, nil
		}

		// Execute the query
		data, err := driver.ExecuteQuery(msg.Query)
		if err != nil {
			logger.Error("Query execution failed", map[string]any{
				"error": err.Error(),
			})
			m.Tabs.SetQueryError(err.Error())
			return m, nil
		}

		// Convert data to table format
		if len(data) > 0 {
			// First row is headers
			columns := make([]table.Column, len(data[0]))
			for i, colName := range data[0] {
				columns[i] = table.Column{
					Title: colName,
					Width: max(10, len(colName)+2),
				}
			}

			// Rest are rows
			var rows []table.Row
			for i := 1; i < len(data); i++ {
				rows = append(rows, table.Row(data[i]))
			}

			m.Tabs.SetQueryResults(columns, rows)
			logger.Info("Query executed successfully", map[string]any{
				"rows": len(rows),
			})
		} else {
			m.Tabs.SetQueryResults([]table.Column{}, []table.Row{})
		}

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
		paginatedResult, err := m.loadTableData(msg.ConnectionName, msg.TableName)
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

		// Set pagination info on the table
		if paginatedResult != nil {
			m.Tabs.SetActiveTabPagination(
				paginatedResult.Page,
				paginatedResult.TotalPages,
				paginatedResult.TotalRows,
				paginatedResult.PageSize,
			)
		}

		// Set tab dimensions (filter bar is always 3 lines with border)
		tableWidth := m.ContentWidth - 4
		tableHeight := m.ContentHeight - 3 - 2
		m.Tabs.SetSize(tableWidth, tableHeight)

		// Update filter UI to show the new tab's filter (which is empty for a new tab)
		m = m.updateFilterForActiveTab()

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

		m.HeaderStyle = headerStyle.Render("SQ [" + t.Name + "]")
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
		m.HelpModal.SetSize(m.TerminalWidth, m.TerminalHeight)

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

		// Always handle filter input
		prevFilter := m.Filter.GetFilter()
		prevActive := m.Filter.Active()
		m.Filter, cmd = m.Filter.Update(msg)
		cmds = append(cmds, cmd)

		// Check if filter was applied (Enter pressed) by checking if filter changed
		currentFilter := m.Filter.GetFilter()
		if currentFilter != nil && (prevFilter == nil || prevFilter.WhereClause != currentFilter.WhereClause) {
			m.Tabs.AddActiveTabFilter(*currentFilter)
			m = m.applyFilterToActiveTab()
		}

		// Check if filter was cleared (Ctrl+C pressed)
		if prevActive && !m.Filter.Active() {
			m.Tabs.ClearActiveTabFilters()
			m = m.applyFilterToActiveTab()
		}

		if m.Sidebar.IsFilterVisible() {
			// Handle sidebar filter input
			cmd := m.Sidebar.UpdateFilterInput(msg)
			cmds = append(cmds, cmd)

			// Check for exit keys
			switch msg.String() {
			case "enter":
				// Apply filter and return to sidebar (keep filter active)
				m.Sidebar.HideFilterInput()
				m.Focus = FocusSidebar
				m = m.updateFooter()
			case "esc":
				// Cancel filter and return to sidebar (clear filter)
				m.Sidebar.SetFilterVisible(false)
				m.Focus = FocusSidebar
				m = m.updateFooter()
			case "ctrl+c":
				// Clear filter
				m.Sidebar.ClearFilterInput()
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

		if m.HelpModal.Visible() {
			m.HelpModal, cmd = m.HelpModal.Update(msg)
			cmds = append(cmds, cmd)

			// Check if modal was closed
			if !m.HelpModal.Visible() {
				// Return to previous focus
				if m.Tabs.HasTabs() {
					m.Focus = FocusMain
					m.Sidebar.SetFocused(false)
					m.Tabs.SetFocused(true)
				} else {
					m.Focus = FocusSidebar
					m.Sidebar.SetFocused(true)
				}
				m = m.updateFooter()
			}
			return m, tea.Batch(cmds...)
		}

		// If query editor is active, pass most keys directly to it
		// Only intercept specific control keys for app-level navigation
		if m.Focus == FocusMain && m.Tabs.HasTabs() && m.Tabs.GetActiveTabType() == tab.TabTypeQuery {
			switch msg.String() {
			case "ctrl+c":
				// Show exit modal
				m.ExitModal.Show()
				m.Focus = FocusExitModal
				m = m.updateFooter()
				return m, nil
			case "tab":
				// Switch to sidebar if not collapsed
				if !m.sidebarCollapsed {
					m.Focus = FocusSidebar
					m.Sidebar.SetFocused(true)
					m.Tabs.SetFocused(false)
					m = m.updateFooter()
				}
				return m, nil
			case "]":
				m.Tabs.NextTab()
				m = m.updateFilterForActiveTab()
				m = m.updateFooter()
				return m, nil
			case "[":
				m.Tabs.PrevTab()
				m = m.updateFilterForActiveTab()
				m = m.updateFooter()
				return m, nil
			case "ctrl+w":
				m.Tabs.CloseTab(m.Tabs.ActiveTabIndex())
				if !m.Tabs.HasTabs() {
					m.Focus = FocusSidebar
					m.Sidebar.SetFocused(true)
					m.Tabs.SetFocused(false)
				}
				m = m.updateFilterForActiveTab()
				m = m.updateFooter()
				return m, nil
			default:
				// Pass all other keys to the query editor
				m.Tabs, cmd = m.Tabs.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}

		// If filter is focused, don't process global shortcuts
		if m.Filter.Focused() {
			return m, nil
		}

		switch msg.String() {
		case "?":
			// Show help modal
			m.HelpModal.Show()
			m.Focus = FocusHelpModal
			m = m.updateFooter()
			return m, nil

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
					// Create new filter with columns (if not already created or empty)
					if !m.Filter.HasText() {
						var filterText string
						if activeFilter := m.Tabs.GetActiveTabFilter(); activeFilter != nil {
							// Use the stored WHERE clause directly
							filterText = activeFilter.WhereClause
						}
						m.Filter = filter.NewWithText(columnNames, filterText)
						m.Filter.SetWidth(m.ContentWidth)
					}
					// Always focus the filter
					m.Filter.Focus()
					m = m.updateFooter()
				}
			} else if m.Focus == FocusSidebar {
				// Toggle sidebar filter
				if !m.Sidebar.IsFilterVisible() {
					// Show filter input
					m.Sidebar.SetFilterVisible(true)
					m.Focus = FocusSidebarFilter
				} else {
					// Hide filter input but keep filter active
					m.Sidebar.HideFilterInput()
					m.Focus = FocusSidebar
				}
				m = m.updateFooter()
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
			if m.Focus == FocusSidebar {
				// Clear sidebar filter
				m.Sidebar.SetFilterText("")
				m.Sidebar.ClearFilterInput()
			} else {
				// Clear table filters
				m.Tabs.ClearActiveTabFilters()
				m = m.applyFilterToActiveTab()
				m = m.updateFilterForActiveTab()
				m = m.updateTabSize()
			}

		case "r", "R":
			if m.Focus == FocusSidebar {
				// Refresh connections
				m.Sidebar.RefreshConnections()
			}

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
			// Check if this is part of 'gd' sequence for go to definition
			if m.gPressed && m.Focus == FocusMain && m.Tabs.HasTabs() {
				m.gPressed = false
				logger.Debug("Goto definition", map[string]any{
					"hasTabs":   m.Tabs.HasTabs(),
					"focusMain": m.Focus == FocusMain,
				})
				err := m.goToForeignKeyDefinition()
				if err != nil {
					logger.Error("Failed to go to foreign key definition", map[string]any{"error": err.Error()})
				} else {
					// Update filter UI for the new tab
					m = m.updateFilterForActiveTab()
				}
				return m, nil
			}

			// Reset gPressed if sequence was broken
			m.gPressed = false

			// Show table structure in a new tab
			if m.Focus == FocusMain && m.Tabs.HasTabs() {
				err := m.loadTableStructure()
				if err != nil {
					logger.Error("Failed to load table structure", map[string]any{"error": err.Error()})
				} else {
					// Update filter UI for the new tab (structure tabs have no filter)
					m = m.updateFilterForActiveTab()
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
							m = m.updateFilterForActiveTab()
							m = m.updateFooter()
						}
						return m, nil
					}
				}
			}

		case "g":
			// Start of 'gd' sequence for go to definition
			if m.Focus == FocusMain && m.Tabs.HasTabs() {
				m.gPressed = true
				logger.Debug("G pressed - waiting for D", nil)
			}

		case "e", "E":
			// Open query editor in a new tab
			activeDB := m.Sidebar.ActiveDatabase()
			if activeDB != nil && activeDB.Connected {
				// Get database name from connection
				connections := m.Sidebar.GetConnections()
				var dbName string
				for _, conn := range connections {
					if conn.Name == activeDB.Name {
						dbName = extractDatabaseName(conn.Host, conn.Type)
						break
					}
				}

				if dbName != "" {
					// Add query tab
					tabName := "Query"
					m.Tabs.AddQueryTab(tabName, activeDB.Name, dbName)

					// Set tab dimensions
					tableWidth := m.ContentWidth - 4
					tableHeight := m.ContentHeight - 3 - 2
					m.Tabs.SetSize(tableWidth, tableHeight)

					// Update filter UI for the new tab (query tabs have no filter)
					m = m.updateFilterForActiveTab()

					// Switch focus to main area
					m.Focus = FocusMain
					m.Sidebar.SetFocused(false)
					m.Tabs.SetFocused(true)
					m = m.updateFooter()

					logger.Info("Query editor opened", map[string]any{
						"connection": activeDB.Name,
						"database":   dbName,
					})
				}
			} else {
				logger.Debug("Cannot open query editor: no active connection", map[string]any{})
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
			// Reset gPressed flag for any key that doesn't continue the sequence
			m.gPressed = false
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
func (m *Model) loadTableData(connectionName, tableName string) (*drivers.PaginatedResult, error) {
	driver, exists := m.dbConnections[connectionName]
	if !exists {
		return nil, fmt.Errorf("no active connection for %s", connectionName)
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
		return nil, fmt.Errorf("could not extract database name from connection")
	}

	// Store current context for filter reloading
	m.currentConnection = connectionName
	m.currentDatabase = dbName
	m.currentTable = tableName

	// Get table columns
	columnsData, err := driver.GetTableColumns(dbName, tableName)
	if err != nil {
		return nil, err
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

	// Add foreign key information to columns
	structure, err := driver.GetTableStructure(dbName, tableName)
	if err == nil { // Don't fail if we can't get structure, just continue without FK info
		for i := range m.columns {
			colName := m.columnNames[i]
			for _, relation := range structure.Relations {
				if relation.Column == colName {
					m.columns[i].IsForeignKey = true
					m.columns[i].ReferencedTable = relation.ReferencedTable
					m.columns[i].ReferencedColumn = relation.ReferencedColumn
					break
				}
			}
		}
	}

	// Get table data with pagination
	pagination := drivers.Pagination{
		Page:     1,
		PageSize: m.pageSize,
	}

	result, err := driver.GetTableDataPaginated(dbName, tableName, pagination)
	if err != nil {
		return nil, err
	}

	// Update pagination state
	m.currentPage = result.Page

	// Convert data to table.Row format (skip header row since we have columns)
	m.allRows = make([]table.Row, len(result.Data)-1)
	for i := 1; i < len(result.Data); i++ {
		m.allRows[i-1] = table.Row(result.Data[i])
	}

	return result, nil
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

	// Reset to page 1 when applying filters
	m.currentPage = 1

	pagination := drivers.Pagination{
		Page:     1,
		PageSize: m.pageSize,
	}

	var result *drivers.PaginatedResult
	var err error

	if len(filters) == 0 {
		logger.Debug("Loading data without filters", map[string]any{})
		// No filters - use paginated query
		result, err = driver.GetTableDataPaginated(dbName, tableName, pagination)
	} else {
		logger.Debug("Loading data with filters", map[string]any{
			"filter_count": len(filters),
		})

		// Get the raw WHERE clause from the filter
		whereClause := ""
		if len(filters) > 0 {
			whereClause = filters[0].WhereClause
		}

		// Load data with filters and pagination
		result, err = driver.GetTableDataWithFilterPaginated(dbName, tableName, whereClause, pagination)
	}

	if err != nil {
		logger.Error("Failed to load filtered data", map[string]any{
			"error": err.Error(),
		})
		return m
	}

	// Convert data to table.Row format (skip header row)
	tableRows := make([]table.Row, len(result.Data)-1)
	for i := 1; i < len(result.Data); i++ {
		tableRows[i-1] = table.Row(result.Data[i])
	}

	logger.Debug("Filter result", map[string]any{
		"filtered_rows": len(tableRows),
		"total_rows":    result.TotalRows,
		"total_pages":   result.TotalPages,
	})

	// Update tab with filtered data and pagination
	if tableModel, ok := activeTab.Content.(table.Model); ok {
		tableModel.SetRows(tableRows)
		tableModel.SetPagination(result.Page, result.TotalPages, result.TotalRows, result.PageSize)
		m.Tabs.UpdateActiveTabContent(tableModel)
	}

	return m
}

// updateStyles refreshes the header and footer styles after theme change
func (m Model) updateStyles() Model {
	t := theme.Current
	m.HeaderStyle = t.Header.Width(m.TerminalWidth).Render("sq [" + t.Name + "]")
	m.FooterStyle = t.Footer.Width(m.TerminalWidth).Render(m.getFooterHelp())
	return m
}

// updateFooter refreshes just the footer with current help text
func (m Model) updateFooter() Model {
	t := theme.Current
	m.FooterStyle = t.Footer.Width(m.TerminalWidth).Render(m.getFooterHelp())
	return m
}

// updateFilterForActiveTab updates the filter input to show the current tab's filter
func (m Model) updateFilterForActiveTab() Model {
	if !m.Tabs.HasTabs() {
		m.Filter.SetFilter(nil)
		return m
	}

	// Get the active tab's filter and set it on the filter UI
	activeFilter := m.Tabs.GetActiveTabFilter()
	m.Filter.SetFilter(activeFilter)

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
		return "?: Help | j/k: Navigate | Enter: Select | e: Query | n: New | /: Filter | Tab: Switch | q: Quit"
	case FocusMain:
		if m.Tabs.HasTabs() {
			tabType := m.Tabs.GetActiveTabType()
			if tabType == tab.TabTypeStructure {
				return "?: Help | j/k/h/l: Navigate | 1-4: Sections | []: Tabs | Ctrl+W: Close | q: Quit"
			}
			if tabType == tab.TabTypeQuery {
				return "?: Help | F5: Execute | Ctrl+R: Results | []: Tabs | Ctrl+W: Close | q: Quit"
			}
			return "?: Help | j/k/h/l: Navigate | </>: Page | /: Filter | []: Tabs | q: Quit"
		}
		return "?: Help | s: Toggle Sidebar | Tab: Switch | q: Quit"

	case FocusSidebarFilter:
		return "Enter: Apply | Esc: Cancel | Ctrl+C: Clear"
	case FocusExitModal:
		return "y: Yes | n/Esc: No | h/l: Switch"
	case FocusCreateConnectionModal:
		return "Tab: Next Field | Enter: Submit | Esc: Cancel"
	case FocusHelpModal:
		return "←→/Tab: Sections | j/k: Scroll | Esc/q: Close"
	default:
		return "?: Help | q: Quit"
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

// goToForeignKeyDefinition navigates to the referenced table for a foreign key
func (m *Model) goToForeignKeyDefinition() error {
	if !m.Tabs.HasTabs() {
		return fmt.Errorf("no active tab")
	}

	activeTab := m.Tabs.ActiveTab()
	tableModel, ok := activeTab.Content.(table.Model)
	if !ok {
		return fmt.Errorf("active tab is not a table")
	}

	// Get selected cell value and column index
	selectedRow := tableModel.SelectedRow()
	cursorCol := tableModel.CursorCol()
	if cursorCol < 0 || cursorCol >= len(selectedRow) {
		return fmt.Errorf("invalid column selection")
	}

	cellValue := tableModel.SelectedCell()
	if cellValue == "" {
		return fmt.Errorf("selected cell is empty")
	}

	// Get table info from tab name
	tabName := m.Tabs.GetActiveTabName()
	parts := strings.Split(tabName, ".")
	if len(parts) < 2 {
		return fmt.Errorf("could not parse table name from tab")
	}
	connectionName := parts[0]
	tableName := parts[1]

	// Get connection
	driver, exists := m.dbConnections[connectionName]
	if !exists {
		return fmt.Errorf("no active connection for %s", connectionName)
	}

	// Get table structure to find foreign key info
	dbName := m.currentDatabase
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
		return fmt.Errorf("could not determine database name")
	}

	structure, err := driver.GetTableStructure(dbName, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table structure: %w", err)
	}

	// Find the column and check if it's a foreign key
	var columnName string
	if cursorCol < len(structure.Columns) {
		columnName = structure.Columns[cursorCol].Name
	}

	var referencedTable, referencedColumn string
	for _, relation := range structure.Relations {
		if relation.Column == columnName {
			referencedTable = relation.ReferencedTable
			referencedColumn = relation.ReferencedColumn
			break
		}
	}

	if referencedTable == "" {
		return fmt.Errorf("selected column is not a foreign key")
	}

	// Create filter for the foreign key value
	whereClause := fmt.Sprintf("%s = '%s'", referencedColumn, strings.ReplaceAll(cellValue, "'", "''"))

	// Get referenced table structure and columns
	targetStructure, err := driver.GetTableStructure(dbName, referencedTable)
	if err != nil {
		return fmt.Errorf("failed to get referenced table structure: %w", err)
	}

	targetColumns := make([]table.Column, len(targetStructure.Columns))
	for i, col := range targetStructure.Columns {
		targetColumns[i] = table.Column{
			Title: col.Name,
			Width: max(10, len(col.Name)+2),
		}
		// Mark foreign keys in the referenced table
		for _, rel := range targetStructure.Relations {
			if rel.Column == col.Name {
				targetColumns[i].IsForeignKey = true
				targetColumns[i].ReferencedTable = rel.ReferencedTable
				targetColumns[i].ReferencedColumn = rel.ReferencedColumn
				break
			}
		}
	}

	// Query referenced table with filter
	result, err := driver.GetTableDataWithFilter(dbName, referencedTable, whereClause)
	if err != nil {
		return fmt.Errorf("failed to query referenced table: %w", err)
	}

	// Convert result data to table rows (skip header row)
	rows := make([]table.Row, len(result)-1)
	for i := 1; i < len(result); i++ {
		rowData := result[i]
		row := make(table.Row, len(rowData))
		for j, cell := range rowData {
			row[j] = cell
		}
		rows[i-1] = row
	}

	// Create new tab for referenced table
	targetTabName := connectionName + "." + referencedTable
	m.Tabs.AddTableTab(targetTabName, targetColumns, rows)

	// Set the filter on the new tab so it shows in the filter bar
	m.Tabs.AddActiveTabFilter(filter.Filter{
		WhereClause: whereClause,
	})

	tableWidth := m.ContentWidth - 4
	tableHeight := m.ContentHeight - 3 - 2
	m.Tabs.SetSize(tableWidth, tableHeight)

	return nil
}

// loadNextPage loads the next page of data for the active table tab
func (m Model) loadNextPage() Model {
	return m.loadPage(m.currentPage + 1)
}

// loadPrevPage loads the previous page of data for the active table tab
func (m Model) loadPrevPage() Model {
	if m.currentPage > 1 {
		return m.loadPage(m.currentPage - 1)
	}
	return m
}

// loadPage loads a specific page of data for the active table tab
func (m Model) loadPage(page int) Model {
	activeTab := m.Tabs.ActiveTab()
	if activeTab == nil {
		return m
	}

	// Only handle table tabs (not structure or query tabs)
	if activeTab.Type != tab.TabTypeTable {
		return m
	}

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

	// Get filters if any
	filters := m.Tabs.GetActiveTabFilters()

	pagination := drivers.Pagination{
		Page:     page,
		PageSize: m.pageSize,
	}

	var result *drivers.PaginatedResult
	var err error

	if len(filters) == 0 {
		result, err = driver.GetTableDataPaginated(dbName, tableName, pagination)
	} else {
		// Get the raw WHERE clause from the filter
		whereClause := ""
		if len(filters) > 0 {
			whereClause = filters[0].WhereClause
		}
		result, err = driver.GetTableDataWithFilterPaginated(dbName, tableName, whereClause, pagination)
	}

	if err != nil {
		logger.Error("Failed to load paginated data", map[string]any{
			"error": err.Error(),
			"page":  page,
		})
		return m
	}

	// Update current page
	m.currentPage = result.Page

	// Convert data to table.Row format (skip header row)
	tableRows := make([]table.Row, len(result.Data)-1)
	for i := 1; i < len(result.Data); i++ {
		tableRows[i-1] = table.Row(result.Data[i])
	}

	logger.Debug("Loaded page", map[string]any{
		"page":        result.Page,
		"total_pages": result.TotalPages,
		"total_rows":  result.TotalRows,
		"rows_loaded": len(tableRows),
	})

	// Update tab with paginated data
	if tableModel, ok := activeTab.Content.(table.Model); ok {
		tableModel.SetRows(tableRows)
		tableModel.SetPagination(result.Page, result.TotalPages, result.TotalRows, result.PageSize)
		m.Tabs.UpdateActiveTabContent(tableModel)
	}

	return m
}
