package tab

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sheenazien8/db-client-tui/drivers"
	"github.com/sheenazien8/db-client-tui/logger"
	"github.com/sheenazien8/db-client-tui/ui/filter"
	"github.com/sheenazien8/db-client-tui/ui/table"
	"github.com/sheenazien8/db-client-tui/ui/theme"
)

// Tab represents a single tab containing content
type Tab struct {
	Name          string
	Content       interface{} // Can be table.Model or query_editor.Model
	Type          TabType
	Active        bool
	AllRows       []table.Row     // Original unfiltered data
	Columns       []table.Column  // Column definitions
	ColumnNames   []string        // Column names for filtering
	ActiveFilters []filter.Filter // Multiple filters applied to this tab (all AND)
}

// TabType represents the type of content in a tab
type TabType int

const (
	TabTypeTable TabType = iota
	TabTypeStructure
)

// StructureSection represents which section of structure is active
type StructureSection int

const (
	SectionColumns StructureSection = iota
	SectionIndexes
	SectionRelations
	SectionTriggers
)

// StructureView holds the table structure data and navigation state
type StructureView struct {
	Structure     *drivers.TableStructure
	ActiveSection StructureSection
	SectionTables map[StructureSection]table.Model
	Width         int
	Height        int
	Focused       bool
}

// NewStructureView creates a new structure view from table structure data
func NewStructureView(structure *drivers.TableStructure, width, height int) StructureView {
	sv := StructureView{
		Structure:     structure,
		ActiveSection: SectionColumns,
		SectionTables: make(map[StructureSection]table.Model),
		Width:         width,
		Height:        height,
		Focused:       false,
	}

	// Create table for columns
	columnsTable := sv.createColumnsTable(structure.Columns)
	columnsTable.SetSize(width, height-4) // Reserve space for section tabs
	sv.SectionTables[SectionColumns] = columnsTable

	// Create table for indexes
	indexesTable := sv.createIndexesTable(structure.Indexes)
	indexesTable.SetSize(width, height-4)
	sv.SectionTables[SectionIndexes] = indexesTable

	// Create table for relations
	relationsTable := sv.createRelationsTable(structure.Relations)
	relationsTable.SetSize(width, height-4)
	sv.SectionTables[SectionRelations] = relationsTable

	// Create table for triggers
	triggersTable := sv.createTriggersTable(structure.Triggers)
	triggersTable.SetSize(width, height-4)
	sv.SectionTables[SectionTriggers] = triggersTable

	return sv
}

func (sv *StructureView) createColumnsTable(columns []drivers.ColumnInfo) table.Model {
	cols := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 20},
		{Title: "Nullable", Width: 10},
		{Title: "Key", Width: 8},
		{Title: "Default", Width: 15},
		{Title: "Extra", Width: 20},
		{Title: "Comment", Width: 25},
	}

	var rows []table.Row
	for _, col := range columns {
		nullable := "NO"
		if col.Nullable {
			nullable = "YES"
		}
		key := ""
		if col.IsPrimaryKey {
			key = "PRI"
		}
		rows = append(rows, table.Row{
			col.Name,
			col.DataType,
			nullable,
			key,
			col.DefaultValue,
			col.Extra,
			col.Comment,
		})
	}

	return table.New(cols, rows)
}

func (sv *StructureView) createIndexesTable(indexes []drivers.IndexInfo) table.Model {
	cols := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Columns", Width: 35},
		{Title: "Type", Width: 12},
		{Title: "Unique", Width: 8},
		{Title: "Primary", Width: 8},
	}

	var rows []table.Row
	for _, idx := range indexes {
		unique := "NO"
		if idx.IsUnique {
			unique = "YES"
		}
		primary := "NO"
		if idx.IsPrimary {
			primary = "YES"
		}
		columnsStr := joinStrings(idx.Columns, ", ")
		rows = append(rows, table.Row{
			idx.Name,
			columnsStr,
			idx.Type,
			unique,
			primary,
		})
	}

	return table.New(cols, rows)
}

