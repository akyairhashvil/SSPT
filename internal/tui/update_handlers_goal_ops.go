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
		m.creatingGoal, m.editingGoalID = true, 0
		m.textInput.Placeholder = "New Objective..."
		m.textInput.Focus()
		return m, nil, true
	case "N":
		if m.validSprintIndex(m.focusedColIdx) && m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
			parent := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
			m.creatingGoal, m.editingGoalID = true, parent.ID
			m.textInput.Placeholder = "New Subtask..."
			m.textInput.Focus()
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalEdit(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "e" {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
		m.editingGoal, m.editingGoalID = true, target.ID
		m.textInput.SetValue(target.Description)
		m.textInput.Focus()
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalDelete(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "d", "backspace":
		if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
			m.confirmingDelete = true
			m.confirmDeleteGoalID = m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx].ID
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalMove(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "m" {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		m.movingGoal = true
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleMoveMode(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			m.movingGoal = false
			return m, nil
		}
		if len(msg.String()) == 1 && strings.Contains("012345678", msg.String()) {
			targetNum := int(msg.String()[0] - '0')
			currentSprint := m.sprints[m.focusedColIdx]
			if len(currentSprint.Goals) > m.focusedGoalIdx {
				goal := currentSprint.Goals[m.focusedGoalIdx]
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
						if m.focusedGoalIdx > 0 {
							m.focusedGoalIdx--
						}
					}
				}
			}
			m.movingGoal = false
			return m, nil
		}
	}
	return m, nil
}

func (m DashboardModel) handleGoalExpandCollapse(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "z" {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
		m.expandedState[target.ID] = !m.expandedState[target.ID]
		m.refreshData(m.day.ID)
	}
	return m, nil, true
}

func (m DashboardModel) handleGoalTaskTimer(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "T" {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
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
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
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
		m.journaling, m.editingGoalID = true, 0
		m.journalInput.Placeholder = "Log your thoughts..."
		m.journalInput.Focus()
		return m, nil, true
	case "J":
		if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
			target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
			m.journaling, m.editingGoalID = true, target.ID
			m.journalInput.Placeholder = fmt.Sprintf("Log for: %s", target.Description)
			m.journalInput.Focus()
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalArchive(key string) (DashboardModel, tea.Cmd, bool) {
	switch key {
	case "A":
		if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
			sprint := m.sprints[m.focusedColIdx]
			if sprint.SprintNumber != -2 {
				if err := m.db.ArchiveGoal(m.ctx, sprint.Goals[m.focusedGoalIdx].ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error archiving goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
					if m.focusedGoalIdx > 0 {
						m.focusedGoalIdx--
					}
				}
			}
		}
		return m, nil, true
	case "u":
		if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
			sprint := m.sprints[m.focusedColIdx]
			if sprint.SprintNumber == -2 {
				if err := m.db.UnarchiveGoal(m.ctx, sprint.Goals[m.focusedGoalIdx].ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error unarchiving goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
					if m.focusedGoalIdx > 0 {
						m.focusedGoalIdx--
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
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
		m.depPicking, m.editingGoalID = true, target.ID
		m.depOptions = m.buildDepOptions(target.ID)
		deps, err := m.db.GetGoalDependencies(m.ctx, target.ID)
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading dependencies: %v", err))
			m.depSelected = make(map[int64]bool)
		} else {
			m.depSelected = deps
		}
		m.depCursor = 0
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalRecurrencePicker(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "R" {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
		m.settingRecurrence, m.editingGoalID = true, target.ID
		m.recurrenceCursor = 0
		m.recurrenceMode = "none"
		m.recurrenceSelected = make(map[string]bool)
		m.recurrenceFocus = "mode"
		m.recurrenceItemCursor = 0
		m.recurrenceDayCursor = 0
		if target.RecurrenceRule != nil {
			rule := strings.ToLower(strings.TrimSpace(*target.RecurrenceRule))
			switch {
			case rule == "daily":
				m.recurrenceMode = "daily"
			case strings.HasPrefix(rule, "weekly:"):
				m.recurrenceMode = "weekly"
				parts := strings.Split(strings.TrimPrefix(rule, "weekly:"), ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						m.recurrenceSelected[p] = true
					}
				}
				for i, d := range m.weekdayOptions {
					if m.recurrenceSelected[d] {
						m.recurrenceItemCursor = i
						break
					}
				}
			case strings.HasPrefix(rule, "monthly:"):
				m.recurrenceMode = "monthly"
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
						m.recurrenceSelected[mo] = true
					}
				}
				if len(days) == 0 {
					days = []string{"1"}
				}
				for _, d := range days {
					d = strings.TrimSpace(d)
					if d != "" {
						m.recurrenceSelected["day:"+d] = true
					}
				}
				for i, mo := range m.monthOptions {
					if m.recurrenceSelected[mo] {
						m.recurrenceItemCursor = i
						break
					}
				}
				for i, d := range m.monthDayOptions {
					if m.recurrenceSelected["day:"+d] {
						m.recurrenceDayCursor = i
						break
					}
				}
			}
		}
		for i, opt := range m.recurrenceOptions {
			if opt == m.recurrenceMode {
				m.recurrenceCursor = i
				break
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalTagging(key string) (DashboardModel, tea.Cmd, bool) {
	if key != "t" {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
		m.tagging, m.editingGoalID = true, target.ID
		m.tagInput.Focus()
		m.tagInput.SetValue("")
		m.tagSelected = make(map[string]bool)
		var customTags []string
		if target.Tags != nil {
			for _, t := range util.JSONToTags(*target.Tags) {
				if containsTag(m.defaultTags, t) {
					m.tagSelected[t] = true
				} else {
					customTags = append(customTags, t)
				}
			}
		}
		if len(customTags) > 0 {
			sort.Strings(customTags)
			m.tagInput.SetValue(strings.Join(customTags, " "))
		}
		m.tagCursor = 0
		return m, nil, true
	}
	return m, nil, false
}

func (m DashboardModel) handleGoalStatusToggle(key string) (DashboardModel, tea.Cmd, bool) {
	if key != " " {
		return m, nil, false
	}
	if m.validSprintIndex(m.focusedColIdx) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
		goal := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
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
