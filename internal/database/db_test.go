package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := InitDB(dbPath, ""); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		if DefaultDB != nil {
			_ = DefaultDB.DB.Close()
			DefaultDB = nil
			DB = nil
		}
	})
	return dbPath
}

func TestInitDB_MigrationsIdempotent(t *testing.T) {
	dbPath := setupTestDB(t)
	if DefaultDB != nil {
		_ = DefaultDB.DB.Close()
		DefaultDB = nil
		DB = nil
	}
	if err := InitDB(dbPath, ""); err != nil {
		t.Fatalf("InitDB second run failed: %v", err)
	}
}

func TestWorkspaceCRUD(t *testing.T) {
	setupTestDB(t)
	wsID, err := EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if wsID == 0 {
		t.Fatalf("EnsureDefaultWorkspace returned zero ID")
	}
	if _, err := CreateWorkspace("Work", "work"); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}
	workspaces, err := GetWorkspaces()
	if err != nil {
		t.Fatalf("GetWorkspaces failed: %v", err)
	}
	if len(workspaces) < 2 {
		t.Fatalf("expected at least 2 workspaces, got %d", len(workspaces))
	}
}

func TestGoalLifecycle(t *testing.T) {
	setupTestDB(t)
	wsID, err := EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := BootstrapDay(wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := CheckCurrentDay()
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}
	sprints, err := GetSprints(dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if len(sprints) == 0 {
		t.Fatalf("expected sprints, got none")
	}
	if err := AddGoal(wsID, "Test Goal", sprints[0].ID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goals, err := GetGoalsForSprint(sprints[0].ID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	if len(goals) == 0 {
		t.Fatalf("expected goal, got none")
	}
	if err := UpdateGoalStatus(goals[0].ID, "completed"); err != nil {
		t.Fatalf("UpdateGoalStatus failed: %v", err)
	}
	completed, err := GetCompletedGoalsForDay(dayID, wsID)
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
	setupTestDB(t)
	wsID, err := EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := BootstrapDay(wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := CheckCurrentDay()
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}
	if err := AddJournalEntry(dayID, wsID, sql.NullInt64{}, sql.NullInt64{}, "Entry"); err != nil {
		t.Fatalf("AddJournalEntry failed: %v", err)
	}
	entries, err := GetJournalEntries(dayID, wsID)
	if err != nil {
		t.Fatalf("GetJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestSetGoalDependenciesRollback(t *testing.T) {
	setupTestDB(t)
	wsID, err := EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := AddGoal(wsID, "Dependency Target", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	var goalID int64
	if err := DefaultDB.DB.QueryRow("SELECT id FROM goals WHERE description = ?", "Dependency Target").Scan(&goalID); err != nil {
		t.Fatalf("lookup goal failed: %v", err)
	}
	if _, err := DefaultDB.DB.Exec("DROP TABLE task_deps"); err != nil {
		t.Fatalf("drop task_deps failed: %v", err)
	}
	if err := SetGoalDependencies(goalID, []int64{goalID}); err == nil {
		t.Fatalf("expected error when task_deps table missing")
	}
}
