package tui

import "testing"

func TestHandleGoalCreate(t *testing.T) {
	m := setupTestDashboard(t)
	next, _, handled := m.handleGoalCreate("n")
	if !handled {
		t.Fatalf("expected handler to handle create")
	}
	if !next.modal.creatingGoal {
		t.Fatalf("expected creatingGoal to be set")
	}
}

func TestHandleGoalEditAndDelete(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := m.db.AddGoal(m.ctx, wsID, "Editable", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	found := false
	for i, s := range m.sprints {
		for j, g := range s.Goals {
			if g.Description == "Editable" {
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

	next, _, handled := m.handleGoalEdit("e")
	if !handled {
		t.Fatalf("expected edit handler to handle")
	}
	if !next.modal.editingGoal {
		t.Fatalf("expected editingGoal to be true")
	}

	next, _, handled = m.handleGoalDelete("d")
	if !handled {
		t.Fatalf("expected delete handler to handle")
	}
	if !next.modal.confirmingDelete {
		t.Fatalf("expected confirmingDelete to be true")
	}
}

func TestHandleGoalMove(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := m.db.AddGoal(m.ctx, wsID, "Movable", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	found := false
	for i, s := range m.sprints {
		for j, g := range s.Goals {
			if g.Description == "Movable" {
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

	next, _, handled := m.handleGoalMove("m")
	if !handled {
		t.Fatalf("expected move handler to handle")
	}
	if !next.modal.movingGoal {
		t.Fatalf("expected movingGoal to be true")
	}
}
