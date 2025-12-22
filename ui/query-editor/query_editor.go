package queryeditor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/config"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/lsp"
	"github.com/sheenazien8/sq/storage"
	"github.com/sheenazien8/sq/ui/completion"
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
	textarea        textarea.Model
	resultTable     table.Model
	lspClient       *lsp.Client
	completionUI    completion.Model
	connectionName  string
	databaseName    string
	width           int
	height          int
	focused         bool
	showResults     bool
	lastError       string
	editorHeight    int // Height of the editor area
	resultHeight    int // Height of the result area
	vimMode         VimMode
	vimEnabled      bool
	documentURI     string
	documentVersion int
	lspInitialized  bool
	lspInitPending  bool
}

// New creates a new query editor model
func New(connectionName, databaseName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your SQL query here...\nPress F5 or Ctrl+E to execute\nVim mode enabled (press i to insert, Esc for normal)\nLSP: F2 or Ctrl+Space for completion (requires database connections)"
	ta.SetWidth(80)
	ta.SetHeight(5)
	ta.CharLimit = 0 // No character limit
	// Keep textarea focused so cursor is visible
	ta.Focus()

	// Generate sqls config and create LSP client
	if err := config.SaveSQLSConfig(); err != nil {
		logger.Debug("Failed to save sqls config", map[string]any{"error": err.Error()})
	}

	configPath, _ := config.GetSQLSConfigPath()
	lspClient := lsp.NewClientWithConfig(configPath)

	// Create completion UI
	compUI := completion.New()

	// Generate document URI
	documentURI := fmt.Sprintf("file:///%s_%s.sql", connectionName, databaseName)

	return Model{
		textarea:        ta,
		lspClient:       lspClient,
		completionUI:    compUI,
		connectionName:  connectionName,
		databaseName:    databaseName,
		focused:         true,
		showResults:     false,
		editorHeight:    8,
		vimMode:         VimNormal,
		vimEnabled:      true,
		documentURI:     documentURI,
		documentVersion: 1,
		lspInitialized:  false,
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

	// Set textarea size (account for borders and padding)
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(m.editorHeight - 2)

	// Set result table size if showing results
	if m.showResults && m.resultHeight > 0 {
		m.resultTable.SetSize(width-4, m.resultHeight-2)
	}

	// Set completion UI size
	m.completionUI.SetSize(40, 10, 10, m.editorHeight+2)
}

// SetFocused sets whether the query editor is focused
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.textarea.Focus()
		// Mark LSP init as pending if not initialized
		if !m.lspInitialized && !m.lspInitPending {
			m.lspInitPending = true
		}
	} else {
		m.textarea.Blur()
	}
}

// Focused returns whether the query editor is focused
func (m Model) Focused() bool {
	return m.focused
}

// GetQuery returns the current query text
func (m Model) GetQuery() string {
	return strings.TrimSpace(m.textarea.Value())
}

// SetQuery sets the query text
func (m *Model) SetQuery(query string) {
	m.textarea.SetValue(query)
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

// InitLSP initializes the LSP client and server
func (m *Model) InitLSP() tea.Cmd {
	return func() tea.Msg {
		logger.Debug("Starting LSP initialization", map[string]any{
			"connectionName": m.connectionName,
			"databaseName":   m.databaseName,
			"documentURI":    m.documentURI,
		})

		// Check if we have any database connections first
		connections, err := storage.GetAllConnections()
		if err != nil {
			logger.Debug("Failed to check database connections", map[string]any{"error": err.Error()})
			return nil
		}

		if len(connections) == 0 {
			logger.Debug("No database connections found - LSP will not initialize", nil)
			return nil
		}

		logger.Debug("Found database connections, proceeding with LSP init", map[string]any{"count": len(connections)})

		// Generate and save sqls config
		if err := config.SaveSQLSConfig(); err != nil {
			logger.Debug("Failed to save SQLS config", map[string]any{"error": err.Error()})
			return nil
		}

		configPath, err := config.GetSQLSConfigPath()
		if err != nil {
			logger.Debug("Failed to get SQLS config path", map[string]any{"error": err.Error()})
			return nil
		}

		logger.Debug("SQLS config saved", map[string]any{"configPath": configPath})

		// Check if sqls command exists
		if m.lspClient == nil {
			m.lspClient = lsp.NewClientWithConfig(configPath)
		}

		logger.Debug("Starting LSP client", nil)
		if err := m.lspClient.Start(); err != nil {
			logger.Debug("Failed to start LSP client", map[string]any{"error": err.Error()})
			return nil
		}

		logger.Debug("LSP client started, initializing server", nil)

		// Initialize LSP server
		capabilities := map[string]interface{}{
			"textDocument": map[string]interface{}{
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": false,
					},
				},
			},
		}

		response, err := m.lspClient.Initialize("file:///"+m.connectionName, capabilities)
		if err != nil {
			logger.Debug("Failed to initialize LSP server", map[string]any{"error": err.Error()})
			return nil
		}

		logger.Debug("LSP server initialized", map[string]any{"response": response})

		// Send initialized notification
		if err := m.lspClient.Initialized(); err != nil {
			logger.Debug("Failed to send initialized notification", map[string]any{"error": err.Error()})
			return nil
		}

		// Send didOpen for current document
		text := m.GetQuery()
		logger.Debug("Sending didOpen notification", map[string]any{"textLength": len(text)})
		if err := m.lspClient.DidOpen(m.documentURI, "sql", text); err != nil {
			logger.Debug("Failed to send didOpen notification", map[string]any{"error": err.Error()})
			return nil
		}

		m.lspInitialized = true
		logger.Debug("LSP initialization completed successfully", nil)

		return nil
	}
}

