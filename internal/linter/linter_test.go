package linter

import "testing"

func TestLintSQLDetectsDangerousPatterns(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		pattern  string
		severity string
	}{
		{
			name:     "drop column",
			sql:      "ALTER TABLE users DROP COLUMN email;",
			pattern:  "DROP_COLUMN",
			severity: "error",
		},
		{
			name:     "rename column",
			sql:      "ALTER TABLE users RENAME COLUMN name TO full_name;",
			pattern:  "RENAME_COLUMN",
			severity: "error",
		},
		{
			name:     "set not null without default",
			sql:      "ALTER TABLE users ALTER COLUMN email SET NOT NULL;",
			pattern:  "SET_NOT_NULL_WITHOUT_DEFAULT",
			severity: "error",
		},
		{
			name:     "drop table",
			sql:      "DROP TABLE users;",
			pattern:  "DROP_TABLE",
			severity: "error",
		},
		{
			name:     "create index without concurrently",
			sql:      "CREATE INDEX users_email_idx ON users (email);",
			pattern:  "CREATE_INDEX_WITHOUT_CONCURRENTLY",
			severity: "warning",
		},
		{
			name:     "foreign key without not valid",
			sql:      "ALTER TABLE posts ADD CONSTRAINT posts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);",
			pattern:  "FOREIGN_KEY_WITHOUT_NOT_VALID",
			severity: "warning",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			warnings := LintSQL(test.sql)
			if len(warnings) != 1 {
				t.Fatalf("expected 1 warning, got %d: %+v", len(warnings), warnings)
			}
			warning := warnings[0]
			if warning.Pattern != test.pattern {
				t.Fatalf("expected pattern %q, got %q", test.pattern, warning.Pattern)
			}
			if warning.Severity != test.severity {
				t.Fatalf("expected severity %q, got %q", test.severity, warning.Severity)
			}
			if warning.Line != 1 {
				t.Fatalf("expected line 1, got %d", warning.Line)
			}
			if warning.Message == "" || warning.Suggestion == "" {
				t.Fatalf("expected message and suggestion, got %+v", warning)
			}
		})
	}
}

func TestLintSQLIgnoresSafeAlternatives(t *testing.T) {
	safeSQL := `
		CREATE INDEX CONCURRENTLY users_email_idx ON users (email);
		ALTER TABLE users ADD COLUMN email TEXT DEFAULT '';
		ALTER TABLE posts ADD CONSTRAINT posts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
	`
	warnings := LintSQL(safeSQL)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", warnings)
	}
}

func TestLintSQLReportsStatementStartLine(t *testing.T) {
	warnings := LintSQL("CREATE TABLE users (id BIGSERIAL PRIMARY KEY);\n\nALTER TABLE users DROP COLUMN legacy;")
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %+v", warnings)
	}
	if warnings[0].Line != 3 {
		t.Fatalf("expected warning on line 3, got %d", warnings[0].Line)
	}
}
