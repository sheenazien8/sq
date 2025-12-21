package drivers

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sheenazien8/db-client-tui/logger"
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

func (db *MySQL) GetTableData(database, table string) ([][]string, error) {
	query := "SELECT * FROM " + database + "." + table + " LIMIT 1000"
	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var data [][]string
	// Add header row
	data = append(data, columns)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = formatSQLValue(val)
			}
		}
		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

func (db *MySQL) GetTableDataWithFilter(database, table string, filters []FilterCondition) ([][]string, error) {
	query := "SELECT * FROM " + database + "." + table

	var whereClauses []string
	var args []interface{}

	// Build WHERE clause from filters
	for _, f := range filters {
		switch f.Operator {
		case "=":
			whereClauses = append(whereClauses, f.Column+" = ?")
			args = append(args, f.Value)
		case "!=":
			whereClauses = append(whereClauses, f.Column+" != ?")
			args = append(args, f.Value)
		case "LIKE":
			whereClauses = append(whereClauses, f.Column+" LIKE ?")
			args = append(args, "%"+f.Value+"%")
		case "NOT LIKE":
			whereClauses = append(whereClauses, f.Column+" NOT LIKE ?")
			args = append(args, "%"+f.Value+"%")
		case ">":
			whereClauses = append(whereClauses, f.Column+" > ?")
			args = append(args, f.Value)
		case "<":
			whereClauses = append(whereClauses, f.Column+" < ?")
			args = append(args, f.Value)
		case ">=":
			whereClauses = append(whereClauses, f.Column+" >= ?")
			args = append(args, f.Value)
		case "<=":
			whereClauses = append(whereClauses, f.Column+" <= ?")
			args = append(args, f.Value)
		case "IS":
			// IS operator typically used for NULL checks - don't use placeholder
			whereClauses = append(whereClauses, f.Column+" IS "+f.Value)
		case "IS NOT":
			// IS NOT operator typically used for NULL checks - don't use placeholder
			whereClauses = append(whereClauses, f.Column+" IS NOT "+f.Value)
		}
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			query += " AND " + whereClauses[i]
		}
	}

	query += " LIMIT 1000"

	// Log the SQL query with parameters
	logger.Debug("Executing filtered query", map[string]any{
		"query":      query,
		"parameters": args,
	})

	rows, err := db.Connection.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var data [][]string
	// Add header row
	data = append(data, columns)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = formatSQLValue(val)
			}
		}
		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// formatSQLValue converts various SQL types to string
func formatSQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		// For any other types, use fmt.Sprintf
		return fmt.Sprintf("%v", v)
	}
}
