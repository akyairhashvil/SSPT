package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
)

func TestExportVaultPlain(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("XDG_DOCUMENTS_DIR", dir)

	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Open(ctx, dbPath, "")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("db close failed: %v", err)
		}
	})

	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Export Me", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}

	path, err := ExportVault(ctx, db, "")
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var out vaultExport
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(out.Workspaces) == 0 {
		t.Fatalf("expected workspaces in export")
	}
	if len(out.Goals) == 0 {
		t.Fatalf("expected goals in export")
	}
}
