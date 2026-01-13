package database

import "testing"

func TestSprintLifecycle(t *testing.T) {
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
	if len(sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(sprints))
	}

	sprintID := sprints[0].ID
	if err := db.StartSprint(sprintID); err != nil {
		t.Fatalf("StartSprint failed: %v", err)
	}
	if err := db.PauseSprint(sprintID, 10); err != nil {
		t.Fatalf("PauseSprint failed: %v", err)
	}
	if err := db.CompleteSprint(sprintID); err != nil {
		t.Fatalf("CompleteSprint failed: %v", err)
	}

	updated, err := db.GetSprints(dayID, wsID)
	if err != nil {
		t.Fatalf("GetSprints after updates failed: %v", err)
	}
	if updated[0].Status != "completed" {
		t.Fatalf("expected completed status, got %q", updated[0].Status)
	}
	if updated[0].EndTime == nil {
		t.Fatalf("expected end time to be set")
	}
}
