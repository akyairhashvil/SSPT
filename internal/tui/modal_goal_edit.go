package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmDelete() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.GoalDeleteState()
	if !ok {
		return m, nil, false
	}
	if state.GoalID > 0 {
		if err := m.db.DeleteGoal(m.ctx, state.GoalID); err != nil {
			m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
		} else {
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
	}
	m.modal.Close()
	return m, nil, true
}

func (m DashboardModel) handleModalInputConfirmDelete(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.GoalDeleteState()
	if !ok {
		return m, nil, false
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "a":
			if state.GoalID > 0 {
				if err := m.db.ArchiveGoal(m.ctx, state.GoalID); err != nil {
					m.setStatusError(fmt.Sprintf("Error archiving goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
				}
			}
			m.modal.Close()
			return m, nil, true
		case "d", "backspace":
			if state.GoalID > 0 {
				if err := m.db.DeleteGoal(m.ctx, state.GoalID); err != nil {
					m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
				}
			}
			m.modal.Close()
			return m, nil, true
		}
	}
	return m, nil, true
}

func (m DashboardModel) handleModalConfirmJournaling() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.JournalState()
	if !ok {
		return m, nil, false
	}
	text := m.inputs.journalInput.Value()
	if strings.TrimSpace(text) != "" {
		var sID, gID *int64
		if m.timer.ActiveSprint != nil {
			id := m.timer.ActiveSprint.ID
			sID = &id
		}
		if state.GoalID > 0 {
			id := state.GoalID
			gID = &id
		}
		activeWS := m.workspaces[m.activeWorkspaceIdx]
		if err := m.db.AddJournalEntry(m.ctx, m.day.ID, activeWS.ID, sID, gID, text); err != nil {
			m.setStatusError(fmt.Sprintf("Error saving journal entry: %v", err))
		} else {
			m.refreshData(m.day.ID)
		}
	}
	m.modal.Close()
	m.inputs.journalInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalInputJournaling(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	if !m.modal.Is(ModalJournaling) {
		return m, nil, false
	}
	m.inputs.journalInput, cmd = m.inputs.journalInput.Update(msg)
	return m, cmd, true
}

func (m DashboardModel) handleModalConfirmWorkspaceCreate() (DashboardModel, tea.Cmd, bool) {
	if !m.modal.Is(ModalWorkspaceCreate) {
		return m, nil, false
	}
	name := m.inputs.textInput.Value()
	if name != "" {
		newID, err := m.db.CreateWorkspace(m.ctx, name, strings.ToLower(name))
		if err == nil {
			m.modal.Open(&WorkspaceInitState{WorkspaceID: newID})
			m.inputs.textInput.Placeholder = "How many sprints?"
			m.inputs.textInput.Reset()
		} else {
			m.err = err
			m.modal.Close()
		}
	}
	return m, nil, true
}

func (m DashboardModel) handleModalConfirmInitializeSprints() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.WorkspaceInitState()
	if !ok {
		return m, nil, false
	}
	val := m.inputs.textInput.Value()
	if num, err := strconv.Atoi(val); err == nil && num > 0 && num <= 8 {
		if err := m.db.BootstrapDay(m.ctx, state.WorkspaceID, num); err != nil {
			m.setStatusError(fmt.Sprintf("Error creating sprints: %v", err))
		} else if err := m.loadWorkspaces(); err != nil {
			m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
		} else {
			for i, ws := range m.workspaces {
				if ws.ID == state.WorkspaceID {
					m.activeWorkspaceIdx = i
					break
				}
			}
			if dayID := m.db.CheckCurrentDay(m.ctx); dayID > 0 {
				if day, err := m.db.GetDay(m.ctx, dayID); err == nil {
					m.day = day
				}
				m.refreshData(dayID)
			}
		}
	}
	m.modal.Close()
	m.inputs.textInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalConfirmGoalEdit() (DashboardModel, tea.Cmd, bool) {
	text := m.inputs.textInput.Value()
	if state, ok := m.modal.GoalEditState(); ok {
		if text != "" {
			if err := m.db.EditGoal(m.ctx, state.GoalID, text); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating goal: %v", err))
			}
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
		m.modal.Close()
		m.inputs.textInput.Reset()
		return m, nil, true
	}
	if state, ok := m.modal.GoalCreateState(); ok {
		if text != "" {
			if state.ParentID > 0 {
				if err := m.db.AddSubtask(m.ctx, text, state.ParentID); err != nil {
					m.setStatusError(fmt.Sprintf("Error adding subtask: %v", err))
				} else {
					m.view.expandedState[state.ParentID] = true
				}
			} else {
				if err := m.db.AddGoal(m.ctx, m.workspaces[m.activeWorkspaceIdx].ID, text, m.sprints[m.view.focusedColIdx].ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error adding goal: %v", err))
				}
			}
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
		m.modal.Close()
		m.inputs.textInput.Reset()
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleModalInputGoalText(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	if m.modal.Is(ModalGoalCreate) || m.modal.Is(ModalGoalEdit) || m.modal.Is(ModalWorkspaceCreate) || m.modal.Is(ModalWorkspaceInit) {
		m.inputs.textInput, cmd = m.inputs.textInput.Update(msg)
		return m, cmd, true
	}
	return m, nil, false
}
