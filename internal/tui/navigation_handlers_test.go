package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestHandleTabFocus(t *testing.T) {
	m := setupTestDashboard(t)
	m.sprints = []SprintView{
		{Sprint: models.Sprint{ID: 1, SprintNumber: 1, Status: models.StatusPending}},
		{Sprint: models.Sprint{ID: 2, SprintNumber: 2, Status: models.StatusPending}},
	}
	m.view.focusedColIdx = 0
	startIdx := m.view.focusedColIdx
	next, handled := m.handleTabFocus("tab")
	if !handled {
		t.Fatalf("expected tab to be handled")
	}
	if next.view.focusedColIdx == startIdx {
		t.Fatalf("expected focused column to advance")
	}
}

func TestHandleArrowKeys(t *testing.T) {
	m := setupTestDashboard(t)
	wsID := m.workspaces[m.activeWorkspaceIdx].ID
	if err := m.db.AddGoal(m.ctx, wsID, "Nav Goal", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	m.invalidateGoalCache()
	m.refreshData(m.day.ID)

	found := false
	for i, s := range m.sprints {
		if len(s.Goals) > 0 {
			m.view.focusedColIdx = i
			m.view.focusedGoalIdx = 0
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a goal to navigate")
	}

	next, handled := m.handleArrowKeys("down")
	if !handled {
		t.Fatalf("expected down to be handled")
	}
	if next.view.focusedGoalIdx < 0 {
		t.Fatalf("expected focusedGoalIdx to be non-negative")
	}
}

func TestHandleScrolling(t *testing.T) {
	m := setupTestDashboard(t)
	m.showAnalytics = false
	m.search.Active = true
	m.modal.Open(&JournalState{})

	next, handled := m.handleScrolling("G")
	if !handled {
		t.Fatalf("expected scrolling toggle to be handled")
	}
	if !next.showAnalytics {
		t.Fatalf("expected analytics to be enabled")
	}
	if next.search.Active || next.modal.Is(ModalJournaling) {
		t.Fatalf("expected search and journaling to be disabled")
	}
}
