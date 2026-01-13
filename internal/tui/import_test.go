package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
)

func TestImportSeedJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Open(dbPath, "")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("db close failed: %v", err)
		}
	})

	wsID, err := db.EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.BootstrapDay(wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := db.CheckCurrentDay()
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}

	seed := `{
  "backlog": [
    {"description": "Backlog Task"}
  ],
  "sprints": [
    {"number": 1, "tasks": [
      {"description": "Sprint Task"}
    ]}
  ]
}`
	seedPath := filepath.Join(dir, "seed.json")
	if err := os.WriteFile(seedPath, []byte(seed), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	count, _, backlogFallback, err := ImportSeed(db, seedPath, wsID, dayID)
	if err != nil {
		t.Fatalf("ImportSeed failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 imported tasks, got %d", count)
	}
	if backlogFallback != 0 {
		t.Fatalf("expected no backlog fallback, got %d", backlogFallback)
	}
}