func (sv *StructureView) createRelationsTable(relations []drivers.RelationInfo) table.Model {
	cols := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Column", Width: 20},
		{Title: "Ref Table", Width: 20},
		{Title: "Ref Column", Width: 20},
		{Title: "On Update", Width: 12},
		{Title: "On Delete", Width: 12},
	}

	var rows []table.Row
	for _, rel := range relations {
		rows = append(rows, table.Row{
			rel.Name,
			rel.Column,
			rel.ReferencedTable,
			rel.ReferencedColumn,
			rel.OnUpdate,
			rel.OnDelete,
		})
	}

	return table.New(cols, rows)
}

func (sv *StructureView) createTriggersTable(triggers []drivers.TriggerInfo) table.Model {
	cols := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Event", Width: 12},
		{Title: "Timing", Width: 10},
		{Title: "Statement", Width: 50},
	}

	var rows []table.Row
	for _, trig := range triggers {
		// Truncate statement if too long
		stmt := trig.Statement
		if len(stmt) > 50 {
			stmt = stmt[:47] + "..."
		}
		rows = append(rows, table.Row{
			trig.Name,
			trig.Event,
			trig.Timing,
			stmt,
		})
	}

	return table.New(cols, rows)
}

func (sv *StructureView) SetSize(width, height int) {
	sv.Width = width
	sv.Height = height
	for section, tbl := range sv.SectionTables {
		tbl.SetSize(width, height-4)
		sv.SectionTables[section] = tbl
	}
}

func (sv *StructureView) SetFocused(focused bool) {
	sv.Focused = focused
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(focused)
		sv.SectionTables[sv.ActiveSection] = tbl
	}
}

func (sv *StructureView) NextSection() {
	// Unfocus current section table
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(false)
		sv.SectionTables[sv.ActiveSection] = tbl
	}

	sv.ActiveSection = (sv.ActiveSection + 1) % 4

	// Focus new section table
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(sv.Focused)
		sv.SectionTables[sv.ActiveSection] = tbl
	}
}

func (sv *StructureView) PrevSection() {
	// Unfocus current section table
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(false)
		sv.SectionTables[sv.ActiveSection] = tbl
	}

	if sv.ActiveSection == 0 {
		sv.ActiveSection = SectionTriggers
	} else {
		sv.ActiveSection--
	}

	// Focus new section table
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(sv.Focused)
		sv.SectionTables[sv.ActiveSection] = tbl
	}
}

func (sv StructureView) Update(msg tea.Msg) (StructureView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			sv.switchToSection(SectionColumns)
		case "2":
			sv.switchToSection(SectionIndexes)
		case "3":
			sv.switchToSection(SectionRelations)
		case "4":
			sv.switchToSection(SectionTriggers)
		case "tab":
			sv.NextSection()
		case "shift+tab":
			sv.PrevSection()
		default:
			// Pass to active section table
			if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
				var cmd tea.Cmd
				tbl, cmd = tbl.Update(msg)
				sv.SectionTables[sv.ActiveSection] = tbl
				return sv, cmd
			}
		}
	}
	return sv, nil
}

func (sv *StructureView) switchToSection(section StructureSection) {
	// Unfocus current
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(false)
		sv.SectionTables[sv.ActiveSection] = tbl
	}

	sv.ActiveSection = section

	// Focus new
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		tbl.SetFocused(sv.Focused)
		sv.SectionTables[sv.ActiveSection] = tbl
	}
}

