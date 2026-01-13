package database

import (
	"context"
	"testing"
)

func TestNullableHelpers(t *testing.T) {
	if got := nullableInt64(0); got.Valid {
		t.Fatalf("expected nullableInt64(0) to be invalid, got valid")
	}
	if got := nullableInt64(42); !got.Valid || got.Int64 != 42 {
		t.Fatalf("expected nullableInt64(42) to be valid with 42, got %+v", got)
	}
	if got := nullableString(""); got.Valid {
		t.Fatalf("expected nullableString(\"\") to be invalid, got valid")
	}
	if got := nullableString("note"); !got.Valid || got.String != "note" {
		t.Fatalf("expected nullableString(\"note\") to be valid, got %+v", got)
	}
	if got := toNullableArg[int](nil); got != nil {
		t.Fatalf("expected toNullableArg(nil) to return nil, got %v", got)
	}
	value := int64(7)
	if got := toNullableArg(&value); got != int64(7) {
		t.Fatalf("expected toNullableArg(&7) to return 7, got %v", got)
	}
}

func TestRankHelpers(t *testing.T) {
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
	sprintID := sprints[0].ID

	if _, err := db.DB.ExecContext(ctx,
		"INSERT INTO goals (workspace_id, description, sprint_id, status, rank) VALUES (?, ?, ?, 'pending', ?)",
		wsID, "Sprint goal 1", sprintID, 2); err != nil {
		t.Fatalf("insert sprint goal 1 failed: %v", err)
	}
	if _, err := db.DB.ExecContext(ctx,
		"INSERT INTO goals (workspace_id, description, sprint_id, status, rank) VALUES (?, ?, ?, 'pending', ?)",
		wsID, "Sprint goal 2", sprintID, 5); err != nil {
		t.Fatalf("insert sprint goal 2 failed: %v", err)
	}

	maxRank, err := db.getMaxGoalRank(ctx, sprintID)
	if err != nil {
		t.Fatalf("getMaxGoalRank failed: %v", err)
	}
	if maxRank != 5 {
		t.Fatalf("expected max sprint rank 5, got %d", maxRank)
	}

	result, err := db.DB.ExecContext(ctx,
		"INSERT INTO goals (workspace_id, description, sprint_id, status, rank) VALUES (?, ?, ?, 'pending', ?)",
		wsID, "Parent goal", sprintID, 1)
	if err != nil {
		t.Fatalf("insert parent goal failed: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("parent goal last insert id failed: %v", err)
	}
	if _, err := db.DB.ExecContext(ctx,
		"INSERT INTO goals (workspace_id, description, sprint_id, parent_id, status, rank) VALUES (?, ?, ?, ?, 'pending', ?)",
		wsID, "Subtask 1", sprintID, parentID, 3); err != nil {
		t.Fatalf("insert subtask 1 failed: %v", err)
	}
	if _, err := db.DB.ExecContext(ctx,
		"INSERT INTO goals (workspace_id, description, sprint_id, parent_id, status, rank) VALUES (?, ?, ?, ?, 'pending', ?)",
		wsID, "Subtask 2", sprintID, parentID, 4); err != nil {
		t.Fatalf("insert subtask 2 failed: %v", err)
	}

	subtaskMax, err := db.getMaxSubtaskRank(ctx, parentID)
	if err != nil {
		t.Fatalf("getMaxSubtaskRank failed: %v", err)
	}
	if subtaskMax != 4 {
		t.Fatalf("expected max subtask rank 4, got %d", subtaskMax)
	}

	if _, err := db.DB.ExecContext(ctx,
		"INSERT INTO goals (workspace_id, description, sprint_id, status, rank) VALUES (?, ?, NULL, 'pending', ?)",
		wsID, "Backlog goal 1", 7); err != nil {
		t.Fatalf("insert backlog goal 1 failed: %v", err)
	}
	backlogMax, err := db.getMaxBacklogRank(ctx, wsID)
	if err != nil {
		t.Fatalf("getMaxBacklogRank failed: %v", err)
	}
	if backlogMax != 7 {
		t.Fatalf("expected max backlog rank 7, got %d", backlogMax)
	}
}
