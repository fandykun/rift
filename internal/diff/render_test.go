package diff

import (
	"strings"
	"testing"
)

func TestApplyMigrationSQLBuildsExpectedSnapshot(t *testing.T) {
	before := &SchemaSnapshot{Schema: "public", Tables: []TableDef{
		{Name: "users", Columns: []ColumnDef{
			{TableName: "users", Name: "id", DataType: "bigint", Nullable: false},
			{TableName: "users", Name: "legacy", DataType: "text", Nullable: true},
		}},
	}, Indexes: []IndexDef{{Name: "users_id_idx", TableName: "users", Definition: "CREATE INDEX users_id_idx ON users (id)"}}}

	after, err := ApplyMigrationSQL(before, `
		ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';
		ALTER TABLE users DROP COLUMN legacy;
		CREATE INDEX users_email_idx ON users (email);
		CREATE TABLE projects (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);
	`)
	if err != nil {
		t.Fatalf("ApplyMigrationSQL() error = %v", err)
	}

	schemaDiff := ComputeDiff(before, after)
	if len(schemaDiff.TablesAdded) != 1 || schemaDiff.TablesAdded[0].Name != "projects" {
		t.Fatalf("expected projects table addition, got %+v", schemaDiff.TablesAdded)
	}
	if len(schemaDiff.TablesModified) != 1 {
		t.Fatalf("expected one modified table, got %+v", schemaDiff.TablesModified)
	}
	modification := schemaDiff.TablesModified[0]
	if len(modification.ColumnsAdded) != 1 || modification.ColumnsAdded[0].Name != "email" {
		t.Fatalf("expected email column addition, got %+v", modification.ColumnsAdded)
	}
	if len(modification.ColumnsDropped) != 1 || modification.ColumnsDropped[0].Name != "legacy" {
		t.Fatalf("expected legacy column drop, got %+v", modification.ColumnsDropped)
	}
	if len(schemaDiff.IndexChanges) != 1 || schemaDiff.IndexChanges[0].Name != "users_email_idx" {
		t.Fatalf("expected users_email_idx addition, got %+v", schemaDiff.IndexChanges)
	}
}

func TestRenderTerminalAndJSON(t *testing.T) {
	schemaDiff := &SchemaDiff{
		TablesAdded: []TableDef{{Name: "projects"}},
		TablesModified: []TableModification{{
			TableName:    "users",
			ColumnsAdded: []ColumnDef{{Name: "email", DataType: "text", Nullable: false}},
		}},
		IndexChanges: []IndexChange{{Name: "users_email_idx", Kind: "added", After: &IndexDef{TableName: "users"}}},
	}

	var terminal strings.Builder
	if err := RenderTerminal(&terminal, schemaDiff); err != nil {
		t.Fatalf("RenderTerminal() error = %v", err)
	}
	output := terminal.String()
	for _, expected := range []string{"DIFF", "1 table modified", "1 table added", "1 column added", "1 index changed", "projects", "email", "users_email_idx"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected terminal output to contain %q, got %q", expected, output)
		}
	}

	jsonOutput, err := RenderJSON(schemaDiff)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	if !strings.Contains(string(jsonOutput), `"TablesAdded"`) || !strings.Contains(string(jsonOutput), `"projects"`) {
		t.Fatalf("expected JSON output to include diff fields, got %s", jsonOutput)
	}
}

func TestRenderTerminalNoChanges(t *testing.T) {
	var terminal strings.Builder
	if err := RenderTerminal(&terminal, &SchemaDiff{}); err != nil {
		t.Fatalf("RenderTerminal() error = %v", err)
	}
	if !strings.Contains(terminal.String(), "OK no schema changes") {
		t.Fatalf("expected no changes output, got %q", terminal.String())
	}
}