func (sv StructureView) View() string {
	t := theme.Current

	// Build section tabs
	sections := []struct {
		name    string
		section StructureSection
		count   int
	}{
		{"1:Columns", SectionColumns, len(sv.Structure.Columns)},
		{"2:Indexes", SectionIndexes, len(sv.Structure.Indexes)},
		{"3:Relations", SectionRelations, len(sv.Structure.Relations)},
		{"4:Triggers", SectionTriggers, len(sv.Structure.Triggers)},
	}

	var tabItems []string
	for _, sec := range sections {
		var tabStyle lipgloss.Style
		label := sec.name + " (" + intToStr(sec.count) + ")"
		if sec.section == sv.ActiveSection {
			tabStyle = t.TableHeader.Copy().
				Background(t.Colors.Primary).
				Foreground(t.Colors.Background).
				Padding(0, 1)
		} else {
			tabStyle = t.TableHeader.Copy().
				Foreground(t.Colors.ForegroundDim).
				Padding(0, 1)
		}
		tabItems = append(tabItems, tabStyle.Render(label))
	}

	sectionBar := lipgloss.JoinHorizontal(lipgloss.Left, tabItems...)

	// Get active section content
	var content string
	if tbl, ok := sv.SectionTables[sv.ActiveSection]; ok {
		content = tbl.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, sectionBar, content)
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// intToStr converts int to string
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Model represents a tabbed interface for multiple tables
type Model struct {
	tabs      []Tab
	activeTab int
	width     int
	height    int
	focused   bool
}

// New creates a new tab model
func New() Model {
	return Model{
		tabs:      []Tab{},
		activeTab: -1,
		focused:   false,
	}
}

// SetSize sets the tab container dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Update all tab content sizes - tab bar takes 1 line
	for i := range m.tabs {
		switch m.tabs[i].Type {
		case TabTypeTable:
			if table, ok := m.tabs[i].Content.(table.Model); ok {
				table.SetSize(width, height-1)
				m.tabs[i].Content = table
			}
		case TabTypeStructure:
			if sv, ok := m.tabs[i].Content.(StructureView); ok {
				sv.SetSize(width, height-1)
				m.tabs[i].Content = sv
			}
		}
	}
}

// SetFocused sets whether the tabs are focused
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		switch m.tabs[m.activeTab].Type {
		case TabTypeTable:
			if table, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
				table.SetFocused(focused)
				m.tabs[m.activeTab].Content = table
			}
		case TabTypeStructure:
			if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
				sv.SetFocused(focused)
				m.tabs[m.activeTab].Content = sv
			}
		}
	}
}

// Focused returns whether the tabs are focused
func (m Model) Focused() bool {
	return m.focused
}

// HasTabs returns true if there are any tabs
func (m Model) HasTabs() bool {
	return len(m.tabs) > 0
}

// ActiveTab returns the currently active tab
func (m Model) ActiveTab() *Tab {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return &m.tabs[m.activeTab]
	}
	return nil
}

// GetActiveTabName returns the name of the active tab
func (m Model) GetActiveTabName() string {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab].Name
	}
	return ""
}

// GetActiveTabType returns the type of the active tab
func (m Model) GetActiveTabType() TabType {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab].Type
	}
	return TabTypeTable
}

// UpdateActiveTabContent updates the content of the active tab
func (m *Model) UpdateActiveTabContent(content interface{}) {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].Content = content
	}
}

// GetActiveTabData returns the original data and columns for the active tab
func (m Model) GetActiveTabData() ([]table.Row, []table.Column, []string) {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		tab := m.tabs[m.activeTab]
		return tab.AllRows, tab.Columns, tab.ColumnNames
	}
	return nil, nil, nil
}

// GetActiveTabFilter returns the active filter for the current tab
func (m Model) GetActiveTabFilter() *filter.Filter {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		filters := m.tabs[m.activeTab].ActiveFilters
		if len(filters) > 0 {
			return &filters[0]
		}
	}
	return nil
}

// GetActiveTabFilters returns all active filters for the current tab
func (m Model) GetActiveTabFilters() []filter.Filter {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab].ActiveFilters
	}
	return nil
}

