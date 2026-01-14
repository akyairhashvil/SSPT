package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleNormalModeLockAndClear(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.PassphraseHash = ""
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	if !m.security.lock.Locked {
		t.Fatalf("expected locked state")
	}
	if !strings.Contains(m.security.lock.Message, "passphrase") {
		t.Fatalf("expected lock message")
	}

	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if !m.security.confirmingClearDB {
		t.Fatalf("expected confirming clear db")
	}
}

func TestHandleNormalModeExportAndSearch(t *testing.T) {
	m := setupTestDashboard(t)
	t.Setenv("XDG_DOCUMENTS_DIR", t.TempDir())
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyCtrlE})
	if !strings.Contains(m.Message, "Export") {
		t.Fatalf("expected export message")
	}

	m.search.Input.SetValue("")
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.search.Active {
		t.Fatalf("expected search active")
	}
}

func TestHandleNormalModePassphraseAndQuit(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.PassphraseHash = ""
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !m.security.changingPassphrase {
		t.Fatalf("expected passphrase change mode")
	}

	_, cmd := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit message")
	}
}

func TestHandleNormalModeDayNavigation(t *testing.T) {
	m := setupTestDashboard(t)
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}})
	if m.Message == "" {
		t.Fatalf("expected previous day message")
	}
	m.Message = ""
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'>'}})
	if m.Message == "" {
		t.Fatalf("expected next day message")
	}
}

func TestHandleNormalModeDispatchesGoalCreate(t *testing.T) {
	m := setupTestDashboard(t)
	m, _ = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.modal.creatingGoal {
		t.Fatalf("expected goal create to be handled by dispatcher")
	}
}
