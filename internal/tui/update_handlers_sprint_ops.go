package tui

import (
	"fmt"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleSprintCompletion() (DashboardModel, bool) {
	if !m.hasActiveSprint() {
		return m, false
	}
	if err := m.db.CompleteSprint(m.ctx, m.timer.ActiveSprint.ID); err != nil {
		m.setStatusError(fmt.Sprintf("Error completing sprint: %v", err))
		return m, true
	}
	if err := m.db.MovePendingToBacklog(m.ctx, m.timer.ActiveSprint.ID); err != nil {
		m.setStatusError(fmt.Sprintf("Error moving pending tasks: %v", err))
	}
	m.timer.ActiveSprint, m.timer.BreakActive, m.timer.BreakStart = nil, true, time.Now()
	m.refreshData(m.day.ID)
	return m, true
}

func (m DashboardModel) handleSprintPause(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "s" {
		return m, nil, false
	}
	if m.timer.BreakActive || !m.validSprintIndex(m.focusedColIdx) {
		return m, nil, true
	}
	target := m.sprints[m.focusedColIdx]
	if target.SprintNumber <= 0 {
		return m, nil, true
	}
	if m.hasActiveSprint() && m.timer.ActiveSprint.ID == target.ID {
		startedAt := time.Now()
		if m.timer.ActiveSprint.StartTime != nil {
			startedAt = *m.timer.ActiveSprint.StartTime
		}
		elapsed := int(time.Since(startedAt).Seconds()) + m.timer.ActiveSprint.ElapsedSeconds
		if err := m.db.PauseSprint(m.ctx, target.ID, int(elapsed)); err != nil {
			m.setStatusError(fmt.Sprintf("Error pausing sprint: %v", err))
		} else {
			m.refreshData(m.day.ID)
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleSprintStart(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "s" {
		return m, nil, false
	}
	if m.timer.BreakActive || !m.validSprintIndex(m.focusedColIdx) {
		return m, nil, true
	}
	target := m.sprints[m.focusedColIdx]
	if target.SprintNumber <= 0 {
		return m, nil, true
	}
	if !m.hasActiveSprint() && (target.Status == models.StatusPending || target.Status == models.StatusPaused) {
		if err := m.db.StartSprint(m.ctx, target.ID); err != nil {
			m.setStatusError(fmt.Sprintf("Error starting sprint: %v", err))
		} else {
			m.refreshData(m.day.ID)
			return m, tickCmd(), true
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleSprintReset(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "x" {
		return m, nil, false
	}
	if m.hasActiveSprint() {
		if err := m.db.ResetSprint(m.ctx, m.timer.ActiveSprint.ID); err != nil {
			m.setStatusError(fmt.Sprintf("Error resetting sprint: %v", err))
		} else {
			m.timer.ActiveSprint = nil
			m.refreshData(m.day.ID)
		}
	}
	return m, nil, true
}
