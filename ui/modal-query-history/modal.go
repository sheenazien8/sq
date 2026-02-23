package modalqueryhistory

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/storage"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/query-editor"
	"github.com/sheenazien8/sq/ui/theme"
)

// QueryHistoryContent implements modal.Content and shows recent queries
type QueryHistoryContent struct {
	entries        []storage.QueryHistory
	cursor         int
	closed         bool
	width          int
	visibleLines   int
	start          int // start index for visible window
	connectionName string
}

func NewQueryHistoryContent() *QueryHistoryContent {
	return &QueryHistoryContent{visibleLines: 10, start: 0}
}

func (c *QueryHistoryContent) LoadForConnection(connName string, limit int) error {
	// Find connection ID by name
	conns, err := storage.GetAllConnections()
	if err != nil {
		return err
	}
	var connID int64 = 0
	for _, conn := range conns {
		if conn.Name == connName {
			connID = conn.ID
			break
		}
	}

	var entries []storage.QueryHistory
	if connID == 0 {
		entries, err = storage.GetRecentQueryHistory(limit)
	} else {
		entries, err = storage.GetQueryHistory(connID, limit)
	}
	if err != nil {
		return err
	}
	c.entries = entries
	c.cursor = 0
	c.start = 0
	c.closed = false
	c.connectionName = connName
	return nil
}

func (c *QueryHistoryContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if c.cursor < len(c.entries)-1 {
				c.cursor++
				if c.cursor >= c.start+c.visibleLines {
					c.start = c.cursor - c.visibleLines + 1
				}
			}
		case "k", "up":
			if c.cursor > 0 {
				c.cursor--
				if c.cursor < c.start {
					c.start = c.cursor
				}
			}
		case "enter":
			if c.cursor >= 0 && c.cursor < len(c.entries) {
				q := c.entries[c.cursor].Query
				c.closed = true
				return c, func() tea.Msg {
					return queryeditor.QueryLoadFromHistoryMsg{Query: q}
				}
			}
		case "pgdown", "pagedown":
			// page down
			if c.start+c.visibleLines < len(c.entries) {
				c.start = min(c.start+c.visibleLines, len(c.entries)-c.visibleLines)
				c.cursor = min(c.cursor+c.visibleLines, len(c.entries)-1)
			}
		case "pgup", "pageup":
			// page up
			if c.start > 0 {
				c.start = max(0, c.start-c.visibleLines)
				c.cursor = max(0, c.cursor-c.visibleLines)
			}
		case "esc", "q":
			c.closed = true
		case "y":
			// Copy selected query to clipboard via modal/parent handling
			if c.cursor >= 0 && c.cursor < len(c.entries) {
				q := c.entries[c.cursor].Query
				return c, func() tea.Msg {
					return queryeditor.YankQueryMsg{Content: q}
				}
			}
		}
	}
	return c, nil
}

func (c *QueryHistoryContent) View() string {
	t := theme.Current

	header := lipgloss.NewStyle().Foreground(t.Colors.Primary).Bold(true).Render(fmt.Sprintf("Query History — %s", c.connectionName))

	var lines []string
	lines = append(lines, header, "")

	limit := c.visibleLines
	start := c.start
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(c.entries) {
		end = len(c.entries)
	}

	for i := start; i < end; i++ {
		e := c.entries[i]
		ts := e.ExecutedAt.Format(time.RFC3339)
		prefix := "  "
		if i == c.cursor {
			prefix = "> "
		}
		// show preview of query (first line)
		preview := e.Query
		if len(preview) > 120 {
			preview = preview[:117] + "..."
		}
		lines = append(lines, prefix+ts+" — "+preview)
	}

	if len(c.entries) == 0 {
		lines = append(lines, "(no history)")
	}

	help := lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Render("j/k: navigate | PgUp/PgDn: page | Enter: load | y: copy | q/Esc: close")
	lines = append(lines, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (c *QueryHistoryContent) Result() modal.Result { return modal.ResultNone }
func (c *QueryHistoryContent) ShouldClose() bool    { return c.closed }
func (c *QueryHistoryContent) SetWidth(width int)   { c.width = width }

// Model wraps modal.Model for query history
type Model struct {
	modal   modal.Model
	content *QueryHistoryContent
}

func New() Model {
	content := NewQueryHistoryContent()
	m := modal.New("Query History", content)
	return Model{modal: m, content: content}
}

func (m *Model) ShowFor(connectionName string) error {
	// Load entries (limit 100)
	if err := m.content.LoadForConnection(connectionName, 100); err != nil {
		return err
	}
	m.modal.SetContent(m.content)
	m.modal.Show()
	return nil
}

func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.modal, cmd = m.modal.Update(msg)
	return *m, cmd
}

func (m *Model) View() string     { return m.modal.View() }
func (m *Model) Visible() bool    { return m.modal.Visible() }
func (m *Model) Show()            { m.modal.Show() }
func (m *Model) Hide()            { m.modal.Hide() }
func (m *Model) SetSize(w, h int) { m.modal.SetSize(w, h) }
