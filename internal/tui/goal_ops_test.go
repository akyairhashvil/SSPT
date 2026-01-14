package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestHandleGoalPriority(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	sprintID := m.sprints[0].ID
	if sprintID <= 0 && len(m.sprints) > 1 {
		sprintID = m.sprints[1].ID
	}
	if err := m.db.AddGoal(m.ctx, wsID, "Priority Goal", sprintID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	found := false
	for i, s := range m.sprints {
		for j, g := range s.Goals {
			if g.Description == "Priority Goal" {
				m.view.focusedColIdx = i
				m.view.focusedGoalIdx = j
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("expected to find goal in sprint list")
	}

	_, _, handled := m.handleGoalPriority("P")
	if !handled {
		t.Fatalf("expected priority handler to handle")
	}
}

func TestHandleGoalStatusToggle(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := m.db.AddGoal(m.ctx, wsID, "Toggle Goal", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	found := false
	for i, s := range m.sprints {
		for j, g := range s.Goals {
			if g.Description == "Toggle Goal" {
				m.view.focusedColIdx = i
				m.view.focusedGoalIdx = j
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("expected to find goal in sprint list")
	}

	next, _, handled := m.handleGoalStatusToggle(" ")
	if !handled {
		t.Fatalf("expected status toggle handler to handle")
	}
	if next.Message != "" && next.Message == "Blocked by dependency. Complete dependencies first." {
		t.Fatalf("unexpected blocked message")
	}
}

func TestHandleGoalTagging(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	var sprintID int64
	for _, s := range m.sprints {
		if s.SprintNumber > 0 {
			sprintID = s.ID
			break
		}
	}
	if sprintID == 0 {
		t.Fatalf("expected a sprint column")
	}
	if err := m.db.AddGoal(m.ctx, wsID, "Tag Goal", sprintID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	found := false
	for i, s := range m.sprints {
		for j, g := range s.Goals {
			if g.Description == "Tag Goal" {
				m.view.focusedColIdx = i
				m.view.focusedGoalIdx = j
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("expected to find goal in sprint list")
	}

	next, _, handled := m.handleGoalTagging("t")
	if !handled {
		t.Fatalf("expected tagging handler to handle")
	}
	if !next.modal.Is(ModalTagging) {
		t.Fatalf("expected tagging modal to open")
	}
	if next.inputs.tagInput.Focused() == false {
		t.Fatalf("expected tag input to be focused")
	}
	state, ok := next.modal.TaggingState()
	if !ok || state.GoalID == 0 {
		t.Fatalf("expected goal id to be set")
	}
	if next.sprints[next.view.focusedColIdx].Goals[next.view.focusedGoalIdx].Status == models.GoalStatusCompleted {
		t.Fatalf("expected pending goal")
	}
}
