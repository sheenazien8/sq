package drivers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	"github.com/sheenazien8/sq/logger"
	"github.com/xo/dburl"
)

type PostgreSQL struct {
	Connection       *sql.DB
	Provider         string
	Schema           string // Current schema (for backward compatibility)
	CurrentDatabase  string // Current database name
	PreviousDatabase string // Previous database name for reverting
}

func (db *PostgreSQL) Connect(urlstr string) (err error) {
	db.SetProvider(DriverPostgreSQL)

	db.Connection, err = dburl.Open(urlstr)
	if err != nil {
		return err
	}

	err = db.Connection.Ping()
	if err != nil {
		return err
	}

	// Detect and set the schema
	err = db.detectSchema()
	if err != nil {
		return err
	}

	return nil
}

func (db *PostgreSQL) SetProvider(provider string) {
	db.Provider = provider
}

// SwitchDatabase switches to a different database (in PostgreSQL, databases are separate)
// For PostgreSQL, this is primarily for tracking which database is currently active
func (db *PostgreSQL) SwitchDatabase(database string) error {
	if database == "" {
		return fmt.Errorf("database name is required")
	}

	db.PreviousDatabase = db.CurrentDatabase
	db.CurrentDatabase = database

	logger.Debug("Switched database", map[string]any{
		"database": database,
	})

	return nil
}

// detectSchema attempts to find an appropriate schema to use
// Priority: public schema > first user-created schema > first available schema
func (db *PostgreSQL) detectSchema() error {
	// First, try to use the public schema
	query := `SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'public'`
	var schemaName string
	err := db.Connection.QueryRow(query).Scan(&schemaName)
	if err == nil {
		db.Schema = "public"
		return nil
	}

	// If public doesn't exist, try to find user-created schemas (exclude system schemas)
	query = `SELECT schema_name FROM information_schema.schemata 
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast') 
		ORDER BY schema_name LIMIT 1`
	err = db.Connection.QueryRow(query).Scan(&schemaName)
	if err == nil {
		db.Schema = schemaName
		logger.Debug("Using schema", map[string]any{"schema": schemaName})
		return nil
	}

	// Fallback: use public if nothing else works
	db.Schema = "public"
	return nil
}

