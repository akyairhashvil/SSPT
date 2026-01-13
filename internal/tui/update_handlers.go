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
		target := config.TargetTitleWidth
		if m.width < config.CompactModeThreshold {
			target = m.width / 2
		}
		if target < config.MinTitleWidth {
			target = config.MinTitleWidth
		}
		m.progress.Width = target
	}
	return m, nil
}

func (m DashboardModel) handleTick(msg TickMsg) (DashboardModel, tea.Cmd) {
	if !m.security.lock.Locked && m.security.lock.PassphraseHash != "" && time.Since(m.security.lock.LastInput) >= config.AutoLockAfter {
		m.security.lock.Locked = true
		m.security.lock.Message = "Session locked (idle)"
		m.security.lock.PassphraseInput.Reset()
		m.security.lock.PassphraseInput.Focus()
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
		if m.security.lock.PassphraseHash == "" {
			m.security.lock.Message = "Set passphrase to unlock"
		} else {
			m.security.lock.Message = "Enter passphrase to unlock"
		}
		m.security.lock.Locked = true
		m.security.lock.PassphraseInput.Reset()
		m.security.lock.PassphraseInput.Focus()
		return m, nil
	case "ctrl+e":
		path, err := ExportVault(m.ctx, m.db, m.security.lock.PassphraseHash)
		if err != nil {
			m.Message = fmt.Sprintf("Export failed: %v", err)
		} else {
			m.Message = fmt.Sprintf("Export saved: %s", path)
		}
		return m, nil
	case "/":
		m.search.Active = true
		m.search.ArchiveOnly = m.view.focusedColIdx < len(m.sprints) && m.sprints[m.view.focusedColIdx].SprintNumber == -2
		m.search.Cursor = 0
		m.search.Input.Focus()
		if m.search.ArchiveOnly && len(m.workspaces) > 0 {
			query := util.ParseSearchQuery(m.search.Input.Value())
			query.Status = []string{"archived"}
			m.search.Results, m.err = m.db.Search(m.ctx, query, m.workspaces[m.activeWorkspaceIdx].ID)
		}
		return m, nil
	case "C":
		m.security.confirmingClearDB = true
		m.security.clearDBNeedsPass = m.security.lock.PassphraseHash != ""
		m.security.clearDBStatus = ""
		m.security.lock.PassphraseInput.Reset()
		m.security.lock.PassphraseInput.Placeholder = "Passphrase"
		m.security.lock.PassphraseInput.Focus()
		return m, nil
	case "p":
		m.security.changingPassphrase = true
		m.security.passphraseStatus = ""
		m.security.passphraseStage = 0
		m.inputs.passphraseCurrent.Reset()
		m.inputs.passphraseNew.Reset()
		m.inputs.passphraseConfirm.Reset()
		if m.security.lock.PassphraseHash == "" {
			m.security.passphraseStage = 1
			m.inputs.passphraseNew.Focus()
		} else {
			m.inputs.passphraseCurrent.Focus()
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
