package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
)

func setupImportDB(t *testing.T) (*database.Database, context.Context, int64, int64) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "import.db")
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
	if err := db.BootstrapDay(ctx, wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := db.CheckCurrentDay(ctx)
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}
	return db, ctx, wsID, dayID
}

func TestImportSeedJSON(t *testing.T) {
	db, ctx, wsID, dayID := setupImportDB(t)
	payload := seedConfig{
		Backlog: []database.GoalSeed{
			{Description: "Backlog task"},
		},
		Sprints: []struct {
			Number int                 `json:"number"`
			Tasks  []database.GoalSeed `json:"tasks"`
		}{
			{
				Number: 1,
				Tasks: []database.GoalSeed{
					{Description: "Sprint task"},
				},
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "seed.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write seed failed: %v", err)
	}
	imported, hash, backlogFallback, err := ImportSeed(ctx, db, path, wsID, dayID)
	if err != nil {
		t.Fatalf("ImportSeed JSON failed: %v", err)
	}
	if imported < 2 {
		t.Fatalf("expected imported goals, got %d", imported)
	}
	if hash == "" {
		t.Fatalf("expected hash")
	}
	if backlogFallback != 0 {
		t.Fatalf("expected backlog fallback 0, got %d", backlogFallback)
	}
}

func TestImportSeedDSL(t *testing.T) {
	db, ctx, wsID, dayID := setupImportDB(t)
	dsl := `
= Work Space
+ 1
* First task #Tag !2 @m
- Subtask
* Second task

* Backlog item
`
	path := filepath.Join(t.TempDir(), "seed.txt")
	if err := os.WriteFile(path, []byte(dsl), 0o600); err != nil {
		t.Fatalf("write seed failed: %v", err)
	}
	imported, hash, backlogFallback, err := ImportSeed(ctx, db, path, wsID, dayID)
	if err != nil {
		t.Fatalf("ImportSeed DSL failed: %v", err)
	}
	if imported < 3 {
		t.Fatalf("expected imported goals, got %d", imported)
	}
	if hash == "" {
		t.Fatalf("expected hash")
	}
	if backlogFallback != 0 {
		t.Fatalf("expected backlog fallback 0, got %d", backlogFallback)
	}
}