func (db *PostgreSQL) TestConnection(urlstr string) error {
	conn, err := dburl.Open(urlstr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Ping()
}

// GetTables returns all tables for a given database, organized by schema
func (db *PostgreSQL) GetTables(database string) (map[string][]string, error) {
	if database == "" {
		return nil, fmt.Errorf("database name is required")
	}

	// Store the current database
	db.PreviousDatabase = db.CurrentDatabase
	db.CurrentDatabase = database

	// Query all tables from all schemas in the current database, excluding system schemas
	query := `SELECT table_name, table_schema FROM information_schema.tables 
		WHERE table_catalog = $1 AND table_type = 'BASE TABLE'
		AND table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast', 'pg_temp_1')
		ORDER BY table_schema, table_name`
	rows, err := db.Connection.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make(map[string][]string)
	for rows.Next() {
		var tableName, tableSchema string
		if err := rows.Scan(&tableName, &tableSchema); err != nil {
			return nil, err
		}
		// Organize tables by schema
		tables[tableSchema] = append(tables[tableSchema], tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If public schema has tables, set it as the default
	if _, exists := tables["public"]; exists {
		db.Schema = "public"
	} else if len(tables) > 0 {
		// Otherwise use the first available schema
		for schema := range tables {
			db.Schema = schema
			break
		}
	}

	return tables, nil
}

// GetTableColumns returns basic column information for a table
func (db *PostgreSQL) GetTableColumns(database, table string) ([][]string, error) {
	query := `
		SELECT 
			column_name, 
			data_type, 
			is_nullable,
			column_default,
			''::text as key_type
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`
	rows, err := db.Connection.Query(query, db.Schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns [][]string
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault, keyType sql.NullString
		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault, &keyType); err != nil {
			return nil, err
		}

		defaultValue := ""
		if columnDefault.Valid {
			defaultValue = columnDefault.String
		}

		keyTypeValue := ""
		if keyType.Valid {
			keyTypeValue = keyType.String
		}

		columns = append(columns, []string{columnName, dataType, isNullable, keyTypeValue, defaultValue, ""})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// GetTableData returns all data from a table with a limit
func (db *PostgreSQL) GetTableData(database, table string) ([][]string, error) {
	query := `SELECT * FROM "` + db.Schema + `"."` + table + `" LIMIT 1000`
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
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
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
func (db *PostgreSQL) GetTableDataWithFilter(database, table string, whereClause string) ([][]string, error) {
	query := `SELECT * FROM "` + db.Schema + `"."` + table + `"`

	// Use raw WHERE clause if provided
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	query += " LIMIT 1000"

	// Log the SQL query
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
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
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
func (db *PostgreSQL) GetTableDataPaginated(database, table string, pagination Pagination) (*PaginatedResult, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM "` + db.Schema + `"."` + table + `"`
	var totalRows int
	if err := db.Connection.QueryRow(countQuery).Scan(&totalRows); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := max((pagination.Page-1)*pagination.PageSize, 0)

	// Get paginated data
	query := `SELECT * FROM "` + db.Schema + `"."` + table + `"`

	// Add ORDER BY if sort column is specified
	if pagination.SortColumn != "" {
		sortOrder := pagination.SortOrder
		if sortOrder != "DESC" {
			sortOrder = "ASC"
		}
		query += ` ORDER BY "` + pagination.SortColumn + `" ` + sortOrder
	}

	query += " LIMIT " + strconv.Itoa(pagination.PageSize) + " OFFSET " + strconv.Itoa(offset)

	logger.Debug("Executing paginated query", map[string]any{
		"query":    query,
		"page":     pagination.Page,
		"pageSize": pagination.PageSize,
		"offset":   offset,
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
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
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
func (db *PostgreSQL) GetTableDataWithFilterPaginated(database, table string, whereClause string, pagination Pagination) (*PaginatedResult, error) {
	baseQuery := `SELECT * FROM "` + db.Schema + `"."` + table + `"`
	countQuery := `SELECT COUNT(*) FROM "` + db.Schema + `"."` + table + `"`

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
		query += ` ORDER BY "` + pagination.SortColumn + `" ` + sortOrder
	}

	query += " LIMIT " + strconv.Itoa(pagination.PageSize) + " OFFSET " + strconv.Itoa(offset)

	logger.Debug("Executing filtered paginated query", map[string]any{
		"query":    query,
		"page":     pagination.Page,
		"pageSize": pagination.PageSize,
		"offset":   offset,
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
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
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

// GetTableStructure returns complete table structure including columns, indexes, relations, and triggers
func (db *PostgreSQL) GetTableStructure(database, table string) (*TableStructure, error) {
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
func (db *PostgreSQL) GetColumnInfo(database, table string) ([]ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			CASE WHEN c.is_nullable = 'YES' THEN true ELSE false END as is_nullable,
			c.column_default,
			false as is_primary_key,
			''::text as extra,
			''::text as comment
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := db.Connection.Query(query, db.Schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable, isPrimaryKey bool
		var defaultValue sql.NullString
		var comment sql.NullString

		if err := rows.Scan(&col.Name, &col.DataType, &isNullable, &defaultValue, &isPrimaryKey, &col.Extra, &comment); err != nil {
			return nil, err
		}

		col.Nullable = isNullable
		col.IsPrimaryKey = isPrimaryKey
		col.DefaultValue = defaultValue.String
		col.Comment = comment.String

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// GetIndexInfo returns index information for a table
func (db *PostgreSQL) GetIndexInfo(database, table string) ([]IndexInfo, error) {
	query := `
		SELECT
			indexname,
			indexdef,
			CASE WHEN indexdef ~* 'UNIQUE' THEN true ELSE false END as is_unique,
			CASE WHEN indexdef ~* 'PRIMARY KEY' THEN true ELSE false END as is_primary
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2
		ORDER BY indexname
	`

	rows, err := db.Connection.Query(query, db.Schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var indexDef string
		var isUnique, isPrimary bool

		if err := rows.Scan(&idx.Name, &indexDef, &isUnique, &isPrimary); err != nil {
			return nil, err
		}

		idx.IsUnique = isUnique
		idx.IsPrimary = isPrimary
		idx.Type = "BTREE" // Default type for PostgreSQL

		// Try to extract column names from CREATE INDEX statement
		// This is a simplified approach
		if strings.Contains(indexDef, "(") && strings.Contains(indexDef, ")") {
			start := strings.Index(indexDef, "(") + 1
			end := strings.LastIndex(indexDef, ")")
			if start > 0 && end > start {
				colStr := indexDef[start:end]
				colStr = strings.TrimSpace(colStr)
				idx.Columns = []string{colStr}
			}
		}

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// GetRelationInfo returns foreign key relationships for a table
func (db *PostgreSQL) GetRelationInfo(database, table string) ([]RelationInfo, error) {
	query := `
		SELECT
			constraint_name,
			column_name,
			foreign_table_name,
			foreign_column_name,
			update_rule,
			delete_rule
		FROM (
			SELECT
				tc.constraint_name,
				kcu.column_name,
				ccu.table_name AS foreign_table_name,
				ccu.column_name AS foreign_column_name,
				rc.update_rule,
				rc.delete_rule
			FROM information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			JOIN information_schema.referential_constraints AS rc
				ON rc.constraint_name = tc.constraint_name
				AND rc.constraint_schema = tc.table_schema
			WHERE tc.constraint_type = 'FOREIGN KEY'
				AND tc.table_schema = $1
				AND tc.table_name = $2
		) AS fks
		ORDER BY constraint_name, column_name
	`

	rows, err := db.Connection.Query(query, db.Schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []RelationInfo
	for rows.Next() {
		var rel RelationInfo

		if err := rows.Scan(&rel.Name, &rel.Column, &rel.ReferencedTable, &rel.ReferencedColumn, &rel.OnUpdate, &rel.OnDelete); err != nil {
			return nil, err
		}

		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// GetTriggerInfo returns trigger information for a table
func (db *PostgreSQL) GetTriggerInfo(database, table string) ([]TriggerInfo, error) {
	query := `
		SELECT
			trigger_name,
			event_manipulation,
			action_timing,
			action_statement,
			event_object_table
		FROM information_schema.triggers
		WHERE trigger_schema = $1 AND event_object_table = $2
		ORDER BY trigger_name
	`

	rows, err := db.Connection.Query(query, db.Schema, table)
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

// ExecuteQuery executes a raw SQL query and returns the results
func (db *PostgreSQL) ExecuteQuery(query string) ([][]string, error) {
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
