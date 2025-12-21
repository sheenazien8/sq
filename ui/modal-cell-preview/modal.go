package modalcellpreview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

// Model wraps the generic modal with cell preview content
type Model struct {
	modal   modal.Model
	content *PreviewContent
}

// New creates a new cell preview modal
func New() Model {
	content := NewPreviewContent()
	m := modal.New("Cell Preview", content)
	return Model{
		modal:   m,
		content: content,
	}
}

// Show displays the modal with the given cell content
func (m *Model) Show(cellContent string) {
	m.content.SetContent(cellContent)
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

// PreviewContent implements Content for cell preview
type PreviewContent struct {
	viewport viewport.Model
	content  string
	width    int
	height   int
	closed   bool
}

// NewPreviewContent creates a new preview content
func NewPreviewContent() *PreviewContent {
	vp := viewport.New(60, 15) // Start with reasonable defaults
	vp.Style = theme.Current.TableCell.Copy()
	return &PreviewContent{
		viewport: vp,
		closed:   false,
	}
}

// SetContent sets the content to preview
func (p *PreviewContent) SetContent(content string) {
	p.content = content
	p.closed = false
	p.viewport.SetContent(content)
}

// Update handles input
func (p *PreviewContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			// Close the modal
			p.closed = true
			return p, nil
		default:
			// Pass other keys to viewport for scrolling
			p.viewport, cmd = p.viewport.Update(msg)
		}
	case tea.WindowSizeMsg:
		// Update viewport size when terminal resizes
		p.SetWidth(p.width)
	}
	return p, cmd
}

// View renders the content
func (p *PreviewContent) View() string {
	if p.content == "" {
		return "No content to preview"
	}

	// Show some basic info
	t := theme.Current
	infoStyle := t.StatusBar.Copy().Padding(0, 1)
	info := infoStyle.Render("Press Esc or Enter to close • Arrow keys to scroll")

	return strings.Join([]string{
		p.viewport.View(),
		info,
	}, "\n")
}

// Result returns the content's result
func (p *PreviewContent) Result() modal.Result {
	return modal.ResultNone
}

// ShouldClose returns true if the modal should close
func (p *PreviewContent) ShouldClose() bool {
	return p.closed
}

// SetWidth sets the content width
func (p *PreviewContent) SetWidth(width int) {
	p.width = width
	p.height = 20 // Fixed height for preview
	p.viewport.Width = width
	p.viewport.Height = p.height - 2 // Account for info line
}
