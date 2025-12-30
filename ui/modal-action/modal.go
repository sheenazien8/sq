package modalaction

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

// Action represents the type of action selected
type Action int

const (
	ActionNone Action = iota
	ActionDeleteRow
	ActionSetNull
	ActionSetEmpty
	ActionEditCell
	ActionCopyCell
	ActionCopyJSON
	ActionCopySQL
)

// Model wraps the generic modal with action content
type Model struct {
	modal   modal.Model
	content *ActionContent
}

// New creates a new action modal
func New() Model {
	content := NewActionContent()
	m := modal.New("Cell Actions", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal with the given cell and row context
func (m *Model) Show(cellValue string, rowData []string, columnNames []string, selectedCol int, tableName string) {
	m.content.SetContext(cellValue, rowData, columnNames, selectedCol, tableName)
	m.modal.Show()
}

// Hide hides the modal
func (m *Model) Hide() {
	m.modal.Hide()
}

// Visible returns whether the modal is visible
func (m Model) Visible() bool {
	return m.modal.Visible()
}

// SetSize sets the terminal size for centering
func (m *Model) SetSize(width, height int) {
	m.modal.SetSize(width, height)
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.modal, cmd = m.modal.Update(msg)
	return m, cmd
}

// View renders the modal
func (m Model) View() string {
	return m.modal.View()
}

// Result returns the modal result
func (m Model) Result() modal.Result {
	return m.modal.Result()
}

// SelectedAction returns the action that was selected
func (m Model) SelectedAction() Action {
	return m.content.selectedAction
}

// GetCellValue returns the cell value for the selected action
func (m Model) GetCellValue() string {
	return m.content.cellValue
}

// GetRowData returns the row data for the selected action
func (m Model) GetRowData() []string {
	return m.content.rowData
}

// GetColumnNames returns the column names
func (m Model) GetColumnNames() []string {
	return m.content.columnNames
}

// GetSelectedColumn returns the selected column index
func (m Model) GetSelectedColumn() int {
	return m.content.selectedCol
}

// GetTableName returns the table name
func (m Model) GetTableName() string {
	return m.content.tableName
}

// GetActionData returns formatted data for the selected action
func (m Model) GetActionData(action Action) string {
	return m.content.GetActionData(action)
}

// ActionContent implements Content for action selection
type ActionContent struct {
	actions []ActionItem

	selectedIndex  int
	selectedAction Action
	confirmed      bool

	// Context data
	cellValue   string
	rowData     []string
	columnNames []string
	selectedCol int
	tableName   string

	width  int
	closed bool
}

// ActionItem represents an action with description
type ActionItem struct {
	Action      Action
	Label       string
	Description string
	Shortcut    string
}

// NewActionContent creates a new action content
func NewActionContent() *ActionContent {
	return &ActionContent{
		actions: []ActionItem{
			{ActionDeleteRow, "Delete Row", "Delete this entire row/record", "d"},
			{ActionSetNull, "Set NULL", "Set this cell value to NULL", "n"},
			{ActionSetEmpty, "Set Empty", "Set this cell value to empty string", "e"},
			{ActionEditCell, "Edit Cell", "Edit this cell value", "i"},
			{ActionCopyCell, "Copy Cell", "Copy cell value to clipboard", "c"},
			{ActionCopyJSON, "Copy as JSON", "Copy row data as JSON", "j"},
			{ActionCopySQL, "Copy as SQL", "Copy row data as SQL syntax", "s"},
		},
		selectedIndex:  4, // Default to copy cell
		selectedAction: ActionNone,
		closed:         false,
	}
}

// SetContext sets the cell and row context for the actions
func (a *ActionContent) SetContext(cellValue string, rowData []string, columnNames []string, selectedCol int, tableName string) {
	a.cellValue = cellValue
	a.rowData = make([]string, len(rowData))
	copy(a.rowData, rowData)
	a.columnNames = make([]string, len(columnNames))
	copy(a.columnNames, columnNames)
	a.selectedCol = selectedCol
	a.tableName = tableName
	a.selectedIndex = 4 // Reset to copy cell
	a.selectedAction = ActionNone
	a.confirmed = false
	a.closed = false
}

// Update handles input
func (a *ActionContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if a.selectedIndex > 0 {
				a.selectedIndex--
			}
		case "down", "j":
			if a.selectedIndex < len(a.actions)-1 {
				a.selectedIndex++
			}
		case "enter":
			a.selectedAction = a.actions[a.selectedIndex].Action
			a.closed = true
			return a, nil
		case "esc":
			a.selectedAction = ActionNone
			a.closed = true
			return a, nil
		default:
			// Check for shortcut keys
			for i, action := range a.actions {
				if action.Shortcut == msg.String() {
					a.selectedIndex = i
					a.selectedAction = action.Action
					a.closed = true
					return a, nil
				}
			}
		}
	}
	return a, nil
}

