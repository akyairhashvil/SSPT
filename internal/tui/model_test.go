package tui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
	tea "github.com/charmbracelet/bubbletea"
)

func setupModelDB(t *testing.T) *database.Database {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "model.db")
	db, err := database.Open(ctx, dbPath, "")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("db close failed: %v", err)
		}
	})
	return db
}

func TestNewMainModelInitializing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := setupModelDB(t)
	m := NewMainModel(context.Background(), db)
	if m.state != StateInitializing {
		t.Fatalf("expected initializing state, got %v", m.state)
	}
	if m.View() == "" {
		t.Fatalf("expected non-empty view")
	}
}

func TestMainModelUpdateInitializingInvalidInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := setupModelDB(t)
	m := NewMainModel(context.Background(), db)
	m.textInput.SetValue("x")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(MainModel)
	if updated.err == nil {
		t.Fatalf("expected error for invalid input")
	}
	if updated.state != StateInitializing {
		t.Fatalf("expected to stay in initializing state")
	}
}

func TestMainModelUpdateInitializingValidInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := setupModelDB(t)
	m := NewMainModel(context.Background(), db)
	m.textInput.SetValue("2")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(MainModel)
	if updated.err != nil {
		t.Fatalf("unexpected error: %v", updated.err)
	}
	if updated.state != StateDashboard {
		t.Fatalf("expected dashboard state")
	}
}

func TestNewMainModelWithExistingDay(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := setupModelDB(t)
	ctx := context.Background()
	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.BootstrapDay(ctx, wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	m := NewMainModel(ctx, db)
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state when day exists")
	}
}

func TestMainModelInitAndResize(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := setupModelDB(t)
	ctx := context.Background()
	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.BootstrapDay(ctx, wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	m := NewMainModel(ctx, db)
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected init cmd")
	}
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := model.(MainModel)
	if updated.width != 120 || updated.height != 40 {
		t.Fatalf("expected size updated")
	}
}

func TestMainModelUpdateCtrlC(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := setupModelDB(t)
	m := NewMainModel(context.Background(), db)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
}
