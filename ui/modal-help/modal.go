package modalhelp

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/keys"
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
					{keys.AppKeys.Help.Help().Key, keys.AppKeys.Help.Help().Desc},
					{keys.AppKeys.Quit.Help().Key, keys.AppKeys.Quit.Help().Desc},
					{keys.AppKeys.FocusNext.Help().Key, keys.AppKeys.FocusNext.Help().Desc},
					{keys.AppKeys.ToggleTheme.Help().Key, keys.AppKeys.ToggleTheme.Help().Desc},
					{keys.AppKeys.ToggleSidebar.Help().Key, keys.AppKeys.ToggleSidebar.Help().Desc},
					{keys.AppKeys.NextTab.Help().Key, keys.AppKeys.NextTab.Help().Desc},
					{keys.AppKeys.PrevTab.Help().Key, keys.AppKeys.PrevTab.Help().Desc},
					{keys.AppKeys.CloseTab.Help().Key, keys.AppKeys.CloseTab.Help().Desc},
				},
			},
			{
				Title: "Sidebar",
				Keymaps: []Keymap{
					{keys.AppKeys.Down.Help().Key, keys.AppKeys.Down.Help().Desc},
					{keys.AppKeys.Up.Help().Key, keys.AppKeys.Up.Help().Desc},
					{keys.AppKeys.Confirm.Help().Key, "Select/Connect database"},
					{keys.AppKeys.OpenQueryEditor.Help().Key, keys.AppKeys.OpenQueryEditor.Help().Desc},
					{keys.AppKeys.ViewStructure.Help().Key, keys.AppKeys.ViewStructure.Help().Desc},
					{keys.AppKeys.NewConnection.Help().Key, keys.AppKeys.NewConnection.Help().Desc},
					{keys.AppKeys.Filter.Help().Key, keys.AppKeys.Filter.Help().Desc},
					{keys.AppKeys.ClearFilter.Help().Key, keys.AppKeys.ClearFilter.Help().Desc},
					{keys.AppKeys.Refresh.Help().Key, keys.AppKeys.Refresh.Help().Desc},
				},
			},
			{
				Title: "Table View",
				Keymaps: []Keymap{
					{keys.AppKeys.Down.Help().Key, "Move down one row"},
					{keys.AppKeys.Up.Help().Key, "Move up one row"},
					{keys.AppKeys.Left.Help().Key, "Move left one column"},
					{keys.AppKeys.Right.Help().Key, "Move right one column"},
					{keys.AppKeys.PageDown.Help().Key, "Page down"},
					{keys.AppKeys.PageUp.Help().Key, "Page up"},
					{keys.AppKeys.JumpToFirstColumn.Help().Key, keys.AppKeys.JumpToFirstColumn.Help().Desc},
					{keys.AppKeys.JumpToLastColumn.Help().Key, keys.AppKeys.JumpToLastColumn.Help().Desc},
					{keys.AppKeys.Home.Help().Key, keys.AppKeys.Home.Help().Desc},
					{keys.AppKeys.End.Help().Key, keys.AppKeys.End.Help().Desc},
					{keys.AppKeys.NextPage.Help().Key, keys.AppKeys.NextPage.Help().Desc},
					{keys.AppKeys.PrevPage.Help().Key, keys.AppKeys.PrevPage.Help().Desc},
					{"Space", "Sort by column (toggle ASC/DESC)"},
					{keys.AppKeys.Yank.Help().Key, keys.AppKeys.Yank.Help().Desc},
					{keys.AppKeys.Preview.Help().Key, keys.AppKeys.Preview.Help().Desc},
					{keys.AppKeys.ActionMenu.Help().Key, keys.AppKeys.ActionMenu.Help().Desc},
					{keys.AppKeys.GotoDefinition.Help().Key, keys.AppKeys.GotoDefinition.Help().Desc},
					{keys.AppKeys.ColumnVisibility.Help().Key, keys.AppKeys.ColumnVisibility.Help().Desc},
					{keys.AppKeys.Filter.Help().Key, keys.AppKeys.Filter.Help().Desc},
					{keys.AppKeys.ClearFilter.Help().Key, keys.AppKeys.ClearFilter.Help().Desc},
					{keys.AppKeys.OpenQueryEditor.Help().Key, keys.AppKeys.OpenQueryEditor.Help().Desc},
					{keys.AppKeys.ViewStructure.Help().Key, keys.AppKeys.ViewStructure.Help().Desc},
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
					{keys.AppKeys.ExecuteQuery.Help().Key, keys.AppKeys.ExecuteQuery.Help().Desc},
					{keys.AppKeys.FormatQuery.Help().Key, keys.AppKeys.FormatQuery.Help().Desc},
					{keys.AppKeys.YankQuery.Help().Key, keys.AppKeys.YankQuery.Help().Desc},
					{keys.AppKeys.SwitchPane.Help().Key, keys.AppKeys.SwitchPane.Help().Desc},
				},
			},
			{
				Title: "Filter",
				Keymaps: []Keymap{
					{keys.AppKeys.Filter.Help().Key, keys.AppKeys.Filter.Help().Desc},
					{"Tab", "Complete current word"},
					{"Ctrl+N", "Next suggestion"},
					{"Ctrl+P", "Previous suggestion"},
					{keys.AppKeys.Confirm.Help().Key, "Apply filter & blur"},
					{keys.AppKeys.Cancel.Help().Key, "Blur without applying"},
					{keys.AppKeys.Quit.Help().Key, "Clear filter & refresh"},
				},
			},
			{
				Title: "Structure View",
				Keymaps: []Keymap{
					{"1", "Columns section"},
					{"2", "Indexes section"},
					{"3", "Relations section"},
					{"4", "Triggers section"},
					{keys.AppKeys.FocusNext.Help().Key, "Next section"},
					{keys.AppKeys.Down.Help().Key + "/" + keys.AppKeys.Up.Help().Key, "Navigate rows"},
					{keys.AppKeys.Left.Help().Key + "/" + keys.AppKeys.Right.Help().Key, "Navigate columns"},
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
		switch {
		case key.Matches(msg, keys.AppKeys.Cancel), key.Matches(msg, keys.AppKeys.Quit), key.Matches(msg, keys.AppKeys.Help):
			c.closed = true
		case key.Matches(msg, keys.AppKeys.FocusNext), key.Matches(msg, keys.AppKeys.Right):
			c.activeSection = (c.activeSection + 1) % len(c.sections)
			c.scrollOffset = 0
		case msg.String() == "shift+tab", key.Matches(msg, keys.AppKeys.Left):
			c.activeSection--
			if c.activeSection < 0 {
				c.activeSection = len(c.sections) - 1
			}
			c.scrollOffset = 0
		case key.Matches(msg, keys.AppKeys.Down):
			maxOffset := len(c.sections[c.activeSection].Keymaps) - c.visibleLines
			if maxOffset < 0 {
				maxOffset = 0
			}
			if c.scrollOffset < maxOffset {
				c.scrollOffset++
			}
		case key.Matches(msg, keys.AppKeys.Up):
			if c.scrollOffset > 0 {
				c.scrollOffset--
			}
		case msg.String() == "1":
			if len(c.sections) > 0 {
				c.activeSection = 0
				c.scrollOffset = 0
			}
		case msg.String() == "2":
			if len(c.sections) > 1 {
				c.activeSection = 1
				c.scrollOffset = 0
			}
		case msg.String() == "3":
			if len(c.sections) > 2 {
				c.activeSection = 2
				c.scrollOffset = 0
			}
		case msg.String() == "4":
			if len(c.sections) > 3 {
				c.activeSection = 3
				c.scrollOffset = 0
			}
		case msg.String() == "5":
			if len(c.sections) > 4 {
				c.activeSection = 4
				c.scrollOffset = 0
			}
		case msg.String() == "6":
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
