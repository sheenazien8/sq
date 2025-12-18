package sidebar

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	Type     string // mysql, postgres, sqlite, etc.
	Host     string
	Selected bool
	Tables   []Table
}

// Model represents the sidebar with database list
type Model struct {
	databases []Connection
	cursor    int
	offset    int
	width     int
	height    int
	focused   bool
}

// New creates a new sidebar model with sample databases
func New() Model {
	return Model{
		databases: getConnections(),
		cursor:    0,
		offset:    0,
		focused:   false,
	}
}

func getConnections() (data []Connection) {
	connections, err := storage.GetAllConnections()
	if err != nil {
		for _, connection := range connections {
			data = append(data, Connection{
				Name: connection.Name,
				Type: connection.Driver,
				Host: connection.URL,
			})
		}

		return data
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

// SelectedDatabase returns the currently selected database (cursor position)
func (m Model) SelectedDatabase() *Connection {
	if m.cursor >= 0 && m.cursor < len(m.databases) {
		return &m.databases[m.cursor]
	}
	return nil
}

// ActiveDatabase returns the database that has been activated (via Enter key)
func (m Model) ActiveDatabase() *Connection {
	for i := range m.databases {
		if m.databases[i].Selected {
			return &m.databases[i]
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
	m.databases = databases
	if m.cursor >= len(databases) {
		m.cursor = max(0, len(databases)-1)
	}
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
			if m.cursor < len(m.databases)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.visibleItems() {
					m.offset = m.cursor - m.visibleItems() + 1
				}
			}
		case "home":
			m.cursor = 0
			m.offset = 0
		case "end":
			m.cursor = max(0, len(m.databases)-1)
			maxOffset := max(0, len(m.databases)-m.visibleItems())
			m.offset = maxOffset
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.databases) {
				// Clear previous selection
				for i := range m.databases {
					m.databases[i].Selected = false
				}
				m.databases[m.cursor].Selected = true
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
	title := t.SidebarTitle.Width(innerWidth).Render(" Connection name")
	lines = append(lines, title)

	// Separator
	separatorStyle := lipgloss.NewStyle().Foreground(t.Colors.BorderUnfocused)
	lines = append(lines, separatorStyle.Render(strings.Repeat("─", innerWidth)))

	// Database items
	visibleCount := m.visibleItems()
	endIdx := min(m.offset+visibleCount, len(m.databases))

	for i := m.offset; i < endIdx; i++ {
		db := m.databases[i]
		isSelected := i == m.cursor

		// Icon based on type
		icon := getDbIcon(db.Type)
		text := icon + " " + truncateString(db.Name, innerWidth-4)

		var line string
		if isSelected && m.focused {
			line = t.SidebarSelected.Width(innerWidth).Render(text)
		} else if db.Selected {
			line = t.SidebarActive.Width(innerWidth).Render(text)
		} else {
			line = t.SidebarItem.Width(innerWidth).Render(text)
		}
		lines = append(lines, line)
	}

	// Fill empty space
	for i := endIdx - m.offset; i < visibleCount; i++ {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	// Status bar
	status := t.StatusBar.Width(innerWidth).Align(lipgloss.Right).
		Render(intToStr(m.cursor+1) + "/" + intToStr(len(m.databases)))
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

// getDbIcon returns an icon for the database type
func getDbIcon(dbType string) string {
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
