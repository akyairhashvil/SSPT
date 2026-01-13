package tui

import "github.com/akyairhashvil/SSPT/internal/models"

// GoalView wraps a goal with UI-only state.
type GoalView struct {
	models.Goal
	Subtasks []GoalView
	Expanded bool
	Level    int
	Blocked  bool
}

// SprintView wraps a sprint with UI-only goal state.
type SprintView struct {
	models.Sprint
	Goals []GoalView
}
