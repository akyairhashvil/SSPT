package tui

import "github.com/akyairhashvil/SSPT/internal/models"

func (m DashboardModel) handleTabFocus(key string) (DashboardModel, bool) {
	switch key {
	case "tab", "right", "l":
		nextIdx := -1
		for i := m.view.focusedColIdx + 1; i < len(m.sprints); i++ {
			if m.sprints[i].Status != models.StatusCompleted || i < 2 {
				nextIdx = i
				break
			}
		}
		if nextIdx != -1 {
			m.view.focusedColIdx, m.view.focusedGoalIdx = nextIdx, 0
			if m.view.focusedColIdx >= 2 {
				m.view.colScrollOffset++
			}
		}
		return m, true
	case "shift+tab", "left", "h":
		if m.view.focusedColIdx > 0 {
			m.view.focusedColIdx--
			m.view.focusedGoalIdx = 0
			if m.view.colScrollOffset > 0 {
				m.view.colScrollOffset--
			}
		}
		return m, true
	}
	return m, false
}

func (m DashboardModel) handleArrowKeys(key string) (DashboardModel, bool) {
	switch key {
	case "up", "k":
		if m.view.focusedGoalIdx > 0 {
			m.view.focusedGoalIdx--
		}
		return m, true
	case "down", "j":
		if m.validSprintIndex(m.view.focusedColIdx) && m.view.focusedGoalIdx < len(m.sprints[m.view.focusedColIdx].Goals)-1 {
			m.view.focusedGoalIdx++
		}
		return m, true
	}
	return m, false
}

func (m DashboardModel) handleScrolling(key string) (DashboardModel, bool) {
	if key != "G" {
		return m, false
	}
	m.showAnalytics = !m.showAnalytics
	m.search.Active = false
	if m.modal.Is(ModalJournaling) {
		m.modal.Close()
	}
	return m, true
}
