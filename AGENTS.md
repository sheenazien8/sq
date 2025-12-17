# AGENTS.md

## Project Overview

**db-client-tui** is a terminal-based database client built with the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework for Go. The project implements a multi-pane layout with keyboard-driven navigation following vim-like patterns.

**Status**: Early development - currently displays mock data. Planned support for PostgreSQL, MySQL, and SQLite.

**Technology Stack**:
- Go 1.22
- Bubble Tea v0.25.0 (TUI framework)
- Bubbles v0.17.1 (TUI components)
- Lipgloss v0.9.1 (styling)

## Essential Commands

### Running the Application
```bash
go run .
```

### Building
```bash
go build -o db-client-tui
./db-client-tui
```

### Dependencies
```bash
go mod download
go mod tidy
```

### Debugging
- Debug logs are written to `debug.log` in the current directory (configured in `main.go:12`)
- Use `tea.LogToFile()` for debugging TUI applications

## Project Structure

```
db-client-tui/
├── main.go              # Entry point, initializes Bubble Tea program
├── app/                 # Main application logic (Bubble Tea Model-View-Update)
│   ├── init.go          # Init() - initialization command
│   ├── model.go         # Model struct and constructor
│   ├── update.go        # Update() - handles messages and input
│   └── view.go          # View() - renders the UI
├── config/              # Application configuration
│   └── config.go        # Config loading/saving (~/.config/db-client-tui/config.json)
└── ui/                  # UI components (separate Bubble Tea models)
    ├── sidebar/         # Database list sidebar
    ├── table/           # Scrollable table widget
    ├── filter/          # Filter input component
    ├── modal/           # Modal dialogs (e.g., exit confirmation)
    ├── theme/           # Theme system and color definitions
    ├── main/            # (future) Main record view
    └── detail/          # (future) Detail pane
```

## Architecture Patterns

### Bubble Tea Model-View-Update (MVU)

The app follows Bubble Tea's MVU architecture:

1. **Model** (`app/model.go`): Application state
   - Contains sub-models for each UI component (sidebar, table, filter, modal)
   - Tracks focus state, dimensions, theme, and configuration
   - Initialized with `app.New()`

2. **Update** (`app/update.go`): Event handling
   - Processes `tea.Msg` (keyboard, window resize, etc.)
   - Routes messages to focused component
   - Updates state and returns new model + commands
   - Pattern: `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`

3. **View** (`app/view.go`): Rendering
   - Composes sub-component views using `lipgloss.JoinHorizontal/Vertical`
   - Applies styles based on focus state
   - Pattern: `func (m Model) View() string`

4. **Init** (`app/init.go`): Initial commands
   - Currently returns `nil` (no initial async operations)
   - Pattern: `func (m Model) Init() tea.Cmd`

### Component Structure

Each UI component (sidebar, table, filter, modal) follows the same pattern:
- Separate package under `ui/`
- Own `Model` struct
- Own `Update(msg tea.Msg)` and `View()` methods
- Exported methods for parent to control: `SetSize()`, `SetFocused()`, etc.

### Focus Management

Focus is managed via the `Focus` type (iota) in `app/model.go`:
```go
const (
    FocusSidebar Focus = iota
    FocusMain
    FocusFilter
    FocusModal
)
```

Only the focused component receives keyboard input. Tab key switches focus between sidebar and main table.

### State Updates

**Important**: Bubble Tea models are immutable. Always return a new/updated model:
```go
// CORRECT
m.Focus = FocusMain
return m, cmd

// CORRECT (with pointer receiver for helper methods)
func (m *Model) SetSize(w, h int) {
    m.width = w
    m.height = h
}
```

## Code Conventions

### File Organization
- One component per package under `ui/`
- Split app logic into `init.go`, `model.go`, `update.go`, `view.go`
- Keep `main.go` minimal (program initialization only)

