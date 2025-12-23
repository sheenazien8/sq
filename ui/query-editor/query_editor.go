package queryeditor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/mjibson/sqlfmt"
	"github.com/sheenazien8/sq/logger"
	syntaxeditor "github.com/sheenazien8/sq/ui/syntax-editor"
	"github.com/sheenazien8/sq/ui/table"
	"github.com/sheenazien8/sq/ui/theme"
)

// VimMode represents the current vim mode
type VimMode int

const (
	VimNormal VimMode = iota
	VimInsert
	VimVisual
)

// QueryExecuteMsg is sent when the user executes a query
type QueryExecuteMsg struct {
	Query          string
	ConnectionName string
	DatabaseName   string
}

// QueryResultMsg is sent when a query has been executed
type QueryResultMsg struct {
	Columns []table.Column
	Rows    []table.Row
	Error   error
}

// CellPreviewMsg is sent when user wants to preview a cell in the results
type CellPreviewMsg struct {
	Content string
}

// YankCellMsg is sent when user wants to copy a cell content
type YankCellMsg struct {
	Content string
}

// Model represents the query editor component
type Model struct {
	syntaxEditor   syntaxeditor.Model
	resultTable    table.Model
	connectionName string
	databaseName   string
	width          int
	height         int
	focused        bool
	showResults    bool
	lastError      string
	editorHeight   int // Height of the editor area
	resultHeight   int // Height of the result area
	vimMode        VimMode
	vimEnabled     bool
	pendingCommand string // Pending vim command (e.g., "d" for dd)
	yankBuffer     string // Buffer for yanked text
	visualStartX   int    // Start X for visual selection
	visualStartY   int    // Start Y for visual selection
}

// New creates a new query editor model
func New(connectionName, databaseName string) Model {
	se := syntaxeditor.New()
	se.SetPlaceholder("Enter your SQL query here...\nPress F5 or Ctrl+E to execute\nVim mode enabled (press i to insert, Esc for normal)")
	se.SetBorder(false) // Query editor provides its own border
	se.SetSize(80, 5)
	se.SetCharLimit(0) // No character limit
	// Keep editor focused so cursor is visible
	se.Focus()

	return Model{
		syntaxEditor:   se,
		connectionName: connectionName,
		databaseName:   databaseName,
		focused:        true,
		showResults:    false,
		editorHeight:   8,
		vimMode:        VimNormal,
		vimEnabled:     true,
		pendingCommand: "",
		yankBuffer:     "",
		visualStartX:   0,
		visualStartY:   0,
	}
}

// SetSize sets the query editor dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Calculate heights: editor gets top portion, results get the rest
	if m.showResults {
		m.editorHeight = max(5, height/3)
		m.resultHeight = height - m.editorHeight - 3 // 3 for borders and separator
	} else {
		m.editorHeight = height - 2 // Leave room for status bar
		m.resultHeight = 0
	}

	// Set syntax editor size (account for borders and padding)
	m.syntaxEditor.SetSize(width-4, m.editorHeight-2)

	// Set result table size if showing results
	if m.showResults && m.resultHeight > 0 {
		m.resultTable.SetSize(width-4, m.resultHeight-2)
	}
}

// SetFocused sets whether the query editor is focused
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.syntaxEditor.Focus()
	} else {
		m.syntaxEditor.Blur()
	}
}

// Focused returns whether the query editor is focused
func (m Model) Focused() bool {
	return m.focused
}

// GetQuery returns the current query text
func (m Model) GetQuery() string {
	return strings.TrimSpace(m.syntaxEditor.Value())
}

// SetQuery sets the query text
func (m *Model) SetQuery(query string) {
	m.syntaxEditor.SetValue(query)
}

// GetConnectionName returns the connection name
func (m Model) GetConnectionName() string {
	return m.connectionName
}

// GetDatabaseName returns the database name
func (m Model) GetDatabaseName() string {
	return m.databaseName
}

// SetResults sets the query results
func (m *Model) SetResults(columns []table.Column, rows []table.Row) {
	m.resultTable = table.New(columns, rows)
	m.resultTable.SetSize(m.width-4, m.resultHeight-2)
	m.resultTable.SetFocused(false)
	m.showResults = true
	m.lastError = ""
	m.SetSize(m.width, m.height) // Recalculate sizes
}

// SetError sets an error message
func (m *Model) SetError(err string) {
	m.lastError = err
	m.showResults = false
	m.SetSize(m.width, m.height) // Recalculate sizes
}

