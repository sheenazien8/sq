package drivers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/sheenazien8/sq/logger"
	_ "modernc.org/sqlite"
)

type SQLite struct {
	Connection *sql.DB
	Provider   string
	FilePath   string // Path to SQLite database file
}

func (db *SQLite) Connect(urlstr string) error {
	db.SetProvider(DriverSQLite)

	// SQLite URL format: sqlite:///path/to/database.db or file:path/to/database.db
	// We need to extract the file path from the URL
	filePath := strings.TrimPrefix(urlstr, "sqlite://")
	filePath = strings.TrimPrefix(filePath, "file:")
	filePath = strings.TrimPrefix(filePath, "//")

	if filePath == "" {
		return fmt.Errorf("SQLite database file path is required")
	}

	db.FilePath = filePath

	var err error
	db.Connection, err = sql.Open("sqlite", "file:"+filePath)
	if err != nil {
		return err
	}

	// Enable foreign keys support in SQLite
	_, err = db.Connection.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	err = db.Connection.Ping()
	if err != nil {
		return err
	}

	logger.Debug("Connected to SQLite database", map[string]any{
		"filePath": filePath,
	})

	return nil
}

func (db *SQLite) SetProvider(provider string) {
	db.Provider = provider
}

func (db *SQLite) TestConnection(urlstr string) error {
	filePath := strings.TrimPrefix(urlstr, "sqlite://")
	filePath = strings.TrimPrefix(filePath, "file:")
	filePath = strings.TrimPrefix(filePath, "//")

	if filePath == "" {
		return fmt.Errorf("SQLite database file path is required")
	}

	conn, err := sql.Open("sqlite", "file:"+filePath)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Enable foreign keys support
	_, err = conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	return conn.Ping()
}

