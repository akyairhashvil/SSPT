package database

import (
	"context"
	"testing"
)

func TestParseAddColumnMigration(t *testing.T) {
	table, column, ok := parseAddColumnMigration("ALTER TABLE goals ADD COLUMN rank INTEGER DEFAULT 0")
	if !ok {
		t.Fatalf("expected add column migration to parse")
	}
	if table != "goals" || column != "rank" {
		t.Fatalf("expected goals.rank, got %s.%s", table, column)
	}
	if _, _, ok := parseAddColumnMigration("CREATE TABLE goals (id INTEGER)"); ok {
		t.Fatalf("expected non-alter statement to be ignored")
	}
}

func TestColumnExists(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	exists, err := db.columnExists(ctx, "goals", "description")
	if err != nil {
		t.Fatalf("columnExists failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected goals.description to exist")
	}
	exists, err = db.columnExists(ctx, "goals", "not_a_column")
	if err != nil {
		t.Fatalf("columnExists failed: %v", err)
	}
	if exists {
		t.Fatalf("expected non-existent column to be false")
	}
}
