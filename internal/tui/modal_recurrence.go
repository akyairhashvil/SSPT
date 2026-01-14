package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmRecurrence() (DashboardModel, tea.Cmd, bool) {
	if !m.modal.settingRecurrence {
		return m, nil, false
	}
	if m.modal.editingGoalID > 0 {
		rule := m.modal.recurrenceMode
		switch rule {
		case "none":
			if err := m.db.UpdateGoalRecurrence(m.ctx, m.modal.editingGoalID, ""); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
			}
		case "daily":
			if err := m.db.UpdateGoalRecurrence(m.ctx, m.modal.editingGoalID, "daily"); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
			}
		case "weekly":
			var days []string
			for _, d := range m.modal.weekdayOptions {
				if m.modal.recurrenceSelected[d] {
					days = append(days, d)
				}
			}
			if len(days) == 0 {
				m.Message = "Select at least one weekday."
			} else {
				if err := m.db.UpdateGoalRecurrence(m.ctx, m.modal.editingGoalID, "weekly:"+strings.Join(days, ",")); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
				}
			}
		case "monthly":
			var months []string
			var days []string
			for _, mo := range m.modal.monthOptions {
				if m.modal.recurrenceSelected[mo] {
					months = append(months, mo)
				}
			}
			for _, d := range m.modal.monthDayOptions {
				if m.modal.recurrenceSelected["day:"+d] {
					days = append(days, d)
				}
			}
			switch {
			case len(months) == 0:
				m.Message = "Select at least one month."
			case len(days) == 0:
				m.Message = "Select at least one day."
			default:
				rule := fmt.Sprintf("monthly:months=%s;days=%s", strings.Join(months, ","), strings.Join(days, ","))
				if err := m.db.UpdateGoalRecurrence(m.ctx, m.modal.editingGoalID, rule); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
				}
			}
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	m.modal.settingRecurrence, m.modal.editingGoalID = false, 0
	m.modal.recurrenceSelected = make(map[string]bool)
	return m, nil, true
}

func (m DashboardModel) handleModalInputRecurrence(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	if !m.modal.settingRecurrence {
		return m, nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.modal.recurrenceFocus == "mode" {
				if m.modal.recurrenceCursor > 0 {
					m.modal.recurrenceCursor--
				}
			} else if m.modal.recurrenceFocus == "items" {
				if m.modal.recurrenceItemCursor > 0 {
					m.modal.recurrenceItemCursor--
				}
			} else if m.modal.recurrenceFocus == "days" {
				if m.modal.recurrenceDayCursor > 0 {
					m.modal.recurrenceDayCursor--
				}
			}
			return m, nil, true
		case "down", "j":
			if m.modal.recurrenceFocus == "mode" {
				if m.modal.recurrenceCursor < len(m.modal.recurrenceOptions)-1 {
					m.modal.recurrenceCursor++
				}
			} else if m.modal.recurrenceFocus == "items" {
				max := 0
				if m.modal.recurrenceMode == "weekly" {
					max = len(m.modal.weekdayOptions) - 1
				} else if m.modal.recurrenceMode == "monthly" {
					max = len(m.modal.monthOptions) - 1
				}
				if m.modal.recurrenceItemCursor < max {
					m.modal.recurrenceItemCursor++
				}
			} else if m.modal.recurrenceFocus == "days" {
				maxDay := m.monthlyMaxDay()
				if maxDay <= 0 {
					return m, nil, true
				}
				if m.modal.recurrenceDayCursor < maxDay-1 {
					m.modal.recurrenceDayCursor++
				}
			}
			return m, nil, true
		case "tab":
			if m.modal.recurrenceFocus == "items" && m.modal.recurrenceMode == "monthly" {
				m.modal.recurrenceFocus = "days"
			} else if m.modal.recurrenceFocus == "days" {
				m.modal.recurrenceFocus = "mode"
			} else if m.modal.recurrenceFocus == "items" {
				m.modal.recurrenceFocus = "mode"
			} else if len(m.modal.recurrenceOptions) > 0 && m.modal.recurrenceCursor < len(m.modal.recurrenceOptions) {
				m.modal.recurrenceMode = m.modal.recurrenceOptions[m.modal.recurrenceCursor]
				if m.modal.recurrenceMode == "weekly" || m.modal.recurrenceMode == "monthly" {
					m.modal.recurrenceFocus = "items"
				} else {
					m.modal.recurrenceFocus = "mode"
				}
			}
			return m, nil, true
		case " ":
			if m.modal.recurrenceFocus == "items" {
				switch m.modal.recurrenceMode {
				case "weekly":
					if m.modal.recurrenceItemCursor < len(m.modal.weekdayOptions) {
						key := m.modal.weekdayOptions[m.modal.recurrenceItemCursor]
						m.modal.recurrenceSelected[key] = !m.modal.recurrenceSelected[key]
					}
				case "monthly":
					if m.modal.recurrenceItemCursor < len(m.modal.monthOptions) {
						key := m.modal.monthOptions[m.modal.recurrenceItemCursor]
						m.modal.recurrenceSelected[key] = !m.modal.recurrenceSelected[key]
						m.pruneMonthlyDays(m.monthlyMaxDay())
					}
				}
			} else if m.modal.recurrenceFocus == "days" {
				maxDay := m.monthlyMaxDay()
				if maxDay > 0 && m.modal.recurrenceDayCursor < maxDay {
					key := "day:" + m.modal.monthDayOptions[m.modal.recurrenceDayCursor]
					m.modal.recurrenceSelected[key] = !m.modal.recurrenceSelected[key]
				}
			} else if m.modal.recurrenceFocus == "mode" {
				if len(m.modal.recurrenceOptions) > 0 && m.modal.recurrenceCursor < len(m.modal.recurrenceOptions) {
					m.modal.recurrenceMode = m.modal.recurrenceOptions[m.modal.recurrenceCursor]
				}
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
