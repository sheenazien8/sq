package modal

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

// Result represents the modal result
type Result int

const (
	ResultNone Result = iota
	ResultYes
	ResultNo
	ResultSubmit
	ResultCancel
)

// Content is the interface that modal content must implement
type Content interface {
	// Update handles input for the content
	Update(msg tea.Msg) (Content, tea.Cmd)
	// View renders the content (without modal frame)
	View() string
	// Result returns the content's result
	Result() Result
	// ShouldClose returns true if the modal should close
	ShouldClose() bool
	// SetWidth sets the content width
	SetWidth(width int)
}

// Model represents a generic modal
type Model struct {
	Title   string
	Content Content
	visible bool

	width  int
	height int
}

// New creates a new modal with given title and content
func New(title string, content Content) Model {
	return Model{
		Title:   title,
		Content: content,
		visible: false,
	}
}

// Show displays the modal
func (m *Model) Show() {
	m.visible = true
}

// Hide hides the modal
func (m *Model) Hide() {
	m.visible = false
}

// Visible returns whether the modal is visible
func (m Model) Visible() bool {
	return m.visible
}

// Result returns the modal content's result
func (m Model) Result() Result {
	if m.Content != nil {
		return m.Content.Result()
	}
	return ResultNone
}

// SetSize sets the terminal size for centering
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.Content != nil {
		// Give content a reasonable width (modal inner width)
		m.Content.SetWidth(width - 20)
	}
}

// SetContent sets the modal content
func (m *Model) SetContent(content Content) {
	m.Content = content
	if m.width > 0 {
		m.Content.SetWidth(m.width - 20)
	}
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible || m.Content == nil {
		return m, nil
	}

	var cmd tea.Cmd
	m.Content, cmd = m.Content.Update(msg)

	// Check if content wants to close the modal
	if m.Content.ShouldClose() {
		m.visible = false
	}

	return m, cmd
}

// View renders the modal as a full screen with centered dialog
func (m Model) View() string {
	if !m.visible || m.Content == nil {
		return ""
	}

	t := theme.Current

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Colors.Primary).
		Padding(1, 3).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Bold(true).
		Align(lipgloss.Center)

	title := titleStyle.Render(m.Title)
	content := m.Content.View()

	dialogContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		content,
	)

	dialog := dialogStyle.Render(dialogContent)

	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)

	padLeft := (m.width - dialogWidth) / 2
	padTop := (m.height - dialogHeight) / 2

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	var lines []string

	// Top padding (empty lines)
	for range padTop {
		lines = append(lines, "")
	}

	dialogLines := strings.Split(dialog, "\n")
	leftPadding := strings.Repeat(" ", padLeft)
	for _, line := range dialogLines {
		lines = append(lines, leftPadding+line)
	}

	for len(lines) < m.height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// ConfirmContent implements Content for a simple yes/no confirmation
type ConfirmContent struct {
	Message  string
	selected int // 0 = Yes, 1 = No
	result   Result
	closed   bool
	width    int
}

// NewConfirmContent creates a new confirmation content
func NewConfirmContent(message string) *ConfirmContent {
	return &ConfirmContent{
		Message:  message,
		selected: 1, // Default to No
		result:   ResultNone,
		closed:   false,
	}
}

func (c *ConfirmContent) Update(msg tea.Msg) (Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "tab":
			c.selected = 0
		case "right", "l", "shift+tab":
			c.selected = 1
		case "y", "Y":
			c.result = ResultYes
			c.closed = true
		case "n", "N", "esc":
			c.result = ResultNo
			c.closed = true
		case "enter":
			if c.selected == 0 {
				c.result = ResultYes
			} else {
				c.result = ResultNo
			}
			c.closed = true
		}
	}
	return c, nil
}

func (c *ConfirmContent) View() string {
	t := theme.Current

	messageStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Align(lipgloss.Center).
		Padding(1, 0)

	activeButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 2).
		Bold(true)

	inactiveButtonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Background(t.Colors.SelectionBg).
		Padding(0, 2)

	var yesButton, noButton string
	if c.selected == 0 {
		yesButton = activeButtonStyle.Render("[ Yes ]")
		noButton = inactiveButtonStyle.Render("  No  ")
	} else {
		yesButton = inactiveButtonStyle.Render("  Yes  ")
		noButton = activeButtonStyle.Render("[ No ]")
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, yesButton, "   ", noButton)
	buttonRowCentered := lipgloss.NewStyle().Width(40).Align(lipgloss.Center).Render(buttonRow)

	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Align(lipgloss.Center).
		Padding(1, 0, 0, 0)
	help := helpStyle.Render("←→: select | Enter: confirm | Y/N | Esc: cancel")

	message := messageStyle.Render(c.Message)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		message,
		buttonRowCentered,
		help,
	)
}

func (c *ConfirmContent) Result() Result {
	return c.result
}

func (c *ConfirmContent) ShouldClose() bool {
	return c.closed
}

func (c *ConfirmContent) SetWidth(width int) {
	c.width = width
}

// Reset resets the confirmation content to initial state
func (c *ConfirmContent) Reset() {
	c.selected = 1
	c.result = ResultNone
	c.closed = false
}

// NewConfirm creates a new confirmation modal (convenience function)
func NewConfirm(title, message string) Model {
	content := NewConfirmContent(message)
	return New(title, content)
}