// HasResults returns whether there are query results to display
func (m Model) HasResults() bool {
	return m.showResults
}

// GetError returns the last error message
func (m Model) GetError() string {
	return m.lastError
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()
		logger.Debug("QueryEditor received key", map[string]any{
			"key":     keyStr,
			"vimMode": m.vimMode,
		})

		// Global shortcuts that work in any mode
		switch keyStr {
		case "f5", "ctrl+e":
			// Execute the query
			query := m.GetQuery()
			logger.Debug("Execute query triggered", map[string]any{
				"query":      query,
				"connection": m.connectionName,
				"database":   m.databaseName,
			})
			if query != "" {
				return m, func() tea.Msg {
					return QueryExecuteMsg{
						Query:          query,
						ConnectionName: m.connectionName,
						DatabaseName:   m.databaseName,
					}
				}
			}
			return m, nil
		case "ctrl+r":
			// Toggle between editor and results focus
			if m.showResults {
				if m.resultTable.Focused() {
					// Switch from results to editor
					m.resultTable.SetFocused(false)
					m.syntaxEditor.Focus()
					m.vimMode = VimNormal
				} else {
					// Switch from editor to results
					m.syntaxEditor.Blur()
					m.resultTable.SetFocused(true)
				}
			}
			return m, nil
		case "ctrl+f":
			// Format SQL
			m.formatSQL()
			return m, nil
		}

		// If results table is focused, handle its input
		if m.showResults && m.resultTable.Focused() {
			// Allow switching back to editor
			if keyStr == "i" || keyStr == "a" {
				m.resultTable.SetFocused(false)
				m.vimMode = VimInsert
				m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
				m.syntaxEditor.Focus()
				return m, nil
			}
			// Preview cell content
			if keyStr == "p" {
				cellContent := m.resultTable.SelectedCell()
				if cellContent != "" {
					return m, func() tea.Msg {
						return CellPreviewMsg{Content: cellContent}
					}
				}
				return m, nil
			}
			// Yank (copy) cell content
			if keyStr == "y" {
				cellContent := m.resultTable.SelectedCell()
				if cellContent != "" {
					// Return a message that the app can handle for clipboard
					return m, func() tea.Msg {
						return YankCellMsg{Content: cellContent}
					}
				}
				return m, nil
			}
			m.resultTable, cmd = m.resultTable.Update(msg)
			return m, cmd
		}

		// Handle vim modes
		if m.vimEnabled {
			return m.handleVimInput(msg)
		}

		// Non-vim mode: pass directly to syntax editor
		m.syntaxEditor, cmd = m.syntaxEditor.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleVimInput processes input based on current vim mode
func (m Model) handleVimInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	keyStr := msg.String()

	switch m.vimMode {
	case VimNormal:
		return m.handleVimNormal(msg)
	case VimInsert:
		// Escape returns to normal mode
		if keyStr == "esc" {
			m.vimMode = VimNormal
			// Switch to block cursor for normal mode
			m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorBlock)
			return m, nil
		}
		// In insert mode, pass everything to syntax editor
		m.syntaxEditor, cmd = m.syntaxEditor.Update(msg)
		return m, cmd
	case VimVisual:
		return m.handleVimVisual(msg)
	}

	return m, nil
}

