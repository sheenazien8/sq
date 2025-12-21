package drivers

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sheenazien8/sq/logger"
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

// GetTableStructure returns complete table structure including columns, indexes, relations, and triggers
func (db *MySQL) GetTableStructure(database, table string) (*TableStructure, error) {
	columns, err := db.GetColumnInfo(database, table)
	if err != nil {
		return nil, err
	}

	indexes, err := db.GetIndexInfo(database, table)
	if err != nil {
		return nil, err
	}

	relations, err := db.GetRelationInfo(database, table)
	if err != nil {
		return nil, err
	}

	triggers, err := db.GetTriggerInfo(database, table)
	if err != nil {
		return nil, err
	}

	return &TableStructure{
		Columns:   columns,
		Indexes:   indexes,
		Relations: relations,
		Triggers:  triggers,
	}, nil
}

// GetColumnInfo returns detailed column information for a table
func (db *MySQL) GetColumnInfo(database, table string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_KEY,
			COLUMN_DEFAULT,
			EXTRA,
			COLUMN_COMMENT
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? 
		ORDER BY ORDINAL_POSITION`

	rows, err := db.Connection.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable, columnKey string
		var defaultValue, extra, comment sql.NullString

		if err := rows.Scan(&col.Name, &col.DataType, &isNullable, &columnKey, &defaultValue, &extra, &comment); err != nil {
			return nil, err
		}

		col.Nullable = isNullable == "YES"
		col.IsPrimaryKey = columnKey == "PRI"
		col.DefaultValue = defaultValue.String
		col.Extra = extra.String
		col.Comment = comment.String

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// GetIndexInfo returns index information for a table
func (db *MySQL) GetIndexInfo(database, table string) ([]IndexInfo, error) {
	query := `
		SELECT 
			INDEX_NAME,
			GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) as COLUMNS,
			NOT NON_UNIQUE as IS_UNIQUE,
			INDEX_NAME = 'PRIMARY' as IS_PRIMARY,
			INDEX_TYPE
		FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY INDEX_NAME, NON_UNIQUE, INDEX_TYPE
		ORDER BY INDEX_NAME`

	rows, err := db.Connection.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var columnsStr string
		var isUnique, isPrimary bool

		if err := rows.Scan(&idx.Name, &columnsStr, &isUnique, &isPrimary, &idx.Type); err != nil {
			return nil, err
		}

		idx.Columns = splitColumns(columnsStr)
		idx.IsUnique = isUnique
		idx.IsPrimary = isPrimary

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// GetRelationInfo returns foreign key relationships for a table
func (db *MySQL) GetRelationInfo(database, table string) ([]RelationInfo, error) {
	query := `
		SELECT 
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY CONSTRAINT_NAME, ORDINAL_POSITION`

	rows, err := db.Connection.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []RelationInfo
	for rows.Next() {
		var rel RelationInfo

		if err := rows.Scan(&rel.Name, &rel.Column, &rel.ReferencedTable, &rel.ReferencedColumn); err != nil {
			return nil, err
		}

		relations = append(relations, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get ON UPDATE and ON DELETE actions
	for i := range relations {
		actionQuery := `
			SELECT UPDATE_RULE, DELETE_RULE
			FROM information_schema.REFERENTIAL_CONSTRAINTS
			WHERE CONSTRAINT_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = ?`

		var onUpdate, onDelete string
		err := db.Connection.QueryRow(actionQuery, database, table, relations[i].Name).Scan(&onUpdate, &onDelete)
		if err == nil {
			relations[i].OnUpdate = onUpdate
			relations[i].OnDelete = onDelete
		}
	}

	return relations, nil
}

// GetTriggerInfo returns trigger information for a table
func (db *MySQL) GetTriggerInfo(database, table string) ([]TriggerInfo, error) {
	query := `
		SELECT 
			TRIGGER_NAME,
			EVENT_MANIPULATION,
			ACTION_TIMING,
			ACTION_STATEMENT,
			EVENT_OBJECT_TABLE
		FROM information_schema.TRIGGERS
		WHERE TRIGGER_SCHEMA = ? AND EVENT_OBJECT_TABLE = ?
		ORDER BY TRIGGER_NAME`

	rows, err := db.Connection.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []TriggerInfo
	for rows.Next() {
		var trig TriggerInfo

		if err := rows.Scan(&trig.Name, &trig.Event, &trig.Timing, &trig.Statement, &trig.Table); err != nil {
			return nil, err
		}

		triggers = append(triggers, trig)
	}

	return triggers, rows.Err()
}

// splitColumns splits a comma-separated column string into a slice
func splitColumns(s string) []string {
	if s == "" {
		return nil
	}
	var columns []string
	for _, col := range splitString(s, ',') {
		columns = append(columns, trimSpace(col))
	}
	return columns
}

// splitString splits a string by a separator
func splitString(s string, sep rune) []string {
	var result []string
	var current string
	for _, r := range s {
		if r == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// ExecuteQuery executes a raw SQL query and returns the results
func (db *MySQL) ExecuteQuery(query string) ([][]string, error) {
	logger.Debug("Executing raw query", map[string]any{
		"query": query,
	})

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
