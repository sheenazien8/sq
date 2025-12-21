package drivers

const (
	DriverMySQL string = "mysql"
)

// FilterCondition represents a single filter condition
type FilterCondition struct {
	Column   string
	Operator string
	Value    string
}

type Driver interface {
	Connect(urlstr string) error
	TestConnection(urlstr string) error
	GetTables(database string) (map[string][]string, error)
	GetTableColumns(database, table string) ([][]string, error)
	GetTableData(database, table string) ([][]string, error)
	GetTableDataWithFilter(database, table string, filters []FilterCondition) ([][]string, error)

	// Table structure methods
	GetTableStructure(database, table string) (*TableStructure, error)
	GetColumnInfo(database, table string) ([]ColumnInfo, error)
	GetIndexInfo(database, table string) ([]IndexInfo, error)
	GetRelationInfo(database, table string) ([]RelationInfo, error)
	GetTriggerInfo(database, table string) ([]TriggerInfo, error)

	// Query execution
	ExecuteQuery(query string) ([][]string, error)
}
