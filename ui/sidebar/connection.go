package sidebar

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/storage"
	"github.com/sheenazien8/sq/ui/theme"
)

type Table struct {
	Name     string
	RowCount int64
	Selected bool
}

// Connection represents a database item in the sidebar
type Connection struct {
	ID        int64
	Name      string
	Type      string
	Host      string
	Selected  bool
	Expanded  bool
	Connected bool
	Tables    []Table
}

// TreeItem represents an item in the tree (connection or table)
type TreeItem struct {
	ConnectionIndex int
	TableIndex      int
	Level           int
	IsLastChild     bool
}

// TableSelectedMsg is sent when a table is selected in the sidebar
type TableSelectedMsg struct {
	ConnectionName string
	TableName      string
}

// ConnectionSelectedMsg is sent when a connection is selected (expanded/activated)
type ConnectionSelectedMsg struct {
	ConnectionName string
	ConnectionType string
	ConnectionURL  string
}

// Model represents the sidebar with database list
type Model struct {
	connections []Connection
	cursor      int
	offset      int
	width       int
	height      int
	focused     bool

	// Filter state
	filterInput textinput.Model
	filterText  string
	showFilter  bool
}

// New creates a new sidebar model with sample databases
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Search"
	ti.CharLimit = 20
	ti.Width = 1000 // Large width to prevent internal wrapping

	return Model{
		connections: getConnections(),
		cursor:      0,
		offset:      0,
		focused:     false,
		filterInput: ti,
		filterText:  "",
		showFilter:  false,
	}
}

func getConnections() (data []Connection) {
	connections, err := storage.GetAllConnections()
	if err != nil {
		logger.Debug("Error getting connections", map[string]any{
			"error": err.Error(),
		})
		return data
	}

	logger.Debug("Getting connections", map[string]any{
		"data": len(connections),
	})

	for _, connection := range connections {
		// Start with no tables - they will be loaded when connection is established
		data = append(data, Connection{
			ID:        connection.ID,
			Name:      connection.Name,
			Type:      connection.Driver,
			Host:      connection.URL,
			Tables:    []Table{}, // Empty initially
			Expanded:  false,     // start collapsed
			Connected: false,     // start disconnected
		})
	}

	return data
}

// SetSize sets the sidebar dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets whether the sidebar is focused
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// Focused returns whether the sidebar is focused
func (m Model) Focused() bool {
	return m.focused
}

// SetFilterVisible shows/hides the sidebar filter
func (m *Model) SetFilterVisible(visible bool) {
	m.showFilter = visible
	if visible {
		m.filterInput.Focus()
	} else {
		m.filterInput.Blur()
		m.filterText = ""
		// Reset cursor when clearing filter
		m.cursor = 0
		m.offset = 0
	}
	// Adjust scrolling for new layout
	m.adjustScrolling()
}

// HideFilterInput hides the filter input without clearing the filter text
func (m *Model) HideFilterInput() {
	m.showFilter = false
	m.filterInput.Blur()
	// Adjust scrolling for new layout
	m.adjustScrolling()
}

// adjustScrolling adjusts cursor and offset based on current visible items
func (m *Model) adjustScrolling() {
	treeItems := m.getTreeItems()
	visibleCount := m.visibleItems()

	// Adjust cursor if out of bounds
	if m.cursor >= len(treeItems) {
		m.cursor = max(0, len(treeItems)-1)
	}

	// Adjust offset if needed
	maxOffset := max(0, len(treeItems)-visibleCount)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}

	// Ensure cursor is visible
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleCount {
		m.offset = m.cursor - visibleCount + 1
	}
}

// GetFilterText returns the current filter text
func (m Model) GetFilterText() string {
	return m.filterText
}

// SetFilterText sets the filter text
func (m *Model) SetFilterText(text string) {
	m.filterText = text
	// Reset cursor when filter changes
	m.cursor = 0
	m.offset = 0
}

// IsFilterVisible returns whether the sidebar filter is visible
func (m Model) IsFilterVisible() bool {
	return m.showFilter
}

// UpdateFilterInput updates the filter input component
func (m *Model) UpdateFilterInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	return cmd
}

