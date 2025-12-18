package app

import (
	"github.com/sheenazien8/db-client-tui/config"
	"github.com/sheenazien8/db-client-tui/ui/filter"
	modalcreateconnection "github.com/sheenazien8/db-client-tui/ui/modal-create-connection"
	modalexit "github.com/sheenazien8/db-client-tui/ui/modal-exit"
	"github.com/sheenazien8/db-client-tui/ui/sidebar"
	"github.com/sheenazien8/db-client-tui/ui/table"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

// Re-export table types for convenience
type TableColumn = table.Column
type TableRow = table.Row

// Focus represents which panel is currently focused
type Focus int

const (
	FocusSidebar Focus = iota
	FocusMain
	FocusFilter
	FocusExitModal
	FocusCreateConnectionModal
)

type Model struct {
	Sidebar               sidebar.Model
	Main                  table.Model
	Filter                filter.Model
	ExitModal             modalexit.Model
	CreateConnectionModal modalcreateconnection.Model
	Focus                 Focus

	allRows     []table.Row
	columns     []table.Column
	columnNames []string

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

	return Model{
		Sidebar:               s,
		ExitModal:             exitModal,
		CreateConnectionModal: createConnectionModal,
		Focus:                 FocusSidebar,
		themeIndex:            themeIdx,
		config:                cfg,
	}
}