// GetTables returns all tables in the SQLite database
// For SQLite, there's no concept of "databases" within a file, so we use the file name as database
func (db *SQLite) GetTables(database string) (map[string][]string, error) {
	query := `
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.Connection.Query(query)
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

// GetTableColumns returns column information for a table
func (db *SQLite) GetTableColumns(database, table string) ([][]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteIdentifier(table))

	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns [][]string
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notnull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notnull, &defaultValue, &pk); err != nil {
			return nil, err
		}

		isNullable := "YES"
		if notnull == 1 {
			isNullable = "NO"
		}

		columnKey := ""
		if pk == 1 {
			columnKey = "PRI"
		}

		columns = append(columns, []string{
			name,
			dataType,
			isNullable,
			columnKey,
			defaultValue.String,
			"",
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// GetTableData returns all data from a table with a limit
func (db *SQLite) GetTableData(database, table string) ([][]string, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1000", quoteIdentifier(table))

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

// GetTableDataWithFilter returns filtered table data
func (db *SQLite) GetTableDataWithFilter(database, table string, whereClause string) ([][]string, error) {
	query := fmt.Sprintf("SELECT * FROM %s", quoteIdentifier(table))

	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	query += " LIMIT 1000"

	logger.Debug("Executing filtered query", map[string]any{
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

// GetTableDataPaginated returns paginated table data
func (db *SQLite) GetTableDataPaginated(database, table string, pagination Pagination) (*PaginatedResult, error) {
	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdentifier(table))
	var totalRows int
	if err := db.Connection.QueryRow(countQuery).Scan(&totalRows); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.PageSize
	if offset < 0 {
		offset = 0
	}

	// Get paginated data
	query := fmt.Sprintf("SELECT * FROM %s", quoteIdentifier(table))

	// Add ORDER BY if sort column is specified
	if pagination.SortColumn != "" {
		sortOrder := pagination.SortOrder
		if sortOrder != "DESC" {
			sortOrder = "ASC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", quoteIdentifier(pagination.SortColumn), sortOrder)
	}

	query += " LIMIT " + strconv.Itoa(pagination.PageSize) + " OFFSET " + strconv.Itoa(offset)

	logger.Debug("Executing paginated query", map[string]any{
		"query":     query,
		"page":      pagination.Page,
		"pageSize":  pagination.PageSize,
		"offset":    offset,
		"totalRows": totalRows,
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

	// Calculate total pages
	totalPages := totalRows / pagination.PageSize
	if totalRows%pagination.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       data,
		TotalRows:  totalRows,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTableDataWithFilterPaginated returns paginated and filtered table data
func (db *SQLite) GetTableDataWithFilterPaginated(database, table string, whereClause string, pagination Pagination) (*PaginatedResult, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", quoteIdentifier(table))
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdentifier(table))

	// Use raw WHERE clause if provided
	if whereClause != "" {
		baseQuery += " WHERE " + whereClause
		countQuery += " WHERE " + whereClause
	}

	// Get total count with filters
	var totalRows int
	if err := db.Connection.QueryRow(countQuery).Scan(&totalRows); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := max((pagination.Page-1)*pagination.PageSize, 0)

	// Build final query with pagination
	query := baseQuery

	// Add ORDER BY if sort column is specified
	if pagination.SortColumn != "" {
		sortOrder := pagination.SortOrder
		if sortOrder != "DESC" {
			sortOrder = "ASC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", quoteIdentifier(pagination.SortColumn), sortOrder)
	}

	query += " LIMIT " + strconv.Itoa(pagination.PageSize) + " OFFSET " + strconv.Itoa(offset)

	logger.Debug("Executing filtered paginated query", map[string]any{
		"query":     query,
		"page":      pagination.Page,
		"pageSize":  pagination.PageSize,
		"offset":    offset,
		"totalRows": totalRows,
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

	// Calculate total pages
	totalPages := totalRows / pagination.PageSize
	if totalRows%pagination.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       data,
		TotalRows:  totalRows,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTableStructure returns complete table structure including columns, indexes, and relations
func (db *SQLite) GetTableStructure(database, table string) (*TableStructure, error) {
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

	// SQLite doesn't support triggers in the same way, but we can still try to get them
	triggers, err := db.GetTriggerInfo(database, table)
	if err != nil {
		// Don't fail if we can't get triggers
		triggers = []TriggerInfo{}
	}

	return &TableStructure{
		Columns:   columns,
		Indexes:   indexes,
		Relations: relations,
		Triggers:  triggers,
	}, nil
}

// GetColumnInfo returns detailed column information for a table
func (db *SQLite) GetColumnInfo(database, table string) ([]ColumnInfo, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteIdentifier(table))

	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notnull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notnull, &defaultValue, &pk); err != nil {
			return nil, err
		}

		col := ColumnInfo{
			Name:         name,
			DataType:     dataType,
			Nullable:     notnull == 0,
			IsPrimaryKey: pk == 1,
			DefaultValue: defaultValue.String,
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// GetIndexInfo returns index information for a table
func (db *SQLite) GetIndexInfo(database, table string) ([]IndexInfo, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", quoteIdentifier(table))

	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Skip auto-generated indexes
		if origin == "c" {
			continue
		}

		// Get index columns
		indexInfoQuery := fmt.Sprintf("PRAGMA index_info(%s)", quoteIdentifier(name))
		indexRows, err := db.Connection.Query(indexInfoQuery)
		if err != nil {
			continue
		}

		var columns []string
		for indexRows.Next() {
			var seqno int
			var cid int
			var colName string
			if err := indexRows.Scan(&seqno, &cid, &colName); err != nil {
				continue
			}
			columns = append(columns, colName)
		}
		indexRows.Close()

		idx := IndexInfo{
			Name:      name,
			Columns:   columns,
			IsUnique:  unique == 1,
			IsPrimary: origin == "pk",
			Type:      "BTREE", // SQLite primarily uses B-tree indexes
		}

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// GetRelationInfo returns foreign key relationships for a table
func (db *SQLite) GetRelationInfo(database, table string) ([]RelationInfo, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", quoteIdentifier(table))

	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []RelationInfo
	for rows.Next() {
		var id int
		var seq int
		var table_ string
		var from string
		var to string
		var onUpdate string
		var onDelete string
		var match string

		if err := rows.Scan(&id, &seq, &table_, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		rel := RelationInfo{
			Name:             fmt.Sprintf("fk_%s_%s", table, from),
			Column:           from,
			ReferencedTable:  table_,
			ReferencedColumn: to,
			OnUpdate:         onUpdate,
			OnDelete:         onDelete,
		}

		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// GetTriggerInfo returns trigger information for a table
func (db *SQLite) GetTriggerInfo(database, table string) ([]TriggerInfo, error) {
	query := `
		SELECT name, tbl_name, sql FROM sqlite_master 
		WHERE type='trigger' AND tbl_name = ?
		ORDER BY name
	`

	rows, err := db.Connection.Query(query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []TriggerInfo
	for rows.Next() {
		var name string
		var tableName string
		var sql sql.NullString

		if err := rows.Scan(&name, &tableName, &sql); err != nil {
			return nil, err
		}

		// Parse trigger information from SQL
		trigger := TriggerInfo{
			Name:      name,
			Table:     tableName,
			Statement: sql.String,
			Event:     "UNKNOWN",
			Timing:    "UNKNOWN",
		}

		// Try to extract timing and event from trigger SQL
		upperSQL := strings.ToUpper(sql.String)

		if strings.Contains(upperSQL, "BEFORE") {
			trigger.Timing = "BEFORE"
		} else if strings.Contains(upperSQL, "AFTER") {
			trigger.Timing = "AFTER"
		} else if strings.Contains(upperSQL, "INSTEAD OF") {
			trigger.Timing = "INSTEAD OF"
		}

		if strings.Contains(upperSQL, "INSERT") {
			trigger.Event = "INSERT"
		} else if strings.Contains(upperSQL, "UPDATE") {
			trigger.Event = "UPDATE"
		} else if strings.Contains(upperSQL, "DELETE") {
			trigger.Event = "DELETE"
		}

		triggers = append(triggers, trigger)
	}

	return triggers, rows.Err()
}

// ExecuteQuery executes a raw SQL query and returns the results
func (db *SQLite) ExecuteQuery(query string) ([][]string, error) {
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

// quoteIdentifier safely quotes a table or column name for SQLite
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
