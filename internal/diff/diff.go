package diff

import "reflect"

// ComputeDiff compares a live/before schema with an expected/after schema.
func ComputeDiff(before *SchemaSnapshot, after *SchemaSnapshot) *SchemaDiff {
	if before == nil {
		before = &SchemaSnapshot{}
	}
	if after == nil {
		after = &SchemaSnapshot{}
	}

	diff := &SchemaDiff{}
	beforeTables := tableMap(before.Tables)
	afterTables := tableMap(after.Tables)

	for name, afterTable := range afterTables {
		beforeTable, ok := beforeTables[name]
		if !ok {
			diff.TablesAdded = append(diff.TablesAdded, afterTable)
			continue
		}
		if modification := computeTableModification(beforeTable, afterTable); hasTableModification(modification) {
			diff.TablesModified = append(diff.TablesModified, modification)
		}
	}
	for name, beforeTable := range beforeTables {
		if _, ok := afterTables[name]; !ok {
			diff.TablesDropped = append(diff.TablesDropped, beforeTable)
		}
	}

	diff.IndexChanges = computeIndexChanges(before.Indexes, after.Indexes)
	return diff
}

func computeTableModification(before TableDef, after TableDef) TableModification {
	modification := TableModification{TableName: after.Name}
	beforeColumns := columnMap(before.Columns)
	afterColumns := columnMap(after.Columns)

	for name, afterColumn := range afterColumns {
		beforeColumn, ok := beforeColumns[name]
		if !ok {
			modification.ColumnsAdded = append(modification.ColumnsAdded, afterColumn)
			continue
		}
		if !sameColumn(beforeColumn, afterColumn) {
			modification.ColumnsModified = append(modification.ColumnsModified, ColumnModification{Before: beforeColumn, After: afterColumn})
		}
	}
	for name, beforeColumn := range beforeColumns {
		if _, ok := afterColumns[name]; !ok {
			modification.ColumnsDropped = append(modification.ColumnsDropped, beforeColumn)
		}
	}
	return modification
}

func computeIndexChanges(before []IndexDef, after []IndexDef) []IndexChange {
	beforeIndexes := indexMap(before)
	afterIndexes := indexMap(after)
	changes := make([]IndexChange, 0)
	for name, afterIndex := range afterIndexes {
		beforeIndex, ok := beforeIndexes[name]
		if !ok {
			index := afterIndex
			changes = append(changes, IndexChange{Name: name, Kind: "added", After: &index})
			continue
		}
		if beforeIndex.Definition != afterIndex.Definition || beforeIndex.TableName != afterIndex.TableName {
			beforeCopy := beforeIndex
			afterCopy := afterIndex
			changes = append(changes, IndexChange{Name: name, Kind: "modified", Before: &beforeCopy, After: &afterCopy})
		}
	}
	for name, beforeIndex := range beforeIndexes {
		if _, ok := afterIndexes[name]; !ok {
			index := beforeIndex
			changes = append(changes, IndexChange{Name: name, Kind: "dropped", Before: &index})
		}
	}
	return changes
}

func tableMap(tables []TableDef) map[string]TableDef {
	mapped := make(map[string]TableDef, len(tables))
	for _, table := range tables {
		mapped[table.Name] = table
	}
	return mapped
}

func columnMap(columns []ColumnDef) map[string]ColumnDef {
	mapped := make(map[string]ColumnDef, len(columns))
	for _, column := range columns {
		mapped[column.Name] = column
	}
	return mapped
}

func indexMap(indexes []IndexDef) map[string]IndexDef {
	mapped := make(map[string]IndexDef, len(indexes))
	for _, index := range indexes {
		mapped[index.Name] = index
	}
	return mapped
}

func sameColumn(before ColumnDef, after ColumnDef) bool {
	return before.Name == after.Name && before.DataType == after.DataType && before.Nullable == after.Nullable && before.Default == after.Default && reflect.DeepEqual(before.MaxLength, after.MaxLength)
}

func hasTableModification(modification TableModification) bool {
	return len(modification.ColumnsAdded) > 0 || len(modification.ColumnsDropped) > 0 || len(modification.ColumnsModified) > 0
}
