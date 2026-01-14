package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmTagging() (DashboardModel, tea.Cmd, bool) {
	state, ok := m.modal.TaggingState()
	if !ok {
		return m, nil, false
	}
	raw := strings.Fields(m.inputs.tagInput.Value())
	tags := make(map[string]bool)
	for t, selected := range state.Selected {
		if selected {
			tags[t] = true
		}
	}
	for _, t := range raw {
		t = strings.TrimSpace(t)
		t = strings.TrimPrefix(t, "#")
		t = strings.ToLower(t)
		if t != "" {
			tags[t] = true
		}
	}
	var out []string
	for t := range tags {
		out = append(out, t)
	}
	if err := m.db.SetGoalTags(m.ctx, state.GoalID, out); err != nil {
		m.setStatusError(fmt.Sprintf("Error saving tags: %v", err))
	} else {
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	m.modal.Close()
	m.inputs.tagInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalInputTagging(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	state, ok := m.modal.TaggingState()
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
			if state.Cursor < len(m.modal.defaultTags)-1 {
				state.Cursor++
			}
			return m, nil, true
		case "tab":
			if len(m.modal.defaultTags) > 0 && state.Cursor < len(m.modal.defaultTags) {
				tag := m.modal.defaultTags[state.Cursor]
				state.Selected[tag] = !state.Selected[tag]
			}
			return m, nil, true
		}
	}
	m.inputs.tagInput, cmd = m.inputs.tagInput.Update(msg)
	return m, cmd, true
}