// ShutdownLSP shuts down the LSP client
func (m *Model) ShutdownLSP() {
	if m.lspClient != nil {
		m.lspClient.Stop()
	}
}

// getCursorPosition returns the current cursor position as line and character
func (m Model) getCursorPosition() (int, int) {
	// Get current line and column from textarea
	// This is a simplified implementation - bubbles/textarea doesn't expose
	// cursor position directly, so we'll use line/column info
	line := m.textarea.Line()
	col := m.textarea.LineInfo().ColumnOffset

	return line, col
}

// triggerCompletion triggers LSP completion at current cursor position
func (m *Model) triggerCompletion() tea.Cmd {
	if !m.lspInitialized {
		return nil
	}

	return func() tea.Msg {
		line, character := m.getCursorPosition()

		response, err := m.lspClient.Completion(m.documentURI, line, character)
		if err != nil {
			logger.Debug("Failed to get completion", map[string]any{"error": err.Error()})
			return nil
		}

		// Parse completion items from response
		if items, ok := response.Result.(map[string]interface{}); ok {
			if itemList, ok := items["items"].([]interface{}); ok {
				completionItems := make([]completion.CompletionItem, 0, len(itemList))
				for _, item := range itemList {
					if itemMap, ok := item.(map[string]interface{}); ok {
						var compItem completion.CompletionItem
						if label, ok := itemMap["label"].(string); ok {
							compItem.Label = label
						}
						if kind, ok := itemMap["kind"].(float64); ok {
							compItem.Kind = int(kind)
						}
						if detail, ok := itemMap["detail"].(string); ok {
							compItem.Detail = detail
						}
						if insertText, ok := itemMap["insertText"].(string); ok {
							compItem.InsertText = insertText
						} else {
							compItem.InsertText = compItem.Label
						}

						completionItems = append(completionItems, compItem)
					}
				}

				if len(completionItems) > 0 {
					return LSPCompletionMsg{Items: completionItems}
				}
			}
		}

		return nil
	}
}

