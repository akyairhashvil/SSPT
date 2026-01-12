package tui

import (
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
	tea "github.com/charmbracelet/bubbletea"
)

func setupTestDashboard(t *testing.T) DashboardModel {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := database.InitDB(dbPath, ""); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		if database.DefaultDB != nil {
			_ = database.DefaultDB.DB.Close()
			database.DefaultDB = nil
			database.DB = nil
		}
	})
	wsID, err := database.EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := database.BootstrapDay(wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := database.CheckCurrentDay()
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}

	m := NewDashboardModel(database.DB, dayID)
	m.locked = false
	return m
}

func TestDashboardKeyRoutingNewGoal(t *testing.T) {
	m := setupTestDashboard(t)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated, ok := model.(DashboardModel)
	if !ok {
		t.Fatalf("expected DashboardModel, got %T", model)
	}
	if !updated.creatingGoal {
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
	if !updated.searching {
		t.Fatalf("expected searching to be true")
	}
}
