package tui

import (
	"strings"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleLockedStateSuccessAndFailure(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.Locked = true
	m.security.lock.PassphraseHash = util.HashPassphrase("secret")

	m.security.lock.PassphraseInput.SetValue("wrong")
	next, _ := m.handleLockedState(tea.KeyMsg{Type: tea.KeyEnter})
	if !next.security.lock.Locked {
		t.Fatalf("expected still locked on wrong passphrase")
	}
	if strings.TrimSpace(next.security.lock.Message) == "" {
		t.Fatalf("expected error message")
	}

	next.security.lock.PassphraseInput.SetValue("secret")
	next, _ = next.handleLockedState(tea.KeyMsg{Type: tea.KeyEnter})
	if next.security.lock.Locked {
		t.Fatalf("expected unlocked on correct passphrase")
	}
	if next.security.lock.PassphraseInput.Value() != "" {
		t.Fatalf("expected passphrase input cleared")
	}
}

func TestHandleModalConfirmDeleteGoal(t *testing.T) {
	m, goalID, sprintIdx := setupGoalInSprint(t)
	m.view.focusedColIdx = sprintIdx
	m.modal.Open(&GoalDeleteState{GoalID: goalID})

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled confirm")
	}
	if m.modal.Is(ModalGoalDelete) {
		t.Fatalf("expected delete modal closed")
	}
	goals, err := m.db.GetGoalsForSprint(m.ctx, m.sprints[sprintIdx].ID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	if len(goals) != 0 {
		t.Fatalf("expected goal deleted")
	}
}

func TestHandleModalConfirmJournalEntry(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.Open(&JournalState{})
	m.inputs.journalInput.SetValue("note")

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.modal.Is(ModalJournaling) {
		t.Fatalf("expected journaling modal closed")
	}
	entries, err := m.db.GetJournalEntries(m.ctx, m.day.ID, m.workspaces[m.activeWorkspaceIdx].ID)
	if err != nil {
		t.Fatalf("GetJournalEntries failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected journal entry")
	}
}

func TestHandleModalInputArchiveFromConfirm(t *testing.T) {
	m, goalID, sprintIdx := setupGoalInSprint(t)
	m.view.focusedColIdx = sprintIdx
	m.modal.Open(&GoalDeleteState{GoalID: goalID})

	m, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.modal.Is(ModalGoalDelete) {
		t.Fatalf("expected confirm modal closed after archive")
	}
	archived, err := m.db.GetArchivedGoals(m.ctx, m.workspaces[m.activeWorkspaceIdx].ID)
	if err != nil {
		t.Fatalf("GetArchivedGoals failed: %v", err)
	}
	if len(archived) == 0 {
		t.Fatalf("expected archived goal")
	}
}
