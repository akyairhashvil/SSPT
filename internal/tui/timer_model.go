package tui

import (
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

type TimerModel struct {
	ActiveSprint *SprintView
	ActiveTask   *models.Goal
	BreakActive  bool
	BreakStart   time.Time
}

func NewTimerModel() TimerModel {
	return TimerModel{}
}