// SetActiveTabFilter sets the filter for the current tab (replaces all filters)
func (m *Model) SetActiveTabFilter(f *filter.Filter) {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		if f != nil {
			m.tabs[m.activeTab].ActiveFilters = []filter.Filter{*f}
		} else {
			m.tabs[m.activeTab].ActiveFilters = nil
		}
	}
}

// AddActiveTabFilter adds a new filter to the current tab
func (m *Model) AddActiveTabFilter(f filter.Filter) {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].ActiveFilters = append(m.tabs[m.activeTab].ActiveFilters, f)
	}
}

// RemoveActiveTabFilter removes a filter at the given index
func (m *Model) RemoveActiveTabFilter(index int) {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		filters := m.tabs[m.activeTab].ActiveFilters
		if index >= 0 && index < len(filters) {
			m.tabs[m.activeTab].ActiveFilters = append(filters[:index], filters[index+1:]...)
		}
	}
}

// ClearActiveTabFilters clears all filters for the current tab
func (m *Model) ClearActiveTabFilters() {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].ActiveFilters = nil
	}
}

// AddTableTab adds a new tab with table data
func (m *Model) AddTableTab(name string, columns []table.Column, rows []table.Row) {
	logger.Debug("AddTableTab called", map[string]any{
		"name":    name,
		"columns": len(columns),
		"rows":    len(rows),
		"width":   m.width,
		"height":  m.height,
	})

	newTable := table.New(columns, rows)
	newTable.SetSize(m.width, m.height-3)
	newTable.SetFocused(m.focused)
	logger.Info("Adding table to tab", map[string]any{
		"name": name,
		"type": TabTypeTable,
	})

	// Extract column names
	columnNames := make([]string, len(columns))
	for i, col := range columns {
		columnNames[i] = col.Title
	}

	newTab := Tab{
		Name:        name,
		Content:     newTable,
		Type:        TabTypeTable,
		Active:      true,
		AllRows:     rows,
		Columns:     columns,
		ColumnNames: columnNames,
	}

	m.addTab(newTab)
}

// addTab is a helper to add a tab and manage active state
func (m *Model) addTab(newTab Tab) {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].Active = false
		switch m.tabs[m.activeTab].Type {
		case TabTypeTable:
			if table, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
				table.SetFocused(false)
				m.tabs[m.activeTab].Content = table
			}
		case TabTypeStructure:
			if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
				sv.SetFocused(false)
				m.tabs[m.activeTab].Content = sv
			}
		}
	}

	m.tabs = append(m.tabs, newTab)
	m.activeTab = len(m.tabs) - 1
}

// AddStructureTab adds a new tab with table structure data
func (m *Model) AddStructureTab(name string, structure *drivers.TableStructure) {
	logger.Debug("AddStructureTab called", map[string]any{
		"name":      name,
		"columns":   len(structure.Columns),
		"indexes":   len(structure.Indexes),
		"relations": len(structure.Relations),
		"triggers":  len(structure.Triggers),
	})

	sv := NewStructureView(structure, m.width, m.height-3)
	sv.SetFocused(m.focused)

	newTab := Tab{
		Name:    name,
		Content: sv,
		Type:    TabTypeStructure,
		Active:  true,
	}

	m.addTab(newTab)
}

// SwitchTab switches to the tab at the given index
func (m *Model) SwitchTab(index int) {
	if index < 0 || index >= len(m.tabs) {
		return
	}

	// Deactivate current tab
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].Active = false
		switch m.tabs[m.activeTab].Type {
		case TabTypeTable:
			if table, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
				table.SetFocused(false)
				m.tabs[m.activeTab].Content = table
			}
		case TabTypeStructure:
			if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
				sv.SetFocused(false)
				m.tabs[m.activeTab].Content = sv
			}
		}
	}

	m.activeTab = index
	m.tabs[m.activeTab].Active = true
	switch m.tabs[m.activeTab].Type {
	case TabTypeTable:
		if table, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
			table.SetFocused(m.focused)
			m.tabs[m.activeTab].Content = table
		}
	case TabTypeStructure:
		if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
			sv.SetFocused(m.focused)
			m.tabs[m.activeTab].Content = sv
		}
	}
}

