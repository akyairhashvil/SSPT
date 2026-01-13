package database

import (
	"testing"
)

func TestAddGoalBacklog(t *testing.T) {
	db := setupTestDB(t)
	wsID, err := db.EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}

	if err := db.AddGoal(wsID, "Test Goal", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}

	goals, err := db.GetBacklogGoals(wsID)
	if err != nil {
		t.Fatalf("GetBacklogGoals failed: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 backlog goal, got %d", len(goals))
	}
	if goals[0].Description != "Test Goal" {
		t.Fatalf("expected description to match, got %q", goals[0].Description)
	}
	if goals[0].SprintID != nil {
		t.Fatalf("expected backlog goal sprint_id to be nil")
	}
}

func TestMoveGoalToSprint(t *testing.T) {
	db := setupTestDB(t)
	wsID, err := db.EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.BootstrapDay(wsID, 1); err != nil {
		t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := db.CheckCurrentDay()
	if dayID == 0 {
		t.Fatalf("CheckCurrentDay returned zero ID")
	}

	sprints, err := db.GetSprints(dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if len(sprints) == 0 {
		t.Fatalf("expected at least one sprint")
	}

	if err := db.AddGoal(wsID, "Move Me", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	var goalID int64
	if err := db.DB.QueryRow("SELECT id FROM goals WHERE description = ?", "Move Me").Scan(&goalID); err != nil {
		t.Fatalf("query goal id failed: %v", err)
	}

	if err := db.MoveGoal(goalID, sprints[0].ID); err != nil {
		t.Fatalf("MoveGoal failed: %v", err)
	}

	goals, err := db.GetGoalsForSprint(sprints[0].ID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 sprint goal, got %d", len(goals))
	}
	if goals[0].ID != goalID {
		t.Fatalf("expected goal id %d, got %d", goalID, goals[0].ID)
	}
}

func TestUpdateGoalStatusCompleted(t *testing.T) {
	db := setupTestDB(t)
	wsID, err := db.EnsureDefaultWorkspace()
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}

	if err := db.AddGoal(wsID, "Complete Me", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	var goalID int64
	if err := db.DB.QueryRow("SELECT id FROM goals WHERE description = ?", "Complete Me").Scan(&goalID); err != nil {
		t.Fatalf("query goal id failed: %v", err)
	}

	if err := db.UpdateGoalStatus(goalID, "completed"); err != nil {
		t.Fatalf("UpdateGoalStatus failed: %v", err)
	}

	var status string
	var completedAt *string
	if err := db.DB.QueryRow("SELECT status, completed_at FROM goals WHERE id = ?", goalID).Scan(&status, &completedAt); err != nil {
		t.Fatalf("query status failed: %v", err)
	}
	if status != "completed" {
		t.Fatalf("expected status completed, got %q", status)
	}
	if completedAt == nil {
		t.Fatalf("expected completed_at to be set")
	}
}
