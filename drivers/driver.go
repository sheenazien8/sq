package drivers

const (
	DriverMySQL string = "mysql"
)

// Pagination represents pagination parameters
type Pagination struct {
	Page       int
	PageSize   int
	SortColumn string // Column name to sort by (empty = no sort)
	SortOrder  string // "ASC" or "DESC"
}

// PaginatedResult represents paginated query results
type PaginatedResult struct {
	Data       [][]string
	TotalRows  int
	Page       int
	PageSize   int
	TotalPages int
}

type Driver interface {
	Connect(urlstr string) error
	TestConnection(urlstr string) error
	GetTables(database string) (map[string][]string, error)
	GetTableColumns(database, table string) ([][]string, error)
	GetTableData(database, table string) ([][]string, error)
	GetTableDataWithFilter(database, table string, whereClause string) ([][]string, error)

	// Paginated data methods
	GetTableDataPaginated(database, table string, pagination Pagination) (*PaginatedResult, error)
	GetTableDataWithFilterPaginated(database, table string, whereClause string, pagination Pagination) (*PaginatedResult, error)

	// Table structure methods
	GetTableStructure(database, table string) (*TableStructure, error)
	GetColumnInfo(database, table string) ([]ColumnInfo, error)
	GetIndexInfo(database, table string) ([]IndexInfo, error)
	GetRelationInfo(database, table string) ([]RelationInfo, error)
	GetTriggerInfo(database, table string) ([]TriggerInfo, error)

	// Query execution
	ExecuteQuery(query string) ([][]string, error)
}
