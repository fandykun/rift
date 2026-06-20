package diff

// SchemaSnapshot is a normalized PostgreSQL schema representation used for diffing.
type SchemaSnapshot struct {
	Schema      string
	Tables      []TableDef
	Indexes     []IndexDef
	ForeignKeys []ForeignKeyDef
}

// TableDef describes one database table and its columns.
type TableDef struct {
	Name    string
	Columns []ColumnDef
}

// ColumnDef describes a PostgreSQL table column.
type ColumnDef struct {
	TableName string
	Name      string
	DataType  string
	Nullable  bool
	Default   string
	MaxLength *int
}

// IndexDef describes one PostgreSQL index.
type IndexDef struct {
	Name       string
	TableName  string
	Definition string
}

// ForeignKeyDef describes one PostgreSQL foreign key constraint.
type ForeignKeyDef struct {
	Name              string
	TableName         string
	ColumnName        string
	ForeignTableName  string
	ForeignColumnName string
}
