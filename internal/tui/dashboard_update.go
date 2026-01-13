package tui

import tea "github.com/charmbracelet/bubbletea"

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Clear error on keypress
	if m.err != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.err = nil
			return m, nil
		}
	}
	// Clear transient messages on keypress
	if m.Message != "" {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.Message = ""
			return m, nil
		}
	}

	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		return m.handleWindowSize(msg)
	}
	if msg, ok := msg.(TickMsg); ok {
		return m.handleTick(msg)
	}

	if m.security.lock.Locked {
		return m.handleLockedState(msg)
	}
	if m.inInputMode() {
		return m.handleModalState(msg)
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.modal.movingGoal {
			return m.handleMoveMode(msg)
		}
		return m.handleNormalMode(msg)
	}

	return m, nil
}
