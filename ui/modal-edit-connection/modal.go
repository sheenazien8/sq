package modaleditconnection

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/drivers"
	"github.com/sheenazien8/sq/logger"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

type FocusField int

const (
	FocusNameInput FocusField = iota
	FocusHostInput
	FocusPortInput
	FocusUsernameInput
	FocusPasswordInput
	FocusDatabaseInput
	FocusUriInput
	FocusSubmitButton
	FocusCancelButton
)

// ConnectionFields holds all connection input fields
type ConnectionFields struct {
	nameInput     textinput.Model
	hostInput     textinput.Model
	portInput     textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model
	databaseInput textinput.Model
	uriInput      textinput.Model // For MongoDB Atlas direct URL input
}

// Content implements modal.Content for editing a connection
type Content struct {
	connectionID int64 // ID of the connection being edited
	driverType   string
	focusField   FocusField
	result       modal.Result
	closed       bool
	width        int
	fields       ConnectionFields
	errorMsg     string
}

// NewContent creates a new edit connection content
func NewContent() *Content {
	fields := createConnectionFields()
	return &Content{
		focusField: FocusNameInput,
		result:     modal.ResultNone,
		closed:     false,
		fields:     fields,
	}
}

func createConnectionFields() ConnectionFields {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g., My Database"
	nameInput.CharLimit = 256
	nameInput.Width = 40

	hostInput := textinput.New()
	hostInput.Placeholder = "localhost"
	hostInput.CharLimit = 256
	hostInput.Width = 40

	portInput := textinput.New()
	portInput.CharLimit = 5
	portInput.Width = 40

	usernameInput := textinput.New()
	usernameInput.Placeholder = "root"
	usernameInput.CharLimit = 256
	usernameInput.Width = 40

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.CharLimit = 256
	passwordInput.Width = 40
	passwordInput.EchoMode = textinput.EchoPassword

	databaseInput := textinput.New()
	databaseInput.Placeholder = "database name"
	databaseInput.CharLimit = 256
	databaseInput.Width = 40

	uriInput := textinput.New()
	uriInput.Placeholder = "mongodb+srv://user:pass@cluster.mongodb.net/database?retryWrites=true&w=majority"
	uriInput.CharLimit = 512
	uriInput.Width = 40

	return ConnectionFields{
		nameInput:     nameInput,
		hostInput:     hostInput,
		portInput:     portInput,
		usernameInput: usernameInput,
		passwordInput: passwordInput,
		databaseInput: databaseInput,
		uriInput:      uriInput,
	}
}

// LoadConnection loads a connection's data into the form
func (c *Content) LoadConnection(id int64, driverType, name, host, port, username, password, database, uri string) {
	c.connectionID = id
	c.driverType = driverType
	c.fields.nameInput.SetValue(name)
	c.fields.hostInput.SetValue(host)
	c.fields.portInput.SetValue(port)
	c.fields.usernameInput.SetValue(username)
	c.fields.passwordInput.SetValue(password)
	c.fields.databaseInput.SetValue(database)
	c.fields.uriInput.SetValue(uri)
	c.focusField = FocusNameInput
	c.errorMsg = ""
	c.closed = false
	c.result = modal.ResultNone
	c.updateFocus()
}

// validate checks if the connection fields are valid
func (c *Content) validate() string {
	if name := c.fields.nameInput.Value(); name == "" {
		return "Connection name is required"
	}

	// SQLite only needs name and file path
	if c.driverType == drivers.DriverTypeSQLite {
		if filePath := c.fields.databaseInput.Value(); filePath == "" {
			return "File path is required"
		}
		return ""
	}

	// MongoDB Atlas only needs the connection URI
	if c.driverType == drivers.DriverTypeMongoDBAtlas {
		if uri := c.fields.uriInput.Value(); uri == "" {
			return "MongoDB Atlas connection URI is required"
		}
		return ""
	}

	// MySQL, PostgreSQL, and MongoDB need host, port, username, and database
	if host := c.fields.hostInput.Value(); host == "" {
		return "Host is required"
	}

	if portStr := c.fields.portInput.Value(); portStr == "" {
		return "Port is required"
	} else if port, err := strconv.Atoi(portStr); err != nil {
		return "Port must be a valid number"
	} else if port < 1 || port > 65535 {
		return "Port must be between 1 and 65535"
	}

	if username := c.fields.usernameInput.Value(); username == "" {
		return "Username is required"
	}

	if database := c.fields.databaseInput.Value(); database == "" {
		return "Database name is required"
	}

	return ""
}

// getDefaultPort returns the default port for the current driver
func (c *Content) getDefaultPort() string {
	switch c.driverType {
	case drivers.DriverTypeMySQL:
		return "3306"
	case drivers.DriverTypePostgreSQL:
		return "5432"
	case drivers.DriverTypeMongoDB:
		return "27017"
	default:
		return "5432"
	}
}

