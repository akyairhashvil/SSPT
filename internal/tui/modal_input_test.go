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

	m.modal.confirmingDelete = true
	m.modal.confirmDeleteGoalID = goalID
	next, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if next.modal.confirmingDelete {
		t.Fatalf("expected confirmingDelete to reset")
	}
	if next.modal.confirmDeleteGoalID != 0 {
		t.Fatalf("expected confirmDeleteGoalID to reset")
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
	if !next.modal.initializingSprints {
		t.Fatalf("expected initializingSprints to be set")
	}
}

func TestHandleModalInputRecurrenceNavigation(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.settingRecurrence = true
	m.modal.recurrenceOptions = []string{"none", "weekly"}
	m.modal.recurrenceCursor = 1
	m.modal.weekdayOptions = []string{"mon", "tue"}
	m.modal.recurrenceMode = "weekly"
	m.modal.recurrenceFocus = "items"
	m.modal.recurrenceItemCursor = 0
	m.modal.recurrenceSelected = make(map[string]bool)

	next, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !next.modal.recurrenceSelected["mon"] {
		t.Fatalf("expected recurrence selection to toggle")
	}

	next, _ = next.handleModalInput(tea.KeyMsg{Type: tea.KeyDown})
	if next.modal.recurrenceItemCursor != 1 {
		t.Fatalf("expected recurrenceItemCursor to advance")
	}
}
