package drivers

// ColumnInfo represents detailed column information
type ColumnInfo struct {
	Name         string
	DataType     string
	Nullable     bool
	IsPrimaryKey bool
	DefaultValue string
	Extra        string // e.g., auto_increment
	Comment      string
}

// IndexInfo represents index information
type IndexInfo struct {
	Name      string
	Columns   []string
	IsUnique  bool
	IsPrimary bool
	Type      string // e.g., BTREE, HASH, FULLTEXT
}

// RelationInfo represents foreign key relationships
type RelationInfo struct {
	Name             string
	Column           string
	ReferencedTable  string
	ReferencedColumn string
	OnUpdate         string
	OnDelete         string
}

// TriggerInfo represents trigger information
type TriggerInfo struct {
	Name      string
	Event     string // INSERT, UPDATE, DELETE
	Timing    string // BEFORE, AFTER
	Statement string
	Table     string
}

// TableStructure holds all structure information for a table
type TableStructure struct {
	Columns   []ColumnInfo
	Indexes   []IndexInfo
	Relations []RelationInfo
	Triggers  []TriggerInfo
}
