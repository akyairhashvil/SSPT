package tui

import (
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/config"
)

func TestHandleTickAutoLock(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.Locked = false
	m.security.lock.PassphraseHash = "hash"
	m.security.lock.LastInput = time.Now().Add(-config.AutoLockAfter - time.Second)

	next, _ := m.handleTick(TickMsg{})
	if !next.security.lock.Locked {
		t.Fatalf("expected session to auto-lock")
	}
	if next.security.lock.Message == "" {
		t.Fatalf("expected lock message to be set")
	}
}

func TestHandleTickBreakEnds(t *testing.T) {
	m := setupTestDashboard(t)
	m.timer.BreakActive = true
	m.timer.BreakStart = time.Now().Add(-config.BreakDuration - time.Second)

	next, _ := m.handleTick(TickMsg{})
	if next.timer.BreakActive {
		t.Fatalf("expected break to end after duration")
	}
}
