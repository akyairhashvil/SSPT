package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleGoalCreate(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "n":
		m.modal.Open(&GoalCreateState{})
		m.inputs.textInput.Placeholder = "New Objective..."
		m.inputs.textInput.Focus()
		return m, nil, true
	case "N":
		if m.validSprintIndex(m.view.focusedColIdx) && m.view.focusedColIdx > 0 && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
			parent := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
			m.modal.Open(&GoalCreateState{ParentID: parent.ID})
			m.inputs.textInput.Placeholder = "New Subtask..."
			m.inputs.textInput.Focus()
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalEdit(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "e" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		m.modal.Open(&GoalEditState{GoalID: target.ID})
		m.inputs.textInput.SetValue(target.Description)
		m.inputs.textInput.Focus()
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalDelete(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "d", "backspace":
		if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
			m.modal.Open(&GoalDeleteState{GoalID: m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx].ID})
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalMove(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "m" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		m.modal.Open(&GoalMoveState{})
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleMoveMode(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			m.modal.Close()
			return m, nil
		}
		if len(msg.String()) == 1 && strings.Contains("012345678", msg.String()) {
			targetNum := int(msg.String()[0] - '0')
			currentSprint := m.sprints[m.view.focusedColIdx]
			if len(currentSprint.Goals) > m.view.focusedGoalIdx {
				goal := currentSprint.Goals[m.view.focusedGoalIdx]
				var targetID int64 = 0 // Default to Backlog
				found := false
				if targetNum == 0 {
					found = true
				} else {
					for _, s := range m.sprints {
						if s.SprintNumber == targetNum {
							targetID = s.ID
							found = true
							break
						}
					}
				}
				if found {
					if err := m.db.MoveGoal(m.ctx, goal.ID, targetID); err != nil {
						m.setStatusError(fmt.Sprintf("Error moving goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
						if m.view.focusedGoalIdx > 0 {
							m.view.focusedGoalIdx--
						}
					}
				}
			}
			m.modal.Close()
			return m, nil
		}
	}
	return m, nil
}

func (m DashboardModel) handleGoalExpandCollapse(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "z" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		m.view.expandedState[target.ID] = !m.view.expandedState[target.ID]
		m.refreshData(m.day.ID)
	}
	return m, nil, true
}

func (m DashboardModel) handleGoalTaskTimer(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "T" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		if target.TaskActive {
			if err := m.db.PauseTaskTimer(m.ctx, target.ID); err != nil {
				m.setStatusError(fmt.Sprintf("Error pausing task timer: %v", err))
			} else {
				m.Message = "Task timer paused."
			}
		} else {
			if err := m.db.StartTaskTimer(m.ctx, target.ID); err != nil {
				m.setStatusError(fmt.Sprintf("Error starting task timer: %v", err))
			} else {
				m.Message = "Task timer started."
			}
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	return m, nil, true
}

func (m DashboardModel) handleGoalPriority(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "P" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		next := target.Priority + 1
		if next < 1 || next > 5 {
			next = 1
		}
		if err := m.db.UpdateGoalPriority(m.ctx, target.ID, next); err != nil {
			m.setStatusError(fmt.Sprintf("Error updating priority: %v", err))
		} else {
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
	}
	return m, nil, true
}

func (m DashboardModel) handleGoalJournalStart(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "ctrl+j":
		m.modal.Open(&JournalState{})
		m.inputs.journalInput.Placeholder = "Log your thoughts..."
		m.inputs.journalInput.Focus()
		return m, nil, true
	case "J":
		if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
			target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
			m.modal.Open(&JournalState{GoalID: target.ID})
			m.inputs.journalInput.Placeholder = fmt.Sprintf("Log for: %s", target.Description)
			m.inputs.journalInput.Focus()
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalArchive(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "A":
		if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
			sprint := m.sprints[m.view.focusedColIdx]
			if sprint.SprintNumber != -2 {
				if err := m.db.ArchiveGoal(m.ctx, sprint.Goals[m.view.focusedGoalIdx].ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error archiving goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
					if m.view.focusedGoalIdx > 0 {
						m.view.focusedGoalIdx--
					}
				}
			}
		}
		return m, nil, true
	case "u":
		if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
			sprint := m.sprints[m.view.focusedColIdx]
			if sprint.SprintNumber == -2 {
				if err := m.db.UnarchiveGoal(m.ctx, sprint.Goals[m.view.focusedGoalIdx].ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error unarchiving goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
					if m.view.focusedGoalIdx > 0 {
						m.view.focusedGoalIdx--
					}
				}
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalDependencyPicker(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "D" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		state := &DependencyState{
			GoalID:  target.ID,
			Options: m.buildDepOptions(target.ID),
		}
		deps, err := m.db.GetGoalDependencies(m.ctx, target.ID)
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading dependencies: %v", err))
			state.Selected = make(map[int64]bool)
		} else {
			state.Selected = deps
		}
		state.Cursor = 0
		m.modal.Open(state)
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalRecurrencePicker(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "R" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		state := &RecurrenceState{
			GoalID:          target.ID,
			Options:         m.modal.recurrenceOptions,
			Mode:            "none",
			WeekdayOptions:  m.modal.weekdayOptions,
			MonthOptions:    m.modal.monthOptions,
			Selected:        make(map[string]bool),
			Focus:           "mode",
			ItemCursor:      0,
			DayCursor:       0,
			MonthDayOptions: m.modal.monthDayOptions,
		}
		if target.RecurrenceRule != nil {
			rule := strings.ToLower(strings.TrimSpace(*target.RecurrenceRule))
			switch {
			case rule == "daily":
				state.Mode = "daily"
			case strings.HasPrefix(rule, "weekly:"):
				state.Mode = "weekly"
				parts := strings.Split(strings.TrimPrefix(rule, "weekly:"), ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						state.Selected[p] = true
					}
				}
				for i, d := range state.WeekdayOptions {
					if state.Selected[d] {
						state.ItemCursor = i
						break
					}
				}
			case strings.HasPrefix(rule, "monthly:"):
				state.Mode = "monthly"
				payload := strings.TrimPrefix(rule, "monthly:")
				var months []string
				var days []string
				if strings.Contains(payload, "months=") || strings.Contains(payload, "days=") {
					chunks := strings.Split(payload, ";")
					for _, chunk := range chunks {
						chunk = strings.TrimSpace(chunk)
						switch {
						case strings.HasPrefix(chunk, "months="):
							months = strings.Split(strings.TrimPrefix(chunk, "months="), ",")
						case strings.HasPrefix(chunk, "days="):
							days = strings.Split(strings.TrimPrefix(chunk, "days="), ",")
						}
					}
				} else if payload != "" {
					months = strings.Split(payload, ",")
				}
				for _, mo := range months {
					mo = strings.TrimSpace(mo)
					if mo != "" {
						state.Selected[mo] = true
					}
				}
				if len(days) == 0 {
					days = []string{"1"}
				}
				for _, d := range days {
					d = strings.TrimSpace(d)
					if d != "" {
						state.Selected["day:"+d] = true
					}
				}
				for i, mo := range state.MonthOptions {
					if state.Selected[mo] {
						state.ItemCursor = i
						break
					}
				}
				for i, d := range state.MonthDayOptions {
					if state.Selected["day:"+d] {
						state.DayCursor = i
						break
					}
				}
			}
		}
		for i, opt := range state.Options {
			if opt == state.Mode {
				state.Cursor = i
				break
			}
		}
		m.modal.Open(state)
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalTagging(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "t" {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && m.view.focusedColIdx > 0 && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		target := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		state := &TaggingState{
			GoalID:   target.ID,
			Selected: make(map[string]bool),
		}
		m.inputs.tagInput.Focus()
		m.inputs.tagInput.SetValue("")
		var customTags []string
		if target.Tags != nil {
			for _, t := range util.JSONToTags(*target.Tags) {
				if containsTag(m.modal.defaultTags, t) {
					state.Selected[t] = true
				} else {
					customTags = append(customTags, t)
				}
			}
		}
		if len(customTags) > 0 {
			sort.Strings(customTags)
			m.inputs.tagInput.SetValue(strings.Join(customTags, " "))
		}
		state.Cursor = 0
		m.modal.Open(state)
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalStatusToggle(key string) (DashboardModel, tea.Cmd, bool) {
	if key != " " {
		return m, nil, false
	}
	if m.validSprintIndex(m.view.focusedColIdx) && len(m.sprints[m.view.focusedColIdx].Goals) > m.view.focusedGoalIdx {
		goal := m.sprints[m.view.focusedColIdx].Goals[m.view.focusedGoalIdx]
		blocked, err := m.db.IsGoalBlocked(m.ctx, goal.ID)
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error checking dependencies: %v", err))
		} else if blocked {
			m.Message = "Blocked by dependency. Complete dependencies first."
			return m, nil, true
		}
		canToggle := true
		if goal.Status == models.GoalStatusPending {
			for _, sub := range goal.Subtasks {
				if sub.Status != models.GoalStatusCompleted {
					canToggle = false
					break
				}
			}
		}
		if canToggle {
			newStatus := models.GoalStatusPending
			if goal.Status == models.GoalStatusPending {
				newStatus = models.GoalStatusCompleted
			}
			if err := m.db.UpdateGoalStatus(m.ctx, goal.ID, newStatus); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating goal status: %v", err))
			} else {
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
		} else {
			m.Message = "Cannot complete task with pending subtasks!"
		}
	}
	return m, nil, true
}
