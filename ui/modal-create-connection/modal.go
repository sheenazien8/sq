package modalcreateconnection

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/ui/modal"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

type FocusField int

const (
	FocusDriverInput FocusField = iota
	FocusUrlInput
	FocusSubmitButton
	FocusCancelButton
)

// Content implements modal.Content for creating a new connection
type Content struct {
	drivers     []string
	driverIndex int
	urlInput    textinput.Model

	focusField FocusField

	result modal.Result
	closed bool
	width  int
}

// NewContent creates a new create connection content
func NewContent() *Content {
	ti := textinput.New()
	ti.Placeholder = "mysql://user:password@localhost:3306/dbname"
	ti.CharLimit = 256
	ti.Width = 50

	return &Content{
		drivers:     []string{"mysql", "postgres", "sqlite"},
		driverIndex: 0,
		urlInput:    ti,
		focusField:  FocusDriverInput,
		result:      modal.ResultNone,
		closed:      false,
	}
}

func (c *Content) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.result = modal.ResultCancel
			c.closed = true
			return c, nil

		case "tab", "down", "j":
			// Cycle forward through fields
			c.focusField = (c.focusField + 1) % 4
			c.updateFocus()

		case "shift+tab", "up", "k":
			// Cycle backward through fields
			c.focusField = (c.focusField - 1 + 4) % 4
			c.updateFocus()

		case "left", "h":
			if c.focusField == FocusDriverInput {
				c.driverIndex = (c.driverIndex - 1 + len(c.drivers)) % len(c.drivers)
			} else if c.focusField == FocusSubmitButton {
				c.focusField = FocusCancelButton
			} else if c.focusField == FocusCancelButton {
				c.focusField = FocusSubmitButton
			}

		case "right", "l":
			if c.focusField == FocusDriverInput {
				c.driverIndex = (c.driverIndex + 1) % len(c.drivers)
			} else if c.focusField == FocusSubmitButton {
				c.focusField = FocusCancelButton
			} else if c.focusField == FocusCancelButton {
				c.focusField = FocusSubmitButton
			}

		case "enter":
			if c.focusField == FocusSubmitButton {
				c.result = modal.ResultSubmit
				c.closed = true
			} else if c.focusField == FocusCancelButton {
				c.result = modal.ResultCancel
				c.closed = true
			}

		default:
			// Handle text input when focused on URL field
			if c.focusField == FocusUrlInput {
				c.urlInput, cmd = c.urlInput.Update(msg)
				return c, cmd
			}
		}
	}

	return c, nil
}

func (c *Content) updateFocus() {
	if c.focusField == FocusUrlInput {
		c.urlInput.Focus()
	} else {
		c.urlInput.Blur()
	}
}

func (c *Content) View() string {
	t := theme.Current

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Bold(true).
		Width(10)

	focusedStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 1)

	unfocusedStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Padding(0, 1)

	activeButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 2).
		Bold(true)

	inactiveButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Background(t.Colors.SelectionBg).
		Padding(0, 2)

	// Driver selector
	var driverDisplay string
	if c.focusField == FocusDriverInput {
		driverDisplay = focusedStyle.Render("< " + c.drivers[c.driverIndex] + " >")
	} else {
		driverDisplay = unfocusedStyle.Render("  " + c.drivers[c.driverIndex] + "  ")
	}
	driverRow := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render("Driver:"),
		driverDisplay,
	)

	// URL input
	urlLabel := labelStyle.Render("URL:")
	var urlDisplay string
	if c.focusField == FocusUrlInput {
		c.urlInput.TextStyle = lipgloss.NewStyle().Foreground(t.Colors.Foreground)
		c.urlInput.PromptStyle = lipgloss.NewStyle().Foreground(t.Colors.Primary)
	} else {
		c.urlInput.TextStyle = lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
		c.urlInput.PromptStyle = lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
	}
	urlDisplay = c.urlInput.View()
	urlRow := lipgloss.JoinHorizontal(lipgloss.Center, urlLabel, urlDisplay)

	// Buttons
	var submitButton, cancelButton string
	if c.focusField == FocusSubmitButton {
		submitButton = activeButtonStyle.Render("[ Submit ]")
	} else {
		submitButton = inactiveButtonStyle.Render("  Submit  ")
	}
	if c.focusField == FocusCancelButton {
		cancelButton = activeButtonStyle.Render("[ Cancel ]")
	} else {
		cancelButton = inactiveButtonStyle.Render("  Cancel  ")
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, submitButton, "   ", cancelButton)
	buttonRowCentered := lipgloss.NewStyle().Width(50).Align(lipgloss.Center).Render(buttonRow)

	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Align(lipgloss.Center).
		Padding(1, 0, 0, 0)
	help := helpStyle.Render("Tab/↑↓: navigate | ←→: select | Enter: confirm | Esc: cancel")

	contentStyle := lipgloss.NewStyle().
		Padding(1, 0)

	return contentStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		driverRow,
		"",
		urlRow,
		"",
		buttonRowCentered,
		help,
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
	if width > 20 {
		c.urlInput.Width = width - 15
	}
}

// GetDriver returns the selected driver
func (c *Content) GetDriver() string {
	return c.drivers[c.driverIndex]
}

// GetURL returns the entered URL
func (c *Content) GetURL() string {
	return c.urlInput.Value()
}

// Reset resets the content to initial state
func (c *Content) Reset() {
	c.driverIndex = 0
	c.urlInput.SetValue("")
	c.focusField = FocusDriverInput
	c.result = modal.ResultNone
	c.closed = false
	c.urlInput.Blur()
}

// Model wraps the generic modal with create connection content
type Model struct {
	modal   modal.Model
	content *Content
}

// New creates a new create connection modal
func New() Model {
	content := NewContent()
	m := modal.New("Create New Connection", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal
func (m *Model) Show() {
	m.content.Reset()
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

// GetDriver returns the selected driver
func (m Model) GetDriver() string {
	return m.content.GetDriver()
}

// GetURL returns the entered URL
func (m Model) GetURL() string {
	return m.content.GetURL()
}
