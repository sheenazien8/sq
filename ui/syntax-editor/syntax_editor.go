package syntaxeditor

import (
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/sq/ui/theme"
	"slices"
)

// HighlightedText represents a piece of text with syntax highlighting
type HighlightedText struct {
	Text  string
	Style lipgloss.Style
}

// CursorStyle represents the cursor display style
type CursorStyle int

const (
	CursorBlock CursorStyle = iota // Block cursor (vim normal mode style)
	CursorLine                     // Line/beam cursor (vim insert mode style)
)

// Model represents a syntax-highlighting text editor
type Model struct {
	content      []string      // Lines of text
	cursorX      int           // Cursor column position
	cursorY      int           // Cursor line position
	width        int           // Editor width
	height       int           // Editor height
	focused      bool          // Whether editor is focused
	lexer        chroma.Lexer  // Syntax lexer
	style        *chroma.Style // Chroma style
	scrollOffset int           // Vertical scroll offset
	charLimit    int           // Character limit (0 = unlimited)
	placeholder  string        // Placeholder text
	showBorder   bool          // Whether to show border around editor
	cursorStyle  CursorStyle   // Block or line cursor
	inVisualMode bool          // Whether in visual mode
	visualStartX int           // Visual selection start X
	visualStartY int           // Visual selection start Y
}

// New creates a new syntax-highlighting text editor
func New() Model {
	lexer := lexers.Get("sql")
	if lexer == nil {
		// Fallback to no highlighting if SQL lexer not found
		lexer = nil
	}

	style := styles.Get("monokai")
	if style == nil {
		// Fallback to default style
		style = styles.Get("default")
	}

	return Model{
		content:      []string{""},
		cursorX:      0,
		cursorY:      0,
		width:        80,
		height:       10,
		focused:      false,
		lexer:        lexer,
		style:        style,
		scrollOffset: 0,
		charLimit:    0,
		placeholder:  "",
		showBorder:   true,
		cursorStyle:  CursorBlock,
		inVisualMode: false,
		visualStartX: 0,
		visualStartY: 0,
	}
}

// SetSize sets the editor dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets whether the editor is focused
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// Focused returns whether the editor is focused
func (m Model) Focused() bool {
	return m.focused
}

// SetValue sets the editor content
func (m *Model) SetValue(value string) {
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	m.content = lines

	// Ensure we have at least one line
	if len(m.content) == 0 {
		m.content = []string{""}
	}

	// Reset cursor if it's beyond the content
	if m.cursorY >= len(m.content) {
		m.cursorY = len(m.content) - 1
	}
	if m.cursorY < 0 {
		m.cursorY = 0
	}
	if m.cursorX > len(m.content[m.cursorY]) {
		m.cursorX = len(m.content[m.cursorY])
	}
	if m.cursorX < 0 {
		m.cursorX = 0
	}

	m.adjustScroll()
}

// Value returns the current content as a string
func (m Model) Value() string {
	return strings.Join(m.content, "\n")
}

// SetLexer sets the syntax lexer
func (m *Model) SetLexer(lexer chroma.Lexer) {
	m.lexer = lexer
}

// SetStyle sets the highlighting style
func (m *Model) SetStyle(style *chroma.Style) {
	m.style = style
}

// SetPlaceholder sets the placeholder text
func (m *Model) SetPlaceholder(placeholder string) {
	m.placeholder = placeholder
}

// SetBorder sets whether to show a border around the editor
func (m *Model) SetBorder(showBorder bool) {
	m.showBorder = showBorder
}

// SetCursorStyle sets the cursor style (block or line)
func (m *Model) SetCursorStyle(style CursorStyle) {
	m.cursorStyle = style
}

// SetCharLimit sets the character limit
func (m *Model) SetCharLimit(limit int) {
	m.charLimit = limit
}

// Focus focuses the editor
func (m *Model) Focus() {
	m.focused = true
}

// Blur blurs the editor
func (m *Model) Blur() {
	m.focused = false
}

