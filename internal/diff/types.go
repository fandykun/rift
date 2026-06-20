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

// SchemaDiff describes structural changes between two schema snapshots.
type SchemaDiff struct {
	TablesAdded    []TableDef
	TablesDropped  []TableDef
	TablesModified []TableModification
	IndexChanges   []IndexChange
}

// TableModification describes column-level changes for one table.
type TableModification struct {
	TableName       string
	ColumnsAdded    []ColumnDef
	ColumnsDropped  []ColumnDef
	ColumnsModified []ColumnModification
}

// ColumnModification describes a changed column definition.
type ColumnModification struct {
	Before ColumnDef
	After  ColumnDef
}

// IndexChange describes an added, dropped, or modified index.
type IndexChange struct {
	Name   string
	Kind   string
	Before *IndexDef
	After  *IndexDef
}
