package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleModalInputRecurrenceToggle(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.settingRecurrence = true
	m.modal.recurrenceOptions = []string{"none", "weekly"}
	m.modal.recurrenceCursor = 1
	m.modal.recurrenceMode = "none"
	m.modal.recurrenceFocus = "mode"

	m, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyTab})
	if m.modal.recurrenceMode != "weekly" {
		t.Fatalf("expected recurrence mode weekly")
	}
	if m.modal.recurrenceFocus != "items" {
		t.Fatalf("expected recurrence focus items")
	}

	m.modal.recurrenceItemCursor = 0
	m, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if len(m.modal.weekdayOptions) == 0 {
		t.Fatalf("expected weekday options")
	}
	if !m.modal.recurrenceSelected[m.modal.weekdayOptions[0]] {
		t.Fatalf("expected weekday selected")
	}
}
