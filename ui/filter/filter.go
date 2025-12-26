package filter

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/theme"
)

// Filter represents a filter with raw WHERE clause
type Filter struct {
	WhereClause string // Raw WHERE clause text (e.g., "name = 'John'")
}

type MapKeyMsg struct {
	Key string
}

// Model represents the filter input component
type Model struct {
	columns     []string // Available column names
	filterInput textinput.Model

	width  int
	active bool // Whether filter is actively filtering

	// Current filter
	currentFilter *Filter

	// Error handling
	errorMessage string // Error message to display
	showError    bool   // Whether to show error message

	// Word completion state
	currentWord string
	wordStart   int // Position where current word starts
	wordEnd     int // Position where current word ends
}

// New creates a new filter model
func New(columns []string) Model {
	return NewWithText(columns, "")
}

// NewWithText creates a new filter model with initial text
func NewWithText(columns []string, initialText string) Model {
	ti := textinput.New()
	ti.Placeholder = "column = value or column value"
	ti.CharLimit = 200
	ti.Width = 50
	ti.SetValue(initialText)
	ti.Blur() // Start blurred

	// Enable autocomplete suggestions for column names
	ti.ShowSuggestions = true

	// Sort columns alphabetically
	sortedColumns := make([]string, len(columns))
	copy(sortedColumns, columns)
	sort.Strings(sortedColumns)

	m := Model{
		columns:     sortedColumns,
		filterInput: ti,
		active:      false,
	}

	// Set column suggestions - textinput automatically filters based on input
	m.filterInput.SetSuggestions(sortedColumns)

	return m
}

// SetColumns updates the available columns
func (m *Model) SetColumns(columns []string) {
	// Sort columns alphabetically
	sortedColumns := make([]string, len(columns))
	copy(sortedColumns, columns)
	sort.Strings(sortedColumns)

	m.columns = sortedColumns
	// Update autocomplete suggestions
	m.filterInput.SetSuggestions(sortedColumns)
	// Update word completion
	m.updateWordCompletion()
}

// SetWidth sets the component width
func (m *Model) SetWidth(width int) {
	m.width = width
	if width > 60 {
		m.filterInput.Width = 50
	} else {
		m.filterInput.Width = 30
	}
}

// Focus focuses the filter input
func (m *Model) Focus() {
	m.filterInput.Focus()
	m.updateWordCompletion()
}

// Blur blurs the filter input
func (m *Model) Blur() {
	m.filterInput.Blur()
}

// HasText returns true if the filter input has text
func (m Model) HasText() bool {
	return m.filterInput.Value() != ""
}

// Focused returns true if the filter input is focused
func (m Model) Focused() bool {
	return m.filterInput.Focused()
}

// GetColumns returns the available columns
func (m Model) GetColumns() []string {
	return m.columns
}

// SetText sets the filter input text
func (m *Model) SetText(text string) {
	m.filterInput.SetValue(text)
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
	m.filterInput.SetValue("")
	m.currentFilter = nil
	m.active = false
	m.ClearError()
}

// SetError sets an error message for the filter
func (m *Model) SetError(msg string) {
	m.errorMessage = msg
	m.showError = true
}

// ClearError clears the error message
func (m *Model) ClearError() {
	m.errorMessage = ""
	m.showError = false
}

// HasError returns true if there's an error message
func (m Model) HasError() bool {
	return m.showError && m.errorMessage != ""
}

// GetError returns the current error message
func (m Model) GetError() string {
	return m.errorMessage
}

// Apply applies the current filter settings
func (m *Model) Apply() error {
	input := strings.TrimSpace(m.filterInput.Value())
	if input == "" {
		m.active = false
		m.currentFilter = nil
		m.ClearError()
		return nil
	}

	// Clear any previous errors when applying
	m.ClearError()

	// Store the raw WHERE clause directly - user is responsible for proper SQL syntax
	m.currentFilter = &Filter{
		WhereClause: input,
	}
	m.active = true
	return nil
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle enter to apply and blur
		if key == "enter" {
			m.Apply()
			m.Blur()
			return m, func() tea.Msg {
				return MapKeyMsg{
					Key: key,
				}
			}
		}

		// Handle escape to blur without applying
		if key == "esc" {
			m.Blur()
			return m, func() tea.Msg {
				return MapKeyMsg{
					Key: key,
				}
			}
		}

		// Handle clear
		if key == "ctrl+c" {
			m.Clear()
			return m, func() tea.Msg {
				return MapKeyMsg{
					Key: key,
				}
			}
		}

		// Update word completion before processing other keys
		m.updateWordCompletion()

		// Handle tab completion for current word
		if key == "tab" && m.currentWord != "" {
			availableSuggestions := m.filterInput.AvailableSuggestions()
			if len(availableSuggestions) > 0 {
				currentSuggestion := availableSuggestions[0] // Use first available suggestion
				// Insert the suggestion into the current word position
				text := m.filterInput.Value()
				beforeWord := text[:m.wordStart]
				afterWord := text[m.wordEnd:]
				newText := beforeWord + currentSuggestion + afterWord
				m.filterInput.SetValue(newText)

				// Move cursor to end of completed word
				newCursorPos := m.wordStart + len(currentSuggestion)
				m.filterInput.SetCursor(newCursorPos)

				// Update word completion for new position
				m.updateWordCompletion()

				return m, nil
			}
		}

		// Pass other keys to text input
		m.filterInput, cmd = m.filterInput.Update(msg)

		return m, cmd
	}

	return m, cmd
}

