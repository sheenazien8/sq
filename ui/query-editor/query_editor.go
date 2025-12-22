package queryeditor

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/config"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/lsp"
	"github.com/sheenazien8/sq/storage"
	"github.com/sheenazien8/sq/ui/completion"
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
	syntaxEditor    syntaxeditor.Model
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
	lspInitStarted  bool // Prevent multiple initialization attempts
}

// New creates a new query editor model
func New(connectionName, databaseName string) Model {
	se := syntaxeditor.New()
	se.SetPlaceholder("Enter your SQL query here...\nPress F5 or Ctrl+E to execute\nVim mode enabled (press i to insert, Esc for normal)\nLSP: F2 or Ctrl+Space for completion (requires database connections)")
	se.SetBorder(false) // Query editor provides its own border
	se.SetSize(80, 5)
	se.SetCharLimit(0) // No character limit
	// Keep editor focused so cursor is visible
	se.Focus()

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
		syntaxEditor:    se,
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
		lspInitStarted:  false,
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

	// Set completion UI size
	m.completionUI.SetSize(40, 10, 10, m.editorHeight+2)
}

// SetFocused sets whether the query editor is focused
func (m *Model) SetFocused(focused bool) {
	logger.Debug("QueryEditor SetFocused called", map[string]any{
		"focused":        focused,
		"lspInitialized": m.lspInitialized,
		"lspInitPending": m.lspInitPending,
		"lspInitStarted": m.lspInitStarted,
	})
	m.focused = focused
	if focused {
		m.syntaxEditor.Focus()
		// Don't auto-initialize LSP - wait for user to request completion
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

// InitLSP initializes the LSP client and server
func (m *Model) InitLSP() tea.Cmd {
	return func() tea.Msg {
		logger.Debug("InitLSP command function STARTED", nil)

		// Check if we have any database connections first
		connections, err := storage.GetAllConnections()
		if err != nil {
			logger.Debug("Failed to check database connections", map[string]any{"error": err.Error()})
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		if len(connections) == 0 {
			logger.Debug("No database connections found - LSP will not initialize", nil)
			return LSPInitializedMsg{Success: false, Error: "no database connections"}
		}

		// Generate and save sqls config
		if err := config.SaveSQLSConfig(); err != nil {
			logger.Debug("Failed to save SQLS config", map[string]any{"error": err.Error()})
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		configPath, err := config.GetSQLSConfigPath()
		if err != nil {
			logger.Debug("Failed to get SQLS config path", map[string]any{"error": err.Error()})
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		// Create new LSP client with config
		lspClient := lsp.NewClientWithConfig(configPath)

		logger.Debug("Starting LSP client", nil)
		// Start the client (this might be what's slow)
		if err := lspClient.Start(); err != nil {
			logger.Debug("Failed to start LSP client", map[string]any{"error": err.Error()})
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		logger.Debug("LSP client started, doing quick init", nil)

		// Do minimal initialization - just initialize, don't do full handshake yet
		capabilities := map[string]interface{}{
			"textDocument": map[string]interface{}{
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": false,
					},
				},
			},
		}

		_, err = lspClient.Initialize(m.documentURI, capabilities)
		if err != nil {
			logger.Debug("Failed to initialize LSP server", map[string]any{"error": err.Error()})
			lspClient.Stop()
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		// Send initialized notification
		if err := lspClient.Initialized(); err != nil {
			logger.Debug("Failed to send initialized notification", map[string]any{"error": err.Error()})
			lspClient.Stop()
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		// Send didOpen for current document
		text := m.GetQuery()
		logger.Debug("Sending didOpen notification", map[string]any{"uri": m.documentURI, "textLength": len(text)})
		if err := lspClient.DidOpen(m.documentURI, "sql", text); err != nil {
			logger.Debug("Failed to send didOpen notification", map[string]any{"error": err.Error()})
			lspClient.Stop()
			return LSPInitializedMsg{Success: false, Error: err.Error()}
		}

		logger.Debug("LSP initialization completed with didOpen", nil)

		msg := LSPInitializedMsg{
			Success: true,
			Client:  lspClient,
			DocURI:  m.documentURI,
		}
		logger.Debug("Returning LSPInitializedMsg", map[string]any{
			"success":   msg.Success,
			"hasClient": msg.Client != nil,
			"docURI":    msg.DocURI,
		})
		logger.Debug("InitLSP command function ENDING", nil)
		return msg
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
	// Get cursor position from syntax editor
	line, col := m.syntaxEditor.CursorPosition()
	return line, col
}

// triggerCompletion triggers LSP completion at current cursor position
func (m *Model) triggerCompletion() tea.Cmd {
	// Initialize LSP lazily if not already initialized
	if !m.lspInitialized {
		logger.Debug("LSP not initialized, initializing now", nil)

		// Synchronous LSP initialization - return a command that will send LSPInitializedMsg
		return func() tea.Msg {
			connections, err := storage.GetAllConnections()
			if err != nil || len(connections) == 0 {
				logger.Debug("No database connections for LSP", nil)
				return nil
			}

			if err := config.SaveSQLSConfig(); err != nil {
				logger.Debug("Failed to save SQLS config", map[string]any{"error": err.Error()})
				return nil
			}

			configPath, err := config.GetSQLSConfigPath()
			if err != nil {
				logger.Debug("Failed to get SQLS config path", map[string]any{"error": err.Error()})
				return nil
			}

			lspClient := lsp.NewClientWithConfig(configPath)
			if err := lspClient.Start(); err != nil {
				logger.Debug("Failed to start LSP client", map[string]any{"error": err.Error()})
				return nil
			}

			capabilities := map[string]interface{}{
				"textDocument": map[string]interface{}{
					"completion": map[string]interface{}{
						"completionItem": map[string]interface{}{
							"snippetSupport": false,
						},
					},
				},
			}

			_, err = lspClient.Initialize(m.documentURI, capabilities)
			if err != nil {
				logger.Debug("Failed to initialize LSP server", map[string]any{"error": err.Error()})
				lspClient.Stop()
				return nil
			}

			if err := lspClient.Initialized(); err != nil {
				logger.Debug("Failed to send initialized notification", map[string]any{"error": err.Error()})
				lspClient.Stop()
				return nil
			}

			text := m.GetQuery()
			if err := lspClient.DidOpen(m.documentURI, "sql", text); err != nil {
				logger.Debug("Failed to send didOpen notification", map[string]any{"error": err.Error()})
				lspClient.Stop()
				return nil
			}

			logger.Debug("LSP initialized for completion", nil)
			return LSPInitializedMsg{
				Success: true,
				Client:  lspClient,
				DocURI:  m.documentURI,
			}
		}
	}

	// LSP is already initialized, do completion
	return func() tea.Msg {
		line, character := m.getCursorPosition()
		logger.Debug("Requesting completion", map[string]any{
			"uri":       m.documentURI,
			"line":      line,
			"character": character,
		})

		response, err := m.lspClient.Completion(m.documentURI, line, character)
		if err != nil {
			logger.Debug("Failed to get completion", map[string]any{"error": err.Error()})
			return nil
		}

		logger.Debug("Received completion response", map[string]any{"response": response})

		// Parse completion items from response
		if items, ok := response.Result.(map[string]interface{}); ok {
			if itemList, ok := items["items"].([]interface{}); ok {
				logger.Debug("Found completion items", map[string]any{"count": len(itemList)})
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
					logger.Debug("Returning completion items", map[string]any{"count": len(completionItems)})
					return LSPCompletionMsg{Items: completionItems}
				} else {
					logger.Debug("No completion items found", nil)
				}
			} else {
				logger.Debug("No 'items' field in completion response", map[string]any{"result": items})
			}
		} else {
			logger.Debug("Unexpected completion response format", map[string]any{"result": response.Result})
		}

		return nil
	}
}

// LSPCompletionMsg is sent when completion items are received
type LSPCompletionMsg struct {
	Items []completion.CompletionItem
}

// LSPInitializedMsg is sent when LSP initialization completes
type LSPInitializedMsg struct {
	Success bool
	Error   string
	Client  *lsp.Client
	DocURI  string
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case LSPInitializedMsg:
		// Handle LSP initialization completion
		logger.Debug("LSPInitializedMsg received", map[string]any{
			"success":   msg.Success,
			"error":     msg.Error,
			"hasClient": msg.Client != nil,
		})
		m.lspInitStarted = false // Reset the started flag
		if msg.Success && msg.Client != nil {
			m.lspClient = msg.Client
			m.lspInitialized = true
			m.documentURI = msg.DocURI
			logger.Debug("LSP client stored in model", map[string]any{
				"initialized": true,
				"documentURI": msg.DocURI,
			})
		} else {
			logger.Debug("LSP initialization failed", map[string]any{"error": msg.Error})
		}
		return m, nil
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
					m.syntaxEditor.Focus()
					m.vimMode = VimNormal
				} else {
					// Switch from editor to results
					m.syntaxEditor.Blur()
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

		// Handle completion UI if visible
		if m.completionUI.Visible() {
			var compCmd tea.Cmd
			m.completionUI, compCmd = m.completionUI.Update(msg)
			cmds = append(cmds, compCmd)

			if m.completionUI.Selected() {
				// Insert selected completion item
				selectedItem := m.completionUI.SelectedItem()
				if selectedItem.InsertText != "" {
					// Insert the completion text into the syntax editor
					m.syntaxEditor.InsertText(selectedItem.InsertText)

					// Send LSP didChange notification
					if m.lspInitialized {
						newText := m.syntaxEditor.Value()
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

		// Non-vim mode: pass directly to syntax editor
		oldText := m.syntaxEditor.Value()
		m.syntaxEditor, cmd = m.syntaxEditor.Update(msg)
		cmds = append(cmds, cmd)

		// Send LSP didChange notification if text changed
		newText := m.syntaxEditor.Value()
		if oldText != newText && m.lspInitialized {
			m.documentVersion++
			go func() {
				if err := m.lspClient.DidChange(m.documentURI, newText, m.documentVersion); err != nil {
					logger.Debug("Failed to send didChange notification", map[string]any{"error": err.Error()})
				}
			}()
		}

		return m, tea.Batch(cmds...)
	default:
		return m, nil
	}
}

// handleVimInput processes input based on current vim mode
func (m Model) handleVimInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
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
		oldText := m.syntaxEditor.Value()
		m.syntaxEditor, cmd = m.syntaxEditor.Update(msg)
		cmds = append(cmds, cmd)

		// Send LSP didChange notification if text changed
		newText := m.syntaxEditor.Value()
		if oldText != newText && m.lspInitialized {
			m.documentVersion++
			go func() {
				if err := m.lspClient.DidChange(m.documentURI, newText, m.documentVersion); err != nil {
					logger.Debug("Failed to send didChange notification", map[string]any{"error": err.Error()})
				}
			}()
		}
		return m, tea.Batch(cmds...)
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
		// For simplicity, 'd' alone does nothing - would need 'dd' detection
		return m, nil

	// Undo (if supported by editor)
	case "u":
		// Editor doesn't have built-in undo, but try ctrl+z
		m.syntaxEditor, _ = m.syntaxEditor.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
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