// highlightText applies syntax highlighting to text and returns styled segments
func (m Model) highlightText(text string) []HighlightedText {
	if m.lexer == nil || text == "" {
		return []HighlightedText{{Text: text, Style: lipgloss.NewStyle()}}
	}

	t := theme.Current

	// Create a style map from chroma style to lipgloss styles
	styleMap := map[chroma.TokenType]lipgloss.Style{
		chroma.Keyword:       lipgloss.NewStyle().Foreground(t.Colors.Primary).Bold(true),
		chroma.KeywordType:   lipgloss.NewStyle().Foreground(t.Colors.Primary),
		chroma.Literal:       lipgloss.NewStyle().Foreground(t.Colors.Success),
		chroma.LiteralString: lipgloss.NewStyle().Foreground(t.Colors.Success),
		chroma.LiteralNumber: lipgloss.NewStyle().Foreground(t.Colors.Warning),
		chroma.Comment:       lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim).Italic(true),
		chroma.Name:          lipgloss.NewStyle().Foreground(t.Colors.Foreground),
		chroma.NameFunction:  lipgloss.NewStyle().Foreground(t.Colors.Primary),
		chroma.NameBuiltin:   lipgloss.NewStyle().Foreground(t.Colors.Primary),
		chroma.Operator:      lipgloss.NewStyle().Foreground(t.Colors.Warning),
		chroma.Punctuation:   lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim),
	}

	iterator, err := m.lexer.Tokenise(nil, text)
	if err != nil {
		// Return plain text if tokenization fails
		return []HighlightedText{{Text: text, Style: lipgloss.NewStyle()}}
	}

	var segments []HighlightedText
	for token := iterator(); token != chroma.EOF; token = iterator() {
		style, exists := styleMap[token.Type]
		if !exists {
			style = lipgloss.NewStyle().Foreground(t.Colors.Foreground)
		}
		segments = append(segments, HighlightedText{
			Text:  token.Value,
			Style: style,
		})
	}

	return segments
}

// isEmpty returns true if the editor has no content
func (m Model) isEmpty() bool {
	return len(m.content) == 1 && m.content[0] == ""
}

// renderLine renders a single line with syntax highlighting
func (m Model) renderLine(line string, lineY int, isCursorLine bool, cursorX int) string {
	if line == "" {
		line = " "
	}
	runes := []rune(line)
	segments := m.highlightText(line)

	// Build a map of position -> style for quick lookup
	positionStyles := make(map[int]lipgloss.Style)
	segPos := 0
	for _, segment := range segments {
		segRunes := []rune(segment.Text)
		for i := 0; i < len(segRunes); i++ {
			positionStyles[segPos+i] = segment.Style
		}
		segPos += len(segRunes)
	}

	var renderedParts []string

	for pos, r := range runes {
		// Get style for this position
		style, found := positionStyles[pos]
		if !found {
			style = lipgloss.NewStyle()
		}

		// Apply selection style if selected
		if m.isPositionSelected(lineY, pos) {
			t := theme.Current
			style = style.Background(t.Colors.SelectionBg).Foreground(t.Colors.Foreground)
		}

		// Handle cursor
		if isCursorLine && pos == cursorX {
			if m.focused && m.cursorStyle == CursorBlock {
				t := theme.Current
				cursorStyle := style.Copy().
					Underline(true).
					Background(t.Colors.Primary).
					Foreground(t.Colors.Background)
				renderedParts = append(renderedParts, cursorStyle.Render(string(r)))
			} else if m.focused && m.cursorStyle == CursorLine {
				t := theme.Current
				cursorChar := lipgloss.NewStyle().
					Foreground(t.Colors.Primary).
					Bold(true).
					Render("|")
				renderedParts = append(renderedParts, cursorChar)
				renderedParts = append(renderedParts, style.Render(string(r)))
			} else {
				renderedParts = append(renderedParts, style.Render(string(r)))
			}
		} else {
			renderedParts = append(renderedParts, style.Render(string(r)))
		}
	}

	// If cursor at end of line
	if isCursorLine && cursorX >= len(runes) && m.focused {
		t := theme.Current
		if m.cursorStyle == CursorBlock {
			cursorStyle := lipgloss.NewStyle().
				Underline(true).
				Background(t.Colors.Primary).
				Foreground(t.Colors.Background)
			renderedParts = append(renderedParts, cursorStyle.Render(" "))
		} else if m.cursorStyle == CursorLine {
			cursorChar := lipgloss.NewStyle().
				Foreground(t.Colors.Primary).
				Bold(true).
				Render("|")
			renderedParts = append(renderedParts, cursorChar)
		}
	}

	return strings.Join(renderedParts, "")
}

