package tui

import (
	"strings"
	"testing"
)

func TestRenderLockScreenTitle(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 60
	m.height = 20
	m.security.lock.PassphraseHash = ""
	output := m.renderLockScreen()
	if !strings.Contains(output, "Set Passphrase") {
		t.Fatalf("expected lock screen to prompt for passphrase setup")
	}
}

func TestRenderJournalPaneAnalyticsNoWorkspaces(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.showAnalytics = true
	m.workspaces = nil
	output := m.renderJournalPane()
	if !strings.Contains(output, "(no workspaces)") {
		t.Fatalf("expected analytics pane to show no workspaces message")
	}
}

func TestRenderJournalPaneAnalyticsData(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.showAnalytics = true

	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	var sprintID int64
	for _, s := range m.sprints {
		if s.SprintNumber > 0 {
			sprintID = s.ID
			break
		}
	}
	if sprintID == 0 {
		t.Fatalf("expected sprint id")
	}
	if err := m.db.AddGoal(m.ctx, wsID, "Analytics Goal", sprintID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	output := m.renderJournalPane()
	if !strings.Contains(output, "Total:") {
		t.Fatalf("expected analytics totals")
	}
}

func TestRenderJournalPaneChangePassphrase(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.security.changingPassphrase = true
	m.security.passphraseStage = 1
	output := m.renderJournalPane()
	if !strings.Contains(output, "Change Passphrase") {
		t.Fatalf("expected change passphrase header")
	}
	if !strings.Contains(output, "New passphrase") {
		t.Fatalf("expected new passphrase label")
	}
}
