package migration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFilesParsesAndSortsMigrations(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "20260620_120002_create_posts.up.sql", "CREATE TABLE posts (id BIGSERIAL PRIMARY KEY);\n")
	writeMigrationFile(t, dir, "20260620_120002_create_posts.down.sql", "DROP TABLE posts;\n")
	writeMigrationFile(t, dir, "20260620_120000_create_users.up.sql", "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);\n")
	writeMigrationFile(t, dir, "20260620_120000_create_users.down.sql", "DROP TABLE users;\n")
	writeMigrationFile(t, dir, "20260620_120001_add_email.up.sql", "ALTER TABLE users ADD COLUMN email TEXT;\n")
	writeMigrationFile(t, dir, "20260620_120001_add_email.down.sql", "ALTER TABLE users DROP COLUMN email;\n")
	writeMigrationFile(t, dir, "README.md", "ignored\n")

	files, err := LoadFiles(dir)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 migration files, got %d", len(files))
	}

	wantVersions := []string{"20260620_120000", "20260620_120001", "20260620_120002"}
	for i, want := range wantVersions {
		if files[i].Version != want {
			t.Fatalf("files[%d].Version = %q, want %q", i, files[i].Version, want)
		}
		if files[i].Checksum == "" {
			t.Fatalf("files[%d].Checksum is empty", i)
		}
	}
	if files[0].Filename != "20260620_120000_create_users" {
		t.Fatalf("unexpected filename: %s", files[0].Filename)
	}
	if files[0].UpSQL == "" || files[0].DownSQL == "" {
		t.Fatal("expected up and down SQL content to be populated")
	}
}

func TestLoadFilesRequiresPairs(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "20260620_120000_create_users.up.sql", "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);\n")

	_, err := LoadFiles(dir)
	if err == nil {
		t.Fatal("expected missing down migration error")
	}
}

func TestLoadFilesRejectsInvalidTimestampPrefix(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "create_users.up.sql", "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);\n")
	writeMigrationFile(t, dir, "create_users.down.sql", "DROP TABLE users;\n")

	_, err := LoadFiles(dir)
	if err == nil {
		t.Fatal("expected invalid timestamp prefix error")
	}
}

func writeMigrationFile(t *testing.T, dir string, name string, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing migration file %s: %v", name, err)
	}
}