// GetFilterInput returns the filter input model
func (m Model) GetFilterInput() textinput.Model {
	return m.filterInput
}

// ClearFilterInput clears the filter input
func (m *Model) ClearFilterInput() {
	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "Search"
	m.filterInput.CharLimit = 50
	m.filterInput.Width = 1000 // Large width to prevent internal wrapping
	if m.showFilter {
		m.filterInput.Focus()
	}
	m.filterText = ""
}

// Cursor returns the current cursor position
func (m Model) Cursor() int {
	return m.cursor
}

// SelectedItem returns the currently selected tree item
func (m Model) SelectedItem() *TreeItem {
	treeItems := m.getTreeItems()
	if m.cursor >= 0 && m.cursor < len(treeItems) {
		item := treeItems[m.cursor]
		return &item
	}
	return nil
}

// SelectedDatabase returns the currently selected database (cursor position)
func (m Model) SelectedDatabase() *Connection {
	selectedItem := m.SelectedItem()
	if selectedItem != nil && selectedItem.Level == 0 {
		return &m.connections[selectedItem.ConnectionIndex]
	}
	return nil
}

// ActiveDatabase returns the database that has been activated (via Enter key)
func (m Model) ActiveDatabase() *Connection {
	for i := range m.connections {
		if m.connections[i].Selected {
			return &m.connections[i]
		}
	}
	return nil
}

// HasActiveDatabase returns true if a database has been selected/activated
func (m Model) HasActiveDatabase() bool {
	return m.ActiveDatabase() != nil
}

// SelectedTable returns the name of the currently selected table (if cursor is on a table)
func (m Model) SelectedTable() string {
	selectedItem := m.SelectedItem()
	if selectedItem != nil && selectedItem.Level == 1 {
		conn := m.connections[selectedItem.ConnectionIndex]
		if selectedItem.TableIndex >= 0 && selectedItem.TableIndex < len(conn.Tables) {
			return conn.Tables[selectedItem.TableIndex].Name
		}
	}
	return ""
}

// SetDatabases updates the database list
func (m *Model) SetDatabases(databases []Connection) {
	m.connections = databases
	treeItems := m.getTreeItems()
	if m.cursor >= len(treeItems) {
		m.cursor = max(0, len(treeItems)-1)
	}
}

// GetConnections returns the current connections
func (m Model) GetConnections() []Connection {
	return m.connections
}

// UpdateConnection updates a specific connection with new table data and connection status
func (m *Model) UpdateConnection(name string, tableNames []string, connected bool) {
	for i := range m.connections {
		if m.connections[i].Name == name {
			m.connections[i].Connected = connected
			m.connections[i].Tables = make([]Table, len(tableNames))
			for j, tableName := range tableNames {
				m.connections[i].Tables[j] = Table{
					Name:     tableName,
					RowCount: 0, // TODO: Get actual row count
					Selected: false,
				}
			}
			break
		}
	}

	// Update tree items and cursor position
	treeItems := m.getTreeItems()
	if m.cursor >= len(treeItems) {
		m.cursor = max(0, len(treeItems)-1)
	}
}

// RefreshConnections reloads the connections from storage
func (m *Model) RefreshConnections() {
	m.connections = getConnections()
	treeItems := m.getTreeItems()
	if m.cursor >= len(treeItems) {
		m.cursor = max(0, len(treeItems)-1)
	}
}

