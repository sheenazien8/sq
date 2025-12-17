package filter

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

// Operator represents a filter operator
type Operator string

const (
	OpEquals      Operator = "equals"
	OpNotEquals   Operator = "not equals"
	OpContains    Operator = "LIKE"
	OpNotContains Operator = "NOT LIKE"
	OpGreater     Operator = "greater than"
	OpLess        Operator = "less than"
	OpGreaterEq   Operator = "greater than equals"
	OpLessEq      Operator = "less than equals"
)

var operators = []Operator{
	OpEquals,
	OpNotEquals,
	OpContains,
	OpNotContains,
	OpGreater,
	OpLess,
	OpGreaterEq,
	OpLessEq,
}

// Filter represents a single filter condition
type Filter struct {
	Column   string
	Operator Operator
	Value    string
}

// FocusField represents which field is currently focused
type FocusField int

const (
	FocusColumn FocusField = iota
	FocusOperator
	FocusValue
)

// Model represents the filter input component
type Model struct {
	columns       []string // Available column names
	columnIndex   int      // Selected column index
	operatorIndex int      // Selected operator index
	valueInput    textinput.Model

	focusField FocusField
	width      int
	visible    bool
	active     bool // Whether filter is actively filtering

	// Current filter
	currentFilter *Filter
}

// New creates a new filter model
func New(columns []string) Model {
	ti := textinput.New()
	ti.Placeholder = "enter value"
	ti.CharLimit = 100
	ti.Width = 30

	return Model{
		columns:       columns,
		columnIndex:   0,
		operatorIndex: 0,
		valueInput:    ti,
		focusField:    FocusColumn,
		visible:       false,
		active:        false,
	}
}

// SetColumns updates the available columns
func (m *Model) SetColumns(columns []string) {
	m.columns = columns
	if m.columnIndex >= len(columns) {
		m.columnIndex = 0
	}
}

// SetWidth sets the component width
func (m *Model) SetWidth(width int) {
	m.width = width
	if width > 60 {
		m.valueInput.Width = 30
	} else {
		m.valueInput.Width = 15
	}
}

// SetVisible shows/hides the filter
func (m *Model) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.focusField = FocusColumn
		m.valueInput.Blur()
	}
}

// Visible returns whether the filter is visible
func (m Model) Visible() bool {
	return m.visible
}

// Toggle toggles visibility
func (m *Model) Toggle() {
	m.visible = !m.visible
	if m.visible {
		m.focusField = FocusColumn
		m.valueInput.Blur()
	}
}

// Active returns whether a filter is active
func (m Model) Active() bool {
	return m.active && m.currentFilter != nil
}

// GetFilter returns the current filter
func (m Model) GetFilter() *Filter {
	if m.active {
		return m.currentFilter
	}
	return nil
}

// Clear clears the current filter
func (m *Model) Clear() {
	m.valueInput.SetValue("")
	m.currentFilter = nil
	m.active = false
	m.columnIndex = 0
	m.operatorIndex = 0
}

// Apply applies the current filter settings
func (m *Model) Apply() {
	if len(m.columns) == 0 {
		return
	}

	value := strings.TrimSpace(m.valueInput.Value())
	if value == "" {
		m.active = false
		m.currentFilter = nil
		return
	}

	m.currentFilter = &Filter{
		Column:   m.columns[m.columnIndex],
		Operator: operators[m.operatorIndex],
		Value:    value,
	}
	m.active = true
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle escape first
		if key == "esc" {
			m.visible = false
			m.valueInput.Blur()
			return m, nil
		}

		// Handle enter to apply
		if key == "enter" {
			m.Apply()
			m.visible = false
			m.valueInput.Blur()
			return m, nil
		}

		// Handle clear
		if key == "ctrl+c" {
			m.Clear()
			m.valueInput.Blur()
			return m, nil
		}

		// If on value field, handle text input
		if m.focusField == FocusValue {
			// Tab moves to next field
			if key == "tab" {
				m.focusField = FocusColumn
				m.valueInput.Blur()
				return m, nil
			}
			if key == "shift+tab" {
				m.focusField = FocusOperator
				m.valueInput.Blur()
				return m, nil
			}
			// Pass other keys to text input
			m.valueInput, cmd = m.valueInput.Update(msg)
			return m, cmd
		}

		// Navigation for column and operator fields
		switch key {
		case "tab", "l", "right":
			m.focusField = (m.focusField + 1) % 3
			if m.focusField == FocusValue {
				m.valueInput.Focus()
			}
		case "shift+tab", "h", "left":
			if m.focusField == 0 {
				m.focusField = 2
			} else {
				m.focusField--
			}
			if m.focusField == FocusValue {
				m.valueInput.Focus()
			}
		case "up", "k":
			if m.focusField == FocusColumn && len(m.columns) > 0 {
				if m.columnIndex > 0 {
					m.columnIndex--
				} else {
					m.columnIndex = len(m.columns) - 1
				}
			} else if m.focusField == FocusOperator {
				if m.operatorIndex > 0 {
					m.operatorIndex--
				} else {
					m.operatorIndex = len(operators) - 1
				}
			}
		case "down", "j":
			if m.focusField == FocusColumn && len(m.columns) > 0 {
				m.columnIndex = (m.columnIndex + 1) % len(m.columns)
			} else if m.focusField == FocusOperator {
				m.operatorIndex = (m.operatorIndex + 1) % len(operators)
			}
		}
	}

	return m, cmd
}

