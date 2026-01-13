package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleWorkspaceSwitch(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "w" {
		return m, nil, false
	}
	if len(m.workspaces) > 1 {
		m.activeWorkspaceIdx = (m.activeWorkspaceIdx + 1) % len(m.workspaces)
		m.refreshData(m.day.ID)
		m.focusedColIdx = 1
	} else {
		m.Message = "No other workspaces. Use Shift+W to create a new one."
	}
	return m, nil, true
}

func (m DashboardModel) handleWorkspaceCreate(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "W" {
		return m, nil, false
	}
	m.creatingWorkspace = true
	m.textInput.Focus()
	return m, nil, true
}

func (m DashboardModel) handleWorkspaceVisibility(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "b":
		if len(m.workspaces) > 0 {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			activeWS.ShowBacklog = !activeWS.ShowBacklog
			if err := m.db.UpdateWorkspacePaneVisibility(m.ctx, activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating workspace view: %v", err))
			}
			m.workspaces[m.activeWorkspaceIdx].ShowBacklog = activeWS.ShowBacklog
			m.refreshData(m.day.ID)
		}
		return m, nil, true
	case "c":
		if len(m.workspaces) > 0 {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			activeWS.ShowCompleted = !activeWS.ShowCompleted
			if err := m.db.UpdateWorkspacePaneVisibility(m.ctx, activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating workspace view: %v", err))
			}
			m.workspaces[m.activeWorkspaceIdx].ShowCompleted = activeWS.ShowCompleted
			m.refreshData(m.day.ID)
		}
		return m, nil, true
	case "a":
		if len(m.workspaces) > 0 {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			activeWS.ShowArchived = !activeWS.ShowArchived
			if err := m.db.UpdateWorkspacePaneVisibility(m.ctx, activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating workspace view: %v", err))
			}
			m.workspaces[m.activeWorkspaceIdx].ShowArchived = activeWS.ShowArchived
			m.refreshData(m.day.ID)
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleWorkspaceViewMode(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "v" {
		return m, nil, false
	}
	m.viewMode = (m.viewMode + 1) % 3
	if len(m.workspaces) > 0 {
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		if err := m.db.UpdateWorkspaceViewMode(m.ctx, activeWS.ID, m.viewMode); err != nil {
			m.setStatusError(fmt.Sprintf("Error updating view mode: %v", err))
		}
		m.workspaces[m.activeWorkspaceIdx].ViewMode = m.viewMode
	}
	if m.viewMode == ViewModeFocused && m.sprints[m.focusedColIdx].SprintNumber == -1 {
		m.focusedColIdx = 1
	} else if m.viewMode == ViewModeMinimal && m.sprints[m.focusedColIdx].SprintNumber <= 0 {
		m.focusedColIdx = 2
		if len(m.sprints) <= 2 {
			m.focusedColIdx = 0
		}
	}
	return m, nil, true
}

func (m DashboardModel) handleWorkspaceTheme(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "Y" {
		return m, nil, false
	}
	if len(m.workspaces) > 0 {
		m.themePicking = true
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		for i, t := range m.themeNames {
			if t == activeWS.Theme {
				m.themeCursor = i
				break
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleWorkspaceSeedImport(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "I" {
		return m, nil, false
	}
	if len(m.workspaces) > 0 {
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		seedPath, err := EnsureSeedFile()
		if err != nil {
			m.Message = fmt.Sprintf("Seed file error: %v", err)
			return m, nil, true
		}
		count, _, backlogFallback, err := ImportSeed(m.ctx, m.db, seedPath, activeWS.ID, m.day.ID)
		if err != nil {
			m.Message = fmt.Sprintf("Seed import failed: %v", err)
			return m, nil, true
		}
		if count == 0 {
			if backlogFallback > 0 {
				m.Message = "Seed import complete. Some sprint tasks moved to backlog (max 8)."
			} else {
				m.Message = "Seed already imported."
			}
		} else {
			if backlogFallback > 0 {
				m.Message = fmt.Sprintf("Imported %d tasks (some moved to backlog: max 8).", count)
			} else {
				m.Message = fmt.Sprintf("Imported %d tasks from seed.", count)
			}
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	return m, nil, true
}

func (m DashboardModel) handleWorkspaceSprintCount(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "+":
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		if err := m.db.AppendSprint(m.ctx, m.day.ID, activeWS.ID); err != nil {
			m.Message = fmt.Sprintf("Add sprint failed: %v", err)
			return m, nil, true
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
		return m, nil, true
	case "-":
		if len(m.workspaces) > 0 {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			if err := m.db.RemoveLastSprint(m.ctx, m.day.ID, activeWS.ID); err != nil {
				m.Message = fmt.Sprintf("Remove sprint failed: %v", err)
			} else {
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
				if m.focusedColIdx >= len(m.sprints) {
					m.focusedColIdx = len(m.sprints) - 1
				}
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleWorkspaceReport(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "ctrl+r" {
		return m, nil, false
	}
	activeWS := m.workspaces[m.activeWorkspaceIdx]
	path, err := GeneratePDFReport(m.ctx, m.db, m.day.ID, activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Report failed: %v", err))
		return m, nil, true
	}
	fmt.Printf("\nPDF Report generated: %s\n", path)
	return m, tea.Quit, true
}
