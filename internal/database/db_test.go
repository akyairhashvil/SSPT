package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func setupTestDB(t *testing.T, ctx context.Context) *Database {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(ctx, dbPath, "")
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

func TestInitDB_MigrationsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	if err := db.Close(); err != nil {
		t.Fatalf("db close failed: %v", err)
	}
	if _, err := Open(ctx, db.dbFile, ""); err != nil {
		t.Fatalf("Open second run failed: %v", err)
	}
}

func TestWorkspaceCRUD(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if wsID == 0 {
		t.Fatalf("EnsureDefaultWorkspace returned zero ID")
	}
	if _, err := db.CreateWorkspace(ctx, "Work", "work"); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}
	workspaces, err := db.GetWorkspaces(ctx)
	if err != nil {
		t.Fatalf("GetWorkspaces failed: %v", err)
	}
	if len(workspaces) < 2 {
		t.Fatalf("expected at least 2 workspaces, got %d", len(workspaces))
	}
}

func TestGoalLifecycle(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
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
	sprints, err := db.GetSprints(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if len(sprints) == 0 {
		t.Fatalf("expected sprints, got none")
	}
	if err := db.AddGoal(ctx, wsID, "Test Goal", sprints[0].ID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goals, err := db.GetGoalsForSprint(ctx, sprints[0].ID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	if len(goals) == 0 {
		t.Fatalf("expected goal, got none")
	}
	if err := db.UpdateGoalStatus(ctx, goals[0].ID, models.GoalStatusCompleted); err != nil {
		t.Fatalf("UpdateGoalStatus failed: %v", err)
	}
	completed, err := db.GetCompletedGoalsForDay(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetCompletedGoalsForDay failed: %v", err)
	}
	found := false
	for _, g := range completed {
		if g.ID == goals[0].ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected completed goal to appear in results")
	}
}

func TestJournalEntries(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
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
	if err := db.AddJournalEntry(ctx, dayID, wsID, nil, nil, "Entry"); err != nil {
		t.Fatalf("AddJournalEntry failed: %v", err)
	}
	entries, err := db.GetJournalEntries(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestSetGoalDependenciesRollback(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Dependency Target", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	var goalID int64
	if err := db.DB.QueryRowContext(ctx, "SELECT id FROM goals WHERE description = ?", "Dependency Target").Scan(&goalID); err != nil {
		t.Fatalf("lookup goal failed: %v", err)
	}
	if _, err := db.DB.Exec("DROP TABLE task_deps"); err != nil {
		t.Fatalf("drop task_deps failed: %v", err)
	}
	if err := db.SetGoalDependencies(ctx, goalID, []int64{goalID}); err == nil {
		t.Fatalf("expected error when task_deps table missing")
	}
}