// View renders the editor
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	t := theme.Current
	var lines []string

	// Show placeholder if editor is empty and not focused
	if m.isEmpty() && !m.focused && m.placeholder != "" {
		placeholderStyle := lipgloss.NewStyle().Foreground(t.Colors.ForegroundDim)
		placeholderLines := strings.Split(m.placeholder, "\n")
		for i := 0; i < m.height && i < len(placeholderLines); i++ {
			line := placeholderStyle.Render(placeholderLines[i])
			if lipgloss.Width(line) < m.width {
				line += strings.Repeat(" ", m.width-lipgloss.Width(line))
			}
			lines = append(lines, line)
		}
		// Fill remaining height
		for len(lines) < m.height {
			lines = append(lines, strings.Repeat(" ", m.width))
		}
		return strings.Join(lines, "\n")
	}

	// Calculate visible line range
	startLine := m.scrollOffset
	endLine := min(startLine+m.height, len(m.content))

	// Ensure we don't go out of bounds
	if startLine < 0 {
		startLine = 0
	}
	if endLine < startLine {
		endLine = startLine
	}

	// Render visible lines
	for i := startLine; i < endLine && i < len(m.content); i++ {
		line := m.content[i]
		isCursorLine := (i == m.cursorY)
		renderedLine := m.renderLine(line, i, isCursorLine, m.cursorX)

		// Pad line to editor width
		if lipgloss.Width(renderedLine) < m.width {
			padding := strings.Repeat(" ", m.width-lipgloss.Width(renderedLine))
			renderedLine += padding
		}

		lines = append(lines, renderedLine)
	}

	// Fill remaining height with empty lines
	for len(lines) < m.height {
		emptyLine := strings.Repeat(" ", m.width)
		lines = append(lines, emptyLine)
	}

	content := strings.Join(lines, "\n")

	// Apply border if enabled and focused
	if m.showBorder && m.focused {
		borderStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Colors.Primary).
			Width(m.width).
			Height(m.height)
		return borderStyle.Render(content)
	}

	return content
}

// isPositionSelected checks if a position is within the visual selection
func (m Model) isPositionSelected(y, x int) bool {
	if !m.inVisualMode {
		return false
	}
	startY, startX := m.visualStartY, m.visualStartX
	endY, endX := m.cursorY, m.cursorX

	// Normalize start < end
	if startY > endY || (startY == endY && startX > endX) {
		startY, endY = endY, startY
		startX, endX = endX, startX
	}

	if y < startY || y > endY {
		return false
	}
	if y == startY && x < startX {
		return false
	}
	if y == endY && x >= endX {
		return false
	}
	return true
}

