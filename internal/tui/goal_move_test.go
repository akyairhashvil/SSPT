package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleMoveModeToBacklog(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	dayID := m.day.ID

	sprints, err := m.db.GetSprints(m.ctx, dayID, wsID)
	if err != nil || len(sprints) == 0 {
		t.Fatalf("GetSprints failed: %v", err)
	}
	// Seed a goal in the first real sprint.
	if err := m.db.AddGoal(m.ctx, wsID, "Move me", sprints[0].ID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(dayID)

	targetIdx := -1
	for i, s := range m.sprints {
		if s.SprintNumber > 0 && len(s.Goals) > 0 {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		t.Fatalf("expected sprint with goals")
	}
	m.view.focusedColIdx = targetIdx
	m.view.focusedGoalIdx = 0
	m.modal.movingGoal = true

	m, _ = m.handleMoveMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	if m.modal.movingGoal {
		t.Fatalf("expected movingGoal cleared")
	}
	backlog, err := m.db.GetBacklogGoals(m.ctx, wsID)
	if err != nil {
		t.Fatalf("GetBacklogGoals failed: %v", err)
	}
	if len(backlog) == 0 {
		t.Fatalf("expected backlog to have moved goal")
	}
}

func TestHandleMoveModeEsc(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.movingGoal = true
	m, _ = m.handleMoveMode(tea.KeyMsg{Type: tea.KeyEsc})
	if m.modal.movingGoal {
		t.Fatalf("expected movingGoal false after esc")
	}
}
