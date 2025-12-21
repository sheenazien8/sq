# DB Client TUI

A terminal-based database client built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework for Go. Features a multi-pane layout with keyboard-driven navigation following vim-like patterns.

**Status**: Early development - currently displays mock data.

## Demo

See `demo/demo.mov` for a demonstration of the current functionality.

## Installation

### Prerequisites
- Go 1.22 or later

### Setup
```bash
go mod download
go mod tidy
```

### Building
```bash
go build -o db-client-tui
./db-client-tui
```

### Running
```bash
go run .
```

## Features

### Current Features
- Multi-pane layout with sidebar (databases), table view, and filter dialog
- Vim-like keyboard navigation
- Theme switching (default, dracula, nord, gruvbox, tokyo-night, catppuccin, monokai)
- Cell preview and yank functionality
- Filter dialogs for advanced querying

### Planned Features
- Database connection management for PostgreSQL, MySQL, and SQLite
- Real data querying and display
- Detail pane for selected records
- Edit/insert/delete operations
- Connection configuration UI
- Query history

## Usage

### Global Shortcuts
- `q` / `Ctrl+C` - Show exit modal
- `Tab` - Switch focus between sidebar and main table
- `T` - Cycle themes
- `C` - Clear active filter

### Sidebar Navigation (when focused)
- `j` / `↓` - Move down
- `k` / `↑` - Move up
- `Home` - Jump to first item
- `End` - Jump to last item
- `Enter` - Select database

### Table Navigation (when focused)
- `j` / `↓` - Move down one row
- `k` / `↑` - Move up one row
- `h` / `←` - Scroll columns left
- `l` / `→` - Scroll columns right
- `H` - Jump to first column
- `L` - Jump to last column
- `PgUp` / `PgDn` - Page up/down
- `Home` / `End` - Jump to first/last row
- `y` - Yank (copy) selected cell content to clipboard
- `p` - Preview selected cell content
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
├── drivers/             # Database drivers (future implementation)
├── logger/              # Logging utilities
├── storage/             # Data storage utilities
├── ui/                  # UI components (separate Bubble Tea models)
│   ├── sidebar/         # Database list sidebar
│   ├── table/           # Scrollable table widget
│   ├── filter/          # Filter input component
│   ├── modal/           # Modal dialogs (e.g., exit confirmation)
│   ├── theme/           # Theme system and color definitions
│   ├── main/            # (future) Main record view
│   └── detail/          # (future) Detail pane
└── demo/                # Demo files
    └── demo.mov         # Demonstration video
```

## Architecture

The app follows Bubble Tea's Model-View-Update (MVU) architecture:

- **Model**: Application state with sub-models for each UI component
- **Update**: Event handling, routing messages to focused components
- **View**: Rendering, composing sub-component views using Lipgloss
- **Init**: Initial commands (currently nil)

Each UI component (sidebar, table, filter, modal) follows the same pattern with own Model, Update(), and View() methods.

## Configuration

Configuration is saved to `~/.config/db-client-tui/config.json`:

```json
{
  "theme": "default"
}
```

Available themes: default, dracula, nord, gruvbox, tokyo-night, catppuccin, monokai.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests (when implemented)
5. Submit a pull request

## License

This project is open source. See LICENSE file for details.