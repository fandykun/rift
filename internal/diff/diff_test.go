package diff

import "testing"

func TestComputeDiffDetectsTableColumnAndIndexChanges(t *testing.T) {
	before := &SchemaSnapshot{
		Tables: []TableDef{
			{
				Name: "users",
				Columns: []ColumnDef{
					{TableName: "users", Name: "id", DataType: "bigint", Nullable: false},
					{TableName: "users", Name: "name", DataType: "text", Nullable: false},
					{TableName: "users", Name: "legacy_code", DataType: "text", Nullable: true},
				},
			},
			{Name: "old_logs", Columns: []ColumnDef{{TableName: "old_logs", Name: "id", DataType: "bigint", Nullable: false}}},
		},
		Indexes: []IndexDef{{Name: "idx_users_name", TableName: "users", Definition: "CREATE INDEX idx_users_name ON users(name)"}},
	}
	after := &SchemaSnapshot{
		Tables: []TableDef{
			{
				Name: "users",
				Columns: []ColumnDef{
					{TableName: "users", Name: "id", DataType: "bigint", Nullable: false},
					{TableName: "users", Name: "name", DataType: "varchar", Nullable: false},
					{TableName: "users", Name: "email", DataType: "text", Nullable: false},
				},
			},
			{Name: "posts", Columns: []ColumnDef{{TableName: "posts", Name: "id", DataType: "bigint", Nullable: false}}},
		},
		Indexes: []IndexDef{{Name: "idx_users_email", TableName: "users", Definition: "CREATE INDEX idx_users_email ON users(email)"}},
	}

	diff := ComputeDiff(before, after)
	if len(diff.TablesAdded) != 1 || diff.TablesAdded[0].Name != "posts" {
		t.Fatalf("expected posts table added, got %#v", diff.TablesAdded)
	}
	if len(diff.TablesDropped) != 1 || diff.TablesDropped[0].Name != "old_logs" {
		t.Fatalf("expected old_logs table dropped, got %#v", diff.TablesDropped)
	}
	if len(diff.TablesModified) != 1 {
		t.Fatalf("expected one modified table, got %#v", diff.TablesModified)
	}
	users := diff.TablesModified[0]
	if len(users.ColumnsAdded) != 1 || users.ColumnsAdded[0].Name != "email" {
		t.Fatalf("expected email column added, got %#v", users.ColumnsAdded)
	}
	if len(users.ColumnsDropped) != 1 || users.ColumnsDropped[0].Name != "legacy_code" {
		t.Fatalf("expected legacy_code column dropped, got %#v", users.ColumnsDropped)
	}
	if len(users.ColumnsModified) != 1 || users.ColumnsModified[0].After.Name != "name" {
		t.Fatalf("expected name column modified, got %#v", users.ColumnsModified)
	}
	if len(diff.IndexChanges) != 2 {
		t.Fatalf("expected added and dropped index changes, got %#v", diff.IndexChanges)
	}
}
