package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleModalInputConfirmingDeleteArchive(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := m.db.AddGoal(m.ctx, wsID, "Archive Me", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	var goalID int64
	for _, sprint := range m.sprints {
		for _, g := range sprint.Goals {
			if g.Description == "Archive Me" {
				goalID = g.ID
				break
			}
		}
	}
	if goalID == 0 {
		t.Fatalf("expected goal to exist")
	}

	m.modal.Open(&GoalDeleteState{GoalID: goalID})
	next, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if next.modal.Is(ModalGoalDelete) {
		t.Fatalf("expected delete modal to close")
	}
}

func TestHandleModalInputConfirmingClearDB(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.confirmingClearDB = true
	m.security.clearDBNeedsPass = false

	next, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if next.security.confirmingClearDB {
		t.Fatalf("expected confirmingClearDB to reset")
	}
	if !next.modal.Is(ModalWorkspaceInit) {
		t.Fatalf("expected workspace init modal to be set")
	}
}

func TestHandleModalInputRecurrenceNavigation(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.Open(&RecurrenceState{
		Options:        []string{"none", "weekly"},
		Cursor:         1,
		Mode:           "weekly",
		WeekdayOptions: []string{"mon", "tue"},
		Focus:          "items",
		ItemCursor:     0,
		Selected:       make(map[string]bool),
	})

	next, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	state, ok := next.modal.RecurrenceState()
	if !ok {
		t.Fatalf("expected recurrence modal state")
	}
	if !state.Selected["mon"] {
		t.Fatalf("expected recurrence selection to toggle")
	}

	next, _ = next.handleModalInput(tea.KeyMsg{Type: tea.KeyDown})
	state, ok = next.modal.RecurrenceState()
	if !ok {
		t.Fatalf("expected recurrence modal state")
	}
	if state.ItemCursor != 1 {
		t.Fatalf("expected recurrenceItemCursor to advance")
	}
}
