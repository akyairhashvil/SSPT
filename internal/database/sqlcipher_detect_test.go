package database

import (
	"context"
	"testing"
)

func TestDetectSQLCipher(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	defer db.Close()

	available, _ := db.detectSQLCipher(ctx)
	if SQLCipherCompiled() && !available {
		t.Fatalf("expected SQLCipher detection to be true when compiled")
	}
}
