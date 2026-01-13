package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestRenderBoardIncludesGoal(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80

	now := time.Now()
	tagJSON := `["urgent"]`
	recurrence := "daily"
	goal := GoalView{
		Goal: models.Goal{
			ID:             1,
			Description:    "Board Goal",
			Status:         models.GoalStatusPending,
			Priority:       2,
			Tags:           &tagJSON,
			RecurrenceRule: &recurrence,
			TaskActive:     true,
			TaskStartedAt:  &now,
		},
	}
	m.sprints = []SprintView{
		{Sprint: models.Sprint{ID: 1, SprintNumber: 1, Status: models.StatusPending}, Goals: []GoalView{goal}},
	}
	m.view.focusedColIdx = 0
	m.view.focusedGoalIdx = 0

	layout := m.buildBoardLayout()
	output := m.renderBoard(8, layout)
	if output == "" {
		t.Fatalf("expected board output")
	}
	if !strings.Contains(output, "Board Goal") {
		t.Fatalf("expected goal description in board output")
	}
}
