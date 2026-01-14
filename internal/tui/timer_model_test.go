package tui

import "testing"

func TestNewTimerManagerDefaults(t *testing.T) {
	m := NewTimerManager()
	if m.ActiveSprint != nil {
		t.Fatalf("expected ActiveSprint to be nil")
	}
	if m.ActiveTask != nil {
		t.Fatalf("expected ActiveTask to be nil")
	}
	if m.BreakActive {
		t.Fatalf("expected BreakActive to be false")
	}
}
