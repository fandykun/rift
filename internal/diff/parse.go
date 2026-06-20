package diff

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	createTablePattern = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][\w.]*)\s*\((.*?)\)\s*;`)
	alterAddPattern    = regexp.MustCompile(`(?is)ALTER\s+TABLE\s+([a-zA-Z_][\w.]*)\s+ADD\s+(?:COLUMN\s+)?([a-zA-Z_][\w]*)\s+([^;]+);`)
	alterDropPattern   = regexp.MustCompile(`(?is)ALTER\s+TABLE\s+([a-zA-Z_][\w.]*)\s+DROP\s+(?:COLUMN\s+)?(?:IF\s+EXISTS\s+)?([a-zA-Z_][\w]*)\s*;`)
	createIndexPattern = regexp.MustCompile(`(?is)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][\w]*)\s+ON\s+([a-zA-Z_][\w.]*)\s*\((.*?)\)\s*;`)
)

// ParseMigrationSQL parses common PostgreSQL DDL into a schema snapshot.
func ParseMigrationSQL(sql string) (*SchemaSnapshot, error) {
	snapshot := &SchemaSnapshot{Schema: "public"}
	tables := make(map[string]*TableDef)

	for _, match := range createTablePattern.FindAllStringSubmatch(sql, -1) {
		tableName := normalizeIdentifier(match[1])
		table := ensureTable(tables, tableName)
		columns, err := parseCreateTableColumns(tableName, match[2])
		if err != nil {
			return nil, err
		}
		table.Columns = append(table.Columns, columns...)
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

	for _, match := range createIndexPattern.FindAllStringSubmatch(sql, -1) {
		snapshot.Indexes = append(snapshot.Indexes, IndexDef{
			Name:       normalizeIdentifier(match[1]),
			TableName:  normalizeIdentifier(match[2]),
			Definition: strings.Join(strings.Fields(match[0]), " "),
		})
	}

	for _, table := range tables {
		snapshot.Tables = append(snapshot.Tables, *table)
	}
	sortSnapshot(snapshot)
	return snapshot, nil
}

func parseCreateTableColumns(tableName string, body string) ([]ColumnDef, error) {
	parts := splitSQLList(body)
	columns := make([]ColumnDef, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || isTableConstraint(part) {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) < 2 {
			return nil, fmt.Errorf("parsing CREATE TABLE column %q: expected name and type", part)
		}
		columns = append(columns, parseColumnDef(tableName, fields[0], strings.Join(fields[1:], " ")))
	}
	return columns, nil
}

func parseColumnDef(tableName string, rawName string, rawDefinition string) ColumnDef {
	definition := strings.TrimSpace(rawDefinition)
	fields := strings.Fields(definition)
	dataType := ""
	if len(fields) > 0 {
		dataType = strings.ToLower(fields[0])
		if len(fields) > 1 {
			second := strings.ToUpper(fields[1])
			if second == "PRECISION" || second == "VARYING" {
				dataType += " " + strings.ToLower(fields[1])
			}
		}
	}

	upperDefinition := strings.ToUpper(definition)
	return ColumnDef{
		TableName: tableName,
		Name:      normalizeIdentifier(rawName),
		DataType:  dataType,
		Nullable:  !strings.Contains(upperDefinition, "NOT NULL") && !strings.Contains(upperDefinition, "PRIMARY KEY"),
		Default:   parseDefault(definition),
	}
}

func parseDefault(definition string) string {
	upperDefinition := strings.ToUpper(definition)
	index := strings.Index(upperDefinition, "DEFAULT")
	if index == -1 {
		return ""
	}
	return strings.TrimSpace(definition[index+len("DEFAULT"):])
}

func ensureTable(tables map[string]*TableDef, tableName string) *TableDef {
	if table, ok := tables[tableName]; ok {
		return table
	}
	table := &TableDef{Name: tableName}
	tables[tableName] = table
	return table
}

func upsertColumn(columns []ColumnDef, column ColumnDef) []ColumnDef {
	for i := range columns {
		if columns[i].Name == column.Name {
			columns[i] = column
			return columns
		}
	}
	return append(columns, column)
}

func removeColumn(columns []ColumnDef, columnName string) []ColumnDef {
	filtered := columns[:0]
	for _, column := range columns {
		if column.Name != columnName {
			filtered = append(filtered, column)
		}
	}
	return filtered
}

func splitSQLList(body string) []string {
	parts := make([]string, 0)
	depth := 0
	start := 0
	for index, char := range body {
		switch char {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, body[start:index])
				start = index + 1
			}
		}
	}
	parts = append(parts, body[start:])
	return parts
}

func isTableConstraint(part string) bool {
	first := strings.ToUpper(strings.Fields(part)[0])
	return first == "PRIMARY" || first == "FOREIGN" || first == "UNIQUE" || first == "CONSTRAINT" || first == "CHECK"
}

func sortSnapshot(snapshot *SchemaSnapshot) {
	sort.Slice(snapshot.Tables, func(i int, j int) bool { return snapshot.Tables[i].Name < snapshot.Tables[j].Name })
	for i := range snapshot.Tables {
		sort.Slice(snapshot.Tables[i].Columns, func(a int, b int) bool {
			return snapshot.Tables[i].Columns[a].Name < snapshot.Tables[i].Columns[b].Name
		})
	}
	sort.Slice(snapshot.Indexes, func(i int, j int) bool { return snapshot.Indexes[i].Name < snapshot.Indexes[j].Name })
	sort.Slice(snapshot.ForeignKeys, func(i int, j int) bool { return snapshot.ForeignKeys[i].Name < snapshot.ForeignKeys[j].Name })
}

func normalizeIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	identifier = strings.Trim(identifier, `"`)
	if dot := strings.LastIndex(identifier, "."); dot != -1 {
		identifier = identifier[dot+1:]
	}
	return strings.Trim(identifier, `"`)
}
