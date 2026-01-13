package tui

import (
	"testing"
	"time"
)

func TestHandleSprintStartAndPause(t *testing.T) {
	m := setupTestDashboard(t)
	var sprintIdx int
	for i, s := range m.sprints {
		if s.SprintNumber > 0 {
			sprintIdx = i
			break
		}
	}
	m.view.focusedColIdx = sprintIdx

	next, _, handled := m.handleSprintStart("s")
	if !handled {
		t.Fatalf("expected start handler to handle")
	}
	if next.timer.ActiveSprint == nil {
		t.Fatalf("expected active sprint after start")
	}

	started := time.Now().Add(-5 * time.Second)
	next.timer.ActiveSprint.StartTime = &started
	next.view.focusedColIdx = sprintIdx
	paused, _, handled := next.handleSprintPause("s")
	if !handled {
		t.Fatalf("expected pause handler to handle")
	}
	if paused.timer.ActiveSprint != nil {
		t.Fatalf("expected active sprint to be cleared after pause")
	}
}

func TestHandleSprintCompletion(t *testing.T) {
	m := setupTestDashboard(t)
	var sprintIdx int
	for i, s := range m.sprints {
		if s.SprintNumber > 0 {
			sprintIdx = i
			break
		}
	}
	m.timer.ActiveSprint = &m.sprints[sprintIdx]

	next, handled := m.handleSprintCompletion()
	if !handled {
		t.Fatalf("expected completion handler to handle")
	}
	if next.timer.ActiveSprint != nil {
		t.Fatalf("expected active sprint to be cleared")
	}
	if !next.timer.BreakActive {
		t.Fatalf("expected break to start after completion")
	}
}
