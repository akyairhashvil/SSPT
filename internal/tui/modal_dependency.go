package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmDependencies() (DashboardModel, tea.Cmd, bool) {
	if !m.modal.depPicking {
		return m, nil, false
	}
	var deps []int64
	for id, selected := range m.modal.depSelected {
		if selected {
			deps = append(deps, id)
		}
	}
	if m.modal.editingGoalID > 0 {
		if err := m.db.SetGoalDependencies(m.ctx, m.modal.editingGoalID, deps); err != nil {
			m.setStatusError(fmt.Sprintf("Error saving dependencies: %v", err))
		} else {
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
	}
	m.modal.depPicking, m.modal.editingGoalID = false, 0
	m.modal.depSelected = make(map[int64]bool)
	return m, nil, true
}

func (m DashboardModel) handleModalInputDependencies(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	if !m.modal.depPicking {
		return m, nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.modal.depCursor > 0 {
				m.modal.depCursor--
			}
			return m, nil, true
		case "down", "j":
			if m.modal.depCursor < len(m.modal.depOptions)-1 {
				m.modal.depCursor++
			}
			return m, nil, true
		case " ":
			if len(m.modal.depOptions) > 0 && m.modal.depCursor < len(m.modal.depOptions) {
				id := m.modal.depOptions[m.modal.depCursor].ID
				m.modal.depSelected[id] = !m.modal.depSelected[id]
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
