package modalcolumnvisibility

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/table"
	"github.com/sheenazien8/sq/ui/theme"
)

// ColumnVisibilityToggleMsg is sent when user confirms column visibility changes
type ColumnVisibilityToggleMsg struct {
	VisibilityMap map[int]bool // column index -> is_visible
}

// ColumnVisibilityContent implements modal.Content for column visibility management
type ColumnVisibilityContent struct {
	columns            []table.Column
	cursor             int
	scrollOffset       int
	width              int
	visibleLines       int
	closed             bool
	originalVisibility map[int]bool // Store original state for cancel
	result             modal.Result
	toggleResult       *ColumnVisibilityToggleMsg

	// Search functionality
	searchText      string
	searchMode      bool  // True if user is typing in search
	filteredIndices []int // Indices of columns matching search
}

// New creates a new column visibility content
func New() *ColumnVisibilityContent {
	return &ColumnVisibilityContent{
		columns:            []table.Column{},
		cursor:             0,
		scrollOffset:       0,
		width:              50,
		visibleLines:       15,
		closed:             false,
		originalVisibility: make(map[int]bool),
		result:             modal.ResultNone,
	}
}

// SetColumns sets the columns to display
func (c *ColumnVisibilityContent) SetColumns(columns []table.Column) {
	c.columns = columns
	c.cursor = 0
	c.scrollOffset = 0
	c.searchText = ""
	c.originalVisibility = make(map[int]bool)
	for i := range columns {
		c.originalVisibility[i] = !columns[i].Hidden
	}
	c.updateFilteredColumns()
}

// SetWidth implements modal.Content
func (c *ColumnVisibilityContent) SetWidth(width int) {
	// Limit width to a reasonable size (max 50 chars for compact modal)
	if width > 50 {
		width = 50
	}
	c.width = width
	// Show up to 15 lines (columns) at once, or less if fewer columns exist
	c.visibleLines = min(15, len(c.columns))
	if c.visibleLines < 5 {
		c.visibleLines = 5
	}
}

// GetVisibility returns the current visibility state
func (c *ColumnVisibilityContent) GetVisibility() map[int]bool {
	visibility := make(map[int]bool)
	for i := range c.columns {
		visibility[i] = !c.columns[i].Hidden
	}
	return visibility
}

// updateFilteredColumns updates the filtered list based on search text
func (c *ColumnVisibilityContent) updateFilteredColumns() {
	c.filteredIndices = []int{}
	searchLower := strings.ToLower(c.searchText)

	if searchLower == "" {
		// No search, show all columns
		for i := range c.columns {
			c.filteredIndices = append(c.filteredIndices, i)
		}
	} else {
		// Filter columns by name
		for i, col := range c.columns {
			if strings.Contains(strings.ToLower(col.Title), searchLower) {
				c.filteredIndices = append(c.filteredIndices, i)
			}
		}
	}

	// Reset cursor when filtering
	c.cursor = 0
	c.scrollOffset = 0
}

// Update implements modal.Content
func (c *ColumnVisibilityContent) Update(msg tea.Msg) (modal.Content, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode text input
		if c.searchMode {
			switch msg.String() {
			case "backspace":
				if len(c.searchText) > 0 {
					c.searchText = c.searchText[:len(c.searchText)-1]
					c.updateFilteredColumns()
				}
			case "enter", "escape":
				// Exit search mode
				c.searchMode = false
			default:
				// Add character to search text
				if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] < 127 {
					c.searchText += msg.String()
					c.updateFilteredColumns()
				}
			}
			return c, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case "/":
			// Enter search mode
			c.searchMode = true
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
				if c.cursor < c.scrollOffset {
					c.scrollOffset = c.cursor
				}
			}
		case "down", "j":
			if c.cursor < len(c.filteredIndices)-1 {
				c.cursor++
				if c.cursor >= c.scrollOffset+c.visibleLines {
					c.scrollOffset = c.cursor - c.visibleLines + 1
				}
			}
		case " ":
			// Toggle current column visibility
			if c.cursor >= 0 && c.cursor < len(c.filteredIndices) {
				originalIdx := c.filteredIndices[c.cursor]
				c.columns[originalIdx].Hidden = !c.columns[originalIdx].Hidden
			}
		case "a", "A":
			// Toggle all filtered columns visibility
			allVisible := true
			for _, idx := range c.filteredIndices {
				if c.columns[idx].Hidden {
					allVisible = false
					break
				}
			}
			for _, idx := range c.filteredIndices {
				c.columns[idx].Hidden = allVisible
			}
		case "enter":
			// Confirm and close
			c.closed = true
			c.result = modal.ResultSubmit
			c.toggleResult = &ColumnVisibilityToggleMsg{
				VisibilityMap: c.GetVisibility(),
			}
			return c, func() tea.Msg {
				return *c.toggleResult
			}
		case "esc":
			// Cancel - restore original visibility
			for i := range c.columns {
				c.columns[i].Hidden = !c.originalVisibility[i]
			}
			c.closed = true
			c.result = modal.ResultCancel
		case "ctrl+c", "q":
			// Cancel - restore original visibility
			for i := range c.columns {
				c.columns[i].Hidden = !c.originalVisibility[i]
			}
			c.closed = true
			c.result = modal.ResultCancel
		}
	}

	return c, nil
}

