package diff

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
)

func TestIntrospectLiveIntegration(t *testing.T) {
	databaseURL := os.Getenv("RIFT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RIFT_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
CREATE TABLE organizations (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(id),
  email TEXT NOT NULL,
  display_name VARCHAR(120),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_email ON users(email);
CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  body TEXT
);
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE TABLE comments (
  id BIGSERIAL PRIMARY KEY,
  post_id BIGINT NOT NULL REFERENCES posts(id),
  body TEXT NOT NULL
);
CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  entity_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	if err != nil {
		t.Fatalf("creating test schema: %v", err)
	}

	snapshot, err := IntrospectLive(ctx, pool, "public")
	if err != nil {
		t.Fatalf("IntrospectLive() error = %v", err)
	}
	if len(snapshot.Tables) != 5 {
		t.Fatalf("expected 5 tables, got %d", len(snapshot.Tables))
	}
	if !hasColumn(snapshot, "users", "display_name", "character varying") {
		t.Fatalf("expected users.display_name character varying column in snapshot: %#v", snapshot.Tables)
	}
	if !hasIndex(snapshot, "idx_users_email") {
		t.Fatalf("expected idx_users_email index in snapshot: %#v", snapshot.Indexes)
	}
	if !hasForeignKey(snapshot, "users", "organization_id", "organizations", "id") {
		t.Fatalf("expected users.organization_id foreign key in snapshot: %#v", snapshot.ForeignKeys)
	}
}

func hasColumn(snapshot *SchemaSnapshot, tableName string, columnName string, dataType string) bool {
	for _, table := range snapshot.Tables {
		if table.Name != tableName {
			continue
		}
		for _, column := range table.Columns {
			if column.Name == columnName && column.DataType == dataType {
				return true
			}
		}
	}
	return false
}

func hasIndex(snapshot *SchemaSnapshot, indexName string) bool {
	for _, index := range snapshot.Indexes {
		if index.Name == indexName {
			return true
		}
	}
	return false
}

func hasForeignKey(snapshot *SchemaSnapshot, tableName string, columnName string, foreignTableName string, foreignColumnName string) bool {
	for _, foreignKey := range snapshot.ForeignKeys {
		if foreignKey.TableName == tableName && foreignKey.ColumnName == columnName && foreignKey.ForeignTableName == foreignTableName && foreignKey.ForeignColumnName == foreignColumnName {
			return true
		}
	}
	return false
}
