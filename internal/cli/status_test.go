package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/migration"
)

func TestStatusRowsIncludesAppliedRolledBackAndPending(t *testing.T) {
	appliedAt := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	applied := []migration.MigrationRecord{
		{Version: "20260620_120000", Filename: "20260620_120000_create_users", AppliedAt: appliedAt, AppliedBy: "fandy", ExecutionMs: 12},
		{Version: "20260620_120001", Filename: "20260620_120001_create_posts", AppliedAt: appliedAt, AppliedBy: "fandy", ExecutionMs: 7, RolledBack: true},
	}
	files := []migration.MigrationFile{
		{Version: "20260620_120000", Filename: "20260620_120000_create_users"},
		{Version: "20260620_120002", Filename: "20260620_120002_add_email"},
	}

	rows := statusRows(applied, files)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Status != "applied" {
		t.Fatalf("rows[0].Status = %q", rows[0].Status)
	}
	if rows[1].Status != "rolled-back" {
		t.Fatalf("rows[1].Status = %q", rows[1].Status)
	}
	if rows[2].Status != "pending" {
		t.Fatalf("rows[2].Status = %q", rows[2].Status)
	}
}

func TestRunStatusIntegration(t *testing.T) {
	configPath, _ := setupUpIntegration(t)
	var upOutput bytes.Buffer
	if err := RunUp(context.Background(), &upOutput, configPath, false, false); err != nil {
		t.Fatalf("RunUp() setup error = %v", err)
	}

	var stdout bytes.Buffer
	if err := RunStatus(context.Background(), &stdout, configPath); err != nil {
		t.Fatalf("RunStatus() error = %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"VERSION", "FILENAME", "STATUS", "APPLIED_AT", "20260620_120000", "applied"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected status output to contain %q, got %q", want, output)
		}
	}
}
