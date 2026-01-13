package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyBindingLocksSession(t *testing.T) {
	m := setupTestDashboard(t)
	m.lock.Locked = false
	m.lock.PassphraseHash = "hashed"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	updated := model.(DashboardModel)
	if !updated.lock.Locked {
		t.Fatalf("expected session to be locked")
	}
	if updated.lock.Message == "" {
		t.Fatalf("expected lock message to be set")
	}
}
