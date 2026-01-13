package database

import (
	"context"
	"encoding/json"
	"testing"
)

func TestExportVault(t *testing.T) {
	ctx := context.Background()
	builder := NewTestDataBuilder(t).
		WithWorkspace("Work").
		WithSprints(2).
		WithGoals(2)
	db := builder.Build()
	defer db.Close()

	payload, err := db.ExportVault(ctx, ExportOptions{})
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	var export VaultExport
	if err := json.Unmarshal(payload, &export); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if len(export.Workspaces) == 0 {
		t.Fatalf("expected workspaces in export")
	}
	if len(export.Goals) == 0 {
		t.Fatalf("expected goals in export")
	}
}