### Naming Conventions
- **Models**: `Model` struct in each package
- **Constructors**: `New()` returns initialized Model
- **Setters**: `Set*()` prefix (e.g., `SetSize`, `SetFocused`)
- **Getters**: Direct name (e.g., `Focused()`, `Cursor()`, `SelectedRow()`)
- **Type Re-exports**: Used for convenience (see `app/model.go:12-14`)

### Indentation & Style
- **Indentation**: 4 spaces (enforced by `.editorconfig`)
- **Line endings**: LF (Unix-style)
- **Final newline**: Required
- **Trailing whitespace**: Trimmed
- Standard Go formatting (use `go fmt`)

### Import Organization
```go
import (
    "stdlib packages"
    
    "third-party packages"
    
    "local packages"
)
```

Example from `app/model.go`:
```go
import (
    "github.com/sheenazien8/db-client-tui/config"
    "github.com/sheenazien8/db-client-tui/ui/filter"
    // ...
)
```

### Custom Helper Functions

The codebase implements custom helper functions instead of using standard library:
- `intToStr(n int) string` - integer to string conversion (in `table/table.go:346`, `sidebar/database.go:248`)
- `max()`, `min()` - min/max helpers (Go 1.22 doesn't require these anymore, but they're used here)
- `truncateOrPad()` - string truncation with width awareness (in `table/table.go:319`)

**When adding code**: Check if similar helpers exist before adding duplicates.

## Styling & Theming

### Theme System (`ui/theme/`)

- **Global theme**: `theme.Current` is the active theme
- **Switching themes**: `theme.SetTheme(theme.GetThemeByName(name))`
- **Available themes**: `theme.GetAvailableThemes()` returns slice of theme names
  - default, dracula, nord, gruvbox, tokyo-night, catppuccin, monokai

### Theme Structure
```go
type Theme struct {
    Name   string
    Colors Colors  // Color palette
    
    // Pre-built styles
    Header, Footer, Title          lipgloss.Style
    BorderFocused, BorderUnfocused lipgloss.Style
    TableHeader, TableCell, TableSelected lipgloss.Style
    SidebarTitle, SidebarItem, SidebarSelected, SidebarActive lipgloss.Style
    StatusBar lipgloss.Style
}
```

### Using Themes in Components
```go
t := theme.Current

style := t.TableHeader.Render("My Header")
borderStyle := t.BorderFocused  // When focused
borderStyle := t.BorderUnfocused // When not focused
```

### Lipgloss Patterns

**Joining layouts**:
```go
lipgloss.JoinHorizontal(lipgloss.Top, view1, view2)    // Side by side
lipgloss.JoinVertical(lipgloss.Left, view1, view2)     // Stacked
```

**Dynamic styling**:
```go
style := lipgloss.NewStyle().
    Width(100).
    Height(50).
    Padding(1, 2).
    Foreground(t.Colors.Foreground).
    Background(t.Colors.Background)
```

**Width/Height calculation**:
```go
lipgloss.Width(renderedString)   // Get display width
lipgloss.Height(renderedString)  // Get line count
```

## Configuration

### Config File Location
`~/.config/db-client-tui/config.json`

### Config Structure
```go
type Config struct {
    Theme string `json:"theme"`
}
```

### Loading/Saving
```go
cfg, _ := config.Load()           // Loads or returns DefaultConfig()
cfg.SetTheme("dracula")
_ = cfg.Save()                     // Creates config dir if needed
```

**Note**: Config is loaded once at startup (`app/model.go:64`) and saved when theme changes (`app/update.go:127`).

## Keyboard Shortcuts

### Global
- `q` / `Ctrl+C` - Show exit modal
- `Tab` - Switch focus (Sidebar ↔ Main table)
- `T` - Cycle themes
- `C` - Clear active filter

### Sidebar (when focused)
- `j` / `↓` - Move down
- `k` / `↑` - Move up
- `Home` - Jump to first item
- `End` - Jump to last item
- `Enter` - Select database (marks as active)