// handleVimNormal handles keys in normal mode
func (m Model) handleVimNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	keyStr := msg.String()

	// Handle pending commands (e.g., dd, yy)
	if m.pendingCommand != "" {
		if m.pendingCommand == "d" && keyStr == "d" {
			// Delete line and yank it
			content := m.syntaxEditor.Value()
			lines := strings.Split(content, "\n")
			if len(lines) > 0 && m.syntaxEditor.CursorY() < len(lines) {
				m.yankBuffer = lines[m.syntaxEditor.CursorY()]
				lines = append(lines[:m.syntaxEditor.CursorY()], lines[m.syntaxEditor.CursorY()+1:]...)
				if len(lines) == 0 {
					lines = []string{""}
				}
				newContent := strings.Join(lines, "\n")
				m.syntaxEditor.SetValue(newContent)
				// Adjust cursor if necessary
				if m.syntaxEditor.CursorY() >= len(lines) {
					m.syntaxEditor.CursorEnd()
				}
			}
		} else if m.pendingCommand == "y" && keyStr == "y" {
			// Yank line
			content := m.syntaxEditor.Value()
			lines := strings.Split(content, "\n")
			if len(lines) > 0 && m.syntaxEditor.CursorY() < len(lines) {
				m.yankBuffer = lines[m.syntaxEditor.CursorY()]
			}
		}
		m.pendingCommand = ""
		return m, nil
	}

	switch keyStr {
	// Enter insert mode
	case "i":
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		return m, nil
	case "a":
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		// Move cursor right (append) - send right arrow key
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "I":
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		// Move to beginning of line
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyHome})
		return m, nil
	case "A":
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		// Move to end of line
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyEnd})
		return m, nil
	case "o":
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		// Move to end of line and insert newline
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyEnd})
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, nil
	case "O":
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		// Move to beginning of line and insert newline above
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyHome})
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil

	// Navigation - use arrow keys directly
	case "h":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		return m, nil
	case "j":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case "k":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case "l":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "left":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		return m, nil
	case "down":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case "up":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case "right":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "0":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyHome})
		return m, nil
	case "$":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyEnd})
		return m, nil
	case "g":
		// gg - go to beginning (simplified, just g for now)
		m.syntaxEditor.CursorStart()
		return m, nil
	case "G":
		// G - go to end
		m.syntaxEditor.CursorEnd()
		return m, nil
	case "w":
		// Move word forward
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
		return m, nil
	case "b":
		// Move word backward
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
		return m, nil

	// Deletion
	case "x":
		// Delete character under cursor
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyDelete})
		return m, nil
	case "X":
		// Delete character before cursor (backspace)
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		return m, nil
	case "d":
		m.pendingCommand = "d"
		return m, nil

	// Undo (if supported by textarea)
	case "u":
		// Editor doesn't have built-in undo, but try ctrl+z
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
		return m, nil
	case "y":
		m.pendingCommand = "y"
		return m, nil
	case "p":
		// Paste yank buffer after cursor
		if m.yankBuffer != "" {
			// Insert at cursor position
			content := m.syntaxEditor.Value()
			lines := strings.Split(content, "\n")
			cursorY := m.syntaxEditor.CursorY()
			cursorX := m.syntaxEditor.CursorX()
			if cursorY < len(lines) {
				line := lines[cursorY]
				before := line[:cursorX]
				after := line[cursorX:]
				lines[cursorY] = before + m.yankBuffer + after
				newContent := strings.Join(lines, "\n")
				m.syntaxEditor.SetValue(newContent)
				// Move cursor after pasted text
				newCursorX := cursorX + len(m.yankBuffer)
				// But since SetValue resets cursor, need to adjust
				// For simplicity, just set cursor to end of inserted
				for i := 0; i < newCursorX-cursorX; i++ {
					m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
				}
			}
		}
		return m, nil
	case "S":
		// Substitute line: delete line and enter insert mode
		content := m.syntaxEditor.Value()
		lines := strings.Split(content, "\n")
		cursorY := m.syntaxEditor.CursorY()
		if cursorY < len(lines) {
			m.yankBuffer = lines[cursorY]
			lines = append(lines[:cursorY], lines[cursorY+1:]...)
			if len(lines) == 0 {
				lines = []string{""}
			}
			newContent := strings.Join(lines, "\n")
			m.syntaxEditor.SetValue(newContent)
			if cursorY >= len(lines) {
				m.syntaxEditor.CursorEnd()
			}
		}
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		return m, nil
	case "e":
		// Move to end of word
		content := m.syntaxEditor.Value()
		lines := strings.Split(content, "\n")
		cursorY := m.syntaxEditor.CursorY()
		cursorX := m.syntaxEditor.CursorX()
		if cursorY < len(lines) {
			line := lines[cursorY]
			i := cursorX
			for i < len(line) {
				if line[i] != ' ' {
					// Found start of word, go to end
					for i < len(line) && line[i] != ' ' {
						i++
					}
					// Move cursor to i-1 (end of word)
					for cursorX < i-1 {
						m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
						cursorX++
					}
					break
				}
				i++
			}
		}
		return m, nil
	case "C":
		// Change to end of line
		content := m.syntaxEditor.Value()
		lines := strings.Split(content, "\n")
		cursorY := m.syntaxEditor.CursorY()
		cursorX := m.syntaxEditor.CursorX()
		if cursorY < len(lines) {
			line := lines[cursorY]
			m.yankBuffer = line[cursorX:]   // yank from cursor to end
			lines[cursorY] = line[:cursorX] // keep before cursor
			newContent := strings.Join(lines, "\n")
			m.syntaxEditor.SetValue(newContent)
		}
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		return m, nil
	case "v":
		m.vimMode = VimVisual
		m.visualStartX = m.syntaxEditor.CursorX()
		m.visualStartY = m.syntaxEditor.CursorY()
		m.syntaxEditor.SetVisualMode(true)
		m.syntaxEditor.SetVisualStart(m.visualStartX, m.visualStartY)
		return m, nil
	}

	return m, nil
}