// adjustScroll adjusts the scroll offset to keep cursor visible
func (m *Model) adjustScroll() {
	if m.cursorY < m.scrollOffset {
		m.scrollOffset = m.cursorY
	} else if m.cursorY >= m.scrollOffset+m.height {
		m.scrollOffset = m.cursorY - m.height + 1
	}

	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// Update handles input messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		// Ensure cursor bounds are valid
		if m.cursorY < 0 {
			m.cursorY = 0
		}
		if m.cursorY >= len(m.content) {
			m.cursorY = len(m.content) - 1
		}
		if m.cursorX < 0 {
			m.cursorX = 0
		}
		if m.cursorX > len(m.content[m.cursorY]) {
			m.cursorX = len(m.content[m.cursorY])
		}

		// Handle both string-based keys and key types
		keyStr := msg.String()
		keyType := msg.Type

		switch {
		case keyStr == "left" || keyType == tea.KeyLeft:
			if m.cursorX > 0 {
				m.cursorX--
			} else if m.cursorY > 0 {
				// Move to end of previous line
				m.cursorY--
				m.cursorX = len(m.content[m.cursorY])
			}
		case keyStr == "right" || keyType == tea.KeyRight:
			currentLine := m.content[m.cursorY]
			if m.cursorX < len(currentLine) {
				m.cursorX++
			} else if m.cursorY < len(m.content)-1 {
				// Move to start of next line
				m.cursorY++
				m.cursorX = 0
			}
		case keyStr == "up" || keyType == tea.KeyUp:
			if m.cursorY > 0 {
				m.cursorY--
				// Adjust cursor X if new line is shorter
				if m.cursorX > len(m.content[m.cursorY]) {
					m.cursorX = len(m.content[m.cursorY])
				}
			}
		case keyStr == "down" || keyType == tea.KeyDown:
			if m.cursorY < len(m.content)-1 {
				m.cursorY++
				// Adjust cursor X if new line is shorter
				if m.cursorX > len(m.content[m.cursorY]) {
					m.cursorX = len(m.content[m.cursorY])
				}
			}
		case keyStr == "enter" || keyType == tea.KeyEnter:
			// Split line at cursor with auto-indentation
			currentLine := m.content[m.cursorY]
			before := currentLine[:m.cursorX]
			after := currentLine[m.cursorX:]

			// Calculate indentation from current line
			indent := ""
			for _, ch := range currentLine {
				if ch == ' ' || ch == '\t' {
					indent += string(ch)
				} else {
					break
				}
			}

			// Check if we should add extra indentation (after certain SQL keywords)
			trimmedBefore := strings.TrimSpace(strings.ToUpper(before))
			extraIndent := ""
			if strings.HasSuffix(trimmedBefore, "SELECT") ||
				strings.HasSuffix(trimmedBefore, "FROM") ||
				strings.HasSuffix(trimmedBefore, "WHERE") ||
				strings.HasSuffix(trimmedBefore, "AND") ||
				strings.HasSuffix(trimmedBefore, "OR") ||
				strings.HasSuffix(trimmedBefore, "JOIN") ||
				strings.HasSuffix(trimmedBefore, "ON") ||
				strings.HasSuffix(trimmedBefore, "SET") ||
				strings.HasSuffix(trimmedBefore, "VALUES") ||
				strings.HasSuffix(trimmedBefore, "(") ||
				strings.HasSuffix(trimmedBefore, ",") {
				extraIndent = "  "
			}

			// Insert new line with indentation
			m.content = append(m.content[:m.cursorY+1], m.content[m.cursorY:]...)
			m.content[m.cursorY] = before
			m.content[m.cursorY+1] = indent + extraIndent + strings.TrimLeft(after, " \t")

			m.cursorY++
			m.cursorX = len(indent) + len(extraIndent)
		case keyStr == "backspace" || keyType == tea.KeyBackspace:
			if m.cursorX > 0 {
				// Delete character before cursor
				currentLine := m.content[m.cursorY]
				m.content[m.cursorY] = currentLine[:m.cursorX-1] + currentLine[m.cursorX:]
				m.cursorX--
			} else if m.cursorY > 0 {
				// Join with previous line
				currentLine := m.content[m.cursorY]
				m.cursorX = len(m.content[m.cursorY-1])
				m.content[m.cursorY-1] += currentLine
				m.content = slices.Delete(m.content, m.cursorY, m.cursorY+1)
				m.cursorY--
			}
		case keyStr == "delete" || keyType == tea.KeyDelete:
			currentLine := m.content[m.cursorY]
			if m.cursorX < len(currentLine) {
				// Delete character at cursor
				m.content[m.cursorY] = currentLine[:m.cursorX] + currentLine[m.cursorX+1:]
			} else if m.cursorY < len(m.content)-1 {
				// Join with next line
				nextLine := m.content[m.cursorY+1]
				m.content[m.cursorY] += nextLine
				m.content = slices.Delete(m.content, m.cursorY+1, m.cursorY+2)
			}
		case keyStr == "home" || keyType == tea.KeyHome:
			m.cursorX = 0
		case keyStr == "end" || keyType == tea.KeyEnd:
			m.cursorX = len(m.content[m.cursorY])
		case keyType == tea.KeyCtrlRight:
			// Move to next word
			currentLine := m.content[m.cursorY]
			for m.cursorX < len(currentLine) {
				if currentLine[m.cursorX] == ' ' {
					m.cursorX++
					break
				}
				m.cursorX++
			}
		case keyType == tea.KeyCtrlLeft:
			// Move to previous word
			for m.cursorX > 0 {
				m.cursorX--
				if m.cursorX > 0 && m.content[m.cursorY][m.cursorX-1] == ' ' {
					break
				}
			}
		case keyType == tea.KeyCtrlZ:
			// Undo - not implemented for now
		default:
			// Insert character(s) - allows paste to work
			if len(keyStr) > 0 {
				if m.charLimit == 0 || utf8.RuneCountInString(m.Value())+utf8.RuneCountInString(keyStr) <= m.charLimit {
					currentLine := m.content[m.cursorY]
					m.content[m.cursorY] = currentLine[:m.cursorX] + keyStr + currentLine[m.cursorX:]
					m.cursorX += len(keyStr)
				}
			}
		}

		m.adjustScroll()
	}

	return m, nil
}

// CursorStart moves cursor to the beginning of the text
func (m *Model) CursorStart() {
	m.cursorX = 0
	m.cursorY = 0
	m.adjustScroll()
}

// CursorEnd moves cursor to the end of the text
func (m *Model) CursorEnd() {
	m.cursorY = len(m.content) - 1
	m.cursorX = len(m.content[m.cursorY])
	m.adjustScroll()
}

// CursorY returns the current cursor Y position
func (m Model) CursorY() int {
	return m.cursorY
}

// CursorX returns the current cursor X position
func (m Model) CursorX() int {
	return m.cursorX
}

// SetVisualMode sets whether the editor is in visual mode
func (m *Model) SetVisualMode(visual bool) {
	m.inVisualMode = visual
}

// SetVisualStart sets the start position for visual selection
func (m *Model) SetVisualStart(x, y int) {
	m.visualStartX = x
	m.visualStartY = y
}
