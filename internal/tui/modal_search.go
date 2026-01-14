package tui

import (
	"fmt"

	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmSearch() (DashboardModel, tea.Cmd, bool) {
	if !m.search.Active {
		return m, nil, false
	}
	m.search.Active = false
	m.search.Input.Reset()
	m.search.Cursor = 0
	m.search.ArchiveOnly = false
	return m, nil, true
}

func (m DashboardModel) handleModalInputSearch(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	if !m.search.Active {
		return m, nil, false
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.search.Cursor > 0 {
				m.search.Cursor--
			}
			return m, nil, true
		case "down", "j":
			if m.search.Cursor < len(m.search.Results)-1 {
				m.search.Cursor++
			}
			return m, nil, true
		case "u":
			if m.search.ArchiveOnly && len(m.search.Results) > 0 && m.search.Cursor < len(m.search.Results) {
				target := m.search.Results[m.search.Cursor]
				if err := m.db.UnarchiveGoal(m.ctx, target.ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error unarchiving goal: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
					query := util.ParseSearchQuery(m.search.Input.Value())
					query.Status = []string{"archived"}
					m.search.Results, m.err = m.db.Search(m.ctx, query, m.workspaces[m.activeWorkspaceIdx].ID)
					if m.search.Cursor >= len(m.search.Results) {
						m.search.Cursor = len(m.search.Results) - 1
					}
					if m.search.Cursor < 0 {
						m.search.Cursor = 0
					}
				}
			}
			return m, nil, true
		}
	}
	m.search.Input, cmd = m.search.Input.Update(msg)
	if _, ok := msg.(tea.KeyMsg); ok && len(m.workspaces) > 0 {
		query := util.ParseSearchQuery(m.search.Input.Value())
		if m.search.ArchiveOnly {
			query.Status = []string{"archived"}
		}
		m.search.Results, m.err = m.db.Search(m.ctx, query, m.workspaces[m.activeWorkspaceIdx].ID)
		if m.search.Cursor >= len(m.search.Results) {
			m.search.Cursor = len(m.search.Results) - 1
		}
		if m.search.Cursor < 0 {
			m.search.Cursor = 0
		}
	}
	return m, cmd, true
}
