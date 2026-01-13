package tui

import (
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/config"
)

func TestHandleTickAutoLockExtended(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.PassphraseHash = "hash"
	m.security.lock.LastInput = time.Now().Add(-config.AutoLockAfter - time.Second)
	m.security.lock.Locked = false

	next, _ := m.handleTick(TickMsg{})
	if !next.security.lock.Locked {
		t.Fatalf("expected locked after idle")
	}
	if next.security.lock.Message == "" {
		t.Fatalf("expected lock message")
	}
}

func TestHandleTickBreakEndsExtended(t *testing.T) {
	m := setupTestDashboard(t)
	m.timer.BreakActive = true
	m.timer.BreakStart = time.Now().Add(-config.BreakDuration - time.Second)

	next, _ := m.handleTick(TickMsg{})
	if next.timer.BreakActive {
		t.Fatalf("expected break to end")
	}
}

func TestHandleTickActiveSprintProgress(t *testing.T) {
	m := setupTestDashboard(t)
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
	start := time.Now().Add(-time.Minute)
	m.sprints[idx].StartTime = &start
	m.timer.ActiveSprint = &m.sprints[idx]

	next, cmd := m.handleTick(TickMsg{})
	if next.timer.ActiveSprint == nil {
		t.Fatalf("expected active sprint retained")
	}
	if cmd == nil {
		t.Fatalf("expected tick cmd")
	}
}
