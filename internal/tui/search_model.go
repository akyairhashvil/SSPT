package tui

import (
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/charmbracelet/bubbles/textinput"
)

type SearchModel struct {
	Active      bool
	Input       textinput.Model
	Results     []models.Goal
	Cursor      int
	ArchiveOnly bool
}

func NewSearchModel(input textinput.Model) SearchModel {
	return SearchModel{
		Input: input,
	}
}
