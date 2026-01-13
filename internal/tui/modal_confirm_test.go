package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleModalConfirmDelete(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := m.db.AddGoal(m.ctx, wsID, "Delete Me", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	var goalID int64
	for _, sprint := range m.sprints {
		for _, g := range sprint.Goals {
			if g.Description == "Delete Me" {
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
	next, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected confirm to be handled")
	}
	if next.modal.confirmingDelete {
		t.Fatalf("expected confirmingDelete to reset")
	}
	if next.modal.confirmDeleteGoalID != 0 {
		t.Fatalf("expected confirmDeleteGoalID to reset")
	}
}

func TestHandleModalConfirmPassphraseStage0Invalid(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.changingPassphrase = true
	m.security.passphraseStage = 0
	m.security.lock.PassphraseHash = util.HashPassphrase("Abcdefg1")
	m.inputs.passphraseCurrent.SetValue("Wrongpass1")

	next, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected confirm to be handled")
	}
	if next.security.passphraseStatus == "" {
		t.Fatalf("expected passphrase status error")
	}
	if next.security.passphraseStage != 0 {
		t.Fatalf("expected passphrase stage to remain at 0")
	}
}

func TestHandleModalConfirmPassphraseStage0Valid(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.changingPassphrase = true
	m.security.passphraseStage = 0
	m.security.lock.PassphraseHash = util.HashPassphrase("Abcdefg1")
	m.inputs.passphraseCurrent.SetValue("Abcdefg1")

	next, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected confirm to be handled")
	}
	if next.security.passphraseStage != 1 {
		t.Fatalf("expected passphrase stage to advance")
	}
}

func TestHandleModalConfirmPassphraseStage1Invalid(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.changingPassphrase = true
	m.security.passphraseStage = 1
	m.inputs.passphraseNew.SetValue("short")

	next, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected confirm to be handled")
	}
	if next.security.passphraseStatus == "" {
		t.Fatalf("expected passphrase status error")
	}
	if next.security.passphraseStage != 1 {
		t.Fatalf("expected passphrase stage to remain at 1")
	}
}

func TestHandleModalConfirmPassphraseStage2Mismatch(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.changingPassphrase = true
	m.security.passphraseStage = 2
	m.inputs.passphraseNew.SetValue("Abcdefg1")
	m.inputs.passphraseConfirm.SetValue("Abcdefg2")

	next, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected confirm to be handled")
	}
	if next.security.passphraseStatus == "" {
		t.Fatalf("expected passphrase mismatch error")
	}
	if next.security.passphraseStage != 2 {
		t.Fatalf("expected passphrase stage to remain at 2")
	}
}

func TestHandleModalConfirmClearDBNeedsPass(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.confirmingClearDB = true
	m.security.clearDBNeedsPass = true
	m.security.lock.PassphraseHash = util.HashPassphrase("Abcdefg1")
	m.security.lock.PassphraseInput.SetValue("")

	next, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected confirm to be handled")
	}
	if next.security.clearDBStatus == "" {
		t.Fatalf("expected clearDBStatus to be set")
	}
	if !next.security.confirmingClearDB {
		t.Fatalf("expected confirmingClearDB to remain true")
	}
}
