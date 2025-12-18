package modalexit

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sheenazien8/db-client-tui/logger"
	"github.com/sheenazien8/db-client-tui/ui/modal"
)

// Model wraps the generic modal with exit confirmation content
type Model struct {
	modal   modal.Model
	content *modal.ConfirmContent
}

// New creates a new exit confirmation modal
func New() Model {
	content := modal.NewConfirmContent("Are you sure you want to quit?")
	m := modal.New("Exit", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal
func (m *Model) Show() {
	logger.Debug("Exit modal opened", nil)
	m.content.Reset()
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

// Confirmed returns true if the user confirmed exit
func (m Model) Confirmed() bool {
	confirmed := m.modal.Result() == modal.ResultYes
	if confirmed {
		logger.Info("User confirmed exit", nil)
	} else {
		logger.Debug("User cancelled exit", nil)
	}
	return confirmed
}
