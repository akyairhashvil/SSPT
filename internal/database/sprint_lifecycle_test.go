package database

import (
	"context"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestSprintLifecycleFlow(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	defer db.Close()

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
		t.Fatalf("expected at least one sprint")
	}
	sprint := sprints[0]
	if sprint.Status != models.StatusPending {
		t.Fatalf("expected pending status, got %q", sprint.Status)
	}

	if err := db.StartSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("StartSprint failed: %v", err)
	}
	updated, err := db.GetSprints(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints after start failed: %v", err)
	}
	if updated[0].Status != models.StatusActive {
		t.Fatalf("expected active status, got %q", updated[0].Status)
	}

	if err := db.CompleteSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("CompleteSprint failed: %v", err)
	}
	updated, err = db.GetSprints(ctx, dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints after complete failed: %v", err)
	}
	if updated[0].Status != models.StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated[0].Status)
	}
}
