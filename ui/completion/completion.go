package completion

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/theme"
)

// CompletionItem represents a completion item from LSP
type CompletionItem struct {
	Label         string
	Kind          int
	Detail        string
	Documentation string
	InsertText    string
}

// Item implements list.Item interface
func (c CompletionItem) FilterValue() string {
	return c.Label
}

// Completion represents the completion popup component
type Model struct {
	list     list.Model
	visible  bool
	width    int
	height   int
	x        int
	y        int
	selected bool
}

// New creates a new completion model
func New() Model {
	items := []list.Item{}

	l := list.New(items, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return Model{
		list:    l,
		visible: false,
	}
}

// SetSize sets the completion popup dimensions and position
func (m *Model) SetSize(width, height, x, y int) {
	m.width = width
	m.height = height
	m.x = x
	m.y = y
	m.list.SetSize(width-4, height-2)
}

// SetItems sets the completion items
func (m *Model) SetItems(items []CompletionItem) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	m.list.SetItems(listItems)
	m.selected = false
}

// Show shows the completion popup
func (m *Model) Show() {
	m.visible = true
}

// Hide hides the completion popup
func (m *Model) Hide() {
	m.visible = false
}

// Visible returns whether the completion popup is visible
func (m Model) Visible() bool {
	return m.visible
}

// Selected returns whether an item has been selected
func (m Model) Selected() bool {
	return m.selected
}

// SelectedItem returns the currently selected completion item
func (m Model) SelectedItem() CompletionItem {
	if item := m.list.SelectedItem(); item != nil {
		return item.(CompletionItem)
	}
	return CompletionItem{}
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.visible && len(m.list.Items()) > 0 {
				m.selected = true
				return m, nil
			}
		case "esc":
			m.visible = false
			return m, nil
		case "ctrl+c":
			m.visible = false
			return m, nil
		}
	}

	if m.visible {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

// View renders the completion popup
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	t := theme.Current

	// Style the list
	content := m.list.View()

	// Create border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Colors.Primary).
		Width(m.width).
		Height(m.height)

	popup := borderStyle.Render(content)

	// Position the popup (this would need to be handled by parent component)
	return popup
}

// itemDelegate implements list.ItemDelegate
type itemDelegate struct{}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(CompletionItem)
	if !ok {
		return
	}

	t := theme.Current

	var style lipgloss.Style

	if index == m.Index() {
		style = lipgloss.NewStyle().
			Foreground(t.Colors.Background).
			Background(t.Colors.Primary).
			Bold(true)
	} else {
		style = lipgloss.NewStyle().
			Foreground(t.Colors.Foreground)
	}

	label := item.Label
	if item.Detail != "" {
		label += " - " + item.Detail
	}

	fmt.Fprint(w, style.Render(label))
}
