package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

func TestMigrateIdempotent(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	if err := db.migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := db.migrate(ctx); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}
}

func TestWithTxRollback(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	err := db.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO workspaces (name, slug) VALUES (?, ?)", "Tx", "tx-rollback"); err != nil {
			return err
		}
		return fmt.Errorf("force rollback")
	})
	if err == nil {
		t.Fatalf("expected error from WithTx")
	}

	var count int
	if err := db.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM workspaces WHERE slug = ?", "tx-rollback").Scan(&count); err != nil {
		t.Fatalf("query count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove workspace, got count %d", count)
	}
}
