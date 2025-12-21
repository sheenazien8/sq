package drivers

const (
	DriverMySQL string = "mysql"
)

type Driver interface {
	Connect(urlstr string) error
	TestConnection(urlstr string) error
	GetTables(database string) (map[string][]string, error)
	GetTableColumns(database, table string) ([][]string, error)
	GetTableData(database, table string) ([][]string, error)
}