func (c *Content) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle text input fields
		if c.focusField >= FocusNameInput && c.focusField <= FocusDatabaseInput {
			switch msg.String() {
			case "esc":
				logger.Debug("Edit connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				c.focusField = (c.focusField + 1)
				if c.focusField > FocusDatabaseInput {
					if c.driverType == drivers.DriverTypeMongoDBAtlas {
						c.focusField = FocusUriInput
					} else {
						c.focusField = FocusSubmitButton
					}
				}
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				if c.focusField == FocusNameInput {
					c.focusField = FocusNameInput
				} else {
					c.focusField = (c.focusField - 1)
				}
				c.updateFocus()
				return c, nil
			default:
				c.handleInputUpdate(msg, c.focusField)
				return c, nil
			}
		}

		// Handle MongoDB Atlas URI input
		if c.focusField == FocusUriInput && c.driverType == drivers.DriverTypeMongoDBAtlas {
			switch msg.String() {
			case "esc":
				logger.Debug("Edit connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				c.focusField = FocusSubmitButton
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				c.focusField = FocusDatabaseInput
				c.updateFocus()
				return c, nil
			default:
				c.fields.uriInput, cmd = c.fields.uriInput.Update(msg)
				return c, cmd
			}
		}

		switch msg.String() {
		case "esc":
			logger.Debug("Edit connection cancelled", nil)
			c.result = modal.ResultCancel
			c.closed = true
			return c, nil

		case "tab", "down", "j":
			if c.focusField < FocusCancelButton {
				c.focusField = (c.focusField + 1) % (FocusCancelButton + 1)
			}
			c.updateFocus()

		case "shift+tab", "up", "k":
			if c.focusField > FocusNameInput {
				c.focusField = (c.focusField - 1)
			} else {
				c.focusField = FocusCancelButton
			}
			c.updateFocus()

		case "left", "h":
			if c.focusField == FocusSubmitButton {
				c.focusField = FocusCancelButton
			} else if c.focusField == FocusCancelButton {
				c.focusField = FocusSubmitButton
			}

		case "right", "l":
			if c.focusField == FocusSubmitButton {
				c.focusField = FocusCancelButton
			} else if c.focusField == FocusCancelButton {
				c.focusField = FocusSubmitButton
			}

		case "enter":
			if c.focusField == FocusSubmitButton {
				if errMsg := c.validate(); errMsg != "" {
					c.errorMsg = errMsg
					return c, nil
				}
				c.errorMsg = "" // Clear any previous error

				logger.Info("Connection update submitted", map[string]any{
					"id":     c.connectionID,
					"driver": c.driverType,
					"name":   c.fields.nameInput.Value(),
				})
				c.result = modal.ResultSubmit
				c.closed = true
			} else if c.focusField == FocusCancelButton {
				c.errorMsg = "" // Clear error on cancel
				logger.Debug("Edit connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
			}
		}
	}

	return c, nil
}

// handleInputUpdate routes key input to the appropriate text input field
func (c *Content) handleInputUpdate(msg tea.KeyMsg, focusField FocusField) {
	switch focusField {
	case FocusNameInput:
		c.fields.nameInput, _ = c.fields.nameInput.Update(msg)
	case FocusHostInput:
		c.fields.hostInput, _ = c.fields.hostInput.Update(msg)
	case FocusPortInput:
		c.fields.portInput, _ = c.fields.portInput.Update(msg)
	case FocusUsernameInput:
		c.fields.usernameInput, _ = c.fields.usernameInput.Update(msg)
	case FocusPasswordInput:
		c.fields.passwordInput, _ = c.fields.passwordInput.Update(msg)
	case FocusDatabaseInput:
		c.fields.databaseInput, _ = c.fields.databaseInput.Update(msg)
	}
}

func (c *Content) updateFocus() {
	if c.focusField == FocusNameInput {
		c.fields.nameInput.Focus()
	} else {
		c.fields.nameInput.Blur()
	}

	if c.focusField == FocusHostInput {
		c.fields.hostInput.Focus()
	} else {
		c.fields.hostInput.Blur()
	}

	if c.focusField == FocusPortInput {
		c.fields.portInput.Focus()
	} else {
		c.fields.portInput.Blur()
	}

	if c.focusField == FocusUsernameInput {
		c.fields.usernameInput.Focus()
	} else {
		c.fields.usernameInput.Blur()
	}

	if c.focusField == FocusPasswordInput {
		c.fields.passwordInput.Focus()
	} else {
		c.fields.passwordInput.Blur()
	}

	if c.focusField == FocusDatabaseInput {
		c.fields.databaseInput.Focus()
	} else {
		c.fields.databaseInput.Blur()
	}

	if c.focusField == FocusUriInput {
		c.fields.uriInput.Focus()
	} else {
		c.fields.uriInput.Blur()
	}
}

func (c *Content) View() string {
	t := theme.Current

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Bold(true).
		Width(10)

	inputWidth := 40

	focusedInputStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(t.Colors.Primary).
		Padding(0, 1).
		Height(1)

	unfocusedInputStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(t.Colors.SelectionBg).
		Padding(0, 1).
		Height(1)

	activeButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 2).
		Bold(true)

	inactiveButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Background(t.Colors.SelectionBg).
		Padding(0, 2)

	// Helper function to render input field
	renderField := func(label string, input textinput.Model, focused bool) string {
		if focused {
			input.TextStyle = lipgloss.NewStyle().Foreground(t.Colors.Foreground)
			input.PromptStyle = lipgloss.NewStyle().Foreground(t.Colors.Primary)
		} else {
			input.TextStyle = lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
			input.PromptStyle = lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
		}

		inputView := input.View()

		var inputStyle lipgloss.Style
		if focused {
			inputStyle = focusedInputStyle
		} else {
			inputStyle = unfocusedInputStyle
		}

		inputContainer := inputStyle.Width(inputWidth).Render(inputView)

		return lipgloss.JoinHorizontal(lipgloss.Center,
			labelStyle.Render(label+":"),
			"  ",
			inputContainer,
		)
	}

	// Render form fields
	nameRow := renderField("Name", c.fields.nameInput, c.focusField == FocusNameInput)

	var hostRow, portRow, usernameRow, passwordRow, databaseRow, uriRow string

	if c.driverType == drivers.DriverTypeSQLite {
		databaseRow = renderField("Path", c.fields.databaseInput, c.focusField == FocusDatabaseInput)
	} else if c.driverType == drivers.DriverTypeMongoDBAtlas {
		uriRow = renderField("URI", c.fields.uriInput, c.focusField == FocusUriInput)
	} else {
		hostRow = renderField("Host", c.fields.hostInput, c.focusField == FocusHostInput)
		portRow = renderField("Port", c.fields.portInput, c.focusField == FocusPortInput)
		usernameRow = renderField("Username", c.fields.usernameInput, c.focusField == FocusUsernameInput)
		passwordRow = renderField("Password", c.fields.passwordInput, c.focusField == FocusPasswordInput)
		databaseRow = renderField("Database", c.fields.databaseInput, c.focusField == FocusDatabaseInput)
	}

	// Error message
	var errorRow string
	if c.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(t.Colors.Primary).
			Align(lipgloss.Center).
			Padding(0, 0, 1, 0)
		errorRow = errorStyle.Render("Error: " + c.errorMsg)
	}

	// Buttons
	var submitButton, cancelButton string
	if c.focusField == FocusSubmitButton {
		submitButton = activeButtonStyle.Render("[ Update ]")
	} else {
		submitButton = inactiveButtonStyle.Render("  Update  ")
	}
	if c.focusField == FocusCancelButton {
		cancelButton = activeButtonStyle.Render("[ Cancel ]")
	} else {
		cancelButton = inactiveButtonStyle.Render("  Cancel  ")
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, submitButton, "   ", cancelButton)

	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Align(lipgloss.Center).
		Padding(1, 0, 0, 0)
	help := helpStyle.Render("Tab/↑↓: navigate | Enter: update | Esc: cancel")

	contentStyle := lipgloss.NewStyle().Padding(0, 0)

	var content []string
	content = append(content, nameRow)

	if c.driverType == drivers.DriverTypeSQLite {
		content = append(content, databaseRow)
	} else if c.driverType == drivers.DriverTypeMongoDBAtlas {
		content = append(content, uriRow)
	} else {
		content = append(content, hostRow, portRow, usernameRow, passwordRow, databaseRow)
	}

	if errorRow != "" {
		content = append(content, errorRow)
	}
	content = append(content, buttonRow, help)

	return contentStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
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

// GetConnectionData returns the connection data from the form
func (c *Content) GetConnectionData() (name, driverType, host, port, username, password, database, uri string) {
	return c.fields.nameInput.Value(),
		c.driverType,
		c.fields.hostInput.Value(),
		c.fields.portInput.Value(),
		c.fields.usernameInput.Value(),
		c.fields.passwordInput.Value(),
		c.fields.databaseInput.Value(),
		c.fields.uriInput.Value()
}

// Model wraps the generic modal with edit connection content
type Model struct {
	modal   modal.Model
	content *Content
}

// New creates a new edit connection modal
func New() Model {
	content := NewContent()
	m := modal.New("Edit Connection", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal and loads connection data
func (m *Model) Show(id int64, driverType, name, host, port, username, password, database, uri string) {
	logger.Debug("Edit connection modal opened", map[string]any{
		"connectionID": id,
		"name":         name,
	})
	m.content.LoadConnection(id, driverType, name, host, port, username, password, database, uri)
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

// GetConnectionData returns the connection data from the form
func (m Model) GetConnectionData() (name, driverType, host, port, username, password, database, uri string) {
	return m.content.GetConnectionData()
}
