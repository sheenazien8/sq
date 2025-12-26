# sq

A keyboard-first SQL TUI built for VIM users [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework for Go.
It focuses on speed, clarity, and terminal-native workflows—no mouse,
no clutter, just efficient querying inside terminal but with the beautifull UI

**Status**: Active development - MySQL, PostgreSQL, SQLite, MongoDB, and MongoDB Atlas support available with full CRUD operations, foreign key navigation, pagination, and SQL query editor.

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

### Command Line Options
```bash
sq --help                    # Show help
sq --version                 # Show version
sq --create-connection       # Create a new database connection
```

## Features

### Current Features

**Database Support:**
- MySQL database connections with full feature support
- PostgreSQL database connections with full feature support
- SQLite database file connections with full feature support
- MongoDB database connections with full feature support (local and remote)
- MongoDB Atlas cloud connections with full feature support
- Multiple simultaneous connections in sidebar
- Persistent connection storage

**Data Browsing:**
- Table listing with automatic refresh
- Data viewing with pagination (100 rows per page by default)
- Efficient handling of large datasets
- Cell-level data preview with `p` key
- Copy cell data to clipboard with `y` key

**Advanced Features:**
- **Query Editor** with vim-mode support for writing and executing custom SQL queries
  - SQL syntax highlighting (Chroma v2)
  - SQL formatting with `Ctrl+F` (sqlfmt integration)
  - Multi-line query support
  - Query execution with F5 or Ctrl+E
- **Table Structure Viewer** - View columns, indexes, relations, and triggers
  - Column information (type, nullable, default values)
  - Index information (unique, primary, type)
  - Foreign key relationships
  - Triggers and their definitions

**Navigation & Filtering:**
- **Foreign Key Navigation** - Jump to related tables with `gd` (goto definition)
- **Advanced Filtering** - Multi-condition filter dialog with column/operator/value selection
- Vim-like keyboard navigation (hjkl movement, gg/G jump, w/b word movement)
- Tabbed interface for multiple tables/queries
- Collapsible sidebar to maximize table view space

**UI & Theming:**
- 7 built-in themes (default, dracula, nord, gruvbox, tokyo-night, catppuccin, monokai)
- Real-time theme switching with `T` key
- Multi-pane layout with sidebar, table view, and filter dialog
- Built-in help modal accessible with `?` key
- Responsive design that adapts to terminal size

### Planned Features
- Edit/insert/delete operations
- Query history
- Detail pane for selected records
- Export data (CSV, JSON)

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
│   └── config.go        # Config loading/saving (~/.config/sq/config.json)
├── drivers/             # Database drivers
│   ├── driver.go        # Driver interface definition
│   ├── mysql.go         # MySQL driver implementation (with pagination support)
│   ├── postgres.go      # PostgreSQL driver implementation (with pagination support)
│   ├── sqlite.go        # SQLite driver implementation
│   ├── mongodb.go       # MongoDB driver implementation (standard and Atlas support)
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

Connections are stored in `~/.config/sq/storage.db`.

### Quick Start - Creating Your First Connection

1. Launch the application:
    ```bash
    ./sq
    ```

2. Press `n` to create a new connection

3. Select your database driver (MySQL, PostgreSQL, SQLite, MongoDB, or MongoDB Atlas)

4. **For MySQL/PostgreSQL/SQLite**, enter your connection details:
    - **Name**: A descriptive name for this connection (e.g., "Production DB")
    - **Host**: Database server address (default: localhost)
    - **Port**: Database port (MySQL: 3306, PostgreSQL: 5432)
    - **Username**: Database user (MySQL: root, PostgreSQL: postgres)
    - **Password**: User password
    - **Database**: Database name to connect to

5. **For MongoDB**, enter:
    - **Name**: A descriptive name for this connection
    - **Host**: MongoDB server address (default: localhost)
    - **Port**: MongoDB port (default: 27017)
    - **Username**: Database user (optional)
    - **Password**: User password (optional)
    - **Database**: Database name to connect to

6. **For MongoDB Atlas**, paste your connection URI:
    - **Name**: A descriptive name for this connection
    - **Connection URI**: Full MongoDB connection string (e.g., `mongodb+srv://user:pass@cluster.mongodb.net/database`)

7. Press `Enter` to test the connection and save it

8. Once saved, your connection appears in the sidebar and can be selected with `Enter`

### Supported Databases
- **MySQL** - Full support including:
  - Table browsing and data viewing with pagination
  - Table structure (columns, indexes, relations, triggers)
  - Foreign key navigation (goto definition)
  - Custom SQL query execution with syntax highlighting and formatting
  - Filtering with multiple conditions
  - Connection persistence

- **PostgreSQL** - Full support including:
   - Table browsing and data viewing with pagination
   - Table structure (columns, indexes, relations, triggers)
   - Foreign key navigation (goto definition)
   - Custom SQL query execution with syntax highlighting and formatting
   - Filtering with multiple conditions
   - Connection persistence

- **MongoDB** - Full support including:
   - Collection browsing and document viewing with pagination
   - Collection structure (field analysis from samples)
   - Efficient filtering with JSON and simple syntax
   - ObjectId filtering support
   - Both self-hosted and MongoDB Atlas cloud connections
   - Connection persistence

- **MongoDB Atlas** - Cloud database support including:
   - Direct connection string URI input
   - Automatic database detection
   - Full feature parity with standard MongoDB
   - Optimized for cloud connections

### Connection URL Format

**MySQL:**
```
mysql://user:password@tcp(host:port)/database
```

Example:
```
mysql://root:password@tcp(localhost:3306)/mydb
```

**PostgreSQL:**
```
postgres://user:password@host:port/database?sslmode=disable
```

Example:
```
postgres://postgres:password@localhost:5432/mydb?sslmode=disable
```

**MongoDB:**
```
mongodb://user:password@host:port/database
```

Example:
```
mongodb://localhost:27017/mydb
mongodb://admin:password@localhost:27017/mydb
```

**MongoDB Atlas:**
```
mongodb+srv://user:password@cluster.mongodb.net/database?retryWrites=true&w=majority
```

Example:
```
mongodb+srv://admin:securepassword@cluster-name-abc.mongodb.net/mydatabase?retryWrites=true&w=majority
```

### MongoDB Filtering

MongoDB supports two filter syntaxes:

**Simple Syntax** (recommended for basic queries):
```
fieldname=value                 # Match a field
_id=507f1f77bcf86cd799439011  # Match by ObjectId
age=25,status=active           # Multiple conditions (AND)
```

**JSON Syntax** (for advanced queries):
```
{"age": {"$gt": 25}}           # Greater than
{"status": {"$in": ["a", "b"]}}  # In array
{"_id": {"$exists": true}}     # Field exists
```

#### PostgreSQL Schema Support

sq automatically detects and uses the appropriate schema:

1. **Priority Order**:
   - First, checks if the `public` schema exists and uses it
   - If `public` doesn't exist, uses the first user-created schema found
   - Falls back to `public` if detection fails

2. **Schema Detection**:
   - Automatically excludes PostgreSQL system schemas (`pg_catalog`, `information_schema`, `pg_toast`)
   - Selects the first user-accessible schema alphabetically
   - Detection happens automatically on connection

3. **Current Limitations**:
   - Only one schema is supported per connection
   - Tables from multiple schemas cannot be viewed simultaneously
   - To work with tables in different schemas, create separate connections for each schema

If you need to work with a specific non-public schema, the schema is automatically detected on connection. If detection doesn't find your schema, ensure:
- The schema exists in the database
- Your user has permissions to access it
- The schema is not a PostgreSQL system schema

## Troubleshooting

### Connection Issues

**Error: "invalid database scheme"**
- Make sure you're using the correct URI format for your database type
- MySQL: `mysql://user:password@host:port/database`
- PostgreSQL: `postgres://user:password@host:port/database?sslmode=disable`

**PostgreSQL: "relation not found"**
- Check that your tables exist in the detected schema (see logs for which schema was selected)
- Verify the table names match exactly (PostgreSQL is case-sensitive for unquoted identifiers)
- If using a non-public schema, ensure:
  - The schema exists in your database
  - Your user has SELECT permissions on the schema
  - The schema is not a PostgreSQL system schema (`pg_catalog`, `information_schema`, `pg_toast`)
- Check `debug.log` to see which schema was automatically detected

**Connection refused**
- Check that the database server is running and accessible
- Verify host and port are correct
- Check firewall rules if connecting to remote servers
- Ensure username and password are correct

### MongoDB Specific Issues

**MongoDB Atlas Connection Issues:**
- Ensure your IP address is whitelisted in the Atlas security settings
- Use the correct connection string format with `mongodb+srv://` protocol
- Check that your cluster is running and not paused
- Verify database credentials are correct

**MongoDB Collection Display Issues:**
- Collections may take a moment to appear after connection
- Ensure the database exists and contains collections
- Check that your user has permission to list collections
- For MongoDB Atlas, ensure appropriate network access rules are set

**ObjectId Filtering Not Working:**
- ObjectId must be exactly 24 hexadecimal characters
- Use format: `_id=507f1f77bcf86cd799439011`
- For non-24-char IDs, MongoDB will treat them as strings instead

### Debugging

Debug logs are written to `debug.log` in the current directory. This can be helpful for troubleshooting connection issues or unexpected behavior.

To enable detailed logging:
1. Check the `debug.log` file in your current directory after launching sq
2. Look for error messages related to your specific operation
3. Common issues like schema problems, query errors, and connection failures are logged here

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests (when implemented)
5. Submit a pull request

## License

This project is open source. See LICENSE file for details.
