package tui

import "testing"

func TestHandleSprintReset(t *testing.T) {
	m := setupTestDashboard(t)
	idx := -1
	for i, s := range m.sprints {
		if s.SprintNumber > 0 {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatalf("expected sprint")
	}
	m.timer.ActiveSprint = &m.sprints[idx]

	m, _, handled := m.handleSprintReset("x")
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.timer.ActiveSprint != nil {
		t.Fatalf("expected active sprint cleared")
	}
}