// getTreeItems returns a flattened list of all visible tree items
func (m Model) getTreeItems() []TreeItem {
	var items []TreeItem

	filterLower := strings.ToLower(m.filterText)

	for connIdx, conn := range m.connections {
		connLower := strings.ToLower(conn.Name)
		includeConnection := m.filterText == "" || strings.Contains(connLower, filterLower)

		// Check tables for matches
		var matchingTableIndices []int
		for tableIdx, table := range conn.Tables {
			tableLower := strings.ToLower(table.Name)
			if m.filterText == "" || strings.Contains(tableLower, filterLower) {
				matchingTableIndices = append(matchingTableIndices, tableIdx)
			}
		}

		// Handle table display based on expansion and filtering
		var tablesToShow []int

		if m.filterText == "" {
			// No filter: show tables only if connection is expanded
			if conn.Expanded {
				for tableIdx := range conn.Tables {
					tablesToShow = append(tablesToShow, tableIdx)
				}
			}
		} else {
			// With filter: show matching tables
			if len(matchingTableIndices) > 0 {
				tablesToShow = matchingTableIndices
			} else if conn.Expanded && includeConnection {
				// If connection matches but no specific table matches, show all tables if expanded
				for tableIdx := range conn.Tables {
					tablesToShow = append(tablesToShow, tableIdx)
				}
			}
		}

		// Add the connection and its tables if it should be included
		if includeConnection || len(matchingTableIndices) > 0 {
			items = append(items, TreeItem{
				ConnectionIndex: connIdx,
				TableIndex:      -1,
				Level:           0,
				IsLastChild:     false,
			})

			// Add tables
			for i, tableIdx := range tablesToShow {
				isLast := i == len(tablesToShow)-1
				items = append(items, TreeItem{
					ConnectionIndex: connIdx,
					TableIndex:      tableIdx,
					Level:           1,
					IsLastChild:     isLast,
				})
			}
		}
	}

	return items
}

// visibleItems returns the number of items that can be displayed
func (m Model) visibleItems() int {
	// Account for title (1 line), separator (1 line), status (1 line), borders (2 lines)
	// Plus filter bar (1 line) if visible
	extraLines := 0
	if m.showFilter {
		extraLines = 1
	}
	return max(0, m.height-7-extraLines)
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	treeItems := m.getTreeItems()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(treeItems)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.visibleItems() {
					m.offset = m.cursor - m.visibleItems() + 1
				}
			}
		case "home":
			m.cursor = 0
			m.offset = 0
		case "end":
			m.cursor = max(0, len(treeItems)-1)
			maxOffset := max(0, len(treeItems)-m.visibleItems())
			m.offset = maxOffset
		case "enter":
			if m.cursor >= 0 && m.cursor < len(treeItems) {
				item := treeItems[m.cursor]
				if item.Level == 0 {
					conn := &m.connections[item.ConnectionIndex]
					conn.Expanded = !conn.Expanded

					for i := range m.connections {
						m.connections[i].Selected = false
					}
					conn.Selected = true

					logger.Debug("Toggled connection expansion", map[string]any{
						"name":     conn.Name,
						"expanded": conn.Expanded,
					})

					// Recalculate tree items after expansion change
					treeItems = m.getTreeItems()

					// Adjust cursor if it's now out of bounds
					if m.cursor >= len(treeItems) {
						m.cursor = max(0, len(treeItems)-1)
					}

					// Adjust offset if needed
					maxOffset := max(0, len(treeItems)-m.visibleItems())
					if m.offset > maxOffset {
						m.offset = maxOffset
					}

					// Send connection selected message
					return m, func() tea.Msg {
						return ConnectionSelectedMsg{
							ConnectionName: conn.Name,
							ConnectionType: conn.Type,
							ConnectionURL:  conn.Host,
						}
					}
				} else {
					conn := &m.connections[item.ConnectionIndex]
					table := &conn.Tables[item.TableIndex]

					logger.Debug("Selected table", map[string]any{
						"connection": conn.Name,
						"table":      table.Name,
						"row_count":  table.RowCount,
					})

					return m, func() tea.Msg {
						return TableSelectedMsg{
							ConnectionName: conn.Name,
							TableName:      table.Name,
						}
					}
				}
			}
		}
	}

	return m, nil
}

