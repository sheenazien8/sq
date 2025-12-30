package modalhelp

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/theme"
)

// HelpSection represents a section of help keymaps
type HelpSection struct {
	Title   string
	Keymaps []Keymap
}

// Keymap represents a single key mapping
type Keymap struct {
	Key         string
	Description string
}

// HelpContent implements modal.Content for displaying help
type HelpContent struct {
	sections      []HelpSection
	activeSection int
	closed        bool
	width         int
	scrollOffset  int
	visibleLines  int
}

// NewHelpContent creates a new help content with sections
func NewHelpContent() *HelpContent {
	return &HelpContent{
		sections: []HelpSection{
			{
				Title: "Global",
				Keymaps: []Keymap{
					{"?", "Show this help"},
					{"q / Ctrl+C", "Quit application"},
					{"Tab", "Switch focus between panels"},
					{"s", "Toggle sidebar"},
					{"T", "Cycle themes"},
					{"[", "Previous tab"},
					{"]", "Next tab"},
					{"Ctrl+W", "Close current tab"},
				},
			},
			{
				Title: "Sidebar",
				Keymaps: []Keymap{
					{"j / ↓", "Move down"},
					{"k / ↑", "Move up"},
					{"Enter", "Select/Connect database"},
					{"e", "Open query editor"},
					{"d", "View table structure"},
					{"n", "New connection"},
					{"/", "Filter connections/tables"},
					{"C", "Clear filter"},
					{"R", "Refresh connections"},
				},
			},
			{
				Title: "Table View",
				Keymaps: []Keymap{
					{"j / ↓", "Move down one row"},
					{"k / ↑", "Move up one row"},
					{"h / ←", "Move left one column"},
					{"l / →", "Move right one column"},
					{"J / PgDn", "Page down"},
					{"K / PgUp", "Page up"},
					{"H", "Jump to first column"},
					{"L", "Jump to last column"},
					{"Home", "Jump to first row"},
					{"End", "Jump to last row"},
					{">", "Next page (query)"},
					{"<", "Previous page (query)"},
					{"Space", "Sort by column (toggle ASC/DESC)"},
					{"y", "Yank (copy) cell"},
					{"p", "Preview cell content"},
					{"a", "Cell actions menu"},
					{"gd", "Go to definition (FK)"},
					{"/", "Focus filter"},
					{"C", "Clear filter"},
					{"e", "Open query editor"},
					{"d", "View table structure"},
				},
			},
			{
				Title: "Query Editor",
				Keymaps: []Keymap{
					{"", "─── Normal Mode ───"},
					{"i", "Enter insert mode"},
					{"a", "Append after cursor"},
					{"I", "Insert at line start"},
					{"A", "Append at line end"},
					{"o", "New line below"},
					{"O", "New line above"},
					{"h/j/k/l", "Navigate"},
					{"w", "Move word forward"},
					{"b", "Move word backward"},
					{"0", "Go to line start"},
					{"$", "Go to line end"},
					{"g", "Go to start"},
					{"G", "Go to end"},
					{"x", "Delete character"},
					{"dd", "Delete line"},
					{"yy", "Yank line"},
					{"Y", "Yank query to clipboard"},
					{"p", "Paste"},
					{"u", "Undo"},
					{"v", "Visual mode"},
					{"", ""},
					{"", "─── Insert Mode ───"},
					{"Esc", "Return to normal mode"},
					{"", ""},
					{"", "─── Visual Mode ───"},
					{"Esc", "Return to normal mode"},
					{"h/j/k/l", "Extend selection"},
					{"d", "Delete selection"},
					{"y", "Yank selection"},
					{"c", "Change selection"},
					{"", ""},
					{"", "─── All Modes ───"},
					{"F5 / Ctrl+E", "Execute query"},
					{"Ctrl+F", "Format SQL"},
					{"Ctrl+Y", "Copy query to clipboard"},
					{"Ctrl+R", "Toggle results focus"},
				},
			},
			{
				Title: "Filter",
				Keymaps: []Keymap{
					{"/", "Focus filter input"},
					{"Tab", "Complete current word"},
					{"Ctrl+N", "Next suggestion"},
					{"Ctrl+P", "Previous suggestion"},
					{"Enter", "Apply filter & blur"},
					{"Esc", "Blur without applying"},
					{"Ctrl+C", "Clear filter & refresh"},
				},
			},
			{
				Title: "Structure View",
				Keymaps: []Keymap{
					{"1", "Columns section"},
					{"2", "Indexes section"},
					{"3", "Relations section"},
					{"4", "Triggers section"},
					{"Tab", "Next section"},
					{"j/k", "Navigate rows"},
					{"h/l", "Navigate columns"},
				},
			},
		},
		activeSection: 0,
		closed:        false,
		visibleLines:  20,
	}
}

