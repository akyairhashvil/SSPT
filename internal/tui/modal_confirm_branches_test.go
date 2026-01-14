package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func setupTwoGoalsInSprint(t *testing.T) (DashboardModel, int64, int64, int64, int) {
	t.Helper()
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	dayID := m.day.ID
	sprints, err := m.db.GetSprints(m.ctx, dayID, wsID)
	if err != nil || len(sprints) == 0 {
		t.Fatalf("GetSprints failed: %v", err)
	}
	if err := m.db.AddGoal(m.ctx, wsID, "Goal A", sprints[0].ID); err != nil {
		t.Fatalf("AddGoal A failed: %v", err)
	}
	idA, err := m.db.GetLastGoalID(m.ctx)
	if err != nil {
		t.Fatalf("GetLastGoalID failed: %v", err)
	}
	if err := m.db.AddGoal(m.ctx, wsID, "Goal B", sprints[0].ID); err != nil {
		t.Fatalf("AddGoal B failed: %v", err)
	}
	idB, err := m.db.GetLastGoalID(m.ctx)
	if err != nil {
		t.Fatalf("GetLastGoalID failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(dayID)
	targetIdx := -1
	for i, s := range m.sprints {
		if s.SprintNumber > 0 {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		t.Fatalf("expected sprint index")
	}
	return m, idA, idB, sprints[0].ID, targetIdx
}

func TestHandleModalConfirmTagging(t *testing.T) {
	m, goalID, _, _, sprintIdx := setupTwoGoalsInSprint(t)
	m.view.focusedColIdx = sprintIdx
	m.modal.Open(&TaggingState{
		GoalID:   goalID,
		Selected: map[string]bool{"urgent": true},
	})
	m.inputs.tagInput.SetValue("custom")

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.modal.Is(ModalTagging) {
		t.Fatalf("expected tagging modal closed")
	}
	goals, err := m.db.GetGoalsForSprint(m.ctx, m.sprints[sprintIdx].ID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	found := false
	for _, g := range goals {
		if g.ID == goalID && g.Tags != nil {
			tags := util.JSONToTags(*g.Tags)
			if !containsTag(tags, "urgent") || !containsTag(tags, "custom") {
				t.Fatalf("expected tags saved, got %#v", tags)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tagged goal")
	}
}

func TestHandleModalConfirmThemePicking(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.themeNames = []string{"default", "alt"}
	m.modal.Open(&ThemeState{Cursor: 1})

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.modal.Is(ModalTheme) {
		t.Fatalf("expected theme modal closed")
	}
	if m.workspaces[m.activeWorkspaceIdx].Theme != "alt" {
		t.Fatalf("expected theme updated")
	}
}

func TestHandleModalConfirmDepPicking(t *testing.T) {
	m, goalA, goalB, _, _ := setupTwoGoalsInSprint(t)
	m.modal.Open(&DependencyState{
		GoalID:   goalA,
		Selected: map[int64]bool{goalB: true},
	})

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled")
	}
	deps, err := m.db.GetGoalDependencies(m.ctx, goalA)
	if err != nil {
		t.Fatalf("GetGoalDependencies failed: %v", err)
	}
	if !deps[goalB] {
		t.Fatalf("expected dependency saved")
	}
}

func TestHandleModalConfirmRecurrenceWeekly(t *testing.T) {
	m, goalID, _, sprintID, _ := setupTwoGoalsInSprint(t)
	m.modal.Open(&RecurrenceState{
		GoalID:         goalID,
		Mode:           "weekly",
		Selected:       map[string]bool{"mon": true},
		WeekdayOptions: []string{"mon"},
	})

	m, _, handled := m.handleModalConfirm(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.modal.Is(ModalRecurrence) {
		t.Fatalf("expected recurrence modal closed")
	}
	goals, err := m.db.GetGoalsForSprint(m.ctx, sprintID)
	if err != nil {
		t.Fatalf("GetGoalsForSprint failed: %v", err)
	}
	found := false
	for _, g := range goals {
		if g.ID == goalID && g.RecurrenceRule != nil {
			if *g.RecurrenceRule != "weekly:mon" {
				t.Fatalf("expected recurrence rule set, got %q", *g.RecurrenceRule)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("expected goal with recurrence")
	}
}
