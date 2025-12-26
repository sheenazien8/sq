package drivers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sheenazien8/sq/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Provider string
	Database string
}

func (db *MongoDB) Connect(urlstr string) error {
	db.SetProvider(DriverTypeMongoDB)

	// Parse the connection string to extract database name if present
	// MongoDB connection string format: mongodb://user:pass@host:port/database?options
	// or: mongodb+srv://user:pass@cluster/database?options
	if idx := strings.LastIndex(urlstr, "/"); idx != -1 {
		// Check if there's a database name after the last /
		remaining := urlstr[idx+1:]
		if remaining != "" && !strings.HasPrefix(remaining, "?") {
			// Extract database name (stop at ? if present)
			if qIdx := strings.Index(remaining, "?"); qIdx != -1 {
				db.Database = remaining[:qIdx]
			} else {
				db.Database = remaining
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(urlstr))
	if err != nil {
		return err
	}

	db.Client = client
	return nil
}

func (db *MongoDB) SetProvider(provider string) {
	db.Provider = provider
}

func (db *MongoDB) TestConnection(urlstr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(urlstr))
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	return err
}

func (db *MongoDB) GetTables(database string) (map[string][]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if database == "" {
		database = db.Database
	}

	if database == "" {
		// If no database specified, try to use admin database or list databases
		database = "admin"
	}

	dbObj := db.Client.Database(database)
	collections, err := dbObj.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		// If we can't list collections in the specified database,
		// try to list databases and use the first one
		logger.Debug("Failed to list collections in database", map[string]any{
			"database": database,
			"error":    err.Error(),
		})

		// Try to get list of databases as fallback
		adminDB := db.Client.Database("admin")
		adminCollections, dbErr := adminDB.ListCollectionNames(ctx, bson.M{})
		if dbErr == nil && len(adminCollections) > 0 {
			tables := make(map[string][]string)
			tables["admin"] = adminCollections
			return tables, nil
		}

		// If even that fails, return error
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	tables := make(map[string][]string)
	tables[database] = collections

	return tables, nil
}

func (db *MongoDB) GetTableColumns(database, table string) ([][]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if database == "" {
		database = db.Database
	}

	if database == "" {
		return nil, fmt.Errorf("no database specified")
	}

	// Get a sample document to determine schema
	collection := db.Client.Database(database).Collection(table)
	result := collection.FindOne(ctx, bson.M{})

	var doc map[string]interface{}
	err := result.Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return [][]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	// Extract field names and types
	var columns [][]string
	for fieldName, fieldValue := range doc {
		fieldType := getMongoType(fieldValue)
		columns = append(columns, []string{fieldName, fieldType, "YES", "", "", ""})
	}

	return columns, nil
}

func (db *MongoDB) GetTableData(database, table string) ([][]string, error) {
	pagination := Pagination{
		Page:     1,
		PageSize: 1000,
	}
	result, err := db.GetTableDataPaginated(database, table, pagination)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (db *MongoDB) GetTableDataWithFilter(database, table string, whereClause string) ([][]string, error) {
	pagination := Pagination{
		Page:     1,
		PageSize: 1000,
	}
	result, err := db.GetTableDataWithFilterPaginated(database, table, whereClause, pagination)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (db *MongoDB) GetTableDataPaginated(database, table string, pagination Pagination) (*PaginatedResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if database == "" {
		database = db.Database
	}

	if database == "" {
		return nil, fmt.Errorf("no database specified")
	}

	collection := db.Client.Database(database).Collection(table)

	// Get total count
	totalRows, err := collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate offset and pagination
	offset := int64((pagination.Page - 1) * pagination.PageSize)
	opts := options.Find().SetSkip(offset).SetLimit(int64(pagination.PageSize))

	// Execute query
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []map[string]interface{}
	if err = cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	// Get field names from first document for consistent column ordering
	var fieldNames []string
	if len(documents) > 0 {
		for key := range documents[0] {
			fieldNames = append(fieldNames, key)
		}
	}

	// Convert to string array format
	var data [][]string
	data = append(data, fieldNames)

	for _, doc := range documents {
		row := make([]string, len(fieldNames))
		for i, fieldName := range fieldNames {
			row[i] = formatMongoValue(doc[fieldName])
		}
		data = append(data, row)
	}

	// Calculate total pages
	totalPages := int(totalRows) / pagination.PageSize
	if int(totalRows)%pagination.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       data,
		TotalRows:  int(totalRows),
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (db *MongoDB) GetTableDataWithFilterPaginated(database, table string, whereClause string, pagination Pagination) (*PaginatedResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if database == "" {
		database = db.Database
	}

	if database == "" {
		return nil, fmt.Errorf("no database specified")
	}

	collection := db.Client.Database(database).Collection(table)

	// Parse the filter clause (simple JSON-like syntax for MongoDB)
	filter := bson.M{}
	if whereClause != "" {
		err := json.Unmarshal([]byte(whereClause), &filter)
		if err != nil {
			// If JSON parsing fails, try simple key:value parsing
			filter = parseSimpleFilter(whereClause)
		}
	}

	// Get total count with filter
	totalRows, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Calculate offset and pagination
	offset := int64((pagination.Page - 1) * pagination.PageSize)
	opts := options.Find().SetSkip(offset).SetLimit(int64(pagination.PageSize))

	// Execute query with filter
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []map[string]interface{}
	if err = cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	// Get field names from first document
	var fieldNames []string
	if len(documents) > 0 {
		for key := range documents[0] {
			fieldNames = append(fieldNames, key)
		}
	}

	// Convert to string array format
	var data [][]string
	data = append(data, fieldNames)

	for _, doc := range documents {
		row := make([]string, len(fieldNames))
		for i, fieldName := range fieldNames {
			row[i] = formatMongoValue(doc[fieldName])
		}
		data = append(data, row)
	}

	// Calculate total pages
	totalPages := int(totalRows) / pagination.PageSize
	if int(totalRows)%pagination.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       data,
		TotalRows:  int(totalRows),
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (db *MongoDB) GetTableStructure(database, table string) (*TableStructure, error) {
	columns, err := db.GetColumnInfo(database, table)
	if err != nil {
		return nil, err
	}

	indexes, err := db.GetIndexInfo(database, table)
	if err != nil {
		return nil, err
	}

	return &TableStructure{
		Columns:   columns,
		Indexes:   indexes,
		Relations: []RelationInfo{},
		Triggers:  []TriggerInfo{},
	}, nil
}

func (db *MongoDB) GetColumnInfo(database, table string) ([]ColumnInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if database == "" {
		database = db.Database
	}

	if database == "" {
		return nil, fmt.Errorf("no database specified")
	}

	collection := db.Client.Database(database).Collection(table)

	// Sample documents to determine field types
	opts := options.Find().SetLimit(100)
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []map[string]interface{}
	if err = cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	// Collect all field names and their types
	fieldTypes := make(map[string]string)
	for _, doc := range documents {
		for fieldName, fieldValue := range doc {
			if _, exists := fieldTypes[fieldName]; !exists {
				fieldTypes[fieldName] = getMongoType(fieldValue)
			}
		}
	}

	// Convert to ColumnInfo slice
	var columns []ColumnInfo
	for fieldName, fieldType := range fieldTypes {
		columns = append(columns, ColumnInfo{
			Name:         fieldName,
			DataType:     fieldType,
			Nullable:     true,
			IsPrimaryKey: fieldName == "_id",
			DefaultValue: "",
			Extra:        "",
			Comment:      "",
		})
	}

	return columns, nil
}

func (db *MongoDB) GetIndexInfo(database, table string) ([]IndexInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if database == "" {
		database = db.Database
	}

	if database == "" {
		return nil, fmt.Errorf("no database specified")
	}

	collection := db.Client.Database(database).Collection(table)
	indexModel, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}

	var indexes []IndexInfo
	var indexDocs []map[string]interface{}
	if err = indexModel.All(ctx, &indexDocs); err != nil {
		return nil, err
	}

	for _, indexDoc := range indexDocs {
		// Extract index name
		name := ""
		if n, ok := indexDoc["name"]; ok {
			name = n.(string)
		}

		// Extract key information
		var columns []string
		if keyData, ok := indexDoc["key"]; ok {
			if keyMap, ok := keyData.(map[string]interface{}); ok {
				for k := range keyMap {
					columns = append(columns, k)
				}
			}
		}

		// Extract unique flag
		isUnique := false
		if u, ok := indexDoc["unique"]; ok {
			isUnique = u.(bool)
		}

		indexes = append(indexes, IndexInfo{
			Name:      name,
			Columns:   columns,
			IsUnique:  isUnique,
			IsPrimary: name == "_id_",
			Type:      "index",
		})
	}

	return indexes, nil
}

func (db *MongoDB) GetRelationInfo(database, table string) ([]RelationInfo, error) {
	// MongoDB doesn't have built-in foreign key constraints like SQL databases
	// Return empty slice
	return []RelationInfo{}, nil
}

func (db *MongoDB) GetTriggerInfo(database, table string) ([]TriggerInfo, error) {
	// MongoDB doesn't have traditional triggers in the same way SQL databases do
	// Return empty slice
	return []TriggerInfo{}, nil
}

func (db *MongoDB) ExecuteQuery(query string) ([][]string, error) {
	// MongoDB doesn't use SQL, so we'll try to parse it as a simple command
	// For now, return an error indicating MongoDB doesn't support raw SQL queries
	logger.Debug("MongoDB ExecuteQuery called", map[string]any{
		"query": query,
	})

	return nil, fmt.Errorf("MongoDB does not support SQL queries. Use the collection/document interface instead")
}

// Helper functions

func formatMongoValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case []interface{}:
		// Format arrays as JSON
		b, _ := json.Marshal(v)
		return string(b)
	case map[string]interface{}:
		// Format objects as JSON
		b, _ := json.Marshal(v)
		return string(b)
	default:
		// For other types (ObjectID, Date, etc.), convert to string
		return fmt.Sprintf("%v", v)
	}
}

func getMongoType(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// parseSimpleFilter parses simple filter syntax like "field=value,field2=value2"
func parseSimpleFilter(whereClause string) bson.M {
	filter := bson.M{}
	pairs := strings.Split(whereClause, ",")

	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			// Special handling for _id field (ObjectId)
			if key == "_id" {
				// Try to parse as ObjectId
				if oid, err := primitive.ObjectIDFromHex(value); err == nil {
					filter[key] = oid
					continue
				}
				// If not a valid ObjectId, try as string
				filter[key] = value
				continue
			}

			// Try to parse as number
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				filter[key] = intVal
			} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				filter[key] = floatVal
			} else if value == "true" {
				filter[key] = true
			} else if value == "false" {
				filter[key] = false
			} else {
				filter[key] = value
			}
		}
	}

	return filter
}
