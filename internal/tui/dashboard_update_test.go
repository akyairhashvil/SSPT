package tui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
	tea "github.com/charmbracelet/bubbletea"
)

func setupTestDashboard(t *testing.T) DashboardModel {
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
	return m
}

func TestDashboardKeyRoutingNewGoal(t *testing.T) {
	m := setupTestDashboard(t)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated, ok := model.(DashboardModel)
	if !ok {
		t.Fatalf("expected DashboardModel, got %T", model)
	}
	if !updated.modal.creatingGoal {
		t.Fatalf("expected creatingGoal to be true")
	}
}

func TestDashboardKeyRoutingSearch(t *testing.T) {
	m := setupTestDashboard(t)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated, ok := model.(DashboardModel)
	if !ok {
		t.Fatalf("expected DashboardModel, got %T", model)
	}
	if !updated.search.Active {
		t.Fatalf("expected searching to be true")
	}
}
