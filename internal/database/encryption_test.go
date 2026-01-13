package database

import (
	"context"
	"testing"
)

func TestDatabaseEncryption(t *testing.T) {
	if !SQLCipherCompiled() {
		t.Skip("SQLCipher not available")
	}
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Test Goal", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	key := "Pass1234"
	if err := db.EncryptDatabase(ctx, key); err != nil {
		t.Fatalf("EncryptDatabase failed: %v", err)
	}
	path := db.dbFile
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, err := Open(ctx, path, "wrongpass"); err == nil {
		t.Fatalf("expected error with wrong passphrase")
	}
	reopened, err := Open(ctx, path, key)
	if err != nil {
		t.Fatalf("Open with key failed: %v", err)
	}
	defer reopened.Close()
	if reopened.DatabaseHasData(ctx) == false {
		t.Fatalf("expected data after reopen")
	}
}

func TestRekeyDatabase(t *testing.T) {
	if !SQLCipherCompiled() {
		t.Skip("SQLCipher not available")
	}
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Test Goal", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	key := "Pass1234"
	if err := db.EncryptDatabase(ctx, key); err != nil {
		t.Fatalf("EncryptDatabase failed: %v", err)
	}
	newKey := "Pass5678"
	if err := db.RekeyDB(ctx, newKey); err != nil {
		t.Fatalf("RekeyDB failed: %v", err)
	}
	path := db.dbFile
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, err := Open(ctx, path, key); err == nil {
		t.Fatalf("expected old passphrase to fail after rekey")
	}
	reopened, err := Open(ctx, path, newKey)
	if err != nil {
		t.Fatalf("Open with new key failed: %v", err)
	}
	defer reopened.Close()
	if !reopened.DatabaseHasData(ctx) {
		t.Fatalf("expected data after rekey")
	}
}