// View renders the filter component
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	t := theme.Current

	// Focused and unfocused styles
	focusedStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.Primary).
		Padding(0, 1)

	unfocusedStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim)

	// Build column field
	var columnValue string
	if len(m.columns) > 0 {
		columnValue = m.columns[m.columnIndex]
	} else {
		columnValue = "(none)"
	}
	columnLabel := labelStyle.Render("Column: ")
	var columnField string
	if m.focusField == FocusColumn {
		columnField = focusedStyle.Render("< " + columnValue + " >")
	} else {
		columnField = unfocusedStyle.Render("  " + columnValue + "  ")
	}

	// Build operator field
	opValue := string(operators[m.operatorIndex])
	opLabel := labelStyle.Render("  Op: ")
	var opField string
	if m.focusField == FocusOperator {
		opField = focusedStyle.Render("< " + padOperator(opValue) + " >")
	} else {
		opField = unfocusedStyle.Render("  " + padOperator(opValue) + "  ")
	}

	// Build value field
	valueLabel := labelStyle.Render("  Value: ")
	var valueField string
	if m.focusField == FocusValue {
		// Render text input with cursor
		inputStyle := lipgloss.NewStyle().
			Foreground(t.Colors.Foreground).
			Background(t.Colors.SelectionBg).
			Padding(0, 1)
		valueField = inputStyle.Render(m.valueInput.View())
	} else {
		val := m.valueInput.Value()
		if val == "" {
			val = "..."
		}
		valueField = unfocusedStyle.Render("[" + val + "]")
	}

	// Status
	var status string
	if m.active {
		status = lipgloss.NewStyle().
			Foreground(t.Colors.Success).
			Render(" [ACTIVE]")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim)
	help := helpStyle.Render(" | Tab/←→: fields | ↑↓: values | Enter: apply | Esc: close")

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Bold(true)
	title := titleStyle.Render("Filter:")

	// Combine all parts in a single line
	line := title + " " + columnLabel + columnField + opLabel + opField + valueLabel + valueField + status + help

	// Container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Colors.Primary).
		Padding(0, 1)

	return containerStyle.Render(line)
}

func padOperator(op string) string {
	// Pad operator to consistent width
	for len(op) < 2 {
		op = op + " "
	}
	return op
}

// Match checks if a row matches the filter
func (f *Filter) Match(row []string, columns []string) bool {
	if f == nil {
		return true
	}

	// Find column index
	colIdx := -1
	for i, col := range columns {
		if col == f.Column {
			colIdx = i
			break
		}
	}

	if colIdx < 0 || colIdx >= len(row) {
		return true // Column not found, don't filter
	}

	cellValue := strings.ToLower(row[colIdx])
	filterValue := strings.ToLower(f.Value)

	switch f.Operator {
	case OpEquals:
		return cellValue == filterValue
	case OpNotEquals:
		return cellValue != filterValue
	case OpContains:
		return strings.Contains(cellValue, filterValue)
	case OpNotContains:
		return !strings.Contains(cellValue, filterValue)
	case OpGreater, OpLess, OpGreaterEq, OpLessEq:
		// Try numeric comparison
		cellNum, err1 := parseNumber(cellValue)
		filterNum, err2 := parseNumber(filterValue)
		if err1 != nil || err2 != nil {
			// Fall back to string comparison
			return compareStrings(cellValue, filterValue, f.Operator)
		}
		return compareNumbers(cellNum, filterNum, f.Operator)
	}

	return true
}

func parseNumber(s string) (float64, error) {
	// Remove commas from numbers like "37,274,000"
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}

func compareNumbers(a, b float64, op Operator) bool {
	switch op {
	case OpGreater:
		return a > b
	case OpLess:
		return a < b
	case OpGreaterEq:
		return a >= b
	case OpLessEq:
		return a <= b
	}
	return false
}

func compareStrings(a, b string, op Operator) bool {
	switch op {
	case OpGreater:
		return a > b
	case OpLess:
		return a < b
	case OpGreaterEq:
		return a >= b
	case OpLessEq:
		return a <= b
	}
	return false
}

// FilterRows filters rows based on the filter
func FilterRows(rows [][]string, columns []string, f *Filter) [][]string {
	if f == nil {
		return rows
	}

	var filtered [][]string
	for _, row := range rows {
		if f.Match(row, columns) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}
