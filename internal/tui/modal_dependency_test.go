package tui

import "testing"

func TestModalDependency_ConfirmSavesDeps(t *testing.T) {
	m := setupTestDashboard(t)
	activeWS := m.workspaces[m.activeWorkspaceIdx]

	if err := m.db.AddGoal(m.ctx, activeWS.ID, "Goal A", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goalA, err := m.db.GetLastGoalID(m.ctx)
	if err != nil {
		t.Fatalf("GetLastGoalID failed: %v", err)
	}

	if err := m.db.AddGoal(m.ctx, activeWS.ID, "Goal B", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goalB, err := m.db.GetLastGoalID(m.ctx)
	if err != nil {
		t.Fatalf("GetLastGoalID failed: %v", err)
	}

	m.modal.Open(&DependencyState{
		GoalID:   goalA,
		Options:  []depOption{{ID: goalB, Label: "Goal B"}},
		Selected: map[int64]bool{goalB: true},
	})

	updated, _, handled := m.handleModalConfirmDependencies()
	if !handled {
		t.Fatalf("expected modal handler to run")
	}
	if updated.modal.Is(ModalDependency) {
		t.Fatalf("expected dependency modal to close after save")
	}

	deps, err := updated.db.GetGoalDependencies(updated.ctx, goalA)
	if err != nil {
		t.Fatalf("GetGoalDependencies failed: %v", err)
	}
	if !deps[goalB] {
		t.Fatalf("expected goal dependency to be saved")
	}
}
