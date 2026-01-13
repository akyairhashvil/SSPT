package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestNewSearchModel(t *testing.T) {
	input := textinput.New()
	m := NewSearchModel(input)
	if m.Active {
		t.Fatalf("expected Active to be false")
	}
	if m.Input.Value() != "" {
		t.Fatalf("expected input to be empty")
	}
}
