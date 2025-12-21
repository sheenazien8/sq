package sidebar

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/logger"
	"github.com/sheenazien8/db-client-tui/storage"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

type Table struct {
	Name     string
	RowCount int64
	Selected bool
}

// Connection represents a database item in the sidebar
type Connection struct {
	Name     string
	Type     string
	Host     string
	Selected bool
	Expanded bool
	Tables   []Table
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

// Model represents the sidebar with database list
type Model struct {
	connections []Connection
	cursor      int
	offset      int
	width       int
	height      int
	focused     bool
}

// New creates a new sidebar model with sample databases
func New() Model {
	return Model{
		connections: getConnections(),
		cursor:      0,
		offset:      0,
		focused:     false,
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
		// Add mock tables for demonstration
		mockTables := []Table{
			{Name: "users", RowCount: 1250},
			{Name: "products", RowCount: 500},
			{Name: "orders", RowCount: 3200},
			{Name: "categories", RowCount: 45},
		}

		data = append(data, Connection{
			Name:     connection.Name,
			Type:     connection.Driver,
			Host:     connection.URL,
			Tables:   mockTables,
			Expanded: false, // start collapsed
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

// SetDatabases updates the database list
func (m *Model) SetDatabases(databases []Connection) {
	m.connections = databases
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

	for connIdx, conn := range m.connections {
		items = append(items, TreeItem{
			ConnectionIndex: connIdx,
			TableIndex:      -1,
			Level:           0,
			IsLastChild:     false,
		})

		if conn.Expanded {
			for tableIdx := range conn.Tables {
				isLast := tableIdx == len(conn.Tables)-1
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
	return max(0, m.height-5)
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

	// Title
	title := t.SidebarTitle.Width(innerWidth).Render(" Databases")
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

			text = treeChar + " " + icon + " " + truncateString(conn.Name, innerWidth-6)

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
			text = prefix + " " + tableIcon + " " + truncateString(table.Name, innerWidth-8) +
				" (" + intToStr(int(table.RowCount)) + ")"

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
		Width(m.width - 2).
		Height(m.height - 2).
		Render(content)
}

// getConnectionIcon returns an icon for the database type
func getConnectionIcon(dbType string) string {
	switch dbType {
	case "mysql":
		return "[M]"
	case "postgres":
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