// View renders the sidebar
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	// Get current theme
	t := theme.Current

	// Inner content width (minus border)
	innerWidth := m.width - 4

	var lines []string

	// Filter input (if visible)
	if m.showFilter {
		filterStyle := lipgloss.NewStyle().
			Foreground(t.Colors.Primary).
			Background(t.Colors.SelectionBg).
			Padding(0, 1)
		// Replace newlines to prevent wrapping that breaks UI
		filterView := strings.ReplaceAll(m.filterInput.View(), "\n", " ")
		filterLine := filterStyle.Width(innerWidth).Render(filterView)
		lines = append(lines, filterLine)
	}

	// Title
	titleText := " Databases"
	if m.filterText != "" && !m.showFilter {
		titleText = " (filtered: " + m.filterText + ")"
	}
	title := t.SidebarTitle.
		Align(lipgloss.Center, lipgloss.Center).
		Width(innerWidth).
		Height(3).
		Render(titleText)

	lines = append(lines, title)

	// Separator
	separatorStyle := lipgloss.NewStyle().Foreground(t.Colors.BorderUnfocused)
	lines = append(lines, separatorStyle.Render(strings.Repeat("─", innerWidth)))

	// Tree items
	treeItems := m.getTreeItems()
	visibleCount := m.visibleItems()
	endIdx := min(m.offset+visibleCount, len(treeItems))

	for i := m.offset; i < endIdx; i++ {
		item := treeItems[i]
		isSelected := i == m.cursor

		var text string
		var style lipgloss.Style

		if item.Level == 0 {
			conn := m.connections[item.ConnectionIndex]
			icon := getConnectionIcon(conn.Type)

			treeChar := "▶"
			if conn.Expanded {
				treeChar = "▼"
			}

			checkIcon := ""
			if conn.Connected {
				checkIcon = "✓ "
			}

			// Calculate available space for name
			// Account for: treeChar (1) + space + icon (3) + space + checkIcon (0 or 2)
			treeCharLen := lipgloss.Width(treeChar)
			iconLen := lipgloss.Width(icon)
			checkIconLen := lipgloss.Width(checkIcon)
			availableForName := innerWidth - treeCharLen - 1 - iconLen - 1 - checkIconLen

			text = treeChar + " " + icon + " " + checkIcon + truncateString(conn.Name, availableForName)

			if isSelected && m.focused {
				style = t.SidebarSelected
			} else if conn.Selected {
				style = t.SidebarActive
			} else {
				style = t.SidebarItem
			}
		} else { // Table
			conn := m.connections[item.ConnectionIndex]
			table := conn.Tables[item.TableIndex]

			prefix := "  "
			if item.IsLastChild {
				prefix += "└─"
			} else {
				prefix += "├─"
			}

			tableIcon := "󰓫"

			// Calculate row count suffix
			rowCountSuffix := " (" + intToStr(int(table.RowCount)) + ")"

			// Account for: prefix (4-5 chars) + space + icon + space + row count suffix
			// Leave room for all parts
			prefixLen := lipgloss.Width(prefix)
			iconLen := lipgloss.Width(tableIcon)
			suffixLen := lipgloss.Width(rowCountSuffix)
			availableForName := innerWidth - prefixLen - 1 - iconLen - 1 - suffixLen

			text = prefix + " " + tableIcon + " " + truncateString(table.Name, availableForName) + rowCountSuffix

			if isSelected && m.focused {
				style = t.SidebarSelected
			} else {
				style = t.SidebarItem
			}
		}

		line := style.Width(innerWidth).Render(text)
		lines = append(lines, line)
	}

	// Fill empty space
	for i := endIdx - m.offset; i < visibleCount; i++ {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	// Status bar
	status := t.StatusBar.Width(innerWidth).Align(lipgloss.Right).
		Render(intToStr(m.cursor+1) + "/" + intToStr(len(treeItems)))
	lines = append(lines, status)

	// Join content
	content := strings.Join(lines, "\n")

	// Apply border based on focus state
	var borderStyle lipgloss.Style
	if m.focused {
		borderStyle = t.BorderFocused
	} else {
		borderStyle = t.BorderUnfocused
	}

	return borderStyle.
		Width(m.width - 4).
		Height(m.height - 2).
		Render(content)
}

// getConnectionIcon returns an icon for the database type
func getConnectionIcon(dbType string) string {
	switch dbType {
	case "mysql":
		return "[M]"
	case "postgresql":
		return "[P]"
	case "sqlite":
		return "[S]"
	case "redis":
		return "[R]"
	case "mongodb":
		return "[m]"
	default:
		return "[?]"
	}
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen > 3 {
			return string(runes[:maxLen-3]) + "..."
		}
		return string(runes[:maxLen])
	}
	return s
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
