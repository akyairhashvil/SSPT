package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmTagging() (DashboardModel, tea.Cmd, bool) {
	if !m.modal.tagging {
		return m, nil, false
	}
	raw := strings.Fields(m.inputs.tagInput.Value())
	tags := make(map[string]bool)
	for t, selected := range m.modal.tagSelected {
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
	if err := m.db.SetGoalTags(m.ctx, m.modal.editingGoalID, out); err != nil {
		m.setStatusError(fmt.Sprintf("Error saving tags: %v", err))
	} else {
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	m.modal.tagging, m.modal.editingGoalID = false, 0
	m.inputs.tagInput.Reset()
	m.modal.tagSelected = make(map[string]bool)
	m.modal.tagCursor = 0
	return m, nil, true
}

func (m DashboardModel) handleModalInputTagging(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	if !m.modal.tagging {
		return m, nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.modal.tagCursor > 0 {
				m.modal.tagCursor--
			}
			return m, nil, true
		case "down", "j":
			if m.modal.tagCursor < len(m.modal.defaultTags)-1 {
				m.modal.tagCursor++
			}
			return m, nil, true
		case "tab":
			if len(m.modal.defaultTags) > 0 && m.modal.tagCursor < len(m.modal.defaultTags) {
				tag := m.modal.defaultTags[m.modal.tagCursor]
				m.modal.tagSelected[tag] = !m.modal.tagSelected[tag]
			}
			return m, nil, true
		}
	}
	m.inputs.tagInput, cmd = m.inputs.tagInput.Update(msg)
	return m, cmd, true
}
