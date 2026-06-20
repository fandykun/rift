package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	upSuffix   = ".up.sql"
	downSuffix = ".down.sql"
)

// MigrationFile is a paired up/down SQL migration loaded from disk.
type MigrationFile struct {
	Version  string
	Filename string
	UpSQL    string
	DownSQL  string
	Checksum string
}

type migrationPair struct {
	upPath   string
	downPath string
}

// LoadFiles reads paired *.up.sql and *.down.sql migration files sorted by version.
func LoadFiles(dir string) ([]MigrationFile, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("loading migration files: migrations directory is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory %q: %w", dir, err)
	}

	pairs := make(map[string]migrationPair)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(dir, name)
		switch {
		case strings.HasSuffix(name, upSuffix):
			base := strings.TrimSuffix(name, upSuffix)
			pair := pairs[base]
			pair.upPath = path
			pairs[base] = pair
		case strings.HasSuffix(name, downSuffix):
			base := strings.TrimSuffix(name, downSuffix)
			pair := pairs[base]
			pair.downPath = path
			pairs[base] = pair
		}
	}

	bases := make([]string, 0, len(pairs))
	for base := range pairs {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	files := make([]MigrationFile, 0, len(bases))
	for _, base := range bases {
		pair := pairs[base]
		if pair.upPath == "" {
			return nil, fmt.Errorf("migration %q is missing matching .up.sql file", base)
		}
		if pair.downPath == "" {
			return nil, fmt.Errorf("migration %q is missing matching .down.sql file", base)
		}

		upContent, err := os.ReadFile(pair.upPath)
		if err != nil {
			return nil, fmt.Errorf("reading up migration %q: %w", pair.upPath, err)
		}
		downContent, err := os.ReadFile(pair.downPath)
		if err != nil {
			return nil, fmt.Errorf("reading down migration %q: %w", pair.downPath, err)
		}

		version, err := versionFromBase(base)
		if err != nil {
			return nil, err
		}

		files = append(files, MigrationFile{
			Version:  version,
			Filename: base,
			UpSQL:    string(upContent),
			DownSQL:  string(downContent),
			Checksum: Checksum(upContent),
		})
	}

	return files, nil
}

func versionFromBase(base string) (string, error) {
	parts := strings.SplitN(base, "_", 3)
	if len(parts) < 2 || len(parts[0]) != 8 || len(parts[1]) != 6 {
		return "", fmt.Errorf("migration %q must start with timestamp prefix YYYYMMDD_HHmmss", base)
	}
	return parts[0] + "_" + parts[1], nil
}
