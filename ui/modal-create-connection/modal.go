package modalcreateconnection

import (
	"fmt"
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
	FocusDriverSelect FocusField = iota
	FocusNameInput
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

// Content implements modal.Content for creating a new connection
type Content struct {
	drivers            []string
	driverIndex        int // 0 = MySQL, 1 = PostgreSQL, 2 = SQLite, 3 = MongoDB, 4 = MongoDB Atlas
	focusField         FocusField
	result             modal.Result
	closed             bool
	width              int
	mysqlFields        ConnectionFields
	postgresFields     ConnectionFields
	sqliteFields       ConnectionFields
	mongodbFields      ConnectionFields
	mongodbAtlasFields ConnectionFields
	errorMsg           string
}

// NewContent creates a new create connection content
func NewContent() *Content {
	mysql := createConnectionFields()
	postgres := createConnectionFields()
	sqlite := createSQLiteConnectionFields()
	mongodb := createMongoDBConnectionFields()
	mongodbAtlas := createMongoDBAtlasConnectionFields()

	return &Content{
		drivers:            []string{"mysql", "postgresql", "sqlite", "mongodb", "mongodb-atlas"},
		driverIndex:        0,
		focusField:         FocusDriverSelect,
		result:             modal.ResultNone,
		closed:             false,
		mysqlFields:        mysql,
		postgresFields:     postgres,
		sqliteFields:       sqlite,
		mongodbFields:      mongodb,
		mongodbAtlasFields: mongodbAtlas,
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
	hostInput.SetValue("localhost")

	portInput := textinput.New()
	portInput.CharLimit = 5
	portInput.Width = 40
	portInput.SetValue("3306") // Default MySQL port

	usernameInput := textinput.New()
	usernameInput.Placeholder = "root"
	usernameInput.CharLimit = 256
	usernameInput.Width = 40
	usernameInput.SetValue("root")

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.CharLimit = 256
	passwordInput.Width = 40
	passwordInput.EchoMode = textinput.EchoPassword

	databaseInput := textinput.New()
	databaseInput.Placeholder = "database name"
	databaseInput.CharLimit = 256
	databaseInput.Width = 40

	return ConnectionFields{
		nameInput:     nameInput,
		hostInput:     hostInput,
		portInput:     portInput,
		usernameInput: usernameInput,
		passwordInput: passwordInput,
		databaseInput: databaseInput,
	}
}

func createSQLiteConnectionFields() ConnectionFields {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g., My SQLite DB"
	nameInput.CharLimit = 256
	nameInput.Width = 40

	// SQLite uses file path as "database input"
	databaseInput := textinput.New()
	databaseInput.Placeholder = "/path/to/database.db"
	databaseInput.CharLimit = 256
	databaseInput.Width = 40

	// Create dummy inputs for unused fields (host, port, username, password)
	hostInput := textinput.New()
	portInput := textinput.New()
	usernameInput := textinput.New()
	passwordInput := textinput.New()

	return ConnectionFields{
		nameInput:     nameInput,
		hostInput:     hostInput,
		portInput:     portInput,
		usernameInput: usernameInput,
		passwordInput: passwordInput,
		databaseInput: databaseInput,
	}
}

func createMongoDBConnectionFields() ConnectionFields {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g., My MongoDB"
	nameInput.CharLimit = 256
	nameInput.Width = 40

	hostInput := textinput.New()
	hostInput.Placeholder = "localhost"
	hostInput.CharLimit = 256
	hostInput.Width = 40
	hostInput.SetValue("localhost")

	portInput := textinput.New()
	portInput.CharLimit = 5
	portInput.Width = 40
	portInput.SetValue("27017") // Default MongoDB port

	usernameInput := textinput.New()
	usernameInput.Placeholder = "username"
	usernameInput.CharLimit = 256
	usernameInput.Width = 40

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.CharLimit = 256
	passwordInput.Width = 40
	passwordInput.EchoMode = textinput.EchoPassword

	// Database name for MongoDB
	databaseInput := textinput.New()
	databaseInput.Placeholder = "database name"
	databaseInput.CharLimit = 256
	databaseInput.Width = 40

	return ConnectionFields{
		nameInput:     nameInput,
		hostInput:     hostInput,
		portInput:     portInput,
		usernameInput: usernameInput,
		passwordInput: passwordInput,
		databaseInput: databaseInput,
	}
}

func createMongoDBAtlasConnectionFields() ConnectionFields {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g., CompassNFC"
	nameInput.CharLimit = 256
	nameInput.Width = 40

	// MongoDB Atlas URI input
	uriInput := textinput.New()
	uriInput.Placeholder = "mongodb+srv://user:pass@cluster.mongodb.net/database?retryWrites=true&w=majority"
	uriInput.CharLimit = 512
	uriInput.Width = 40

	// Dummy inputs (not used for Atlas URI mode)
	hostInput := textinput.New()
	portInput := textinput.New()
	usernameInput := textinput.New()
	passwordInput := textinput.New()
	databaseInput := textinput.New()

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

// getCurrentFields returns the current driver's fields
func (c *Content) getCurrentFields() *ConnectionFields {
	if c.driverIndex == 0 {
		return &c.mysqlFields
	} else if c.driverIndex == 1 {
		return &c.postgresFields
	} else if c.driverIndex == 2 {
		return &c.sqliteFields
	} else if c.driverIndex == 3 {
		return &c.mongodbFields
	}
	return &c.mongodbAtlasFields
}

// createDriver creates a driver instance for the current driver
func (c *Content) createDriver() (drivers.Driver, error) {
	switch c.GetDriver() {
	case drivers.DriverTypeMySQL:
		return &drivers.MySQL{}, nil
	case drivers.DriverTypePostgreSQL:
		return &drivers.PostgreSQL{}, nil
	case drivers.DriverTypeSQLite:
		return &drivers.SQLite{}, nil
	case drivers.DriverTypeMongoDB, drivers.DriverTypeMongoDBAtlas:
		return &drivers.MongoDB{}, nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s", c.GetDriver())
	}
}

// validate checks if the connection fields are valid
func (c *Content) validate() string {
	fields := c.getCurrentFields()

	if name := fields.nameInput.Value(); name == "" {
		return "Connection name is required"
	}

	// SQLite only needs name and file path
	if c.GetDriver() == drivers.DriverTypeSQLite {
		if filePath := fields.databaseInput.Value(); filePath == "" {
			return "File path is required"
		}
		return ""
	}

	// MongoDB Atlas only needs the connection URI
	if c.GetDriver() == drivers.DriverTypeMongoDBAtlas {
		if uri := fields.uriInput.Value(); uri == "" {
			return "MongoDB Atlas connection URI is required"
		}
		return ""
	}

	// MySQL, PostgreSQL, and MongoDB need host, port, username, and database
	if host := fields.hostInput.Value(); host == "" {
		return "Host is required"
	}

	if portStr := fields.portInput.Value(); portStr == "" {
		return "Port is required"
	} else if port, err := strconv.Atoi(portStr); err != nil {
		return "Port must be a valid number"
	} else if port < 1 || port > 65535 {
		return "Port must be between 1 and 65535"
	}

	if username := fields.usernameInput.Value(); username == "" {
		return "Username is required"
	}

	if database := fields.databaseInput.Value(); database == "" {
		return "Database name is required"
	}

	return ""
}

// getDefaultPort returns the default port for the current driver
func (c *Content) getDefaultPort() string {
	if c.driverIndex == 0 {
		return "3306"
	} else if c.driverIndex == 1 {
		return "5432"
	} else if c.driverIndex == 3 {
		return "27017"
	} else if c.driverIndex == 4 {
		return "" // MongoDB Atlas doesn't use a port
	}
	return "5432"
}

// setDefaultPort sets the port to the default for the current driver
func (c *Content) setDefaultPort() {
	fields := c.getCurrentFields()
	fields.portInput.SetValue(c.getDefaultPort())
}

func (c *Content) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	var cmd tea.Cmd
	fields := c.getCurrentFields()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle text input fields for MySQL/PostgreSQL
		if c.focusField >= FocusHostInput && c.focusField <= FocusDatabaseInput && c.GetDriver() != drivers.DriverTypeSQLite {
			switch msg.String() {
			case "esc":
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				c.focusField = (c.focusField + 1)
				if c.focusField > FocusDatabaseInput {
					c.focusField = FocusSubmitButton
				}
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				if c.focusField == FocusHostInput {
					c.focusField = FocusNameInput
				} else {
					c.focusField = (c.focusField - 1)
				}
				c.updateFocus()
				return c, nil
			default:
				// Pass all other keys to text input
				fields.handleInputUpdate(msg, c.focusField)
				return c, nil
			}
		}

		// Handle text input field for SQLite (only database input for file path)
		if c.focusField == FocusDatabaseInput && c.GetDriver() == drivers.DriverTypeSQLite {
			switch msg.String() {
			case "esc":
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				c.focusField = FocusSubmitButton
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				c.focusField = FocusNameInput
				c.updateFocus()
				return c, nil
			default:
				// Pass all other keys to text input
				fields.handleInputUpdate(msg, c.focusField)
				return c, nil
			}
		}

		// Handle MongoDB Atlas URI input
		if c.focusField == FocusUriInput && c.GetDriver() == drivers.DriverTypeMongoDBAtlas {
			switch msg.String() {
			case "esc":
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab", "down":
				c.focusField = FocusSubmitButton
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				c.focusField = FocusNameInput
				c.updateFocus()
				return c, nil
			default:
				// Pass all other keys to text input
				fields.uriInput, cmd = fields.uriInput.Update(msg)
				return c, cmd
			}
		}

		if c.focusField == FocusDriverSelect {
			switch msg.String() {
			case "esc":
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
				return c, nil
			case "tab":
				c.focusField = FocusNameInput
				c.updateFocus()
				return c, nil
			case "shift+tab":
				c.focusField = FocusDriverSelect
				c.updateFocus()
				return c, nil
			case "k":
				c.driverIndex = (c.driverIndex - 1 + len(c.drivers)) % len(c.drivers)
				c.setDefaultPort()
				return c, nil
			case "j":
				c.driverIndex = (c.driverIndex + 1) % len(c.drivers)
				c.setDefaultPort()
				return c, nil
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
				// For SQLite, skip to database input (file path)
				// For MongoDB Atlas, go to URI input
				// For others, go to host input
				if c.GetDriver() == drivers.DriverTypeSQLite {
					c.focusField = FocusDatabaseInput
				} else if c.GetDriver() == drivers.DriverTypeMongoDBAtlas {
					c.focusField = FocusUriInput
				} else {
					c.focusField = FocusHostInput
				}
				c.updateFocus()
				return c, nil
			case "shift+tab", "up":
				c.focusField = FocusDriverSelect
				c.updateFocus()
				return c, nil
			default:
				// Pass all other keys to text input
				fields.nameInput, cmd = fields.nameInput.Update(msg)
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
			if c.focusField < FocusCancelButton {
				c.focusField = (c.focusField + 1) % (FocusCancelButton + 1)
			}
			c.updateFocus()

		case "shift+tab", "up", "k":
			// Cycle backward through fields
			if c.focusField > FocusNameInput {
				c.focusField = (c.focusField - 1)
			} else {
				c.focusField = FocusCancelButton
			}
			c.updateFocus()

		case "left", "h":
			// Navigate buttons
			if c.focusField == FocusSubmitButton {
				c.focusField = FocusCancelButton
			} else if c.focusField == FocusCancelButton {
				c.focusField = FocusSubmitButton
			}

		case "right", "l":
			// Navigate buttons
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

				// Create driver and test connection
				driver, err := c.createDriver()
				if err != nil {
					c.errorMsg = err.Error()
					return c, nil
				}

				connStr := c.BuildConnectionString()
				if err := driver.TestConnection(connStr); err != nil {
					c.errorMsg = "Connection failed: " + err.Error()
					return c, nil
				}

				logger.Info("Connection submitted", map[string]any{
					"driver": c.drivers[c.driverIndex],
					"name":   fields.nameInput.Value(),
					"host":   fields.hostInput.Value(),
					"port":   fields.portInput.Value(),
				})
				c.result = modal.ResultSubmit
				c.closed = true
			} else if c.focusField == FocusCancelButton {
				c.errorMsg = "" // Clear error on cancel
				logger.Debug("Create connection cancelled", nil)
				c.result = modal.ResultCancel
				c.closed = true
			}
		}
	}

	return c, nil
}

// handleInputUpdate routes key input to the appropriate text input field
func (cf *ConnectionFields) handleInputUpdate(msg tea.KeyMsg, focusField FocusField) {
	switch focusField {
	case FocusHostInput:
		cf.hostInput, _ = cf.hostInput.Update(msg)
	case FocusPortInput:
		cf.portInput, _ = cf.portInput.Update(msg)
	case FocusUsernameInput:
		cf.usernameInput, _ = cf.usernameInput.Update(msg)
	case FocusPasswordInput:
		cf.passwordInput, _ = cf.passwordInput.Update(msg)
	case FocusDatabaseInput:
		cf.databaseInput, _ = cf.databaseInput.Update(msg)
	}
}

func (c *Content) updateFocus() {
	fields := c.getCurrentFields()

	// Focus management for all fields
	if c.focusField == FocusNameInput {
		fields.nameInput.Focus()
	} else {
		fields.nameInput.Blur()
	}

	if c.focusField == FocusHostInput {
		fields.hostInput.Focus()
	} else {
		fields.hostInput.Blur()
	}

	if c.focusField == FocusPortInput {
		fields.portInput.Focus()
	} else {
		fields.portInput.Blur()
	}

	if c.focusField == FocusUsernameInput {
		fields.usernameInput.Focus()
	} else {
		fields.usernameInput.Blur()
	}

	if c.focusField == FocusPasswordInput {
		fields.passwordInput.Focus()
	} else {
		fields.passwordInput.Blur()
	}

	if c.focusField == FocusDatabaseInput {
		fields.databaseInput.Focus()
	} else {
		fields.databaseInput.Blur()
	}

	if c.focusField == FocusUriInput {
		fields.uriInput.Focus()
	} else {
		fields.uriInput.Blur()
	}
}

func (c *Content) View() string {
	t := theme.Current
	fields := c.getCurrentFields()

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Bold(true).
		Width(10)

	// Calculate input width based on modal content width
	// Keep inputs compact for smaller modal
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

	// Driver select field
	var driverSelect string
	if c.focusField == FocusDriverSelect {
		driverSelect = focusedInputStyle.Width(inputWidth).Render(fmt.Sprintf("[%s ▼]", c.drivers[c.driverIndex]))
	} else {
		driverSelect = unfocusedInputStyle.Width(inputWidth).Render(fmt.Sprintf(" %s  ", c.drivers[c.driverIndex]))
	}
	driverRow := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render("Driver:"),
		"  ",
		driverSelect,
	)

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
	nameRow := renderField("Name", fields.nameInput, c.focusField == FocusNameInput)

	var hostRow, portRow, usernameRow, passwordRow, databaseRow string

	if c.GetDriver() == drivers.DriverTypeSQLite {
		// For SQLite, show the database input as file path
		databaseRow = renderField("Path", fields.databaseInput, c.focusField == FocusDatabaseInput)
	} else if c.GetDriver() == drivers.DriverTypeMongoDBAtlas {
		// For MongoDB Atlas, only show URI input
		hostRow = renderField("Connection URI", fields.uriInput, c.focusField == FocusUriInput)
		portRow = ""
		usernameRow = ""
		passwordRow = ""
		databaseRow = ""
	} else {
		// For MySQL, PostgreSQL, and MongoDB, show all fields
		hostRow = renderField("Host", fields.hostInput, c.focusField == FocusHostInput)
		portRow = renderField("Port", fields.portInput, c.focusField == FocusPortInput)
		usernameRow = renderField("Username", fields.usernameInput, c.focusField == FocusUsernameInput)
		passwordRow = renderField("Password", fields.passwordInput, c.focusField == FocusPasswordInput)
		databaseRow = renderField("Database", fields.databaseInput, c.focusField == FocusDatabaseInput)
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

	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Align(lipgloss.Center).
		Padding(1, 0, 0, 0)
	help := helpStyle.Render("Tab/↑↓: navigate | k/j: select driver | Enter: test connection | Esc: cancel")

	contentStyle := lipgloss.NewStyle().
		Padding(0, 0)

	var content []string
	content = append(content, driverRow, nameRow)

	if c.GetDriver() == drivers.DriverTypeSQLite {
		content = append(content, databaseRow)
	} else if c.GetDriver() == drivers.DriverTypeMongoDBAtlas {
		// MongoDB Atlas - only URI input
		content = append(content, hostRow)
	} else {
		// MySQL, PostgreSQL, and MongoDB standard - all fields
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
	// Keep inputs compact
	inputWidth := 35

	// Update both driver field sets
	for _, fields := range []*ConnectionFields{&c.mysqlFields, &c.postgresFields} {
		fields.nameInput.Width = inputWidth
		fields.hostInput.Width = inputWidth
		fields.portInput.Width = inputWidth
		fields.usernameInput.Width = inputWidth
		fields.passwordInput.Width = inputWidth
		fields.databaseInput.Width = inputWidth
	}
}

// GetDriver returns the selected driver
func (c *Content) GetDriver() string {
	return c.drivers[c.driverIndex]
}

// BuildConnectionString builds the connection URL from the fields
func (c *Content) BuildConnectionString() string {
	fields := c.getCurrentFields()
	driver := c.GetDriver()

	if driver == drivers.DriverTypeSQLite {
		// SQLite URL format: sqlite:///path/to/database.db
		filePath := fields.databaseInput.Value()
		if filePath == "" {
			return ""
		}
		return fmt.Sprintf("sqlite://%s", filePath)
	}

	host := fields.hostInput.Value()
	if host == "" {
		host = "localhost"
	}

	port := fields.portInput.Value()
	if port == "" {
		port = c.getDefaultPort()
	}

	username := fields.usernameInput.Value()
	password := fields.passwordInput.Value()
	database := fields.databaseInput.Value()

	if driver == drivers.DriverTypeMySQL {
		// MySQL URL format: mysql://user:password@host:port/database
		if password != "" {
			return fmt.Sprintf("mysql://%s:%s@%s:%s/%s", username, password, host, port, database)
		}
		return fmt.Sprintf("mysql://%s@%s:%s/%s", username, host, port, database)
	} else if driver == drivers.DriverTypePostgreSQL {
		// PostgreSQL URL format: postgres://user:password@host:port/database?sslmode=disable
		if password != "" {
			return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, database)
		}
		return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", username, host, port, database)
	} else if driver == drivers.DriverTypeMongoDB {
		// MongoDB URL format: mongodb://user:password@host:port/database
		if password != "" {
			return fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", username, password, host, port, database)
		}
		return fmt.Sprintf("mongodb://%s@%s:%s/%s", username, host, port, database)
	} else if driver == drivers.DriverTypeMongoDBAtlas {
		// MongoDB Atlas uses direct URI input
		return fields.uriInput.Value()
	}

	return ""
}

func (c *Content) GetName() string {
	fields := c.getCurrentFields()
	return fields.nameInput.Value()
}

// Reset resets the content to initial state
func (c *Content) Reset() {
	c.driverIndex = 0
	c.focusField = FocusDriverSelect
	c.result = modal.ResultNone
	c.closed = false
	c.errorMsg = ""

	// Reset all driver field sets but keep defaults
	c.mysqlFields.nameInput.SetValue("")
	c.mysqlFields.hostInput.SetValue("localhost")
	c.mysqlFields.portInput.SetValue("3306")
	c.mysqlFields.usernameInput.SetValue("root")
	c.mysqlFields.passwordInput.SetValue("")
	c.mysqlFields.databaseInput.SetValue("")

	c.postgresFields.nameInput.SetValue("")
	c.postgresFields.hostInput.SetValue("localhost")
	c.postgresFields.portInput.SetValue("5432")
	c.postgresFields.usernameInput.SetValue("postgres")
	c.postgresFields.passwordInput.SetValue("")
	c.postgresFields.databaseInput.SetValue("")

	c.sqliteFields.nameInput.SetValue("")
	c.sqliteFields.databaseInput.SetValue("")

	c.mongodbFields.nameInput.SetValue("")
	c.mongodbFields.hostInput.SetValue("localhost")
	c.mongodbFields.portInput.SetValue("27017")
	c.mongodbFields.usernameInput.SetValue("")
	c.mongodbFields.passwordInput.SetValue("")
	c.mongodbFields.databaseInput.SetValue("")

	c.mongodbAtlasFields.nameInput.SetValue("")
	c.mongodbAtlasFields.uriInput.SetValue("")

	c.getCurrentFields().nameInput.Focus()
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
	// Set a fixed smaller width for the content
	m.content.SetWidth(60)
	// Update SQLite database input width
	m.content.sqliteFields.databaseInput.Width = 35
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

// GetConnectionString returns the built connection string
func (m Model) GetConnectionString() string {
	return m.content.BuildConnectionString()
}

// GetName returns the entered connection name
func (m Model) GetName() string {
	return m.content.GetName()
}