// View implements modal.Content
func (c *ColumnVisibilityContent) View() string {
	t := theme.Current

	// Build search bar
	searchDisplay := "Search: " + c.searchText
	if c.searchMode {
		searchDisplay += "|"
	}
	searchStyle := lipgloss.NewStyle().
		Foreground(t.Colors.Primary)
	if c.searchMode {
		searchStyle = searchStyle.Bold(true)
	}
	searchBar := searchStyle.Render(searchDisplay)

	// Build column list from filtered columns
	var columnLines []string
	endIdx := min(c.scrollOffset+c.visibleLines, len(c.filteredIndices))

	// Calculate max column name width - be conservative to fit in terminal
	maxWidth := 30
	if c.width > 10 {
		maxWidth = c.width - 8 // Leave room for checkbox and padding
	}

	for i := c.scrollOffset; i < endIdx; i++ {
		originalIdx := c.filteredIndices[i]
		col := c.columns[originalIdx]
		checkbox := "[ ]"
		if !col.Hidden {
			checkbox = "[✓]"
		}

		colName := col.Title
		// Truncate column name if too long
		if len(colName) > maxWidth {
			colName = colName[:maxWidth-3] + "..."
		}
		line := checkbox + " " + colName

		// Highlight current cursor - don't add extra padding
		if i == c.cursor {
			line = t.SidebarSelected.Render(line)
		}

		columnLines = append(columnLines, line)
	}

	// Build stats text - without background color
	visibleCount := 0
	for _, col := range c.columns {
		if !col.Hidden {
			visibleCount++
		}
	}
	totalCount := len(c.columns)
	filteredCount := len(c.filteredIndices)
	statsText := intToStr(visibleCount) + "/" + intToStr(totalCount) + " visible"
	if c.searchText != "" {
		statsText += " (" + intToStr(filteredCount) + " matched)"
	}
	statsStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Padding(1, 0, 0, 0)
	stats := statsStyle.Render(statsText)

	// Build help text - without background color, matching other modals
	helpStr := "/:search | j/k:nav | Spc:tog | A:all | Enter:ok | Esc"
	helpStyle := lipgloss.NewStyle().
		Foreground(t.Colors.ForegroundDim).
		Padding(1, 0, 0, 0)
	helpText := helpStyle.Render(helpStr)

	// Join content
	content := strings.Join(columnLines, "\n")
	if len(columnLines) < c.visibleLines {
		// Pad with empty lines
		for i := len(columnLines); i < c.visibleLines; i++ {
			content += "\n"
		}
	}

	// Wrap with width constraint using lipgloss
	innerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		searchBar,
		content,
		stats,
		helpText,
	)

	// Apply strict width limit to prevent overflow
	if c.width > 0 {
		return lipgloss.NewStyle().MaxWidth(c.width).Render(innerContent)
	}

	return innerContent
}

// Result implements modal.Content
func (c *ColumnVisibilityContent) Result() modal.Result {
	return c.result
}

// ShouldClose implements modal.Content
func (c *ColumnVisibilityContent) ShouldClose() bool {
	return c.closed
}

// Reset resets the modal state for reopening
func (c *ColumnVisibilityContent) Reset() {
	c.closed = false
	c.result = modal.ResultNone
	c.toggleResult = nil
	c.cursor = 0
	c.scrollOffset = 0
	c.searchText = ""
	c.searchMode = false
	c.updateFilteredColumns()
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte(n%10) + '0'}, digits...)
		n /= 10
	}
	return string(digits)
}
