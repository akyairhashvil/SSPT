package tui

import "github.com/akyairhashvil/SSPT/internal/models"

func (m DashboardModel) handleTabFocus(key string) (DashboardModel, bool) {
	switch key {
	case "tab", "right", "l":
		nextIdx := -1
		for i := m.focusedColIdx + 1; i < len(m.sprints); i++ {
			if m.sprints[i].Status != models.StatusCompleted || i < 2 {
				nextIdx = i
				break
			}
		}
		if nextIdx != -1 {
			m.focusedColIdx, m.focusedGoalIdx = nextIdx, 0
			if m.focusedColIdx >= 2 {
				m.colScrollOffset++
			}
		}
		return m, true
	case "shift+tab", "left", "h":
		if m.focusedColIdx > 0 {
			m.focusedColIdx--
			m.focusedGoalIdx = 0
			if m.colScrollOffset > 0 {
				m.colScrollOffset--
			}
		}
		return m, true
	}
	return m, false
}

func (m DashboardModel) handleArrowKeys(key string) (DashboardModel, bool) {
	switch key {
	case "up", "k":
		if m.focusedGoalIdx > 0 {
			m.focusedGoalIdx--
		}
		return m, true
	case "down", "j":
		if m.validSprintIndex(m.focusedColIdx) && m.focusedGoalIdx < len(m.sprints[m.focusedColIdx].Goals)-1 {
			m.focusedGoalIdx++
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
	m.journaling = false
	return m, true
}
