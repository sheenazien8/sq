package modaleditcell

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

// Model wraps the generic modal with cell edit content
type Model struct {
	modal   modal.Model
	content *EditCellContent
}

// New creates a new cell edit modal
func New() Model {
	content := NewEditCellContent()
	m := modal.New("Edit Cell", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal with the current cell value
func (m *Model) Show(currentValue, columnName, tableName string) {
	m.content.SetValue(currentValue, columnName, tableName)
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

// GetNewValue returns the new value entered by the user
func (m Model) GetNewValue() string {
	return m.content.GetValue()
}

// Confirmed returns true if the user confirmed the edit
func (m Model) Confirmed() bool {
	return m.modal.Result() == modal.ResultSubmit
}

// EditCellContent implements Content for cell editing
type EditCellContent struct {
	columnName string
	tableName  string
	input      textinput.Model
	result     modal.Result
	closed     bool
	width      int
}

const maxInputWidth = 60

// NewEditCellContent creates a new edit cell content
func NewEditCellContent() *EditCellContent {
	ti := textinput.New()
	ti.Placeholder = "Enter new value..."
	ti.CharLimit = 1000
	ti.Width = maxInputWidth

	return &EditCellContent{
		input:  ti,
		result: modal.ResultNone,
		closed: false,
	}
}

// SetValue sets the current value and context
func (e *EditCellContent) SetValue(currentValue, columnName, tableName string) {
	e.columnName = columnName
	e.tableName = tableName
	e.input.SetValue(currentValue)
	e.input.Focus()
	e.result = modal.ResultNone
	e.closed = false
}

// GetValue returns the current input value
func (e *EditCellContent) GetValue() string {
	return strings.TrimSpace(e.input.Value())
}

// Update handles input
func (e *EditCellContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Confirm the edit
			e.result = modal.ResultSubmit
			e.closed = true
			return e, nil
		case "esc":
			// Cancel the edit
			e.result = modal.ResultCancel
			e.closed = true
			return e, nil
		default:
			// Pass other keys to the text input
			e.input, cmd = e.input.Update(msg)
		}
	}

	return e, cmd
}

// View renders the content
func (e *EditCellContent) View() string {
	if e.width == 0 {
		return "Loading..."
	}

	t := theme.Current

	var lines []string

	// Context info - left aligned
	contextStyle := t.StatusBar.Copy().Padding(0, 1)
	contextInfo := "Editing cell in table '" + e.tableName + "', column '" + e.columnName + "'"
	contextLine := contextStyle.Width(e.width).Align(lipgloss.Left).Render(contextInfo)
	lines = append(lines, contextLine)

	// Separator
	separatorLine := strings.Repeat(" ", e.width)
	lines = append(lines, separatorLine)

	// Input field with label - left aligned
	inputLabel := "New value:"
	labelStyle := t.TableCell.Copy().Bold(true)
	labelLine := labelStyle.Width(e.width).Align(lipgloss.Left).Render(inputLabel)
	lines = append(lines, labelLine)

	// Input field - left aligned
	inputStyle := t.TableCell.Copy().Padding(0, 1)
	inputDisplay := e.input.View()
	inputLine := inputStyle.Width(e.width).Align(lipgloss.Left).Render(inputDisplay)
	lines = append(lines, inputLine)

	// Help text - left aligned
	helpStyle := lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Padding(1, 0, 0, 0)
	help := helpStyle.Width(e.width).Align(lipgloss.Left).Render("Enter: Confirm | Esc: Cancel")
	lines = append(lines, help)

	return strings.Join(lines, "\n")
}

// Result returns the content's result
func (e *EditCellContent) Result() modal.Result {
	return e.result
}

// ShouldClose returns true if the modal should close
func (e *EditCellContent) ShouldClose() bool {
	return e.closed
}

// SetWidth sets the content width
func (e *EditCellContent) SetWidth(width int) {
	e.width = width
	e.input.Width = min(width-4, maxInputWidth) // Account for padding
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
