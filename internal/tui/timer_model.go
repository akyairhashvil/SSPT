package tui

import (
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

type TimerManager struct {
	ActiveSprint *SprintView
	ActiveTask   *models.Goal
	BreakActive  bool
	BreakStart   time.Time
}

func NewTimerManager() TimerManager {
	return TimerManager{}
}
