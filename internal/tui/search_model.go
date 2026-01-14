package tui

import (
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/charmbracelet/bubbles/textinput"
)

type SearchManager struct {
	Active      bool
	Input       textinput.Model
	Results     []models.Goal
	Cursor      int
	ArchiveOnly bool
}

func NewSearchManager(input textinput.Model) SearchManager {
	return SearchManager{
		Input: input,
	}
}