// View renders the filter component
func (m Model) View() string {
	t := theme.Current

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim)

	inputStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Foreground).
		Background(t.Colors.SelectionBg).
		Padding(0, 1)

	// Title and WHERE label
	titleStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Bold(true)
	title := titleStyle.Render("Filter:")
	whereLabel := labelStyle.Render(" WHERE ")

	// Input field
	inputField := inputStyle.Render(m.filterInput.View())

	// Status
	var status string
	if m.active {
		status = lipgloss.NewStyle().
			Foreground(t.Colors.Success).
			Render(" [ACTIVE]")
	}

	// Combine all parts
	line := title + whereLabel + inputField + status

	// Container - use different border style based on focus state
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(m.width-4).
		Padding(0, 1)

	if m.filterInput.Focused() {
		containerStyle = containerStyle.BorderForeground(t.Colors.BorderFocused)
	} else {
		containerStyle = containerStyle.BorderForeground(t.Colors.BorderUnfocused)
	}

	return containerStyle.Render(line)
}

func padOperator(op string) string {
	// Pad operator to consistent width
	for len(op) < 2 {
		op = op + " "
	}
	return op
}

// SetActive sets the filter active state
func (m *Model) SetActive(active bool) {
	m.active = active
}

// SetFilter sets the filter from an existing filter (used when switching tabs)
func (m *Model) SetFilter(f *Filter) {
	if f != nil {
		m.filterInput.SetValue(f.WhereClause)
		m.currentFilter = f
		m.active = true
	} else {
		m.filterInput.SetValue("")
		m.currentFilter = nil
		m.active = false
	}
}

// findWordBoundaries finds the start and end positions of the word at the cursor
func (m *Model) findWordBoundaries(text string, cursorPos int) (start, end int) {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}

	// Find word start (go backwards until we hit a non-word character)
	start = cursorPos
	for start > 0 && isWordChar(text[start-1]) {
		start--
	}

	// Find word end (go forwards until we hit a non-word character)
	end = cursorPos
	for end < len(text) && isWordChar(text[end]) {
		end++
	}

	return start, end
}

// isWordChar determines if a character is part of a word (alphanumeric, underscore)
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// updateWordCompletion updates the current word and suggestions based on cursor position
func (m *Model) updateWordCompletion() {
	text := m.filterInput.Value()
	cursorPos := m.filterInput.Position()

	// Find word boundaries
	start, end := m.findWordBoundaries(text, cursorPos)
	m.wordStart = start
	m.wordEnd = end

	// Get current word
	if start < end {
		m.currentWord = text[start:end]
	} else {
		m.currentWord = ""
	}

	// Update suggestions based on current word
	if m.currentWord != "" {
		var filteredSuggestions []string
		currentWordLower := strings.ToLower(m.currentWord)
		for _, col := range m.columns {
			if strings.HasPrefix(strings.ToLower(col), currentWordLower) {
				filteredSuggestions = append(filteredSuggestions, col)
			}
		}

		// If no exact prefix matches, also include contains matches for better UX
		if len(filteredSuggestions) == 0 {
			for _, col := range m.columns {
				if strings.Contains(strings.ToLower(col), currentWordLower) {
					filteredSuggestions = append(filteredSuggestions, col)
				}
			}
		}

		// If still no matches, show columns that start with any character from the current word
		if len(filteredSuggestions) == 0 && len(m.currentWord) > 0 {
			firstChar := string(currentWordLower[0])
			for _, col := range m.columns {
				if strings.HasPrefix(strings.ToLower(col), firstChar) {
					filteredSuggestions = append(filteredSuggestions, col)
				}
			}
		}

		// Always show suggestions when typing a word
		m.filterInput.SetSuggestions(filteredSuggestions)
	} else {
		// Show all columns when not actively typing a word
		m.filterInput.SetSuggestions(m.columns)
	}
}
