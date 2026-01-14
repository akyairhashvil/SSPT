package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmTheme() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.ThemeState()
	if !ok {
		return m, nil, false
	}
	if len(m.modal.themeNames) > 0 && state.Cursor < len(m.modal.themeNames) {
		name := m.modal.themeNames[state.Cursor]
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		if err := m.db.UpdateWorkspaceTheme(m.ctx, activeWS.ID, name); err != nil {
			m.setStatusError(fmt.Sprintf("Error updating workspace theme: %v", err))
		} else {
			m.workspaces[m.activeWorkspaceIdx].Theme = name
			m.theme = ResolveTheme(name)
		}
	}
	m.modal.Close()
	return m, nil, true
}

func (m DashboardModel) handleModalInputTheme(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.ThemeState()
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
			if state.Cursor < len(m.modal.themeNames)-1 {
				state.Cursor++
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
