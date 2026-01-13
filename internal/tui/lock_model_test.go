package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
)

func TestNewLockModel(t *testing.T) {
	input := textinput.New()
	lock := NewLockModel(config.AutoLockAfter, input)
	if lock.AutoLockAfter != config.AutoLockAfter {
		t.Fatalf("expected AutoLockAfter %s, got %s", config.AutoLockAfter, lock.AutoLockAfter)
	}
	if lock.LastInput.IsZero() {
		t.Fatalf("expected LastInput to be set")
	}
}
