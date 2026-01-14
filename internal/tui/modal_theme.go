package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmTheme() (DashboardModel, tea.Cmd, bool) {
	if !m.modal.themePicking {
		return m, nil, false
	}
	if len(m.modal.themeNames) > 0 && m.modal.themeCursor < len(m.modal.themeNames) {
		name := m.modal.themeNames[m.modal.themeCursor]
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		if err := m.db.UpdateWorkspaceTheme(m.ctx, activeWS.ID, name); err != nil {
			m.setStatusError(fmt.Sprintf("Error updating workspace theme: %v", err))
		} else {
			m.workspaces[m.activeWorkspaceIdx].Theme = name
			m.theme = ResolveTheme(name)
		}
	}
	m.modal.themePicking = false
	return m, nil, true
}

func (m DashboardModel) handleModalInputTheme(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	if !m.modal.themePicking {
		return m, nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.modal.themeCursor > 0 {
				m.modal.themeCursor--
			}
			return m, nil, true
		case "down", "j":
			if m.modal.themeCursor < len(m.modal.themeNames)-1 {
				m.modal.themeCursor++
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
