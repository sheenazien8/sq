package app

import (
	"github.com/sheenazien8/sq/config"
	"github.com/sheenazien8/sq/drivers"
	"github.com/sheenazien8/sq/ui/modal"
	"github.com/sheenazien8/sq/ui/modal-action"
	"github.com/sheenazien8/sq/ui/modal-cell-preview"
	"github.com/sheenazien8/sq/ui/modal-column-visibility"
	"github.com/sheenazien8/sq/ui/modal-create-connection"
	modaldeleteconnection "github.com/sheenazien8/sq/ui/modal-delete-connection"
	"github.com/sheenazien8/sq/ui/modal-edit-cell"
	modaleditconnection "github.com/sheenazien8/sq/ui/modal-edit-connection"
	"github.com/sheenazien8/sq/ui/modal-exit"
	"github.com/sheenazien8/sq/ui/modal-help"
	modalqueryhistory "github.com/sheenazien8/sq/ui/modal-query-history"
	"github.com/sheenazien8/sq/ui/sidebar"
	"github.com/sheenazien8/sq/ui/tab"
	"github.com/sheenazien8/sq/ui/table"
	"github.com/sheenazien8/sq/ui/theme"
)

// Re-export table types for convenience
type TableColumn = table.Column
type TableRow = table.Row

// Page represents which page the application is currently on
type Page int

const (
	PageConnectionManager Page = iota
	PageDatabaseOperations
)

// Focus represents which panel is currently focused
type Focus int

const (
	FocusConnectionManager Focus = iota
	FocusSidebar
	FocusMain
	FocusSidebarFilter
	FocusExitModal
	FocusCreateConnectionModal
	FocusEditConnectionModal
	FocusDeleteConnectionModal
	FocusCellPreviewModal
	FocusActionModal
	FocusEditCellModal
	FocusConfirmModal
	FocusAlertModal
	FocusHelpModal
	// FocusQueryHistoryModal is used when the query history modal is visible
	FocusQueryHistoryModal
)

type Model struct {
	CurrentPage           Page
	Sidebar               sidebar.Model
	Main                  table.Model
	Tabs                  tab.Model
	ExitModal             modalexit.Model
	CreateConnectionModal modalcreateconnection.Model
	EditConnectionModal   modaleditconnection.Model
	DeleteConnectionModal modaldeleteconnection.Model
	CellPreviewModal      modalcellpreview.Model
	ActionModal           modalaction.Model
	EditCellModal         modaleditcell.Model
	ConfirmModal          modal.Model
	AlertModal            modal.Model
	HelpModal             modalhelp.Model
	ColumnVisibilityModal modal.Model
	QueryHistoryModal     modalqueryhistory.Model
	Focus                 Focus
	previousFocus         Focus

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

	// Action confirmation state
	confirmAction      modalaction.Action
	confirmActionModal *modalaction.Model

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

	// Connection Manager state
	connectionManagerCursor int
	connectionManagerOffset int
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
	editConnectionModal := modaleditconnection.New()
	deleteConnectionModal := modaldeleteconnection.New()
	cellPreviewModal := modalcellpreview.New()
	actionModal := modalaction.New()
	editCellModal := modaleditcell.New()
	confirmModal := modal.NewConfirm("Confirm Action", "Are you sure you want to perform this action?")
	alertModal := modal.NewAlert("Error", "")
	helpModal := modalhelp.New()
	columnVisibilityContent := modalcolumnvisibility.New()
	columnVisibilityModal := modal.New("Column Visibility", columnVisibilityContent)
	queryHistoryModal := modalqueryhistory.New()
	tabs := tab.New()

	return Model{
		CurrentPage:           PageConnectionManager,
		Sidebar:               s,
		Tabs:                  tabs,
		ExitModal:             exitModal,
		CreateConnectionModal: createConnectionModal,
		EditConnectionModal:   editConnectionModal,
		DeleteConnectionModal: deleteConnectionModal,
		CellPreviewModal:      cellPreviewModal,
		ActionModal:           actionModal,
		EditCellModal:         editCellModal,
		ConfirmModal:          confirmModal,
		AlertModal:            alertModal,
		HelpModal:             helpModal,
		ColumnVisibilityModal: columnVisibilityModal,
		QueryHistoryModal:     queryHistoryModal,
		Focus:                 FocusSidebar,
		previousFocus:         FocusSidebar,
		dbConnections:         make(map[string]drivers.Driver),
		themeIndex:            themeIdx,
		config:                cfg,
		currentPage:           1,
		pageSize:              100,
	}
}
