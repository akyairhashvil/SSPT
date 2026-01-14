package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmRecurrence() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.RecurrenceState()
	if !ok {
		return m, nil, false
	}
	if state.GoalID > 0 {
		rule := state.Mode
		switch rule {
		case "none":
			if err := m.db.UpdateGoalRecurrence(m.ctx, state.GoalID, ""); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
			}
		case "daily":
			if err := m.db.UpdateGoalRecurrence(m.ctx, state.GoalID, "daily"); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
			}
		case "weekly":
			var days []string
			for _, d := range state.WeekdayOptions {
				if state.Selected[d] {
					days = append(days, d)
				}
			}
			if len(days) == 0 {
				m.Message = "Select at least one weekday."
			} else {
				if err := m.db.UpdateGoalRecurrence(m.ctx, state.GoalID, "weekly:"+strings.Join(days, ",")); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
				}
			}
		case "monthly":
			var months []string
			var days []string
			for _, mo := range state.MonthOptions {
				if state.Selected[mo] {
					months = append(months, mo)
				}
			}
			for _, d := range state.MonthDayOptions {
				if state.Selected["day:"+d] {
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
				if err := m.db.UpdateGoalRecurrence(m.ctx, state.GoalID, rule); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
				}
			}
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	m.modal.Close()
	return m, nil, true
}

func (m DashboardModel) handleModalInputRecurrence(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.RecurrenceState()
	if !ok {
		return m, nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if state.Focus == "mode" {
				if state.Cursor > 0 {
					state.Cursor--
				}
			} else if state.Focus == "items" {
				if state.ItemCursor > 0 {
					state.ItemCursor--
				}
			} else if state.Focus == "days" {
				if state.DayCursor > 0 {
					state.DayCursor--
				}
			}
			return m, nil, true
		case "down", "j":
			if state.Focus == "mode" {
				if state.Cursor < len(state.Options)-1 {
					state.Cursor++
				}
			} else if state.Focus == "items" {
				max := 0
				if state.Mode == "weekly" {
					max = len(state.WeekdayOptions) - 1
				} else if state.Mode == "monthly" {
					max = len(state.MonthOptions) - 1
				}
				if state.ItemCursor < max {
					state.ItemCursor++
				}
			} else if state.Focus == "days" {
				maxDay := m.monthlyMaxDay(state)
				if maxDay <= 0 {
					return m, nil, true
				}
				if state.DayCursor < maxDay-1 {
					state.DayCursor++
				}
			}
			return m, nil, true
		case "tab":
			if state.Focus == "items" && state.Mode == "monthly" {
				state.Focus = "days"
			} else if state.Focus == "days" {
				state.Focus = "mode"
			} else if state.Focus == "items" {
				state.Focus = "mode"
			} else if len(state.Options) > 0 && state.Cursor < len(state.Options) {
				state.Mode = state.Options[state.Cursor]
				if state.Mode == "weekly" || state.Mode == "monthly" {
					state.Focus = "items"
				} else {
					state.Focus = "mode"
				}
			}
			return m, nil, true
		case " ":
			if state.Focus == "items" {
				switch state.Mode {
				case "weekly":
					if state.ItemCursor < len(state.WeekdayOptions) {
						key := state.WeekdayOptions[state.ItemCursor]
						state.Selected[key] = !state.Selected[key]
					}
				case "monthly":
					if state.ItemCursor < len(state.MonthOptions) {
						key := state.MonthOptions[state.ItemCursor]
						state.Selected[key] = !state.Selected[key]
						m.pruneMonthlyDays(state, m.monthlyMaxDay(state))
					}
				}
			} else if state.Focus == "days" {
				maxDay := m.monthlyMaxDay(state)
				if maxDay > 0 && state.DayCursor < maxDay {
					key := "day:" + state.MonthDayOptions[state.DayCursor]
					state.Selected[key] = !state.Selected[key]
				}
			} else if state.Focus == "mode" {
				if len(state.Options) > 0 && state.Cursor < len(state.Options) {
					state.Mode = state.Options[state.Cursor]
				}
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
