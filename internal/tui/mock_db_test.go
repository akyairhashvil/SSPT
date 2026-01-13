package tui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

type MockDB struct {
	*database.Database
	searchResults []models.Goal
	searchErr     error
}

func (m *MockDB) Search(ctx context.Context, query util.SearchQuery, workspaceID int64) ([]models.Goal, error) {
	return m.searchResults, m.searchErr
}

func TestSearchUsesMockDatabase(t *testing.T) {
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

	mock := &MockDB{
		Database: db,
		searchResults: []models.Goal{
			{ID: 42, Description: "Mocked"},
		},
	}
	m := NewDashboardModel(ctx, mock, dayID)
	m.lock.Locked = false
	m.search.Active = true
	m.search.Input.Focus()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = model.(DashboardModel)
	if len(m.search.Results) != 1 {
		t.Fatalf("expected mocked search results, got %d", len(m.search.Results))
	}
}