// handleVimVisual handles keys in visual mode
func (m Model) handleVimVisual(msg tea.KeyMsg) (Model, tea.Cmd) {
	keyStr := msg.String()

	switch keyStr {
	case "esc":
		m.vimMode = VimNormal
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorBlock)
		m.syntaxEditor.SetVisualMode(false)
		return m, nil
	case "d":
		m.deleteVisualSelection()
		m.vimMode = VimNormal
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorBlock)
		m.syntaxEditor.SetVisualMode(false)
		return m, nil
	case "y":
		m.yankVisualSelection()
		m.vimMode = VimNormal
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorBlock)
		m.syntaxEditor.SetVisualMode(false)
		return m, nil
	case "c":
		m.deleteVisualSelection()
		m.vimMode = VimInsert
		m.syntaxEditor.SetCursorStyle(syntaxeditor.CursorLine)
		m.syntaxEditor.SetVisualMode(false)
		return m, nil
	// Movement keys extend selection
	case "h":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		return m, nil
	case "j":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case "k":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case "l":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "left":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		return m, nil
	case "down":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case "up":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case "right":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "0":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyHome})
		return m, nil
	case "$":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyEnd})
		return m, nil
	case "w":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
		return m, nil
	case "b":
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
		return m, nil
	case "e":
		// Move to end of word (same as normal)
		content := m.syntaxEditor.Value()
		lines := strings.Split(content, "\n")
		cursorY := m.syntaxEditor.CursorY()
		cursorX := m.syntaxEditor.CursorX()
		if cursorY < len(lines) {
			line := lines[cursorY]
			i := cursorX
			for i < len(line) {
				if line[i] != ' ' {
					for i < len(line) && line[i] != ' ' {
						i++
					}
					for cursorX < i-1 {
						m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyRight})
						cursorX++
					}
					break
				}
				i++
			}
		}
		return m, nil
	}

	return m, nil
}

// deleteVisualSelection deletes the current visual selection
func (m *Model) deleteVisualSelection() {
	startY := m.visualStartY
	startX := m.visualStartX
	endY := m.syntaxEditor.CursorY()
	endX := m.syntaxEditor.CursorX()

	if startY > endY || (startY == endY && startX > endX) {
		startY, endY = endY, startY
		startX, endX = endX, startX
	}

	content := m.syntaxEditor.Value()
	lines := strings.Split(content, "\n")

	if startY == endY {
		// Same line
		if startX < len(lines[startY]) && endX <= len(lines[startY]) {
			m.yankBuffer = lines[startY][startX:endX]
			lines[startY] = lines[startY][:startX] + lines[startY][endX:]
		}
	} else {
		// Multi-line
		var yanked []string
		// First line: from startX to end
		yanked = append(yanked, lines[startY][startX:])
		lines[startY] = lines[startY][:startX]
		// Middle lines
		for y := startY + 1; y < endY; y++ {
			yanked = append(yanked, lines[y])
		}
		// Last line: from start to endX
		if endY < len(lines) && endX <= len(lines[endY]) {
			yanked = append(yanked, lines[endY][:endX])
			lines[endY] = lines[endY][endX:]
		}
		// Remove middle lines
		lines = append(lines[:startY+1], lines[endY:]...)
		// Join yanked
		m.yankBuffer = strings.Join(yanked, "\n")
	}

	newContent := strings.Join(lines, "\n")
	m.syntaxEditor.SetValue(newContent)
}

// yankVisualSelection yanks the current visual selection
func (m *Model) yankVisualSelection() {
	startY := m.visualStartY
	startX := m.visualStartX
	endY := m.syntaxEditor.CursorY()
	endX := m.syntaxEditor.CursorX()

	if startY > endY || (startY == endY && startX > endX) {
		startY, endY = endY, startY
		startX, endX = endX, startX
	}

	content := m.syntaxEditor.Value()
	lines := strings.Split(content, "\n")

	if startY == endY {
		if startX < len(lines[startY]) && endX <= len(lines[startY]) {
			m.yankBuffer = lines[startY][startX:endX]
		}
	} else {
		var yanked []string
		yanked = append(yanked, lines[startY][startX:])
		for y := startY + 1; y < endY; y++ {
			yanked = append(yanked, lines[y])
		}
		if endY < len(lines) && endX <= len(lines[endY]) {
			yanked = append(yanked, lines[endY][:endX])
		}
		m.yankBuffer = strings.Join(yanked, "\n")
	}
}

