package modalexport

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/keys"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

// Content implements modal.Content for exporting data
type ExportContent struct {
	formats     []string
	formatIndex int

	pathInput textinput.Model

	result modal.Result
	closed bool
	width  int
}

func NewExportContent(defaultPath string) *ExportContent {
	ti := textinput.New()
	ti.Placeholder = "path/to/file.csv"
	ti.CharLimit = 1024
	ti.Width = 60
	ti.SetValue(defaultPath)
	ti.Focus()

	return &ExportContent{
		formats:     []string{"csv", "json"},
		formatIndex: 0,
		pathInput:   ti,
		result:      modal.ResultNone,
		closed:      false,
	}
}

func (e *ExportContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.AppKeys.Cancel):
			e.result = modal.ResultCancel
			e.closed = true
			return e, nil
		case key.Matches(msg, keys.AppKeys.Confirm):
			e.result = modal.ResultSubmit
			e.closed = true
			return e, nil
		case key.Matches(msg, keys.AppKeys.Left) || msg.String() == "h":
			if e.formatIndex > 0 {
				e.formatIndex--
			}
			return e, nil
		case key.Matches(msg, keys.AppKeys.Right) || msg.String() == "l":
			if e.formatIndex < len(e.formats)-1 {
				e.formatIndex++
			}
			return e, nil
		default:
			var cmd tea.Cmd
			e.pathInput, cmd = e.pathInput.Update(msg)
			return e, cmd
		}
	}
	return e, nil
}

func (e *ExportContent) View() string {
	t := theme.Current

	formatStyle := lipgloss.NewStyle().Foreground(t.Colors.Primary).Bold(true)

	var formatItems []string
	for i, f := range e.formats {
		if i == e.formatIndex {
			formatItems = append(formatItems, formatStyle.Render("["+f+"]"))
		} else {
			formatItems = append(formatItems, f)
		}
	}

	pathView := e.pathInput.View()

	help := lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Render("←/→: format | Enter: export | Esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(t.Colors.Primary).Bold(true).Render("Format:"),
		lipgloss.JoinHorizontal(lipgloss.Left, formatItems...),
		"",
		lipgloss.NewStyle().Foreground(t.Colors.Primary).Bold(true).Render("Path:"),
		pathView,
		"",
		help,
	)
}

func (e *ExportContent) Result() modal.Result {
	return e.result
}

func (e *ExportContent) ShouldClose() bool {
	return e.closed
}

func (e *ExportContent) SetWidth(width int) {
	e.width = width
	e.pathInput.Width = width - 20
}

func (e *ExportContent) GetPath() string {
	return e.pathInput.Value()
}

func (e *ExportContent) GetFormat() string {
	if e.formatIndex < 0 || e.formatIndex >= len(e.formats) {
		return "csv"
	}
	return e.formats[e.formatIndex]
}

// New creates a modal wrapper
func New(defaultPath string) modal.Model {
	content := NewExportContent(defaultPath)
	return modal.New("Export Data", content)
}