### Table (when focused)
- `j` / `↓` - Move down one row
- `k` / `↑` - Move up one row
- `h` / `←` - Scroll columns left
- `l` / `→` - Scroll columns right
- `H` - Jump to first column
- `L` - Jump to last column
- `PgUp` / `PgDn` - Page up/down
- `Home` / `End` - Jump to first/last row
- `/` / `f` - Open filter dialog

### Filter Dialog (when open)
- `Tab` / `→` / `l` - Next field
- `Shift+Tab` / `←` / `h` - Previous field
- `j` / `↓` - Next option (column/operator)
- `k` / `↑` - Previous option (column/operator)
- `Enter` - Apply filter and close
- `Esc` - Close without applying
- `Ctrl+C` - Clear filter

### Modal (when visible)
- `←` / `→` / `h` / `l` / `Tab` - Switch button
- `Enter` - Confirm selection
- `y` / `Y` - Yes
- `n` / `N` / `Esc` - No

## Important Patterns & Gotchas

### Window Size Handling

All components must handle `tea.WindowSizeMsg`:
```go
case tea.WindowSizeMsg:
    m.TerminalWidth = msg.Width
    m.TerminalHeight = msg.Height
    // Calculate component dimensions
    // Initialize lazy-loaded components
    if !m.initialized {
        // First-time setup
        m.initialized = true
    }
```

**Critical**: Components like table are initialized in `WindowSizeMsg` handler, not in `New()`, because dimensions are unknown at construction time.

### Lazy Initialization

The app uses a `initialized` flag (`app/model.go:53`) to defer setup until first window size message:
```go
if !m.initialized {
    m.columns, m.allRows = getTableData()
    m.Main = table.New(m.columns, m.allRows)
    m.Main.SetSize(tableWidth, tableHeight)
    m.initialized = true
}
```

**When adding features**: If they depend on dimensions, initialize in `WindowSizeMsg` handler.

### Modal/Overlay Rendering

Modals render as full-screen overlays:
```go
// In View()
if m.ExitModal.Visible() {
    return m.ExitModal.View()  // Early return, modal takes over entire screen
}
```

**Pattern**: Check overlay visibility first, return early if visible, otherwise render normal layout.

### Message Routing

Messages are routed to focused component:
```go
switch msg := msg.(type) {
case tea.KeyMsg:
    if m.ExitModal.Visible() {
        m.ExitModal, cmd = m.ExitModal.Update(msg)
        return m, tea.Batch(cmds...)
    }
    
    if m.Filter.Visible() {
        m.Filter, cmd = m.Filter.Update(msg)
        return m, tea.Batch(cmds...)
    }
    
    switch msg.String() {
    case "global-shortcut":
        // Handle global shortcuts
    default:
        if m.Focus == FocusSidebar {
            m.Sidebar, cmd = m.Sidebar.Update(msg)
        } else {
            m.Main, cmd = m.Main.Update(msg)
        }
    }
}
```

**Pattern**: Check overlays (modals, dialogs) first, then global shortcuts, then route to focused component.

### Filter Application

Filters work on a copy of data:
```go
// app/model.go stores both:
allRows     []table.Row  // Original data
Main        table.Model  // Table with potentially filtered data

// When applying filter (app/update.go:150):
func (m Model) applyFilter() Model {
    f := m.Filter.GetFilter()
    if f == nil {
        m.Main.SetRows(m.allRows)  // Reset to all data
    } else {
        filtered := filter.FilterRows(rows, columnNames, f)
        m.Main.SetRows(filtered)   // Show filtered subset
    }
    return m
}
```

**Important**: Always keep original data separate from filtered view.

### Scrolling Implementation

