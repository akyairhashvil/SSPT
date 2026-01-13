package tui

import (
	"context"
	"fmt"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// AuthResult represents the outcome of an authentication attempt.
type AuthResult struct {
	Success       bool
	Error         error
	ShouldRetry   bool
	Message       string
	StatusError   string
	PassphraseHash string
}

// authHandler manages passphrase validation and encryption setup.
type authHandler struct {
	db  Database
	ctx context.Context
}

func newAuthHandler(db Database, ctx context.Context) *authHandler {
	return &authHandler{db: db, ctx: ctx}
}

func (h *authHandler) ValidatePassphrase(entered, existingHash string) AuthResult {
	if existingHash == "" {
		return h.setupNewPassphrase(entered)
	}
	return h.validateExistingPassphrase(entered, existingHash)
}

func (h *authHandler) validateExistingPassphrase(entered, existingHash string) AuthResult {
	if entered != "" && util.HashPassphrase(entered) == existingHash {
		return AuthResult{Success: true}
	}
	return AuthResult{
		Success:     false,
		ShouldRetry: true,
		Message:     "Incorrect passphrase",
	}
}

func (h *authHandler) setupNewPassphrase(entered string) AuthResult {
	if entered == "" {
		return AuthResult{
			Success:     false,
			ShouldRetry: true,
			Message:     "Passphrase required",
		}
	}
	if err := util.ValidatePassphrase(entered); err != nil {
		return AuthResult{
			Success:     false,
			ShouldRetry: true,
			Message:     err.Error(),
		}
	}

	encryptErr := error(nil)
	if !h.db.EncryptionStatus().DatabaseEncrypted {
		if err := h.db.EncryptDatabase(h.ctx, entered); err != nil && err != database.ErrSQLCipherUnavailable {
			if !h.db.DatabaseHasData(h.ctx) {
				if recErr := h.db.RecreateEncryptedDatabase(h.ctx, entered); recErr != nil {
					return AuthResult{
						Success:     false,
						ShouldRetry: true,
						Message:     fmt.Sprintf("Failed to encrypt database: %v", recErr),
					}
				}
			} else {
				return AuthResult{
					Success:     false,
					ShouldRetry: true,
					Message:     fmt.Sprintf("Failed to encrypt database: %v", err),
				}
			}
		} else if err == database.ErrSQLCipherUnavailable {
			encryptErr = err
		}
	}

	hash := util.HashPassphrase(entered)
	statusError := ""
	if err := h.db.SetSetting(h.ctx, "passphrase_hash", hash); err != nil {
		statusError = fmt.Sprintf("Error saving passphrase: %v", err)
	}
	if encryptErr == database.ErrSQLCipherUnavailable {
		statusError = "Encryption unavailable in this build; passphrase only locks the UI"
	} else if !h.db.EncryptionStatus().DatabaseEncrypted {
		statusError = "Passphrase set but database remains unencrypted"
	}

	return AuthResult{
		Success:        true,
		PassphraseHash: hash,
		StatusError:    statusError,
	}
}
