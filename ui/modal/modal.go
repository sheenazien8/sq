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
)

// Model represents a confirmation modal
type Model struct {
	Title    string
	Message  string
	visible  bool
	selected int // 0 = Yes, 1 = No
	result   Result

	width  int
	height int
}

// New creates a new modal
func New(title, message string) Model {
	return Model{
		Title:    title,
		Message:  message,
		visible:  false,
		selected: 1,
		result:   ResultNone,
	}
}

// Show displays the modal
func (m *Model) Show() {
	m.visible = true
	m.selected = 1
	m.result = ResultNone
}

// Hide hides the modal
func (m *Model) Hide() {
	m.visible = false
}

// Visible returns whether the modal is visible
func (m Model) Visible() bool {
	return m.visible
}

// Result returns the modal result
func (m Model) Result() Result {
	return m.result
}

// SetSize sets the terminal size for centering
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "tab":
			m.selected = 0
		case "right", "l", "shift+tab":
			m.selected = 1
		case "y", "Y":
			m.result = ResultYes
			m.visible = false
		case "n", "N", "esc":
			m.result = ResultNo
			m.visible = false
		case "enter":
			if m.selected == 0 {
				m.result = ResultYes
			} else {
				m.result = ResultNo
			}
			m.visible = false
		}
	}

	return m, nil
}

// View renders the modal as a full screen with centered dialog
func (m Model) View() string {
	if !m.visible {
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
	if m.selected == 0 {
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

	title := titleStyle.Render(m.Title)
	message := messageStyle.Render(m.Message)

	dialogContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		message,
		buttonRowCentered,
		help,
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
