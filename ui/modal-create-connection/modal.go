package modalcreateconnection

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

type FocusField int

const (
	FocusNameInput FocusField = iota
	FocusDriverInput
	FocusUrlInput
	FocusSubmitButton
	FocusCancelButton
)

// Content implements modal.Content for creating a new connection
type Content struct {
	drivers     []string
	driverIndex int
	urlInput    textinput.Model
	nameInput   textinput.Model

	focusField FocusField

	result modal.Result
	closed bool
	width  int
}

// NewContent creates a new create connection content
func NewContent() *Content {
	nameInput := textinput.New()
	nameInput.Placeholder = "Your connection name"
	nameInput.CharLimit = 256
	nameInput.Width = 1000 // Large width to prevent internal wrapping

	urlInput := textinput.New()
	urlInput.Placeholder = "mysql://user:password@localhost:3306/dbname"
	urlInput.CharLimit = 256
	urlInput.Width = 1000 // Large width to prevent internal wrapping

	nameInput.Focus() // Focus name input by default

	return &Content{
		drivers:     []string{"mysql"},
		driverIndex: 0,
		urlInput:    urlInput,
		nameInput:   nameInput,
		focusField:  FocusNameInput,
		result:      modal.ResultNone,
		closed:      false,
	}
}

func (c *Content) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle text input first when focused on URL field
		// This allows typing h, l, j, k etc. in the text input
		if c.focusField == FocusUrlInput {
			switch msg.String() {
			case "esc":
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				// Allow navigation out of text input
				c.focusField = (c.focusField + 1) % 5
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				// Allow navigation out of text input
				c.focusField = (c.focusField - 1 + 5) % 5
				c.updateFocus()
				return c, nil
			default:
				// Pass all other keys to text input
				c.urlInput, cmd = c.urlInput.Update(msg)
				return c, cmd
			}
		}

		if c.focusField == FocusNameInput {
			switch msg.String() {
			case "esc":
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				// Allow navigation out of text input
				c.focusField = (c.focusField + 1) % 5
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				// Allow navigation out of text input
				c.focusField = (c.focusField - 1 + 5) % 5
				c.updateFocus()
				return c, nil
			default:
				// Pass all other keys to text input
				c.nameInput, cmd = c.nameInput.Update(msg)
				return c, cmd
			}
		}

		switch msg.String() {
		case "esc":
			logger.Debug("Create connection cancelled", nil)
			c.result = modal.ResultCancel
			c.closed = true
			return c, nil

		case "tab", "down", "j":
			// Cycle forward through fields
			c.focusField = (c.focusField + 1) % 5
			c.updateFocus()

		case "shift+tab", "up", "k":
			// Cycle backward through fields
			c.focusField = (c.focusField - 1 + 5) % 5
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
				logger.Info("Connection submitted", map[string]any{
					"driver": c.drivers[c.driverIndex],
					"url":    c.urlInput.Value(),
					"name":   c.nameInput.Value(),
				})
				c.result = modal.ResultSubmit
				c.closed = true
			} else if c.focusField == FocusCancelButton {
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
			}
		}
	}

	return c, nil
}

func (c *Content) updateFocus() {
	// Handle name input focus
	if c.focusField == FocusNameInput {
		c.nameInput.Focus()
	} else {
		c.nameInput.Blur()
	}

	// Handle URL input focus
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

	nameLabel := labelStyle.Render("Name:")
	var nameDisplay string
	if c.focusField == FocusNameInput {
		c.nameInput.TextStyle = lipgloss.NewStyle().Foreground(t.Colors.Foreground)
		c.nameInput.PromptStyle = lipgloss.NewStyle().Foreground(t.Colors.Primary)
	} else {
		c.nameInput.TextStyle = lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
		c.nameInput.PromptStyle = lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
	}
	nameDisplay = strings.ReplaceAll(c.nameInput.View(), "\n", " ")
	nameRow := lipgloss.JoinHorizontal(lipgloss.Center, nameLabel, nameDisplay)

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
	urlDisplay = strings.ReplaceAll(c.urlInput.View(), "\n", " ")
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
		nameRow,
		"",
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

func (c *Content) GetName() string {
	return c.nameInput.Value()
}

// Reset resets the content to initial state
func (c *Content) Reset() {
	c.driverIndex = 0
	c.nameInput.SetValue("")
	c.urlInput.SetValue("")
	c.focusField = FocusNameInput
	c.result = modal.ResultNone
	c.closed = false
	c.nameInput.Focus()
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
	logger.Debug("Create connection modal opened", nil)
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

// GetURL returns the entered URL
func (m Model) GetName() string {
	return m.content.GetName()
}
