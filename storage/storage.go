package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sheenazien8/sq/drivers"
	_ "modernc.org/sqlite"
)

// DB is the global database connection for app storage
var DB *sql.DB

// Connection represents a saved database connection
type Connection struct {
	ID        int64
	Name      string
	Driver    string // mysql, postgres, sqlite
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SavedQuery represents a saved SQL query
type SavedQuery struct {
	ID           int64
	ConnectionID int64
	Name         string
	Query        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// QueryHistory represents a query execution history entry
type QueryHistory struct {
	ID           int64
	ConnectionID int64
	Query        string
	ExecutedAt   time.Time
	Duration     int64 // milliseconds
	RowsAffected int64
	Error        string
}

// storagePath returns the path to the SQLite database file
func storagePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "sq")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "storage.db"), nil
}

// Init initializes the SQLite database connection and creates tables
func Init() error {
	path, err := storagePath()
	if err != nil {
		return err
	}

	DB, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}

	// Create tables
	if err := createTables(); err != nil {
		return err
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// createTables creates the necessary tables if they don't exist
func createTables() error {
	schema := `
    CREATE TABLE IF NOT EXISTS connections (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        driver TEXT NOT NULL,
        url TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS saved_queries (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        connection_id INTEGER,
        name TEXT NOT NULL,
        query TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS query_history (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        connection_id INTEGER,
        query TEXT NOT NULL,
        executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        duration INTEGER DEFAULT 0,
        rows_affected INTEGER DEFAULT 0,
        error TEXT,
        FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_saved_queries_connection ON saved_queries(connection_id);
    CREATE INDEX IF NOT EXISTS idx_query_history_connection ON query_history(connection_id);
    CREATE INDEX IF NOT EXISTS idx_query_history_executed_at ON query_history(executed_at);
    `

	_, err := DB.Exec(schema)
	return err
}

// CreateConnection creates a new connection and returns its ID
// It tests the connection before saving to ensure it's valid
func CreateConnection(name, driverName, url string) (int64, error) {
	// Test connection before saving
	var driver drivers.Driver

	switch driverName {
	case drivers.DriverTypeMySQL:
		driver = &drivers.MySQL{}
	case drivers.DriverTypePostgreSQL:
		driver = &drivers.PostgreSQL{}
	case drivers.DriverTypeSQLite:
		driver = &drivers.SQLite{}
	case drivers.DriverTypeMongoDB, drivers.DriverTypeMongoDBAtlas:
		// Both MongoDB and MongoDB Atlas use the same driver
		driver = &drivers.MongoDB{}
	default:
		return 0, fmt.Errorf("unsupported driver: %s", driverName)
	}

	if err := driver.TestConnection(url); err != nil {
		return 0, fmt.Errorf("connection test failed: %w", err)
	}

	// Connection is valid, save to database
	result, err := DB.Exec(
		"INSERT INTO connections (name, driver, url) VALUES (?, ?, ?)",
		name, driverName, url,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetConnection retrieves a connection by ID
func GetConnection(id int64) (*Connection, error) {
	conn := &Connection{}
	err := DB.QueryRow(
		"SELECT id, name, driver, url, created_at, updated_at FROM connections WHERE id = ?",
		id,
	).Scan(&conn.ID, &conn.Name, &conn.Driver, &conn.URL, &conn.CreatedAt, &conn.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// GetAllConnections retrieves all saved connections
func GetAllConnections() ([]Connection, error) {
	rows, err := DB.Query(
		"SELECT id, name, driver, url, created_at, updated_at FROM connections ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []Connection
	for rows.Next() {
		var conn Connection
		if err := rows.Scan(&conn.ID, &conn.Name, &conn.Driver, &conn.URL, &conn.CreatedAt, &conn.UpdatedAt); err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}
	return connections, rows.Err()
}

// UpdateConnection updates an existing connection
func UpdateConnection(id int64, name, driver, url string) error {
	_, err := DB.Exec(
		"UPDATE connections SET name = ?, driver = ?, url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		name, driver, url, id,
	)
	return err
}

// DeleteConnection deletes a connection by ID
func DeleteConnection(id int64) error {
	_, err := DB.Exec("DELETE FROM connections WHERE id = ?", id)
	return err
}

// =============================================================================
// SavedQuery CRUD operations
// =============================================================================

// CreateSavedQuery creates a new saved query
func CreateSavedQuery(connectionID int64, name, query string) (int64, error) {
	result, err := DB.Exec(
		"INSERT INTO saved_queries (connection_id, name, query) VALUES (?, ?, ?)",
		connectionID, name, query,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetSavedQueriesByConnection retrieves all saved queries for a connection
func GetSavedQueriesByConnection(connectionID int64) ([]SavedQuery, error) {
	rows, err := DB.Query(
		"SELECT id, connection_id, name, query, created_at, updated_at FROM saved_queries WHERE connection_id = ? ORDER BY name",
		connectionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []SavedQuery
	for rows.Next() {
		var q SavedQuery
		if err := rows.Scan(&q.ID, &q.ConnectionID, &q.Name, &q.Query, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	return queries, rows.Err()
}

// GetAllSavedQueries retrieves all saved queries
func GetAllSavedQueries() ([]SavedQuery, error) {
	rows, err := DB.Query(
		"SELECT id, connection_id, name, query, created_at, updated_at FROM saved_queries ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []SavedQuery
	for rows.Next() {
		var q SavedQuery
		if err := rows.Scan(&q.ID, &q.ConnectionID, &q.Name, &q.Query, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	return queries, rows.Err()
}

// UpdateSavedQuery updates an existing saved query
func UpdateSavedQuery(id int64, name, query string) error {
	_, err := DB.Exec(
		"UPDATE saved_queries SET name = ?, query = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		name, query, id,
	)
	return err
}

// DeleteSavedQuery deletes a saved query by ID
func DeleteSavedQuery(id int64) error {
	_, err := DB.Exec("DELETE FROM saved_queries WHERE id = ?", id)
	return err
}

// =============================================================================
// QueryHistory operations
// =============================================================================

// AddQueryHistory adds a new query history entry
func AddQueryHistory(connectionID int64, query string, duration int64, rowsAffected int64, queryError string) (int64, error) {
	result, err := DB.Exec(
		"INSERT INTO query_history (connection_id, query, duration, rows_affected, error) VALUES (?, ?, ?, ?, ?)",
		connectionID, query, duration, rowsAffected, queryError,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetQueryHistory retrieves query history for a connection (most recent first)
func GetQueryHistory(connectionID int64, limit int) ([]QueryHistory, error) {
	rows, err := DB.Query(
		"SELECT id, connection_id, query, executed_at, duration, rows_affected, error FROM query_history WHERE connection_id = ? ORDER BY executed_at DESC LIMIT ?",
		connectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []QueryHistory
	for rows.Next() {
		var h QueryHistory
		var errStr sql.NullString
		if err := rows.Scan(&h.ID, &h.ConnectionID, &h.Query, &h.ExecutedAt, &h.Duration, &h.RowsAffected, &errStr); err != nil {
			return nil, err
		}
		if errStr.Valid {
			h.Error = errStr.String
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// GetRecentQueryHistory retrieves all recent query history (most recent first)
func GetRecentQueryHistory(limit int) ([]QueryHistory, error) {
	rows, err := DB.Query(
		"SELECT id, connection_id, query, executed_at, duration, rows_affected, error FROM query_history ORDER BY executed_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []QueryHistory
	for rows.Next() {
		var h QueryHistory
		var errStr sql.NullString
		if err := rows.Scan(&h.ID, &h.ConnectionID, &h.Query, &h.ExecutedAt, &h.Duration, &h.RowsAffected, &errStr); err != nil {
			return nil, err
		}
		if errStr.Valid {
			h.Error = errStr.String
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// ClearQueryHistory clears all query history for a connection
func ClearQueryHistory(connectionID int64) error {
	_, err := DB.Exec("DELETE FROM query_history WHERE connection_id = ?", connectionID)
	return err
}

// ClearAllQueryHistory clears all query history
func ClearAllQueryHistory() error {
	_, err := DB.Exec("DELETE FROM query_history")
	return err
}

// =============================================================================
// Database Connection operations
// =============================================================================

// Connect establishes a connection to an external database using the saved connection info
func Connect(conn *Connection) (drivers.Driver, error) {
	var driver drivers.Driver

	switch conn.Driver {
	case drivers.DriverTypeMySQL:
		driver = &drivers.MySQL{}
	case drivers.DriverTypePostgreSQL:
		driver = &drivers.PostgreSQL{}
	case drivers.DriverTypeSQLite:
		driver = &drivers.SQLite{}
	case drivers.DriverTypeMongoDB, drivers.DriverTypeMongoDBAtlas:
		// Both MongoDB and MongoDB Atlas use the same driver
		driver = &drivers.MongoDB{}
	default:
		return nil, fmt.Errorf("unsupported driver: %s", conn.Driver)
	}

	if err := driver.Connect(conn.URL); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return driver, nil
}

// TestConnectionByID tests a connection by ID without keeping it open
func TestConnectionByID(id int64) error {
	conn, err := GetConnection(id)
	if err != nil {
		return fmt.Errorf("connection not found: %w", err)
	}

	var driver drivers.Driver

	switch conn.Driver {
	case drivers.DriverTypeMySQL:
		driver = &drivers.MySQL{}
	case drivers.DriverTypePostgreSQL:
		driver = &drivers.PostgreSQL{}
	case drivers.DriverTypeSQLite:
		driver = &drivers.SQLite{}
	case drivers.DriverTypeMongoDB, drivers.DriverTypeMongoDBAtlas:
		// Both MongoDB and MongoDB Atlas use the same driver
		driver = &drivers.MongoDB{}
	default:
		return fmt.Errorf("unsupported driver: %s", conn.Driver)
	}

	return driver.TestConnection(conn.URL)
}
