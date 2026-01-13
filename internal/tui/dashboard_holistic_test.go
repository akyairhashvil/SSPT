package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func findSprintIndex(m DashboardModel, sprintNumber int) int {
	for i := range m.sprints {
		if m.sprints[i].SprintNumber == sprintNumber {
			return i
		}
	}
	return -1
}

func TestDashboardSearchFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := db.AddGoal(wsID, "Search Target", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.refreshData(m.day.ID)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m, _ = model.(DashboardModel)
	if !m.search.Active {
		t.Fatalf("expected search to be active")
	}

	for _, r := range "Search" {
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m, _ = model.(DashboardModel)
	}
	if len(m.search.Results) == 0 {
		t.Fatalf("expected search results")
	}
}

func TestDashboardTagFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := db.AddGoal(wsID, "Tag Target", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	backlogIdx := findSprintIndex(m, 0)
	if backlogIdx == -1 {
		t.Fatalf("expected backlog column")
	}
	m.focusedColIdx = backlogIdx

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m, _ = model.(DashboardModel)
	if !m.tagging {
		t.Fatalf("expected tagging to be true")
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = model.(DashboardModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = model.(DashboardModel)

	var tags *string
	if err := db.DB.QueryRow("SELECT tags FROM goals WHERE description = ?", "Tag Target").Scan(&tags); err != nil {
		t.Fatalf("query tags failed: %v", err)
	}
	if tags == nil || len(util.JSONToTags(*tags)) == 0 {
		t.Fatalf("expected tags to be set")
	}
}

func TestDashboardDependencyFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := db.AddGoal(wsID, "Dep Target", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	if err := db.AddGoal(wsID, "Dep Source", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	backlogIdx := findSprintIndex(m, 0)
	if backlogIdx == -1 {
		t.Fatalf("expected backlog column")
	}
	m.focusedColIdx = backlogIdx
	m.focusedGoalIdx = 0

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	m, _ = model.(DashboardModel)
	if !m.depPicking {
		t.Fatalf("expected dep picking to be true")
	}
	if len(m.depOptions) == 0 {
		t.Fatalf("expected dependency options")
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m, _ = model.(DashboardModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = model.(DashboardModel)

	var targetID int64
	if err := db.DB.QueryRow("SELECT id FROM goals WHERE description = ?", "Dep Target").Scan(&targetID); err != nil {
		t.Fatalf("query target id failed: %v", err)
	}
	deps, err := db.GetGoalDependencies(targetID)
	if err != nil {
		t.Fatalf("GetGoalDependencies failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatalf("expected at least one dependency")
	}
}

func TestDashboardRecurrenceFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := db.AddGoal(wsID, "Recurring Target", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	backlogIdx := findSprintIndex(m, 0)
	if backlogIdx == -1 {
		t.Fatalf("expected backlog column")
	}
	m.focusedColIdx = backlogIdx

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m, _ = model.(DashboardModel)
	if !m.settingRecurrence {
		t.Fatalf("expected recurrence modal to be active")
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = model.(DashboardModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = model.(DashboardModel)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = model.(DashboardModel)

	var rule *string
	if err := db.DB.QueryRow("SELECT recurrence_rule FROM goals WHERE description = ?", "Recurring Target").Scan(&rule); err != nil {
		t.Fatalf("query recurrence failed: %v", err)
	}
	if rule == nil || *rule != "daily" {
		t.Fatalf("expected recurrence rule daily, got %v", rule)
	}
}

func TestDashboardJournalFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m, _ = model.(DashboardModel)
	if !m.journaling {
		t.Fatalf("expected journaling to be true")
	}

	m.journalInput.SetValue("Journal entry")
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = model.(DashboardModel)
	if m.journaling {
		t.Fatalf("expected journaling to end")
	}

	entries, err := db.GetJournalEntries(m.day.ID, m.workspaces[m.activeWorkspaceIdx].ID)
	if err != nil {
		t.Fatalf("GetJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 journal entry, got %d", len(entries))
	}
}

func TestDashboardStartSprintFlow(t *testing.T) {
	m, db := setupIntegrationDashboard(t)

	sprintIdx := findSprintIndex(m, 1)
	if sprintIdx == -1 {
		t.Fatalf("expected sprint #1")
	}
	m.focusedColIdx = sprintIdx
	m.timer.ActiveSprint = nil

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = model.(DashboardModel)

	sprints, err := db.GetSprints(m.day.ID, m.workspaces[m.activeWorkspaceIdx].ID)
	if err != nil {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if len(sprints) == 0 {
		t.Fatalf("expected at least one sprint")
	}
	if sprints[0].Status != "active" {
		t.Fatalf("expected sprint to be active, got %q", sprints[0].Status)
	}
}
