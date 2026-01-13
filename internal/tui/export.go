package tui

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

type vaultExport struct {
	AppVersion string                        `json:"app_version"`
	ExportedAt string                        `json:"exported_at"`
	Workspaces []database.ExportWorkspace    `json:"workspaces"`
	Days       []database.ExportDay          `json:"days"`
	Sprints    []database.ExportSprint       `json:"sprints"`
	Goals      []database.ExportGoal         `json:"goals"`
	Journal    []database.ExportJournalEntry `json:"journal_entries"`
	TaskDeps   []database.ExportTaskDep      `json:"task_deps"`
}

type encryptedExport struct {
	Encrypted  bool   `json:"encrypted"`
	AppVersion string `json:"app_version"`
	ExportedAt string `json:"exported_at"`
	Nonce      string `json:"nonce"`
	Data       string `json:"data"`
}

// ExportVault writes a JSON export of all data, optionally encrypted with a passphrase hash.
func ExportVault(ctx context.Context, db Database, passphraseHash string) (string, error) {
	workspaces, err := db.GetWorkspaces(ctx)
	if err != nil {
		return "", err
	}
	exportWorkspaces := make([]database.ExportWorkspace, 0, len(workspaces))
	for _, ws := range workspaces {
		exportWorkspaces = append(exportWorkspaces, toExportWorkspace(ws))
	}
	days, err := db.GetAllDays(ctx)
	if err != nil {
		return "", err
	}
	sprints, err := db.GetAllSprintsFlat(ctx)
	if err != nil {
		return "", err
	}
	goals, err := db.GetAllGoalsExport(ctx)
	if err != nil {
		return "", err
	}
	journal, err := db.GetAllJournalEntriesExport(ctx)
	if err != nil {
		return "", err
	}
	deps, err := db.GetAllTaskDeps(ctx)
	if err != nil {
		return "", err
	}

	export := vaultExport{
		AppVersion: AppVersion,
		ExportedAt: time.Now().Format(time.RFC3339),
		Workspaces: exportWorkspaces,
		Days:       days,
		Sprints:    sprints,
		Goals:      goals,
		Journal:    journal,
		TaskDeps:   deps,
	}

	raw, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", err
	}

	reportRoot := filepath.Join(util.ReportsDir("sspt"), "exports")
	if err := os.MkdirAll(reportRoot, 0o755); err != nil {
		return "", err
	}

	filename := filepath.Join(reportRoot, fmt.Sprintf("sspt_export_%s.json", time.Now().Format("20060102_150405")))
	if passphraseHash == "" {
		if err := os.WriteFile(filename, raw, 0o600); err != nil {
			return "", err
		}
		return filename, nil
	}

	enc, err := encryptExport(raw, passphraseHash)
	if err != nil {
		return "", err
	}
	encBytes, err := json.MarshalIndent(enc, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filename, encBytes, 0o600); err != nil {
		return "", err
	}
	return filename, nil
}

func toExportWorkspace(ws models.Workspace) database.ExportWorkspace {
	return database.ExportWorkspace{
		ID:            ws.ID,
		Name:          ws.Name,
		Slug:          ws.Slug,
		ViewMode:      ws.ViewMode,
		Theme:         ws.Theme,
		ShowBacklog:   ws.ShowBacklog,
		ShowCompleted: ws.ShowCompleted,
		ShowArchived:  ws.ShowArchived,
	}
}

func encryptExport(plaintext []byte, passphraseHash string) (encryptedExport, error) {
	sum := sha256.Sum256([]byte(passphraseHash))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return encryptedExport{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedExport{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return encryptedExport{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return encryptedExport{
		Encrypted:  true,
		AppVersion: AppVersion,
		ExportedAt: time.Now().Format(time.RFC3339),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Data:       base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}
