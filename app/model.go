package app

import (
	"github.com/sheenazien8/db-client-tui/config"
	"github.com/sheenazien8/db-client-tui/ui/filter"
	"github.com/sheenazien8/db-client-tui/ui/modal"
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
	FocusModal
)

type Model struct {
	Sidebar   sidebar.Model
	Main      table.Model
	Filter    filter.Model
	ExitModal modal.Model
	Focus     Focus

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

	exitModal := modal.New("Exit", "Are you sure you want to quit?")

	return Model{
		Sidebar:    s,
		ExitModal:  exitModal,
		Focus:      FocusSidebar,
		themeIndex: themeIdx,
		config:     cfg,
	}
}
