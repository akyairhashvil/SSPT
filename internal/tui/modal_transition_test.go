package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModalCancelClearsState(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.Open(&GoalDeleteState{GoalID: 123})
	m.security.confirmingClearDB = true
	m.security.clearDBNeedsPass = true
	m.security.clearDBStatus = "pending"
	m.security.lock.PassphraseInput.SetValue("secret")

	next, _, handled := m.handleModalCancel(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Fatalf("expected escape to be handled")
	}
	if next.modal.Is(ModalGoalDelete) {
		t.Fatalf("expected delete modal to be cleared")
	}
	if next.security.confirmingClearDB {
		t.Fatalf("expected confirmingClearDB to be cleared")
	}
	if next.security.clearDBNeedsPass {
		t.Fatalf("expected clearDBNeedsPass to be cleared")
	}
	if next.security.clearDBStatus != "" {
		t.Fatalf("expected clearDBStatus to be cleared, got %q", next.security.clearDBStatus)
	}
	if next.security.lock.PassphraseInput.Value() != "" {
		t.Fatalf("expected passphrase input to be reset, got %q", next.security.lock.PassphraseInput.Value())
	}
}
