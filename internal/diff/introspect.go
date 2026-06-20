package diff

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IntrospectLive reads tables, columns, indexes, and foreign keys from a live PostgreSQL schema.
func IntrospectLive(ctx context.Context, pool *pgxpool.Pool, schema string) (*SchemaSnapshot, error) {
	if pool == nil {
		return nil, fmt.Errorf("introspecting live schema: database pool is nil")
	}
	if schema == "" {
		schema = "public"
	}

	columns, err := introspectColumns(ctx, pool, schema)
	if err != nil {
		return nil, err
	}
	indexes, err := introspectIndexes(ctx, pool, schema)
	if err != nil {
		return nil, err
	}
	foreignKeys, err := introspectForeignKeys(ctx, pool, schema)
	if err != nil {
		return nil, err
	}

	tablesByName := make(map[string][]ColumnDef)
	tableNames := make([]string, 0)
	for _, column := range columns {
		if _, ok := tablesByName[column.TableName]; !ok {
			tableNames = append(tableNames, column.TableName)
		}
		tablesByName[column.TableName] = append(tablesByName[column.TableName], column)
	}

	tables := make([]TableDef, 0, len(tableNames))
	for _, tableName := range tableNames {
		tables = append(tables, TableDef{Name: tableName, Columns: tablesByName[tableName]})
	}

	return &SchemaSnapshot{
		Schema:      schema,
		Tables:      tables,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
	}, nil
}

func introspectColumns(ctx context.Context, pool *pgxpool.Pool, schema string) ([]ColumnDef, error) {
	rows, err := pool.Query(ctx, `
SELECT
    t.table_name,
    c.column_name,
    c.data_type,
    c.is_nullable,
    COALESCE(c.column_default, ''),
    c.character_maximum_length
FROM information_schema.tables t
JOIN information_schema.columns c ON c.table_name = t.table_name AND c.table_schema = t.table_schema
WHERE t.table_schema = $1
  AND t.table_type = 'BASE TABLE'
ORDER BY t.table_name, c.ordinal_position;
`, schema)
	if err != nil {
		return nil, fmt.Errorf("querying schema columns: %w", err)
	}
	defer rows.Close()

	columns := make([]ColumnDef, 0)
	for rows.Next() {
		var column ColumnDef
		var nullable string
		if err := rows.Scan(&column.TableName, &column.Name, &column.DataType, &nullable, &column.Default, &column.MaxLength); err != nil {
			return nil, fmt.Errorf("scanning schema column: %w", err)
		}
		column.Nullable = nullable == "YES"
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating schema columns: %w", err)
	}
	return columns, nil
}

func introspectIndexes(ctx context.Context, pool *pgxpool.Pool, schema string) ([]IndexDef, error) {
	rows, err := pool.Query(ctx, `
SELECT indexname, tablename, indexdef
FROM pg_indexes
WHERE schemaname = $1
ORDER BY tablename, indexname;
`, schema)
	if err != nil {
		return nil, fmt.Errorf("querying schema indexes: %w", err)
	}
	defer rows.Close()

	indexes := make([]IndexDef, 0)
	for rows.Next() {
		var index IndexDef
		if err := rows.Scan(&index.Name, &index.TableName, &index.Definition); err != nil {
			return nil, fmt.Errorf("scanning schema index: %w", err)
		}
		indexes = append(indexes, index)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating schema indexes: %w", err)
	}
	return indexes, nil
}

func introspectForeignKeys(ctx context.Context, pool *pgxpool.Pool, schema string) ([]ForeignKeyDef, error) {
	rows, err := pool.Query(ctx, `
SELECT
    tc.constraint_name,
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
 AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
 AND ccu.constraint_schema = tc.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = $1
ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position;
`, schema)
	if err != nil {
		return nil, fmt.Errorf("querying schema foreign keys: %w", err)
	}
	defer rows.Close()

	foreignKeys := make([]ForeignKeyDef, 0)
	for rows.Next() {
		var foreignKey ForeignKeyDef
		if err := rows.Scan(&foreignKey.Name, &foreignKey.TableName, &foreignKey.ColumnName, &foreignKey.ForeignTableName, &foreignKey.ForeignColumnName); err != nil {
			return nil, fmt.Errorf("scanning schema foreign key: %w", err)
		}
		foreignKeys = append(foreignKeys, foreignKey)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating schema foreign keys: %w", err)
	}
	return foreignKeys, nil
}