// NextTab switches to the next tab
func (m *Model) NextTab() {
	if len(m.tabs) <= 1 {
		return
	}
	nextIndex := (m.activeTab + 1) % len(m.tabs)
	m.SwitchTab(nextIndex)
}

// PrevTab switches to the previous tab
func (m *Model) PrevTab() {
	if len(m.tabs) <= 1 {
		return
	}
	prevIndex := (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
	m.SwitchTab(prevIndex)
}

// CloseTab closes the tab at the given index
func (m *Model) CloseTab(index int) {
	if index < 0 || index >= len(m.tabs) {
		return
	}

	m.tabs = slices.Delete(m.tabs, index, index+1)

	if len(m.tabs) == 0 {
		m.activeTab = -1
	} else if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
		m.tabs[m.activeTab].Active = true
		m.focusActiveTab()
	} else if index <= m.activeTab {
		m.activeTab--
		if m.activeTab < 0 {
			m.activeTab = 0
		}
		m.tabs[m.activeTab].Active = true
		m.focusActiveTab()
	}
}

// focusActiveTab focuses the content of the active tab
func (m *Model) focusActiveTab() {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return
	}
	switch m.tabs[m.activeTab].Type {
	case TabTypeTable:
		if table, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
			table.SetFocused(m.focused)
			m.tabs[m.activeTab].Content = table
		}
	case TabTypeStructure:
		if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
			sv.SetFocused(m.focused)
			m.tabs[m.activeTab].Content = sv
		}
	}
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
			return m, nil
		}

		switch msg.String() {
		case "]":
			m.NextTab()
		case "[":
			m.PrevTab()
		case "ctrl+w":
			m.CloseTab(m.activeTab)
		default:
			switch m.tabs[m.activeTab].Type {
			case TabTypeTable:
				if tbl, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
					var cmd tea.Cmd
					tbl, cmd = tbl.Update(msg)
					m.tabs[m.activeTab].Content = tbl
					return m, cmd
				}
			case TabTypeStructure:
				if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
					var cmd tea.Cmd
					sv, cmd = sv.Update(msg)
					m.tabs[m.activeTab].Content = sv
					return m, cmd
				}
			}
		}
	}

	return m, nil
}

// View renders the tab interface
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if len(m.tabs) == 0 {
		return ""
	}

	t := theme.Current

	var tabItems []string
	for _, tab := range m.tabs {
		var tabStyle lipgloss.Style
		if tab.Active {
			tabStyle = t.TableHeader.Copy().
				Background(t.Colors.Primary).
				Foreground(t.Colors.Background)
		} else {
			tabStyle = t.TableHeader.Copy().
				Foreground(t.Colors.ForegroundDim)
		}

		name := tab.Name
		// Add icon based on tab type
		if tab.Type == TabTypeStructure {
			name = "[S] " + name
		}
		if len(name) > 18 {
			name = name[:15] + "..."
		}

		closeBtn := " ✕"
		if tab.Active {
			closeBtn = lipgloss.NewStyle().
				Foreground(t.Colors.Background).
				Background(t.Colors.Error).
				Render(" ✕")
		} else {
			closeBtn = lipgloss.NewStyle().
				Foreground(t.Colors.ForegroundDim).
				Render(" ✕")
		}

		tabItem := tabStyle.
			Padding(0, 1).
			Render(name + closeBtn)
		tabItems = append(tabItems, tabItem)
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Left, tabItems...)

	var contentView string
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		switch m.tabs[m.activeTab].Type {
		case TabTypeTable:
			if tbl, ok := m.tabs[m.activeTab].Content.(table.Model); ok {
				contentView = tbl.View()
			}
		case TabTypeStructure:
			if sv, ok := m.tabs[m.activeTab].Content.(StructureView); ok {
				contentView = sv.View()
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, contentView)
}
