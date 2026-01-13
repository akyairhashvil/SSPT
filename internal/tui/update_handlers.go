package tui

import (
	"fmt"
	"time"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/akyairhashvil/SSPT/internal/util"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleWindowSize(msg tea.WindowSizeMsg) (DashboardModel, tea.Cmd) {
	m.width, m.height = msg.Width, msg.Height
	if m.width > 0 {
		target := 30
		if m.width < 60 {
			target = m.width / 2
		}
		if target < 10 {
			target = 10
		}
		m.progress.Width = target
	}
	return m, nil
}

func (m DashboardModel) handleTick(msg TickMsg) (DashboardModel, tea.Cmd) {
	if !m.lock.Locked && m.lock.PassphraseHash != "" && time.Since(m.lock.LastInput) >= config.AutoLockAfter {
		m.lock.Locked = true
		m.lock.Message = "Session locked (idle)"
		m.lock.PassphraseInput.Reset()
		m.lock.PassphraseInput.Focus()
		return m, nil
	}
	if m.timer.BreakActive {
		if time.Since(m.timer.BreakStart) >= config.BreakDuration {
			m.timer.BreakActive = false
		}
		return m, tickCmd()
	}
	if m.timer.ActiveSprint != nil {
		startedAt := time.Now()
		if m.timer.ActiveSprint.StartTime != nil {
			startedAt = *m.timer.ActiveSprint.StartTime
		}
		elapsed := time.Since(startedAt) + (time.Duration(m.timer.ActiveSprint.ElapsedSeconds) * time.Second)
		if elapsed >= config.SprintDuration {
			next, _ := m.handleSprintCompletion()
			return next, tickCmd()
		}
		newProg, progCmd := m.progress.Update(msg)
		m.progress = newProg.(progress.Model)
		return m, tea.Batch(tickCmd(), progCmd)
	}
	return m, tickCmd()
}

func (m DashboardModel) handleNormalMode(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	key := msg.String()
	if next, handled := m.handleTabFocus(key); handled {
		return next, nil
	}
	if next, handled := m.handleArrowKeys(key); handled {
		return next, nil
	}
	if next, handled := m.handleScrolling(key); handled {
		return next, nil
	}
	if next, cmd, handled := m.handleGoalCreate(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalEdit(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalDelete(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalMove(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalExpandCollapse(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalTaskTimer(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalPriority(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalJournalStart(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalArchive(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalDependencyPicker(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalRecurrencePicker(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalStatusToggle(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleGoalTagging(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleSprintPause(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleSprintStart(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleSprintReset(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceSprintCount(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceSwitch(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceCreate(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceVisibility(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceViewMode(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceTheme(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceSeedImport(key); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleWorkspaceReport(key); handled {
		return next, cmd
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "L":
		if m.lock.PassphraseHash == "" {
			m.lock.Message = "Set passphrase to unlock"
		} else {
			m.lock.Message = "Enter passphrase to unlock"
		}
		m.lock.Locked = true
		m.lock.PassphraseInput.Reset()
		m.lock.PassphraseInput.Focus()
		return m, nil
	case "ctrl+e":
		path, err := ExportVault(m.ctx, m.db, m.lock.PassphraseHash)
		if err != nil {
			m.Message = fmt.Sprintf("Export failed: %v", err)
		} else {
			m.Message = fmt.Sprintf("Export saved: %s", path)
		}
		return m, nil
	case "/":
		m.search.Active = true
		m.search.ArchiveOnly = m.focusedColIdx < len(m.sprints) && m.sprints[m.focusedColIdx].SprintNumber == -2
		m.search.Cursor = 0
		m.search.Input.Focus()
		if m.search.ArchiveOnly && len(m.workspaces) > 0 {
			query := util.ParseSearchQuery(m.search.Input.Value())
			query.Status = []string{"archived"}
			m.search.Results, m.err = m.db.Search(m.ctx, query, m.workspaces[m.activeWorkspaceIdx].ID)
		}
		return m, nil
	case "C":
		m.confirmingClearDB = true
		m.clearDBNeedsPass = m.lock.PassphraseHash != ""
		m.clearDBStatus = ""
		m.lock.PassphraseInput.Reset()
		m.lock.PassphraseInput.Placeholder = "Passphrase"
		m.lock.PassphraseInput.Focus()
		return m, nil
	case "p":
		m.changingPassphrase = true
		m.passphraseStatus = ""
		m.passphraseStage = 0
		m.passphraseCurrent.Reset()
		m.passphraseNew.Reset()
		m.passphraseConfirm.Reset()
		if m.lock.PassphraseHash == "" {
			m.passphraseStage = 1
			m.passphraseNew.Focus()
		} else {
			m.passphraseCurrent.Focus()
		}
		return m, nil
	case "<":
		prevID, _, err := m.db.GetAdjacentDay(m.ctx, m.day.ID, -1)
		if err == nil {
			m.refreshData(prevID)
		} else {
			m.Message = "No previous days recorded."
		}
	case ">":
		nextID, _, err := m.db.GetAdjacentDay(m.ctx, m.day.ID, 1)
		if err == nil {
			m.refreshData(nextID)
		} else {
			m.Message = "No future days recorded."
		}
	}
	return m, nil
}
