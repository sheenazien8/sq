package drivers

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/xo/dburl"
)

type MySQL struct {
	Connection *sql.DB
	Provider   string
}

func (db *MySQL) Connect(urlstr string) (err error) {
	db.SetProvider(DriverMySQL)

	db.Connection, err = dburl.Open(urlstr)
	if err != nil {
		return err
	}

	err = db.Connection.Ping()
	if err != nil {
		return err
	}

	return nil
}

func (db *MySQL) SetProvider(provider string) {
	db.Provider = provider
}

func (db *MySQL) TestConnection(urlstr string) error {
	conn, err := dburl.Open(urlstr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Ping()
}

func (db *MySQL) GetTables(database string) (map[string][]string, error) {
	query := "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?"
	rows, err := db.Connection.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make(map[string][]string)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables[database] = append(tables[database], tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

func (db *MySQL) GetTableColumns(database, table string) ([][]string, error) {
	query := "SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION"
	rows, err := db.Connection.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns [][]string
	for rows.Next() {
		var columnName, dataType, isNullable, columnKey string
		var columnDefault, extra sql.NullString
		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnKey, &columnDefault, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, []string{columnName, dataType, isNullable, columnKey, columnDefault.String, extra.String})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}
