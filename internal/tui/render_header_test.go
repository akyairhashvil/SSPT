package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestRenderHeaderBreakActive(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.timer.BreakActive = true
	m.timer.BreakStart = time.Now().Add(-time.Minute)
	out := m.renderHeader()
	if !strings.Contains(out, "BREAK TIME") {
		t.Fatalf("expected break header")
	}
}

func TestRenderHeaderActiveSprint(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	idx := -1
	for i, s := range m.sprints {
		if s.SprintNumber > 0 {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatalf("expected sprint")
	}
	now := time.Now().Add(-time.Minute)
	m.sprints[idx].StartTime = &now
	m.timer.ActiveSprint = &m.sprints[idx]
	out := m.renderHeader()
	if !strings.Contains(out, "ACTIVE SPRINT") {
		t.Fatalf("expected active sprint header")
	}
}

func TestRenderHeaderPausedSprint(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	idx := -1
	for i, s := range m.sprints {
		if s.SprintNumber > 0 {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatalf("expected sprint")
	}
	m.sprints[idx].Status = models.StatusPaused
	m.view.focusedColIdx = idx
	out := m.renderHeader()
	if !strings.Contains(out, "PAUSED SPRINT") {
		t.Fatalf("expected paused sprint header")
	}
}
