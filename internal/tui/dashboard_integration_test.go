package tui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/akyairhashvil/SSPT/internal/database"
	tea "github.com/charmbracelet/bubbletea"
)

func setupIntegrationDashboard(t *testing.T) (DashboardModel, *database.Database) {
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

	m := NewDashboardModel(ctx, db, dayID)
	m.security.lock.Locked = false
	return m, db
}

func TestDashboardCreateGoalFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)
	m.view.focusedColIdx = config.DefaultFocusColumn

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m, _ = model.(DashboardModel)
	if !m.modal.creatingGoal {
		t.Fatalf("expected creatingGoal to be true")
	}
	m.inputs.textInput.SetValue("Test Goal")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = model.(DashboardModel)

	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	goals, err := db.GetBacklogGoals(m.ctx, wsID)
	if err != nil {
		t.Fatalf("GetBacklogGoals failed: %v", err)
	}
	if len(goals) == 0 {
		t.Fatalf("expected at least one goal")
	}
	if goals[0].Description != "Test Goal" {
		t.Fatalf("expected goal description to match, got %q", goals[0].Description)
	}
}