Both table and sidebar implement viewport-style scrolling:
```go
// Cursor position (which item is selected)
cursor int

// Scroll offset (first visible item)
offset int

// Calculate visible range:
visible := m.visibleItems()  // How many fit on screen
endIdx := min(m.offset + visible, len(m.items))

// Render only visible items:
for i := m.offset; i < endIdx; i++ {
    // Render item[i]
}

// Auto-scroll when cursor moves off screen:
if m.cursor < m.offset {
    m.offset = m.cursor  // Scroll up
}
if m.cursor >= m.offset + visible {
    m.offset = m.cursor - visible + 1  // Scroll down
}
```

**Pattern**: Track cursor separately from scroll offset. Auto-adjust offset when cursor moves out of viewport.

### Type Aliases for Convenience

The app uses type aliases to simplify imports:
```go
// In app/model.go:
type TableColumn = table.Column
type TableRow = table.Row
```

This lets other code use `app.TableColumn` instead of importing `table` package.

### Component Communication

Parent updates child via exported methods:
```go
m.Main.SetSize(width, height)
m.Main.SetFocused(true)
m.Main.SetRows(filteredRows)
```

Child notifies parent via return values (not via callbacks):
```go
// In filter component:
if !m.Filter.Visible() {
    // Parent checks visibility in next Update cycle
    // Parent reads state via m.Filter.GetFilter()
}
```

**No callbacks**: Components don't call parent methods. Parent polls child state.

### Boolean Flags for State

Common pattern for toggling states:
```go
type Model struct {
    visible    bool  // Is component visible?
    focused    bool  // Is component focused?
    active     bool  // Is feature active? (e.g., filter applied)
    initialized bool // Has component been initialized?
}
```

Exported getter methods:
```go
func (m Model) Visible() bool { return m.visible }
func (m Model) Active() bool { return m.active && m.currentFilter != nil }
```

### Text Input Handling

Using Bubbles' `textinput` component (see `ui/filter/filter.go:59`):
```go
import "github.com/charmbracelet/bubbles/textinput"

ti := textinput.New()
ti.Placeholder = "enter value"
ti.CharLimit = 100
ti.Width = 30

// Focus/blur controls cursor visibility:
ti.Focus()  // Show cursor, accept input
ti.Blur()   // Hide cursor, ignore input

// In Update():
if m.focusField == FocusValue {
    m.valueInput, cmd = m.valueInput.Update(msg)
}
```

**Important**: Text input only processes messages when focused.

### String Truncation with Unicode Awareness

Don't use simple slice (`s[:n]`). Use `lipgloss.Width()` for proper width calculation:
```go
func truncateOrPad(s string, width int) string {
    currentWidth := lipgloss.Width(s)
    if currentWidth > width {
        runes := []rune(s)
        truncated := ""
        w := 0
        for _, r := range runes {
            rw := lipgloss.Width(string(r))
            if w+rw > width-3 {
                break
            }
            truncated += string(r)
            w += rw
        }
        return truncated + "..."
    }
    return s + strings.Repeat(" ", width-currentWidth)
}
```

**Why**: Unicode characters (emoji, CJK) can have width != 1. Lipgloss handles this correctly.

## Data Flow

### Current Implementation (Mock Data)

1. App starts → `main.go` creates `app.New()`
2. First `WindowSizeMsg` → `getTableData()` returns hardcoded city data
3. User interacts → focus switches, filters apply, theme changes
4. All data is static (see `app/update.go:187-253`)

### Future Implementation (Real Database)

Planned pattern for database queries:
1. User selects database in sidebar → sends query command
2. Query runs asynchronously → returns `tea.Msg` with results
3. Update handler receives results → updates table rows
4. View re-renders with new data

**Note**: Bubble Tea uses message-passing for async operations. Don't block Update().

## Testing

**Current Status**: No tests exist yet.

**Recommended Approach** (when adding tests):
- Test pure functions (helpers, filtering logic)
- Test state transitions in Update()
- Use table-driven tests (Go convention)
- Example testable code: `filter.FilterRows()`, `filter.Match()`

## Common Tasks

### Adding a New UI Component

