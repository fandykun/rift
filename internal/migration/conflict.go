package migration

// ConflictType identifies a migration state mismatch between the database and local files.
type ConflictType string

const (
	// ConflictMissingFile means a migration exists in PostgreSQL state but not on disk.
	ConflictMissingFile ConflictType = "MISSING_FILE"
	// ConflictChecksumMismatch means an applied migration's local up.sql content changed.
	ConflictChecksumMismatch ConflictType = "CHECKSUM_MISMATCH"
)

// Conflict describes a migration state mismatch that must be resolved before applying migrations.
type Conflict struct {
	Type             ConflictType
	Version          string
	DatabaseFilename string
	LocalFilename    string
	DatabaseChecksum string
	LocalChecksum    string
	Message          string
}

// DetectConflicts compares applied database records with local migration files.
func DetectConflicts(applied []MigrationRecord, files []MigrationFile) []Conflict {
	localByVersion := make(map[string]MigrationFile, len(files))
	for _, file := range files {
		localByVersion[file.Version] = file
	}

	conflicts := make([]Conflict, 0)
	for _, record := range applied {
		if record.RolledBack {
			continue
		}

		file, ok := localByVersion[record.Version]
		if !ok {
			conflicts = append(conflicts, Conflict{
				Type:             ConflictMissingFile,
				Version:          record.Version,
				DatabaseFilename: record.Filename,
				DatabaseChecksum: record.Checksum,
				Message:          "migration was applied to the database but no matching local file exists",
			})
			continue
		}

		if record.Checksum != file.Checksum {
			conflicts = append(conflicts, Conflict{
				Type:             ConflictChecksumMismatch,
				Version:          record.Version,
				DatabaseFilename: record.Filename,
				LocalFilename:    file.Filename,
				DatabaseChecksum: record.Checksum,
				LocalChecksum:    file.Checksum,
				Message:          "local migration checksum differs from the applied database record",
			})
		}
	}

	return conflicts
}
