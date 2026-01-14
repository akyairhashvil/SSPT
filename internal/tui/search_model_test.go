package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestNewSearchManager(t *testing.T) {
	input := textinput.New()
	m := NewSearchManager(input)
	if m.Active {
		t.Fatalf("expected Active to be false")
	}
	if m.Input.Value() != "" {
		t.Fatalf("expected input to be empty")
	}
}
