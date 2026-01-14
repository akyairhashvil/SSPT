package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleModalInputRecurrenceToggle(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.Open(&RecurrenceState{
		Options:        []string{"none", "weekly"},
		Cursor:         1,
		Mode:           "none",
		Focus:          "mode",
		Selected:       make(map[string]bool),
		WeekdayOptions: []string{"mon"},
	})

	m, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyTab})
	state, ok := m.modal.RecurrenceState()
	if !ok {
		t.Fatalf("expected recurrence modal state")
	}
	if state.Mode != "weekly" {
		t.Fatalf("expected recurrence mode weekly")
	}
	if state.Focus != "items" {
		t.Fatalf("expected recurrence focus items")
	}

	state.ItemCursor = 0
	m, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	state, ok = m.modal.RecurrenceState()
	if !ok {
		t.Fatalf("expected recurrence modal state")
	}
	if len(state.WeekdayOptions) == 0 {
		t.Fatalf("expected weekday options")
	}
	if !state.Selected[state.WeekdayOptions[0]] {
		t.Fatalf("expected weekday selected")
	}
}
