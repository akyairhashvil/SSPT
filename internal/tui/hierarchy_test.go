package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestBuildHierarchy(t *testing.T) {
	parentID := int64(1)
	goals := []models.Goal{
		{ID: 1, Description: "parent"},
		{ID: 2, ParentID: &parentID, Description: "child"},
		{ID: 3, Description: "root"},
	}

	roots := BuildHierarchy(goals)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}

	var parent GoalView
	found := false
	for _, g := range roots {
		if g.ID == 1 {
			parent = g
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find parent goal")
	}
	if len(parent.Subtasks) != 1 || parent.Subtasks[0].ID != 2 {
		t.Fatalf("expected one child with ID 2, got %+v", parent.Subtasks)
	}
}

func TestFlattenMaxDepth(t *testing.T) {
	parentID := int64(1)
	grandParentID := int64(2)
	goals := []models.Goal{
		{ID: 1, Description: "root"},
		{ID: 2, ParentID: &parentID, Description: "child"},
		{ID: 3, ParentID: &grandParentID, Description: "grandchild"},
	}

	roots := BuildHierarchy(goals)
	flat := Flatten(roots, 0, nil, 2)
	if len(flat) != 2 {
		t.Fatalf("expected 2 flattened goals, got %d", len(flat))
	}
	if flat[0].Level != 0 || flat[1].Level != 1 {
		t.Fatalf("expected levels [0,1], got [%d,%d]", flat[0].Level, flat[1].Level)
	}
}
