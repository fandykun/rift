package diff

import "testing"

func TestParseMigrationSQLCommonDDL(t *testing.T) {
	snapshot, err := ParseMigrationSQL(`
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL,
  display_name VARCHAR(120),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE users DROP COLUMN display_name;
CREATE INDEX idx_users_email ON users(email);
`)
	if err != nil {
		t.Fatalf("ParseMigrationSQL() error = %v", err)
	}
	if len(snapshot.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(snapshot.Tables))
	}
	users := snapshot.Tables[0]
	if users.Name != "users" {
		t.Fatalf("expected users table, got %s", users.Name)
	}
	if hasParsedColumn(users, "display_name") {
		t.Fatal("expected display_name to be removed by ALTER TABLE DROP COLUMN")
	}
	if !hasParsedColumn(users, "deleted_at") {
		t.Fatal("expected deleted_at column from ALTER TABLE ADD COLUMN")
	}
	if !hasParsedColumn(users, "email") {
		t.Fatal("expected email column from CREATE TABLE")
	}
	if len(snapshot.Indexes) != 1 || snapshot.Indexes[0].Name != "idx_users_email" {
		t.Fatalf("expected idx_users_email, got %#v", snapshot.Indexes)
	}
}

func TestParseMigrationSQLCreateTableWithEightColumns(t *testing.T) {
	snapshot, err := ParseMigrationSQL(`
CREATE TABLE accounts (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  login_count INTEGER NOT NULL DEFAULT 0,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	if err != nil {
		t.Fatalf("ParseMigrationSQL() error = %v", err)
	}
	if len(snapshot.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(snapshot.Tables))
	}
	if len(snapshot.Tables[0].Columns) != 8 {
		t.Fatalf("expected 8 columns, got %d: %#v", len(snapshot.Tables[0].Columns), snapshot.Tables[0].Columns)
	}
}

func hasParsedColumn(table TableDef, columnName string) bool {
	for _, column := range table.Columns {
		if column.Name == columnName {
			return true
		}
	}
	return false
}
