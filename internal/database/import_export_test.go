package database

import (
	"context"
	"testing"
)

func TestVaultExportImport(t *testing.T) {
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
	if err := db.AddGoal(ctx, wsID, "Goal A #tag", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Goal B", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goals, err := db.GetBacklogGoals(ctx, wsID)
	if err != nil {
		t.Fatalf("GetBacklogGoals failed: %v", err)
	}
	if len(goals) < 2 {
		t.Fatalf("expected at least 2 goals")
	}
	depGoalID := goals[0].ID
	blockerID := goals[1].ID
	if err := db.SetGoalDependencies(ctx, depGoalID, []int64{blockerID}); err != nil {
		t.Fatalf("SetGoalDependencies failed: %v", err)
	}
	if err := db.AddJournalEntry(ctx, dayID, wsID, nil, nil, "Note"); err != nil {
		t.Fatalf("AddJournalEntry failed: %v", err)
	}

	payload, err := db.ExportVault(ctx, ExportOptions{})
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	otherDB := setupTestDB(t, ctx)
	if err := otherDB.ImportVault(ctx, payload); err != nil {
		t.Fatalf("ImportVault failed: %v", err)
	}

	workspaces, err := otherDB.GetWorkspaces(ctx)
	if err != nil {
		t.Fatalf("GetWorkspaces failed: %v", err)
	}
	if len(workspaces) == 0 {
		t.Fatalf("expected workspaces after import")
	}
	importedGoals, err := otherDB.GetAllGoalsExport(ctx)
	if err != nil {
		t.Fatalf("GetAllGoalsExport failed: %v", err)
	}
	if len(importedGoals) != len(goals) {
		t.Fatalf("expected %d goals after import, got %d", len(goals), len(importedGoals))
	}
	deps, err := otherDB.GetGoalDependencies(ctx, depGoalID)
	if err != nil {
		t.Fatalf("GetGoalDependencies failed: %v", err)
	}
	if !deps[blockerID] {
		t.Fatalf("expected dependency %d -> %d after import", depGoalID, blockerID)
	}
	entries, err := otherDB.GetJournalEntries(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 journal entry after import, got %d", len(entries))
	}
}
