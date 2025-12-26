package toast

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/theme"
)

// ToastType represents the type of toast message
type ToastType int

const (
	ToastInfo ToastType = iota
	ToastSuccess
	ToastError
	ToastWarning
)

// Model represents a toast notification popup
type Model struct {
	message   string
	toastType ToastType
	visible   bool
	width     int
	height    int
	buttonPos int // 0 = OK button position (for keyboard navigation)
}

// New creates a new toast model
func New() Model {
	return Model{
		width:     80,
		height:    30,
		buttonPos: 0,
	}
}

// SetSize sets the terminal dimensions for positioning
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Show displays a toast message
func (m *Model) Show(message string, toastType ToastType) {
	m.message = message
	m.toastType = toastType
	m.visible = true
	m.buttonPos = 0
}

// ShowError displays an error toast
func (m *Model) ShowError(message string) {
	m.Show(message, ToastError)
}

// ShowSuccess displays a success toast
func (m *Model) ShowSuccess(message string) {
	m.Show(message, ToastSuccess)
}

// ShowWarning displays a warning toast
func (m *Model) ShowWarning(message string) {
	m.Show(message, ToastWarning)
}

// ShowInfo displays an info toast
func (m *Model) ShowInfo(message string) {
	m.Show(message, ToastInfo)
}

// Visible returns whether the toast is currently visible
func (m Model) Visible() bool {
	return m.visible
}

// Hide hides the toast
func (m *Model) Hide() {
	m.visible = false
}

// Update handles keyboard input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "q":
			m.visible = false
		}
	}
	return m, nil
}

// wrapText wraps text to a specific width
func wrapText(text string, maxWidth int) []string {
	if maxWidth < 10 {
		maxWidth = 10
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		// Check if adding this word would exceed max width
		testLine := currentLine + word
		if currentLine != "" {
			testLine = currentLine + " " + word
		}

		// If the word itself is longer than maxWidth, just add it to new line
		if len(word) > maxWidth {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				lines = append(lines, word)
				currentLine = ""
			}
		} else if len(testLine) <= maxWidth {
			// Word fits in current line
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine = currentLine + " " + word
			}
		} else {
			// Word doesn't fit, start new line
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	// Add any remaining text
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// View renders the toast notification as a popup modal
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	t := theme.Current

	// Determine styling based on type
	var borderColor, fgColor lipgloss.Color
	switch m.toastType {
	case ToastError:
		borderColor = t.Colors.Error
		fgColor = t.Colors.Error
	case ToastSuccess:
		borderColor = t.Colors.Success
		fgColor = t.Colors.Success
	case ToastWarning:
		borderColor = t.Colors.Warning
		fgColor = t.Colors.Warning
	default: // ToastInfo
		borderColor = t.Colors.Primary
		fgColor = t.Colors.Primary
	}

	// Create icon based on type
	var icon string
	switch m.toastType {
	case ToastError:
		icon = "✘"
	case ToastSuccess:
		icon = "✔"
	case ToastWarning:
		icon = "⚠"
	default:
		icon = "ℹ"
	}

	// Calculate max width for dialog (leave margins)
	dialogMaxWidth := m.width - 6 // 3 char margin on each side
	if dialogMaxWidth > 70 {
		dialogMaxWidth = 70 // Cap at 70 chars
	}
	if dialogMaxWidth < 30 {
		dialogMaxWidth = 30 // Minimum 30 chars
	}

	// Calculate message content width (account for padding and icon)
	// Icon (1 char) + space (1 char) + padding (2*2=4 chars) = 8 chars overhead
	messageContentWidth := dialogMaxWidth - 8
	if messageContentWidth < 20 {
		messageContentWidth = 20
	}

	// Wrap the message text
	wrappedLines := wrapText(m.message, messageContentWidth)

	// Build message with icon on first line
	iconStyle := lipgloss.NewStyle().
		Foreground(fgColor).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(fgColor)

	var messageParts []string
	for i, line := range wrappedLines {
		if i == 0 {
			// First line with icon
			messageParts = append(messageParts, iconStyle.Render(icon)+" "+messageStyle.Render(line))
		} else {
			// Subsequent lines indented to align with text
			messageParts = append(messageParts, "  "+messageStyle.Render(line))
		}
	}

	messageContent := strings.Join(messageParts, "\n")

	// Build button section
	buttonStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 2).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Padding(0, 1)

	okButton := buttonStyle.Render("OK")
	helpText := helpStyle.Render("Enter/Esc/Q to close")

	// Combine all sections with proper padding
	dialogContent := lipgloss.JoinVertical(
		lipgloss.Center,
		messageContent,
		"",
		okButton,
		helpText,
	)

	// Apply dialog border (no background)
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(dialogMaxWidth)

	dialog := dialogStyle.Render(dialogContent)

	// Calculate position - centered with slight bias to top
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)

	padLeft := (m.width - dialogWidth) / 2
	padTop := (m.height - dialogHeight) / 3 // Positioned in upper third

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	// Build final output with positioning
	var lines []string

	// Top padding
	for i := 0; i < padTop; i++ {
		lines = append(lines, "")
	}

	// Dialog content with left padding
	dialogLines := strings.Split(dialog, "\n")
	leftPadding := strings.Repeat(" ", padLeft)
	for _, line := range dialogLines {
		lines = append(lines, leftPadding+line)
	}

	// Fill remaining height
	for len(lines) < m.height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
