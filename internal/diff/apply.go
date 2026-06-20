package diff

// ApplyMigrationSQL returns a new schema snapshot with common migration DDL applied.
func ApplyMigrationSQL(before *SchemaSnapshot, sql string) (*SchemaSnapshot, error) {
	after := cloneSnapshot(before)
	tables := tablePointerMap(after.Tables)

	for _, match := range createTablePattern.FindAllStringSubmatch(sql, -1) {
		tableName := normalizeIdentifier(match[1])
		columns, err := parseCreateTableColumns(tableName, match[2])
		if err != nil {
			return nil, err
		}
		tables[tableName] = &TableDef{Name: tableName, Columns: columns}
	}

	for _, match := range alterAddPattern.FindAllStringSubmatch(sql, -1) {
		tableName := normalizeIdentifier(match[1])
		table := ensureTable(tables, tableName)
		column := parseColumnDef(tableName, match[2], match[3])
		table.Columns = upsertColumn(table.Columns, column)
	}

	for _, match := range alterDropPattern.FindAllStringSubmatch(sql, -1) {
		tableName := normalizeIdentifier(match[1])
		table := ensureTable(tables, tableName)
		columnName := normalizeIdentifier(match[2])
		table.Columns = removeColumn(table.Columns, columnName)
	}

	for _, match := range dropTablePattern.FindAllStringSubmatch(sql, -1) {
		delete(tables, normalizeIdentifier(match[1]))
	}

	after.Tables = after.Tables[:0]
	for _, table := range tables {
		after.Tables = append(after.Tables, *table)
	}

	for _, match := range createIndexPattern.FindAllStringSubmatch(sql, -1) {
		index := IndexDef{
			Name:       normalizeIdentifier(match[1]),
			TableName:  normalizeIdentifier(match[2]),
			Definition: normalizeIndexDefinition(match[0]),
		}
		after.Indexes = upsertIndex(after.Indexes, index)
	}

	sortSnapshot(after)
	return after, nil
}

func cloneSnapshot(snapshot *SchemaSnapshot) *SchemaSnapshot {
	if snapshot == nil {
		return &SchemaSnapshot{Schema: "public"}
	}

	clone := &SchemaSnapshot{Schema: snapshot.Schema}
	clone.Tables = make([]TableDef, len(snapshot.Tables))
	for i, table := range snapshot.Tables {
		clone.Tables[i] = TableDef{Name: table.Name}
		clone.Tables[i].Columns = append([]ColumnDef(nil), table.Columns...)
	}
	clone.Indexes = append([]IndexDef(nil), snapshot.Indexes...)
	clone.ForeignKeys = append([]ForeignKeyDef(nil), snapshot.ForeignKeys...)
	return clone
}

func tablePointerMap(tables []TableDef) map[string]*TableDef {
	mapped := make(map[string]*TableDef, len(tables))
	for i := range tables {
		table := tables[i]
		mapped[table.Name] = &table
	}
	return mapped
}

func upsertIndex(indexes []IndexDef, index IndexDef) []IndexDef {
	for i := range indexes {
		if indexes[i].Name == index.Name {
			indexes[i] = index
			return indexes
		}
	}
	return append(indexes, index)
}
