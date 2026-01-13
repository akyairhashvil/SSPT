package tui

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/util"
)

type authStubDB struct {
	Database
	encrypted bool
	setErr    error
	encErr    error
	hasData   bool
}

func (d *authStubDB) EncryptionStatus() database.EncryptionInfo {
	return database.EncryptionInfo{DatabaseEncrypted: d.encrypted}
}

func (d *authStubDB) EncryptDatabase(ctx context.Context, key string) error {
	return d.encErr
}

func (d *authStubDB) DatabaseHasData(ctx context.Context) bool {
	return d.hasData
}

func (d *authStubDB) RecreateEncryptedDatabase(ctx context.Context, key string) error {
	return nil
}

func (d *authStubDB) SetSetting(ctx context.Context, key, value string) error {
	return d.setErr
}

func newAuthStubDB(t *testing.T) *authStubDB {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
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
	return &authStubDB{Database: db}
}

func TestAuthHandlerExistingPassphrase(t *testing.T) {
	db := newAuthStubDB(t)
	handler := newAuthHandler(db, context.Background())
	hash := util.HashPassphrase("Abcdefg1")

	result := handler.ValidatePassphrase("Abcdefg1", hash)
	if !result.Success {
		t.Fatalf("expected success for matching passphrase")
	}

	result = handler.ValidatePassphrase("Wrongpass1", hash)
	if result.Success || !result.ShouldRetry {
		t.Fatalf("expected retry for wrong passphrase")
	}
}

func TestAuthHandlerNewPassphrase(t *testing.T) {
	db := newAuthStubDB(t)
	handler := newAuthHandler(db, context.Background())

	result := handler.ValidatePassphrase("short", "")
	if result.Success {
		t.Fatalf("expected failure for invalid passphrase")
	}

	result = handler.ValidatePassphrase("Abcdefg1", "")
	if !result.Success {
		t.Fatalf("expected success for valid passphrase")
	}
	if result.PassphraseHash == "" {
		t.Fatalf("expected passphrase hash to be set")
	}
}

func TestAuthHandlerEncryptFailure(t *testing.T) {
	db := newAuthStubDB(t)
	db.encErr = errors.New("encrypt failed")
	db.hasData = true
	handler := newAuthHandler(db, context.Background())

	result := handler.ValidatePassphrase("Abcdefg1", "")
	if result.Success {
		t.Fatalf("expected failure when encryption fails with data")
	}
	if result.Message == "" {
		t.Fatalf("expected error message on encryption failure")
	}
}
