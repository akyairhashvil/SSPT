package database

import (
	"context"
	"errors"
	"testing"
)

func TestAddGoalDependencyDetectsCycles(t *testing.T) {
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

	goalA := addGoalForDependencyTest(t, db, ctx, wsID, sprintID, "Goal A")
	goalB := addGoalForDependencyTest(t, db, ctx, wsID, sprintID, "Goal B")
	goalC := addGoalForDependencyTest(t, db, ctx, wsID, sprintID, "Goal C")

	if err := db.AddGoalDependency(ctx, goalA, goalA); !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency for self reference, got %v", err)
	}

	if err := db.AddGoalDependency(ctx, goalA, goalB); err != nil {
		t.Fatalf("expected A->B to succeed, got %v", err)
	}
	if err := db.AddGoalDependency(ctx, goalB, goalA); !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency for A<->B, got %v", err)
	}

	if err := db.AddGoalDependency(ctx, goalB, goalC); err != nil {
		t.Fatalf("expected B->C to succeed, got %v", err)
	}
	if err := db.AddGoalDependency(ctx, goalC, goalA); !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency for A->B->C->A, got %v", err)
	}
}

func addGoalForDependencyTest(t *testing.T, db *Database, ctx context.Context, wsID, sprintID int64, name string) int64 {
	t.Helper()
	if err := db.AddGoal(ctx, wsID, name, sprintID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goals, err := db.GetGoalsForSprint(ctx, sprintID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	for _, g := range goals {
		if g.Description == name {
			return g.ID
		}
	}
	t.Fatalf("expected to find goal %q", name)
	return 0
}
