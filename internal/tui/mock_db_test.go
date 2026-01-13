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

type MockDatabase struct {
	goals      map[int64]*models.Goal
	sprints    map[int64]*models.Sprint
	workspaces map[int64]*models.Workspace

	AddGoalCalls []addGoalCall
}

type addGoalCall struct {
	Description string
	SprintID    int64
	WorkspaceID int64
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		goals:      make(map[int64]*models.Goal),
		sprints:    make(map[int64]*models.Sprint),
		workspaces: make(map[int64]*models.Workspace),
	}
}

func (m *MockDatabase) AddGoal(ctx context.Context, workspaceID int64, description string, sprintID int64) error {
	m.AddGoalCalls = append(m.AddGoalCalls, addGoalCall{
		Description: description,
		SprintID:    sprintID,
		WorkspaceID: workspaceID,
	})
	id := int64(len(m.goals) + 1)
	var sprintPtr *int64
	if sprintID > 0 {
		sprintPtr = &sprintID
	}
	wsID := workspaceID
	m.goals[id] = &models.Goal{ID: id, Description: description, SprintID: sprintPtr, WorkspaceID: &wsID}
	return nil
}

func (m *MockDatabase) GetBacklogGoals(ctx context.Context, workspaceID int64) ([]models.Goal, error) {
	var out []models.Goal
	for _, g := range m.goals {
		if g.WorkspaceID != nil && *g.WorkspaceID == workspaceID && g.SprintID == nil {
			out = append(out, *g)
		}
	}
	return out, nil
}

func (m *MockDatabase) GetGoalsForSprint(ctx context.Context, sprintID int64) ([]models.Goal, error) {
	var out []models.Goal
	for _, g := range m.goals {
		if g.SprintID != nil && *g.SprintID == sprintID {
			out = append(out, *g)
		}
	}
	return out, nil
}

func (m *MockDatabase) EnsureDefaultWorkspace(ctx context.Context) (int64, error) {
	if len(m.workspaces) == 0 {
		id := int64(1)
		m.workspaces[id] = &models.Workspace{ID: id, Name: "Default", Slug: "default"}
		return id, nil
	}
	for id := range m.workspaces {
		return id, nil
	}
	return 0, nil
}

func (m *MockDatabase) CreateWorkspace(ctx context.Context, name, slug string) (int64, error) {
	id := int64(len(m.workspaces) + 1)
	m.workspaces[id] = &models.Workspace{ID: id, Name: name, Slug: slug}
	return id, nil
}

func (m *MockDatabase) GetWorkspaces(ctx context.Context) ([]models.Workspace, error) {
	var out []models.Workspace
	for _, w := range m.workspaces {
		out = append(out, *w)
	}
	return out, nil
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
	m.security.lock.Locked = false
	m.search.Active = true
	m.search.Input.Focus()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = model.(DashboardModel)
	if len(m.search.Results) != 1 {
		t.Fatalf("expected mocked search results, got %d", len(m.search.Results))
	}
}