func (c *HelpContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "?":
			c.closed = true
		case "tab", "l", "right":
			c.activeSection = (c.activeSection + 1) % len(c.sections)
			c.scrollOffset = 0
		case "shift+tab", "h", "left":
			c.activeSection--
			if c.activeSection < 0 {
				c.activeSection = len(c.sections) - 1
			}
			c.scrollOffset = 0
		case "j", "down":
			maxOffset := len(c.sections[c.activeSection].Keymaps) - c.visibleLines
			if maxOffset < 0 {
				maxOffset = 0
			}
			if c.scrollOffset < maxOffset {
				c.scrollOffset++
			}
		case "k", "up":
			if c.scrollOffset > 0 {
				c.scrollOffset--
			}
		case "1":
			if len(c.sections) > 0 {
				c.activeSection = 0
				c.scrollOffset = 0
			}
		case "2":
			if len(c.sections) > 1 {
				c.activeSection = 1
				c.scrollOffset = 0
			}
		case "3":
			if len(c.sections) > 2 {
				c.activeSection = 2
				c.scrollOffset = 0
			}
		case "4":
			if len(c.sections) > 3 {
				c.activeSection = 3
				c.scrollOffset = 0
			}
		case "5":
			if len(c.sections) > 4 {
				c.activeSection = 4
				c.scrollOffset = 0
			}
		case "6":
			if len(c.sections) > 5 {
				c.activeSection = 5
				c.scrollOffset = 0
			}
		}
	}
	return c, nil
}

func (c *HelpContent) View() string {
	t := theme.Current

	// Section tabs
	var tabs []string
	for i, section := range c.sections {
		tabStyle := lipgloss.NewStyle().Padding(0, 1)
		if i == c.activeSection {
			tabStyle = tabStyle.
				Foreground(t.Colors.Background).
				Background(t.Colors.Primary).
				Bold(true)
		} else {
			tabStyle = tabStyle.
				Foreground(t.Colors.ForegroundDim)
		}
		tabs = append(tabs, tabStyle.Render(section.Title))
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Content for active section
	section := c.sections[c.activeSection]

	keyStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Bold(true).
		Width(20)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground)

	var lines []string
	endIdx := c.scrollOffset + c.visibleLines
	if endIdx > len(section.Keymaps) {
		endIdx = len(section.Keymaps)
	}

	for i := c.scrollOffset; i < endIdx; i++ {
		km := section.Keymaps[i]
		line := keyStyle.Render(km.Key) + descStyle.Render(km.Description)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	// Scroll indicator
	scrollInfo := ""
	if len(section.Keymaps) > c.visibleLines {
		scrollInfo = lipgloss.NewStyle().
			Foreground(t.Colors.ForegroundDim).
			Render("\n↑↓ to scroll")
	}

	// Help footer
	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Padding(1, 0, 0, 0)
	help := helpStyle.Render("←→/Tab: sections | 1-8: jump to section | Esc/q: close")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabBar,
		"",
		content,
		scrollInfo,
		help,
	)
}

func (c *HelpContent) Result() modal.Result {
	return modal.ResultNone
}

func (c *HelpContent) ShouldClose() bool {
	return c.closed
}

func (c *HelpContent) SetWidth(width int) {
	c.width = width
}

// Reset resets the help content
func (c *HelpContent) Reset() {
	c.activeSection = 0
	c.scrollOffset = 0
	c.closed = false
}

// Model wraps the generic modal with help content
type Model struct {
	modal   modal.Model
	content *HelpContent
}

// New creates a new help modal
func New() Model {
	content := NewHelpContent()
	m := modal.New("Keyboard Shortcuts", content)
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
func (m Model) Visible() bool {
	return m.modal.Visible()
}

// SetSize sets the terminal size for centering
func (m *Model) SetSize(width, height int) {
	m.modal.SetSize(width, height)
	// Adjust visible lines based on height
	m.content.visibleLines = max(5, height/2-10)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
