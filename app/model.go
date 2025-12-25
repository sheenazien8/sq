package app

import (
	"github.com/sheenazien8/sq/config"
	"github.com/sheenazien8/sq/drivers"
	"github.com/sheenazien8/sq/ui/modal-cell-preview"
	"github.com/sheenazien8/sq/ui/modal-create-connection"
	"github.com/sheenazien8/sq/ui/modal-exit"
	"github.com/sheenazien8/sq/ui/modal-help"
	"github.com/sheenazien8/sq/ui/sidebar"
	"github.com/sheenazien8/sq/ui/tab"
	"github.com/sheenazien8/sq/ui/table"
	"github.com/sheenazien8/sq/ui/theme"
)

// Re-export table types for convenience
type TableColumn = table.Column
type TableRow = table.Row

// Focus represents which panel is currently focused
type Focus int

const (
	FocusSidebar Focus = iota
	FocusMain
	FocusSidebarFilter
	FocusExitModal
	FocusCreateConnectionModal
	FocusCellPreviewModal
	FocusHelpModal
)

type Model struct {
	Sidebar               sidebar.Model
	Main                  table.Model
	Tabs                  tab.Model
	ExitModal             modalexit.Model
	CreateConnectionModal modalcreateconnection.Model
	CellPreviewModal      modalcellpreview.Model
	HelpModal             modalhelp.Model
	Focus                 Focus

	allRows     []table.Row
	columns     []table.Column
	columnNames []string

	// Database connections
	dbConnections map[string]drivers.Driver

	// Track current table context for reloading with filters
	currentConnection string
	currentDatabase   string
	currentTable      string

	// Pagination state
	currentPage int
	pageSize    int

	// Key sequence state for multi-key commands
	gPressed bool // Track if 'g' was pressed for 'gd' sequence

	TerminalWidth  int
	TerminalHeight int

	ContentWidth int
	SidebarWidth int
	FooterWidth  int
	HeaderWidth  int

	ContentHeight int
	SidebarHeight int
	FooterHeight  int
	HeaderHeight  int

	sidebarCollapsed bool

	HeaderStyle string
	FooterStyle string

	initialized bool

	themeIndex int

	config *config.Config
}

func New() Model {
	s := sidebar.New()
	s.SetFocused(true)

	cfg, _ := config.Load()

	theme.SetTheme(theme.GetThemeByName(cfg.Theme))

	themeIdx := 0
	themes := theme.GetAvailableThemes()
	for i, t := range themes {
		if t == cfg.Theme {
			themeIdx = i
			break
		}
	}

	exitModal := modalexit.New()
	createConnectionModal := modalcreateconnection.New()
	cellPreviewModal := modalcellpreview.New()
	helpModal := modalhelp.New()
	tabs := tab.New()

	return Model{
		Sidebar:               s,
		Tabs:                  tabs,
		ExitModal:             exitModal,
		CreateConnectionModal: createConnectionModal,
		CellPreviewModal:      cellPreviewModal,
		HelpModal:             helpModal,
		Focus:                 FocusSidebar,
		dbConnections:         make(map[string]drivers.Driver),
		themeIndex:            themeIdx,
		config:                cfg,
		currentPage:           1,
		pageSize:              100,
	}
}
