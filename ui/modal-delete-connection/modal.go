package modaldeleteconnection

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

type FocusButton int

const (
	FocusDeleteButton FocusButton = iota
	FocusCancelButton
)

// Content implements modal.Content for deleting a connection
type Content struct {
	connectionID   int64
	connectionName string
	focusButton    FocusButton
	result         modal.Result
	closed         bool
	width          int
}

// NewContent creates a new delete connection content
func NewContent() *Content {
	return &Content{
		focusButton: FocusCancelButton, // Default to Cancel for safety
		result:      modal.ResultNone,
		closed:      false,
	}
}

// LoadConnection loads a connection for deletion
func (c *Content) LoadConnection(id int64, name string) {
	c.connectionID = id
	c.connectionName = name
	c.focusButton = FocusCancelButton // Default to Cancel for safety
	c.result = modal.ResultNone
	c.closed = false
}

func (c *Content) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.result = modal.ResultCancel
			c.closed = true
			return c, nil

		case "tab", "left", "h", "right", "l":
			// Toggle between buttons
			if c.focusButton == FocusDeleteButton {
				c.focusButton = FocusCancelButton
			} else {
				c.focusButton = FocusDeleteButton
			}

		case "enter", "y":
			if c.focusButton == FocusDeleteButton {
				logger.Info("Connection delete confirmed", map[string]any{
					"connectionID": c.connectionID,
					"name":         c.connectionName,
				})
				c.result = modal.ResultSubmit
				c.closed = true
			} else {
				logger.Debug("Connection delete cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
			}

		case "n":
			logger.Debug("Connection delete cancelled", nil)
			c.result = modal.ResultCancel
			c.closed = true
		}
	}

	return c, nil
}

func (c *Content) View() string {
	t := theme.Current

	messageStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Align(lipgloss.Center).
		Padding(0, 0, 1, 0)

	warningStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Align(lipgloss.Center).
		Bold(true).
		Padding(0, 0, 1, 0)

	activeButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 2).
		Bold(true)

	inactiveButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Background(t.Colors.SelectionBg).
		Padding(0, 2)

	message := messageStyle.Render("Are you sure you want to delete this connection?")
	connectionName := warningStyle.Render("\"" + c.connectionName + "\"")
	warning := messageStyle.Render("This action cannot be undone.")

	var deleteButton, cancelButton string
	if c.focusButton == FocusDeleteButton {
		deleteButton = activeButtonStyle.Render("[ Delete ]")
	} else {
		deleteButton = inactiveButtonStyle.Render("  Delete  ")
	}
	if c.focusButton == FocusCancelButton {
		cancelButton = activeButtonStyle.Render("[ Cancel ]")
	} else {
		cancelButton = inactiveButtonStyle.Render("  Cancel  ")
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, deleteButton, "   ", cancelButton)

	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Align(lipgloss.Center).
		Padding(1, 0, 0, 0)
	help := helpStyle.Render("Tab/←→: navigate | Enter/y: delete | Esc/n: cancel")

	contentStyle := lipgloss.NewStyle().Padding(0, 0)

	content := []string{
		message,
		connectionName,
		warning,
		buttonRow,
		help,
	}

	return contentStyle.Render(lipgloss.JoinVertical(
		lipgloss.Center,
		content...,
	))
}

func (c *Content) Result() modal.Result {
	return c.result
}

func (c *Content) ShouldClose() bool {
	return c.closed
}

func (c *Content) SetWidth(width int) {
	c.width = width
}

// Model wraps the generic modal with delete connection content
type Model struct {
	modal   modal.Model
	content *Content
}

// New creates a new delete connection modal
func New() Model {
	content := NewContent()
	m := modal.New("Delete Connection", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal for confirmation
func (m *Model) Show(id int64, name string) {
	logger.Debug("Delete connection modal opened", map[string]any{
		"connectionID": id,
		"name":         name,
	})
	m.content.LoadConnection(id, name)
	m.modal.Show()
}

// Hide hides the modal
func (m *Model) Hide() {
	m.modal.Hide()
}

// Visible returns whether the modal is visible
func (m *Model) Visible() bool {
	return m.modal.Visible()
}

// SetSize sets the terminal size for centering
func (m *Model) SetSize(width, height int) {
	m.modal.SetSize(width, height)
	m.content.SetWidth(60)
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

// GetConnectionID returns the connection ID
func (m Model) GetConnectionID() int64 {
	return m.content.connectionID
}
