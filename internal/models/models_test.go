package models

import "testing"

func TestSprintStatusConstants(t *testing.T) {
	if StatusPending != "pending" {
		t.Fatalf("StatusPending = %q", StatusPending)
	}
	if StatusActive != "active" {
		t.Fatalf("StatusActive = %q", StatusActive)
	}
	if StatusCompleted != "completed" {
		t.Fatalf("StatusCompleted = %q", StatusCompleted)
	}
	if StatusInterrupted != "interrupted" {
		t.Fatalf("StatusInterrupted = %q", StatusInterrupted)
	}
}

func TestGoalZeroValues(t *testing.T) {
	var g Goal
	if g.ParentID != nil || g.WorkspaceID != nil || g.SprintID != nil {
		t.Fatalf("expected nil pointer fields by default")
	}
	if g.Notes != nil || g.Effort != nil || g.Tags != nil || g.RecurrenceRule != nil || g.Links != nil {
		t.Fatalf("expected nil optional string fields by default")
	}
	if g.CompletedAt != nil || g.ArchivedAt != nil || g.TaskStartedAt != nil {
		t.Fatalf("expected nil time fields by default")
	}
}