// formatSQL formats the SQL query using sqlfmt
func (m *Model) formatSQL() {
	query := m.syntaxEditor.Value()
	if strings.TrimSpace(query) == "" {
		return
	}

	cfg := tree.DefaultPrettyCfg()
	cfg.LineWidth = 80
	cfg.TabWidth = 2
	cfg.Simplify = true

	formatted, err := sqlfmt.FmtSQL(cfg, []string{query})
	if err != nil {
		// If formatting fails, log the error but don't change the content
		logger.Debug("SQL format error", map[string]any{"error": err.Error()})
		return
	}

	m.syntaxEditor.SetValue(strings.TrimSpace(formatted))
}

// GetVimMode returns the current vim mode as a string
func (m Model) GetVimMode() string {
	switch m.vimMode {
	case VimNormal:
		return "NORMAL"
	case VimInsert:
		return "INSERT"
	case VimVisual:
		return "VISUAL"
	default:
		return "NORMAL"
	}
}

// View renders the query editor
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	t := theme.Current

	// Editor section
	editorTitle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Bold(true).
		Render("Query Editor [" + m.connectionName + "." + m.databaseName + "]")

	editorStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Colors.Primary).
		Width(m.width - 4).
		Height(m.editorHeight - 2)

	editorContent := m.syntaxEditor.View()
	editorSection := lipgloss.JoinVertical(lipgloss.Left,
		editorTitle,
		editorStyle.Render(editorContent),
	)

	// Status bar with vim mode indicator
	var modeIndicator string
	if m.vimEnabled {
		modeStyle := lipgloss.NewStyle().Bold(true)
		switch m.vimMode {
		case VimNormal:
			modeStyle = modeStyle.Foreground(t.Colors.Primary).Background(t.Colors.Background)
			modeIndicator = modeStyle.Render(" NORMAL ")
		case VimInsert:
			modeStyle = modeStyle.Foreground(t.Colors.Background).Background(t.Colors.Success)
			modeIndicator = modeStyle.Render(" INSERT ")
		case VimVisual:
			modeStyle = modeStyle.Foreground(t.Colors.Background).Background(t.Colors.Warning)
			modeIndicator = modeStyle.Render(" VISUAL ")
		}
	}

	var statusText string
	if m.showResults && m.resultTable.Focused() {
		statusText = "hjkl: Navigate | p: Preview | y: Yank | i: Back to Editor | Ctrl+R: Editor"
	} else if m.vimMode == VimNormal {
		statusText = "i: Insert | hjkl: Navigate | F5: Execute | Ctrl+F: Format | Ctrl+R: Results"
	} else if m.vimMode == VimVisual {
		statusText = "hjkl: Select | d: Delete | y: Yank | c: Change | Esc: Normal"
	} else {
		statusText = "Esc: Normal | F5/Ctrl+E: Execute | Ctrl+F: Format | Ctrl+R: Results"
	}
	if m.lastError != "" {
		statusText = lipgloss.NewStyle().
			Foreground(t.Colors.Error).
			Render("Error: " + truncateText(m.lastError, m.width-20))
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Left,
		modeIndicator,
		" ",
		lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Render(statusText),
	)

	// Results section (if showing)
	if m.showResults && m.resultHeight > 0 {
		resultsTitle := lipgloss.NewStyle().
			Foreground(t.Colors.Success).
			Bold(true).
			Render("Results")

		resultsStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Colors.BorderFocused).
			Width(m.width - 4).
			Height(m.resultHeight - 2)

		resultsContent := m.resultTable.View()
		resultsSection := lipgloss.JoinVertical(lipgloss.Left,
			resultsTitle,
			resultsStyle.Render(resultsContent),
		)

		return lipgloss.JoinVertical(lipgloss.Left,
			editorSection,
			statusBar,
			resultsSection,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		editorSection,
		statusBar,
	)
}

// truncateText truncates text to a maximum width
func truncateText(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return "..."
	}
	return s[:maxWidth-3] + "..."
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
