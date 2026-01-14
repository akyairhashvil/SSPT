package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmDependencies() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.DependencyState()
	if !ok {
		return m, nil, false
	}
	var deps []int64
	for id, selected := range state.Selected {
		if selected {
			deps = append(deps, id)
		}
	}
	if state.GoalID > 0 {
		if err := m.db.SetGoalDependencies(m.ctx, state.GoalID, deps); err != nil {
			m.setStatusError(fmt.Sprintf("Error saving dependencies: %v", err))
		} else {
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
	}
	m.modal.Close()
	return m, nil, true
}

func (m DashboardModel) handleModalInputDependencies(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.DependencyState()
	if !ok {
		return m, nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if state.Cursor > 0 {
				state.Cursor--
			}
			return m, nil, true
		case "down", "j":
			if state.Cursor < len(state.Options)-1 {
				state.Cursor++
			}
			return m, nil, true
		case " ":
			if len(state.Options) > 0 && state.Cursor < len(state.Options) {
				id := state.Options[state.Cursor].ID
				state.Selected[id] = !state.Selected[id]
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
