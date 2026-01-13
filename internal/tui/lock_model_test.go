package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestNewLockModel(t *testing.T) {
	input := textinput.New()
	lock := NewLockModel(5*time.Minute, input)
	if lock.AutoLockAfter != 5*time.Minute {
		t.Fatalf("expected AutoLockAfter 5m, got %s", lock.AutoLockAfter)
	}
	if lock.LastInput.IsZero() {
		t.Fatalf("expected LastInput to be set")
	}
}
