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
	m.modal.confirmingDelete = true
	m.modal.confirmDeleteGoalID = goalID

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled confirm")
	}
	if m.modal.confirmingDelete || m.modal.confirmDeleteGoalID != 0 {
		t.Fatalf("expected delete modal cleared")
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
	m.modal.journaling = true
	m.inputs.journalInput.SetValue("note")

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.modal.journaling {
		t.Fatalf("expected journaling closed")
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
	m.modal.confirmingDelete = true
	m.modal.confirmDeleteGoalID = goalID

	m, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.modal.confirmingDelete {
		t.Fatalf("expected confirm cleared after archive")
	}
	archived, err := m.db.GetArchivedGoals(m.ctx, m.workspaces[m.activeWorkspaceIdx].ID)
	if err != nil {
		t.Fatalf("GetArchivedGoals failed: %v", err)
	}
	if len(archived) == 0 {
		t.Fatalf("expected archived goal")
	}
}
