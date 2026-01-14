package tui

import (
	"testing"
)

func setupGoalInSprint(t *testing.T) (DashboardModel, int64, int) {
	t.Helper()
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	dayID := m.day.ID
	sprints, err := m.db.GetSprints(m.ctx, dayID, wsID)
	if err != nil || len(sprints) == 0 {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if err := m.db.AddGoal(m.ctx, wsID, "Goal", sprints[0].ID); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(dayID)
	targetIdx := -1
	var goalID int64
	for i, s := range m.sprints {
		if s.SprintNumber > 0 && len(s.Goals) > 0 {
			targetIdx = i
			goalID = s.Goals[0].ID
			break
		}
	}
	if targetIdx == -1 {
		t.Fatalf("expected sprint with goals")
	}
	return m, goalID, targetIdx
}

func TestHandleGoalExpandCollapse(t *testing.T) {
	m, goalID, sprintIdx := setupGoalInSprint(t)
	m.view.focusedColIdx = sprintIdx
	m.view.focusedGoalIdx = 0

	m, _, handled := m.handleGoalExpandCollapse("z")
	if !handled {
		t.Fatalf("expected handled")
	}
	if !m.view.expandedState[goalID] {
		t.Fatalf("expected expanded state true")
	}
	m, _, _ = m.handleGoalExpandCollapse("z")
	if m.view.expandedState[goalID] {
		t.Fatalf("expected expanded state false")
	}
}

func TestHandleGoalTaskTimerStartPause(t *testing.T) {
	m, _, sprintIdx := setupGoalInSprint(t)
	m.view.focusedColIdx = sprintIdx
	m.view.focusedGoalIdx = 0

	m, _, handled := m.handleGoalTaskTimer("T")
	if !handled {
		t.Fatalf("expected handled for task timer")
	}
	if m.Message == "" {
		t.Fatalf("expected status message")
	}

	m, _, _ = m.handleGoalTaskTimer("T")
	if m.Message == "" {
		t.Fatalf("expected status message on pause")
	}
}

func TestHandleGoalRecurrencePickerWeekly(t *testing.T) {
	m, goalID, sprintIdx := setupGoalInSprint(t)
	if err := m.db.UpdateGoalRecurrence(m.ctx, goalID, "weekly:mon,tue"); err != nil {
		t.Fatalf("UpdateGoalRecurrence failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)
	m.view.focusedColIdx = sprintIdx
	m.view.focusedGoalIdx = 0

	m, _, handled := m.handleGoalRecurrencePicker("R")
	if !handled {
		t.Fatalf("expected handled")
	}
	if !m.modal.Is(ModalRecurrence) {
		t.Fatalf("expected recurrence modal")
	}
	state, ok := m.modal.RecurrenceState()
	if !ok {
		t.Fatalf("expected recurrence state")
	}
	if state.Mode != "weekly" {
		t.Fatalf("expected weekly mode, got %q", state.Mode)
	}
	if !state.Selected["mon"] {
		t.Fatalf("expected monday selected")
	}
}

func TestHandleGoalArchiveUnarchive(t *testing.T) {
	m, _, sprintIdx := setupGoalInSprint(t)
	m.view.focusedColIdx = sprintIdx
	m.view.focusedGoalIdx = 0

	m, _, handled := m.handleGoalArchive("A")
	if !handled {
		t.Fatalf("expected handled for archive")
	}

	active := m.workspaces[m.activeWorkspaceIdx]
	active.ShowArchived = true
	if err := m.db.UpdateWorkspacePaneVisibility(m.ctx, active.ID, active.ShowBacklog, active.ShowCompleted, active.ShowArchived); err != nil {
		t.Fatalf("UpdateWorkspacePaneVisibility failed: %v", err)
	}
	m.workspaces[m.activeWorkspaceIdx].ShowArchived = true
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	archivedIdx := -1
	for i, s := range m.sprints {
		if s.SprintNumber == -2 && len(s.Goals) > 0 {
			archivedIdx = i
			break
		}
	}
	if archivedIdx == -1 {
		t.Fatalf("expected archived goal column")
	}
	m.view.focusedColIdx = archivedIdx
	m.view.focusedGoalIdx = 0

	m, _, handled = m.handleGoalArchive("u")
	if !handled {
		t.Fatalf("expected handled for unarchive")
	}
}