// LSPCompletionMsg is sent when completion items are received
type LSPCompletionMsg struct {
	Items []completion.CompletionItem
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case LSPCompletionMsg:
		// Handle LSP completion response
		m.completionUI.SetItems(msg.Items)
		return m, nil
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
		case "ctrl+space", "ctrl+@", "ctrl+ ", "f2":
			// Trigger LSP completion (Ctrl+Space is often interpreted as Ctrl+@ in terminals, Tab is alternative)
			logger.Debug("LSP completion triggered", map[string]any{
				"key":            keyStr,
				"lspInitialized": m.lspInitialized,
				"vimMode":        m.vimMode,
			})
			if m.lspInitialized && m.vimMode == VimInsert {
				m.completionUI.Show()
				return m, m.triggerCompletion()
			}
			return m, nil
		case "ctrl+r":
			// Toggle between editor and results focus
			if m.showResults {
				if m.resultTable.Focused() {
					// Switch from results to editor
					m.resultTable.SetFocused(false)
					m.textarea.Focus()
					m.vimMode = VimNormal
				} else {
					// Switch from editor to results
					m.textarea.Blur()
					m.resultTable.SetFocused(true)
				}
			}
			return m, nil
		}

		// If results table is focused, handle its input
		if m.showResults && m.resultTable.Focused() {
			// Allow switching back to editor
			if keyStr == "i" || keyStr == "a" {
				m.resultTable.SetFocused(false)
				m.vimMode = VimInsert
				m.textarea.Focus()
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

		// Handle completion UI if visible
		if m.completionUI.Visible() {
			var compCmd tea.Cmd
			m.completionUI, compCmd = m.completionUI.Update(msg)
			cmds = append(cmds, compCmd)

			if m.completionUI.Selected() {
				// Insert selected completion item
				selectedItem := m.completionUI.SelectedItem()
				if selectedItem.InsertText != "" {
					// Use textarea's insert functionality - simpler approach
					// Send the insert text as key input to textarea
					for _, r := range selectedItem.InsertText {
						keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
						m.textarea, _ = m.textarea.Update(keyMsg)
					}

					// Send LSP didChange notification
					if m.lspInitialized {
						newText := m.textarea.Value()
						m.documentVersion++
						go func() {
							if err := m.lspClient.DidChange(m.documentURI, newText, m.documentVersion); err != nil {
								logger.Debug("Failed to send didChange notification", map[string]any{"error": err.Error()})
							}
						}()
					}
				}
				m.completionUI.Hide()
			}
			return m, tea.Batch(cmds...)
		}

		// Handle vim modes
		if m.vimEnabled {
			return m.handleVimInput(msg)
		}

		// Non-vim mode: pass directly to textarea
		oldText := m.textarea.Value()
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)

		// Send LSP didChange notification if text changed
		newText := m.textarea.Value()
		if oldText != newText && m.lspInitialized {
			m.documentVersion++
			go func() {
				if err := m.lspClient.DidChange(m.documentURI, newText, m.documentVersion); err != nil {
					logger.Debug("Failed to send didChange notification", map[string]any{"error": err.Error()})
				}
			}()
		}

		// Initialize LSP if pending
		if m.lspInitPending && !m.lspInitialized {
			m.lspInitPending = false
			initCmd := m.InitLSP()
			cmds = append(cmds, initCmd)
		}
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
			// Keep textarea focused so cursor remains visible
			return m, nil
		}
		// In insert mode, pass everything to textarea
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleVimNormal handles keys in normal mode
func (m Model) handleVimNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	keyStr := msg.String()

	switch keyStr {
	// Enter insert mode
	case "i":
		m.vimMode = VimInsert
		return m, nil
	case "a":
		m.vimMode = VimInsert
		// Move cursor right (append) - send right arrow key
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "I":
		m.vimMode = VimInsert
		// Move to beginning of line
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})
		return m, nil
	case "A":
		m.vimMode = VimInsert
		// Move to end of line
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
		return m, nil
	case "o":
		m.vimMode = VimInsert
		// Move to end of line and insert newline
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, nil
	case "O":
		m.vimMode = VimInsert
		// Move to beginning of line and insert newline above
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil

	// Navigation - use arrow keys directly
	case "h":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
		return m, nil
	case "j":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case "k":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case "l":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "left":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
		return m, nil
	case "down":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	case "up":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	case "right":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, nil
	case "0":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})
		return m, nil
	case "$":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
		return m, nil
	case "g":
		// gg - go to beginning (simplified, just g for now)
		m.textarea.CursorStart()
		return m, nil
	case "G":
		// G - go to end
		m.textarea.CursorEnd()
		return m, nil
	case "w":
		// Move word forward
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
		return m, nil
	case "b":
		// Move word backward
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
		return m, nil

	// Deletion
	case "x":
		// Delete character under cursor
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
		return m, nil
	case "X":
		// Delete character before cursor (backspace)
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		return m, nil
	case "d":
		// For simplicity, 'd' alone does nothing - would need 'dd' detection
		return m, nil

	// Undo (if supported by textarea)
	case "u":
		// Textarea doesn't have built-in undo, but try ctrl+z
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
		return m, nil
	}

	return m, nil
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

	editorContent := m.textarea.View()
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
		statusText = "i: Insert | hjkl: Navigate | F5: Execute | F2: Complete | Ctrl+R: Results"
	} else {
		statusText = "Esc: Normal | F2: Complete | F5/Ctrl+E: Execute | Ctrl+R: Results"
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

	// Render completion UI if visible
	completionView := ""
	if m.completionUI.Visible() {
		// Position completion popup near cursor (simplified positioning)
		completionView = m.completionUI.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		editorSection,
		statusBar,
		completionView,
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
