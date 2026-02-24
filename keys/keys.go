package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all the keybindings for the application
type KeyMap struct {
	// Global
	Quit          key.Binding
	Help          key.Binding
	ToggleTheme   key.Binding
	ToggleSidebar key.Binding
	ToggleDetail  key.Binding
	ClearFilter   key.Binding
	FocusNext     key.Binding

	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Table specific
	JumpToFirstColumn key.Binding
	JumpToLastColumn  key.Binding
	NextPage          key.Binding
	PrevPage          key.Binding
	GotoDefinition    key.Binding
	ViewStructure     key.Binding
	Yank              key.Binding
	Preview           key.Binding
	Filter            key.Binding
	NewConnection     key.Binding
	OpenQueryEditor   key.Binding
	Confirm           key.Binding
	Cancel            key.Binding
	IncreaseColumn    key.Binding
	DecreaseColumn    key.Binding

	EditConnection          key.Binding
	DeleteConnection        key.Binding
	Refresh                 key.Binding
	ActionMenu              key.Binding
	ColumnVisibility        key.Binding
	CloseTab                key.Binding
	NextTab                 key.Binding
	PrevTab                 key.Binding
	ExecuteQuery            key.Binding
	SwitchEditorResultsPane key.Binding
	FormatQuery             key.Binding
	QueryHistory            key.Binding
	YankQuery               key.Binding

	// Detail specific
	DetailScrollUp   key.Binding
	DetailScrollDown key.Binding
}

// AppKeys is the global instance of key bindings
var AppKeys = DefaultKeyMap()

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		ToggleTheme: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "theme"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("s", "S"),
			key.WithHelp("s", "sidebar"),
		),
		ToggleDetail: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "detail pane"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "clear filter"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch focus"),
		),

		// Navigation
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "first"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "last"),
		),

		// Table specific
		JumpToFirstColumn: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "first col"),
		),
		JumpToLastColumn: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "last col"),
		),
		NextPage: key.NewBinding(
			key.WithKeys(">"),
			key.WithHelp(">", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("<"),
			key.WithHelp("<", "prev page"),
		),
		IncreaseColumn: key.NewBinding(
			key.WithKeys("alt+right", "shift+right"),
			key.WithHelp("alt+right/shift+right", "increase column"),
		),
		DecreaseColumn: key.NewBinding(
			key.WithKeys("alt+left", "shift+left"),
			key.WithHelp("alt+left/shift+left", "decrease column"),
		),
		GotoDefinition: key.NewBinding(
			key.WithKeys("gd"),
			key.WithHelp("gd", "goto definition"),
		),
		ViewStructure: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "structure"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yank"),
		),
		Preview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "preview"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/", "f"),
			key.WithHelp("/, f", "filter"),
		),
		NewConnection: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new connection"),
		),
		OpenQueryEditor: key.NewBinding(
			key.WithKeys("e", "E"),
			key.WithHelp("e", "query editor"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),

		EditConnection: key.NewBinding(
			key.WithKeys("w", "W"),
			key.WithHelp("w", "edit connection"),
		),
		DeleteConnection: key.NewBinding(
			key.WithKeys("x", "X"),
			key.WithHelp("x", "delete connection"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "R"),
			key.WithHelp("r", "refresh"),
		),
		ActionMenu: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "action menu"),
		),
		ColumnVisibility: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "column visibility"),
		),
		CloseTab: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "close tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev tab"),
		),
		ExecuteQuery: key.NewBinding(
			key.WithKeys("f5", "ctrl+e"),
			key.WithHelp("f5/ctrl+e", "execute query"),
		),
		SwitchEditorResultsPane: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "switch pane"),
		),
		FormatQuery: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "format query"),
		),
		YankQuery: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl+y", "yank query"),
		),
		DetailScrollUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "detail up"),
		),
		DetailScrollDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "detail down"),
		),
	}

}

// UpdateFromConfig updates the global keymap based on user configuration
func UpdateFromConfig(configMap map[string][]string) {
	if len(configMap) == 0 {
		return
	}

	apply := func(keyName string, binding *key.Binding) {
		if keys, ok := configMap[keyName]; ok && len(keys) > 0 {
			*binding = key.NewBinding(
				key.WithKeys(keys...),
				key.WithHelp(binding.Help().Key, binding.Help().Desc),
			)
		}
	}

	apply("quit", &AppKeys.Quit)
	apply("help", &AppKeys.Help)
	apply("toggle_theme", &AppKeys.ToggleTheme)
	apply("toggle_sidebar", &AppKeys.ToggleSidebar)
	apply("toggle_detail", &AppKeys.ToggleDetail)
	apply("clear_filter", &AppKeys.ClearFilter)
	apply("focus_next", &AppKeys.FocusNext)

	apply("up", &AppKeys.Up)
	apply("down", &AppKeys.Down)
	apply("left", &AppKeys.Left)
	apply("right", &AppKeys.Right)
	apply("page_up", &AppKeys.PageUp)
	apply("page_down", &AppKeys.PageDown)
	apply("home", &AppKeys.Home)
	apply("end", &AppKeys.End)

	apply("jump_first_column", &AppKeys.JumpToFirstColumn)
	apply("jump_last_column", &AppKeys.JumpToLastColumn)
	apply("next_page", &AppKeys.NextPage)
	apply("prev_page", &AppKeys.PrevPage)
	apply("goto_definition", &AppKeys.GotoDefinition)
	apply("view_structure", &AppKeys.ViewStructure)
	apply("increase_column", &AppKeys.IncreaseColumn)
	apply("decrease_column", &AppKeys.DecreaseColumn)
	apply("yank", &AppKeys.Yank)
	apply("preview", &AppKeys.Preview)
	apply("filter", &AppKeys.Filter)
	apply("new_connection", &AppKeys.NewConnection)
	apply("open_query_editor", &AppKeys.OpenQueryEditor)
	apply("confirm", &AppKeys.Confirm)
	apply("cancel", &AppKeys.Cancel)

	apply("edit_connection", &AppKeys.EditConnection)
	apply("delete_connection", &AppKeys.DeleteConnection)
	apply("refresh", &AppKeys.Refresh)
	apply("action_menu", &AppKeys.ActionMenu)
	apply("column_visibility", &AppKeys.ColumnVisibility)
	apply("close_tab", &AppKeys.CloseTab)
	apply("next_tab", &AppKeys.NextTab)
	apply("prev_tab", &AppKeys.PrevTab)
	apply("execute_query", &AppKeys.ExecuteQuery)
	apply("switch_editor_results_pane", &AppKeys.SwitchEditorResultsPane)
	apply("format_query", &AppKeys.FormatQuery)
	apply("query_history", &AppKeys.QueryHistory)
	apply("yank_query", &AppKeys.YankQuery)
	apply("detail_scroll_up", &AppKeys.DetailScrollUp)
	apply("detail_scroll_down", &AppKeys.DetailScrollDown)
}
