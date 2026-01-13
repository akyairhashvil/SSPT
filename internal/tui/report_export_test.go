package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
)

func setupReportDB(t *testing.T) (*database.Database, context.Context, int64, int64) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "report.db")
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
	if err := db.BootstrapDay(ctx, wsID, 2); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := db.CheckCurrentDay(ctx)
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}
	return db, ctx, wsID, dayID
}

func seedReportData(t *testing.T, db *database.Database, ctx context.Context, wsID, dayID int64) int64 {
	t.Helper()
	sprints, err := db.GetSprints(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if len(sprints) == 0 {
		t.Fatalf("expected sprints")
	}
	sprintID := sprints[0].ID
	if err := db.AddGoal(ctx, wsID, "Write tests", sprintID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goalID, err := db.GetLastGoalID(ctx)
	if err != nil || goalID == 0 {
		t.Fatalf("GetLastGoalID failed: %v", err)
	}
	if err := db.UpdateGoalStatus(ctx, goalID, models.GoalStatusCompleted); err != nil {
		t.Fatalf("UpdateGoalStatus failed: %v", err)
	}
	if err := db.AddSubtask(ctx, "Subtask", goalID); err != nil {
		t.Fatalf("AddSubtask failed: %v", err)
	}
	if err := db.AddJournalEntry(ctx, dayID, wsID, &sprintID, &goalID, "Note"); err != nil {
		t.Fatalf("AddJournalEntry failed: %v", err)
	}
	return sprintID
}

func TestGenerateReportCreatesFile(t *testing.T) {
	db, ctx, wsID, dayID := setupReportDB(t)
	seedReportData(t, db, ctx, wsID, dayID)

	docDir := t.TempDir()
	t.Setenv("XDG_DOCUMENTS_DIR", docDir)
	path, err := GenerateReport(ctx, db, dayID, wsID)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Productivity Report") {
		t.Fatalf("expected report header, got: %s", content)
	}
	if !strings.Contains(content, "Write tests") {
		t.Fatalf("expected goal in report")
	}
}

func TestGeneratePDFReportCreatesFile(t *testing.T) {
	db, ctx, wsID, dayID := setupReportDB(t)
	seedReportData(t, db, ctx, wsID, dayID)

	docDir := t.TempDir()
	t.Setenv("XDG_DOCUMENTS_DIR", docDir)
	path, err := GeneratePDFReport(ctx, db, dayID, wsID)
	if err != nil {
		t.Fatalf("GeneratePDFReport failed: %v", err)
	}
	if filepath.Ext(path) != ".pdf" {
		t.Fatalf("expected pdf report, got %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("pdf report missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty pdf report")
	}
}

func TestExportVault(t *testing.T) {
	db, ctx, wsID, dayID := setupReportDB(t)
	seedReportData(t, db, ctx, wsID, dayID)

	docDir := t.TempDir()
	t.Setenv("XDG_DOCUMENTS_DIR", docDir)

	path, err := ExportVault(ctx, db, "")
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export failed: %v", err)
	}
	if !strings.Contains(string(raw), "\"workspaces\"") {
		t.Fatalf("expected workspace export content")
	}

	encPath, err := ExportVault(ctx, db, "hash")
	if err != nil {
		t.Fatalf("ExportVault encrypted failed: %v", err)
	}
	encRaw, err := os.ReadFile(encPath)
	if err != nil {
		t.Fatalf("read encrypted export failed: %v", err)
	}
	if !strings.Contains(string(encRaw), "\"encrypted\": true") {
		t.Fatalf("expected encrypted export marker")
	}
	if wsID == 0 || dayID == 0 {
		t.Fatalf("unexpected zero IDs")
	}
}
