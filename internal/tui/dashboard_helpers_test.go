package tui

import (
	"testing"
	"time"
)

func TestParseYear(t *testing.T) {
	if got := parseYear("2024-01-02"); got != 2024 {
		t.Fatalf("expected year 2024, got %d", got)
	}
	if got := parseYear("invalid"); got != time.Now().Year() {
		t.Fatalf("expected current year fallback, got %d", got)
	}
}

func TestMonthDayLimit(t *testing.T) {
	if got := monthDayLimit("feb", 2024); got != 29 {
		t.Fatalf("expected 29 for leap year, got %d", got)
	}
	if got := monthDayLimit("feb", 2023); got != 28 {
		t.Fatalf("expected 28 for non-leap year, got %d", got)
	}
	if got := monthDayLimit("apr", 2023); got != 30 {
		t.Fatalf("expected 30 for April, got %d", got)
	}
	if got := monthDayLimit("jan", 2023); got != 31 {
		t.Fatalf("expected 31 for January, got %d", got)
	}
}

func TestMonthlySelectionHelpers(t *testing.T) {
	m := setupTestDashboard(t)
	state := &RecurrenceState{
		MonthOptions: []string{"jan", "feb", "mar"},
		Selected: map[string]bool{
			"jan":    true,
			"mar":    true,
			"day:5":  true,
			"day:31": true,
		},
		DayCursor: 0,
	}
	m.day.Date = "2024-02-10"

	months := m.selectedMonths(state)
	if len(months) != 2 {
		t.Fatalf("expected 2 selected months, got %d", len(months))
	}
	maxDay := m.monthlyMaxDay(state)
	if maxDay != 31 {
		t.Fatalf("expected max day 31, got %d", maxDay)
	}

	m.pruneMonthlyDays(state, 30)
	if state.Selected["day:31"] {
		t.Fatalf("expected day:31 to be pruned")
	}
	if state.DayCursor != 0 {
		t.Fatalf("expected recurrenceDayCursor to reset when out of bounds")
	}
}

func TestDashboardCoreHelpers(t *testing.T) {
	m := setupTestDashboard(t)
	if m.currentSprint() == nil {
		t.Fatalf("expected current sprint to be available")
	}
	m.view.focusedColIdx = 999
	if m.currentSprint() != nil {
		t.Fatalf("expected nil for invalid sprint index")
	}

	m.view.focusedColIdx = 0
	m.security.lock.Locked = false
	if !m.canModifyGoals() {
		t.Fatalf("expected canModifyGoals true when unlocked and idle")
	}
	m.security.lock.Locked = true
	if m.canModifyGoals() {
		t.Fatalf("expected canModifyGoals false when locked")
	}

	m.setStatusError("boom")
	if m.statusMessage != "boom" || !m.statusIsError {
		t.Fatalf("expected status error set")
	}
	m.clearStatus()
	if m.statusMessage != "" || m.statusIsError {
		t.Fatalf("expected status cleared")
	}

	if m.hasActiveSprint() {
		t.Fatalf("expected no active sprint by default")
	}
	m.timer.ActiveSprint = &m.sprints[0]
	if !m.hasActiveSprint() {
		t.Fatalf("expected active sprint")
	}
}
