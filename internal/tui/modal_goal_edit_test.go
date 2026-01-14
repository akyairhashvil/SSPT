package tui

import "testing"

func TestModalGoalEdit_UpdatesGoal(t *testing.T) {
	m := setupTestDashboard(t)
	activeWS := m.workspaces[m.activeWorkspaceIdx]

	if err := m.db.AddGoal(m.ctx, activeWS.ID, "Old Goal", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goalID, err := m.db.GetLastGoalID(m.ctx)
	if err != nil {
		t.Fatalf("GetLastGoalID failed: %v", err)
	}

	m.modal.Open(&GoalEditState{GoalID: goalID})
	m.inputs.textInput.SetValue("Updated Goal")

	updated, _, handled := m.handleModalConfirmGoalEdit()
	if !handled {
		t.Fatalf("expected modal handler to run")
	}
	if updated.modal.Is(ModalGoalEdit) {
		t.Fatalf("expected goal edit modal to close after edit")
	}

	goal, err := updated.db.GetGoalByID(updated.ctx, goalID)
	if err != nil {
		t.Fatalf("GetGoalByID failed: %v", err)
	}
	if goal.Description != "Updated Goal" {
		t.Fatalf("expected updated goal description, got %q", goal.Description)
	}
}
