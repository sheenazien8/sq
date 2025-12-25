# sq

A keyboard-first SQL TUI built for VIM users [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework for Go.
It focuses on speed, clarity, and terminal-native workflows—no mouse,
no clutter, just efficient querying inside terminal but with the beautifull UI

**Status**: Active development - MySQL support available with full CRUD operations, foreign key navigation, pagination, and SQL query editor.

## Demo

![Demo](https://vhs.charm.sh/vhs-md530GUinDz9OJHRgSSgq.gif)

## Installation

### Prerequisites
- Go 1.22 or later

### Install
```bash
go install github.com/sheenazien8/sq@latest
```

### From source
```bash
go mod download
go mod tidy
go build -o sq
./sq
```

### Running
```bash
go run .
```

## Features

### Current Features
- Multi-pane layout with sidebar (connections/tables), tabbed table views, and filter dialog
- MySQL database connection support
- **Query Editor** with vim-mode support for writing and executing custom SQL queries
  - SQL syntax highlighting
  - SQL formatting with `Ctrl+F`
  - Multi-line query support
- Tabbed interface for multiple tables/queries
- Table structure viewer (columns, indexes, relations, triggers)
- **Foreign Key Navigation** - Jump to related tables with `gd` (goto definition)
- **Pagination** - Efficient handling of large datasets with configurable page sizes
- Vim-like keyboard navigation throughout
- Theme switching (default, dracula, nord, gruvbox, tokyo-night, catppuccin, monokai)
- Cell preview and yank (copy) functionality
- Filter dialogs for advanced querying with multiple filter support
- Collapsible sidebar
- Built-in help modal with `?` key

### Planned Features
- PostgreSQL and SQLite support
- Detail pane for selected records
- Edit/insert/delete operations
- Query history

## Usage

### Global Shortcuts
| Key | Action |
|-----|--------|
| `?` | Show help modal |
| `q` / `Ctrl+C` | Show exit modal |
| `Tab` | Switch focus between sidebar and main area |
| `T` | Cycle themes |
| `s` / `S` | Toggle sidebar visibility |

### Sidebar Navigation (when focused)
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Home` | Jump to first item |
| `End` | Jump to last item |
| `Enter` | Select/connect to database or open table |
| `e` | Open Query Editor (requires active connection) |
| `d` | View table structure |
| `n` | Create new connection |

### Tab Management
| Key | Action |
|-----|--------|
| `]` | Next tab |
| `[` | Previous tab |
| `Ctrl+W` | Close current tab |

### Table Navigation (when focused)
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down one row |
| `k` / `↑` | Move up one row |
| `h` / `←` | Scroll columns left |
| `l` / `→` | Scroll columns right |
| `H` | Jump to first column |
| `L` | Jump to last column |
| `J` | Next page (pagination) |
| `K` | Previous page (pagination) |
| `PgUp` / `PgDn` | Page up/down |
| `Home` / `End` | Jump to first/last row |
| `y` | Yank (copy) selected cell content to clipboard |
| `p` | Preview selected cell content |
| `/` / `f` | Open filter dialog |
| `C` | Clear all filters |
| `d` | View table structure |
| `e` | Open Query Editor |
| `gd` | Go to definition (navigate to foreign key table) |

### Table Structure View
| Key | Action |
|-----|--------|
| `1` | View Columns |
| `2` | View Indexes |
| `3` | View Relations |
| `4` | View Triggers |
| `Tab` | Next section |
| `Shift+Tab` | Previous section |

### Query Editor
The query editor features full **vim-mode** support for efficient editing.

#### Vim Modes
| Mode | Description |
|------|-------------|
| `NORMAL` | Navigation and commands (default) |
| `INSERT` | Text editing mode |

#### Normal Mode Commands
| Key | Action |
|-----|--------|
| `i` | Enter insert mode at cursor |
| `a` | Enter insert mode after cursor |
| `I` | Enter insert mode at beginning of line |
| `A` | Enter insert mode at end of line |
| `o` | Open new line below and enter insert mode |
| `O` | Open new line above and enter insert mode |
| `h` / `←` | Move cursor left |
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `l` / `→` | Move cursor right |
| `0` | Move to beginning of line |
| `$` | Move to end of line |
| `w` | Move to next word |
| `b` | Move to previous word |
| `g` | Go to beginning of document |
| `G` | Go to end of document |
| `x` | Delete character under cursor |
| `X` | Delete character before cursor |
| `u` | Undo |

#### Insert Mode
| Key | Action |
|-----|--------|
| `Esc` | Return to normal mode |
| Any key | Type text |

#### Query Execution
| Key | Action |
|-----|--------|
| `F5` / `Ctrl+E` | Execute query |
| `Ctrl+R` | Toggle focus between editor and results |
| `Ctrl+F` | Format SQL query |
| `Ctrl+Y` | Copy entire query to clipboard |

#### Results Table (when focused)
| Key | Action |
|-----|--------|
| `h/j/k/l` | Navigate cells |
| `p` | Preview selected cell content |
| `y` | Yank (copy) selected cell to clipboard |
| `i` / `a` | Return to editor in insert mode |
| `Ctrl+R` | Return to editor |

### Filter Dialog (when open)
| Key | Action |
|-----|--------|
| `Tab` / `→` / `l` | Next field |
| `Shift+Tab` / `←` / `h` | Previous field |
| `j` / `↓` | Next option (column/operator) |
| `k` / `↑` | Previous option (column/operator) |
| `Enter` | Apply filter and close |
| `Esc` | Close without applying |
| `Ctrl+C` | Clear filter |

### Modal (when visible)
| Key | Action |
|-----|--------|
| `←` / `→` / `h` / `l` / `Tab` | Switch button |
| `Enter` | Confirm selection |
| `y` / `Y` | Yes |
| `n` / `N` / `Esc` | No |

## Project Structure

```
sq/
├── main.go              # Entry point, initializes Bubble Tea program
├── app/                 # Main application logic (Bubble Tea Model-View-Update)
│   ├── init.go          # Init() - initialization command
│   ├── model.go         # Model struct and constructor
│   ├── update.go        # Update() - handles messages and input
│   └── view.go          # View() - renders the UI
├── config/              # Application configuration
│   └── config.go        # Config loading/saving (~/.config/db-client-tui/config.json)
├── drivers/             # Database drivers
│   ├── driver.go        # Driver interface definition
│   ├── mysql.go         # MySQL driver implementation (with pagination support)
│   └── types.go         # Shared types (TableStructure, ColumnInfo, Pagination, etc.)
├── logger/              # Logging utilities
├── storage/             # Connection storage utilities
├── ui/                  # UI components (separate Bubble Tea models)
│   ├── sidebar/         # Connection and table list sidebar
│   ├── table/           # Scrollable table widget
│   ├── tab/             # Tabbed interface for multiple views
│   ├── filter/          # Filter input component
│   ├── query-editor/    # SQL query editor with vim-mode and formatter
│   ├── syntax-editor/   # Syntax highlighting text editor component
│   ├── modal/           # Base modal component
│   ├── modal-exit/      # Exit confirmation modal
│   ├── modal-cell-preview/  # Cell content preview modal
│   ├── modal-create-connection/  # New connection modal
│   ├── modal-help/      # Help modal with all keybindings
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

Configuration is saved to `~/.config/sq/config.json`:

```json
{
  "theme": "default"
}
```

Available themes: default, dracula, nord, gruvbox, tokyo-night, catppuccin, monokai.

## Database Connections

Connections are stored in `~/.config/sq/connections.json`.

### Supported Databases
- **MySQL** - Full support including:
  - Table browsing and data viewing with pagination
  - Table structure (columns, indexes, relations, triggers)
  - Foreign key navigation (goto definition)
  - Custom SQL query execution with syntax highlighting and formatting
  - Filtering with multiple conditions
  - Connection persistence

### Connection URL Format

**MySQL:**
```
mysql://user:password@tcp(host:port)/database
```

## Debugging

Debug logs are written to `debug.log` in the current directory. This can be helpful for troubleshooting connection issues or unexpected behavior.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests (when implemented)
5. Submit a pull request

## License

This project is open source. See LICENSE file for details.
