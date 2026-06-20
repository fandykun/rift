package migration

import "testing"

func TestDetectConflicts(t *testing.T) {
	files := []MigrationFile{
		{
			Version:  "20260620_120000",
			Filename: "20260620_120000_create_users",
			Checksum: "same-checksum",
		},
		{
			Version:  "20260620_120001",
			Filename: "20260620_120001_add_email",
			Checksum: "local-checksum",
		},
	}
	applied := []MigrationRecord{
		{
			Version:  "20260620_120000",
			Filename: "20260620_120000_create_users",
			Checksum: "same-checksum",
		},
		{
			Version:  "20260620_120001",
			Filename: "20260620_120001_add_email",
			Checksum: "database-checksum",
		},
		{
			Version:  "20260620_120002",
			Filename: "20260620_120002_create_posts",
			Checksum: "missing-file-checksum",
		},
		{
			Version:    "20260620_120003",
			Filename:   "20260620_120003_rolled_back",
			Checksum:   "ignored-checksum",
			RolledBack: true,
		},
	}

	conflicts := DetectConflicts(applied, files)
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictChecksumMismatch {
		t.Fatalf("first conflict type = %s, want %s", conflicts[0].Type, ConflictChecksumMismatch)
	}
	if conflicts[0].Version != "20260620_120001" {
		t.Fatalf("first conflict version = %s", conflicts[0].Version)
	}
	if conflicts[1].Type != ConflictMissingFile {
		t.Fatalf("second conflict type = %s, want %s", conflicts[1].Type, ConflictMissingFile)
	}
	if conflicts[1].Version != "20260620_120002" {
		t.Fatalf("second conflict version = %s", conflicts[1].Version)
	}
}

func TestDetectConflictsNoConflicts(t *testing.T) {
	files := []MigrationFile{{Version: "20260620_120000", Filename: "20260620_120000_create_users", Checksum: "same-checksum"}}
	applied := []MigrationRecord{{Version: "20260620_120000", Filename: "20260620_120000_create_users", Checksum: "same-checksum"}}

	conflicts := DetectConflicts(applied, files)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
}