1. Create package under `ui/mycomponent/`
2. Define `Model` struct
3. Implement `New()` constructor
4. Implement `Update(msg tea.Msg) (Model, tea.Cmd)`
5. Implement `View() string`
6. Add exported methods: `SetSize()`, `SetFocused()`, getters
7. Add to app model (`app/model.go`)
8. Route messages in `app/update.go`
9. Compose view in `app/view.go`

### Adding a New Theme

1. Create color palette in `ui/theme/theme.go`:
   ```go
   func MyTheme() *Theme {
       return buildStyles("my-theme", Colors{
           Background: lipgloss.Color("#..."),
           // ... define all colors
       })
   }
   ```
2. Add to `GetAvailableThemes()` slice
3. Add case in `GetThemeByName()` switch
4. Test with `T` key

### Adding a Keyboard Shortcut

1. Handle in `app/update.go` (global) or component's `Update()` (local)
2. Add to help text in footer (`app/update.go:31`)
3. Document in this file

### Changing Layout

Main layout is in `app/view.go`:
```go
// Current: [Sidebar][Table]
middleSection := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainArea)

// To add detail pane: [Sidebar][Table][Detail]
middleSection := lipgloss.JoinHorizontal(lipgloss.Top, 
    sidebarView, 
    mainArea,
    detailView,
)
```

**Important**: Recalculate widths in `WindowSizeMsg` handler.

## Dependencies

### Core Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework (MVU architecture)
- `github.com/charmbracelet/lipgloss` - Styling and layout
- `github.com/charmbracelet/bubbles` - Pre-built components (textinput, etc.)

### Indirect Dependencies
- Console handling, terminal detection, text rendering (see `go.mod`)

### Adding Dependencies
```bash
go get github.com/some/package
go mod tidy
```

## Future Development

From README.md and codebase structure:

**Planned Features**:
- [ ] Database connection management (PostgreSQL, MySQL, SQLite)
- [ ] Query execution
- [ ] Real data fetching and display
- [ ] Detail pane for selected row
- [ ] Edit/insert/delete operations
- [ ] Connection configuration UI
- [ ] Query history

**Current Placeholders**:
- `ui/main/record.go` - Future main record view
- `ui/detail/detail.go` - Future detail pane
- Mock data in `app/update.go:getTableData()`

## Debugging Tips

1. **Check debug.log**: All `tea.Printf()` calls go here
2. **Add logging**: 
   ```go
   tea.Printf("Debug: %v", someValue)
   ```
3. **Common issues**:
   - Component not updating → Check if focused
   - Wrong dimensions → Add logging in `WindowSizeMsg` handler
   - Styles not applying → Verify theme is set correctly
   - Text input not working → Check if `Focus()` was called

## Quick Reference

### Bubble Tea Message Types
- `tea.WindowSizeMsg` - Terminal resized
- `tea.KeyMsg` - Keyboard input (check with `msg.String()`)
- `tea.MouseMsg` - Mouse events (not used in this app)
- Custom messages - Define your own `type MyMsg struct{}`

### Lipgloss Quick Patterns
```go
// Style
s := lipgloss.NewStyle().Foreground(color).Background(color)

// Border
s := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

// Alignment
s := lipgloss.NewStyle().Align(lipgloss.Center)

// Size
s := lipgloss.NewStyle().Width(100).Height(50)

// Join
lipgloss.JoinHorizontal(align, views...)
lipgloss.JoinVertical(align, views...)
```

### Common Go Patterns in Codebase
```go
// Min/max
min(a, b)  // Use built-in or define helper
max(a, b)

// Bounds checking
if idx >= 0 && idx < len(slice) { ... }

// Iota for constants
const (
    First = iota  // 0
    Second        // 1
    Third         // 2
)

// Pointer receivers for mutations
func (m *Model) SetSize(w, h int) { ... }

// Value receivers for read-only
func (m Model) View() string { ... }
```