// View renders the content
func (a *ActionContent) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	t := theme.Current

	var lines []string

	// Context info - left aligned within available width
	contextStyle := t.StatusBar.Copy().Padding(0, 1)
	contextInfo := fmt.Sprintf("Table: %s | Cell: %s", a.tableName, truncateCell(a.cellValue, 30))
	contextLine := contextStyle.Width(a.width).Align(lipgloss.Left).Render(contextInfo)
	lines = append(lines, contextLine)

	// Separator
	separatorLine := strings.Repeat(" ", a.width)
	lines = append(lines, separatorLine)

	// Actions list - left aligned
	for i, action := range a.actions {
		var style lipgloss.Style
		if i == a.selectedIndex {
			style = t.TableSelected.Copy()
		} else {
			style = t.TableCell.Copy()
		}

		shortcutStyle := lipgloss.NewStyle().Foreground(t.Colors.Primary).Bold(true)
		labelStyle := lipgloss.NewStyle().Bold(true)

		shortcut := shortcutStyle.Render(fmt.Sprintf("[%s]", action.Shortcut))
		label := labelStyle.Render(action.Label)
		desc := lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Render(action.Description)

		line := fmt.Sprintf(" %s %s - %s", shortcut, label, desc)
		// Ensure the line fills the width and is left-aligned
		actionLine := style.Width(a.width).Align(lipgloss.Left).Render(line)
		lines = append(lines, actionLine)
	}

	// Help text - left aligned
	helpStyle := lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Padding(1, 0, 0, 0)
	help := helpStyle.Width(a.width).Align(lipgloss.Left).Render("↑↓/j/k: navigate | Enter: select | Esc: cancel | [keys]: quick select")
	lines = append(lines, help)

	return strings.Join(lines, "\n")
}

// Result returns the content's result
func (a *ActionContent) Result() modal.Result {
	if a.selectedAction != ActionNone {
		return modal.ResultSubmit
	}
	return modal.ResultCancel
}

// ShouldClose returns true if the modal should close
func (a *ActionContent) ShouldClose() bool {
	return a.closed
}

// SetWidth sets the content width
func (a *ActionContent) SetWidth(width int) {
	a.width = width
}

// Helper function to truncate cell value for display
func truncateCell(cell string, maxLen int) string {
	if len(cell) <= maxLen {
		return cell
	}
	if maxLen > 3 {
		return cell[:maxLen-3] + "..."
	}
	return cell[:maxLen]
}

// GetActionData returns formatted data for the selected action
func (a *ActionContent) GetActionData(action Action) string {
	switch action {
	case ActionCopyCell:
		return a.cellValue
	case ActionCopyJSON:
		return a.getRowAsJSON()
	case ActionCopySQL:
		return a.getRowAsSQL()
	default:
		return ""
	}
}

// getRowAsJSON returns the row data as JSON
func (a *ActionContent) getRowAsJSON() string {
	if len(a.rowData) == 0 || len(a.columnNames) == 0 {
		return "{}"
	}

	rowMap := make(map[string]interface{})
	minLen := len(a.rowData)
	if len(a.columnNames) < minLen {
		minLen = len(a.columnNames)
	}

	for i := 0; i < minLen; i++ {
		rowMap[a.columnNames[i]] = a.rowData[i]
	}

	jsonBytes, err := json.MarshalIndent(rowMap, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\": \"Failed to marshal JSON: %v\"}", err)
	}
	return string(jsonBytes)
}

// getRowAsSQL returns the row data as SQL INSERT syntax
func (a *ActionContent) getRowAsSQL() string {
	if len(a.rowData) == 0 || len(a.columnNames) == 0 || a.tableName == "" {
		return "-- No data available"
	}

	minLen := len(a.rowData)
	if len(a.columnNames) < minLen {
		minLen = len(a.columnNames)
	}

	var columns []string
	var values []string

	for i := 0; i < minLen; i++ {
		// Use double quotes for identifiers (PostgreSQL/SQL standard)
		columns = append(columns, fmt.Sprintf("\"%s\"", a.columnNames[i]))
		// Escape single quotes in the value
		escapedValue := strings.ReplaceAll(a.rowData[i], "'", "''")
		values = append(values, fmt.Sprintf("'%s'", escapedValue))
	}

	return fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s);",
		a.tableName,
		strings.Join(columns, ", "),
		strings.Join(values, ", "))
}
