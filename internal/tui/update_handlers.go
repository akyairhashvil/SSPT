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

type KeyHandler func(DashboardModel, string) (DashboardModel, tea.Cmd, bool)

type HandlerChain []KeyHandler

func (chain HandlerChain) Handle(m DashboardModel, key string) (DashboardModel, tea.Cmd, bool) {
	for _, handler := range chain {
		if next, cmd, handled := handler(m, key); handled {
			return next, cmd, true
		}
	}
	return m, nil, false
}

func wrapKeyHandler(fn func(DashboardModel, string) (DashboardModel, bool)) KeyHandler {
	return func(m DashboardModel, key string) (DashboardModel, tea.Cmd, bool) {
		next, handled := fn(m, key)
		return next, nil, handled
	}
}

var normalModeHandlers = HandlerChain{
	wrapKeyHandler(DashboardModel.handleTabFocus),
	wrapKeyHandler(DashboardModel.handleArrowKeys),
	wrapKeyHandler(DashboardModel.handleScrolling),
	DashboardModel.handleGoalCreate,
	DashboardModel.handleGoalEdit,
	DashboardModel.handleGoalDelete,
	DashboardModel.handleGoalMove,
	DashboardModel.handleGoalExpandCollapse,
	DashboardModel.handleGoalTaskTimer,
	DashboardModel.handleGoalPriority,
	DashboardModel.handleGoalJournalStart,
	DashboardModel.handleGoalArchive,
	DashboardModel.handleGoalDependencyPicker,
	DashboardModel.handleGoalRecurrencePicker,
	DashboardModel.handleGoalStatusToggle,
	DashboardModel.handleGoalTagging,
	DashboardModel.handleSprintPause,
	DashboardModel.handleSprintStart,
	DashboardModel.handleSprintReset,
	DashboardModel.handleWorkspaceSprintCount,
	DashboardModel.handleWorkspaceSwitch,
	DashboardModel.handleWorkspaceCreate,
	DashboardModel.handleWorkspaceVisibility,
	DashboardModel.handleWorkspaceViewMode,
	DashboardModel.handleWorkspaceTheme,
	DashboardModel.handleWorkspaceSeedImport,
	DashboardModel.handleWorkspaceReport,
}

var normalModeKeyHandlers = map[string]KeyHandler{
	"q":      handleNormalQuit,
	"ctrl+c": handleNormalQuit,
	"L":      handleNormalLock,
	"ctrl+e": handleNormalExport,
	"/":      handleNormalSearch,
	"C":      handleNormalClearDB,
	"p":      handleNormalPassphrase,
	"<":      handleNormalPrevDay,
	">":      handleNormalNextDay,
}

func (m DashboardModel) handleNormalMode(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	key := msg.String()
	if next, cmd, handled := normalModeHandlers.Handle(m, key); handled {
		return next, cmd
	}
	if handler, ok := normalModeKeyHandlers[key]; ok {
		next, cmd, _ := handler(m, key)
		return next, cmd
	}
	return m, nil
}

func handleNormalQuit(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	return m, tea.Quit, true
}

func handleNormalLock(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	if m.security.lock.PassphraseHash == "" {
		m.security.lock.Message = "Set passphrase to unlock"
	} else {
		m.security.lock.Message = "Enter passphrase to unlock"
	}
	m.security.lock.Locked = true
	m.security.lock.PassphraseInput.Reset()
	m.security.lock.PassphraseInput.Focus()
	return m, nil, true
}

func handleNormalExport(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	path, err := ExportVault(m.ctx, m.db, m.security.lock.PassphraseHash)
	if err != nil {
		m.Message = fmt.Sprintf("Export failed: %v", err)
	} else {
		m.Message = fmt.Sprintf("Export saved: %s", path)
	}
	return m, nil, true
}

func handleNormalSearch(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	m.search.Active = true
	m.search.ArchiveOnly = m.view.focusedColIdx < len(m.sprints) && m.sprints[m.view.focusedColIdx].SprintNumber == -2
	m.search.Cursor = 0
	m.search.Input.Focus()
	if m.search.ArchiveOnly && len(m.workspaces) > 0 {
		query := util.ParseSearchQuery(m.search.Input.Value())
		query.Status = []string{"archived"}
		m.search.Results, m.err = m.db.Search(m.ctx, query, m.workspaces[m.activeWorkspaceIdx].ID)
	}
	return m, nil, true
}

func handleNormalClearDB(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	m.security.confirmingClearDB = true
	m.security.clearDBNeedsPass = m.security.lock.PassphraseHash != ""
	m.security.clearDBStatus = ""
	m.security.lock.PassphraseInput.Reset()
	m.security.lock.PassphraseInput.Placeholder = "Passphrase"
	m.security.lock.PassphraseInput.Focus()
	return m, nil, true
}

func handleNormalPassphrase(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
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
	return m, nil, true
}

func handleNormalPrevDay(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	prevID, _, err := m.db.GetAdjacentDay(m.ctx, m.day.ID, -1)
	if err == nil {
		m.refreshData(prevID)
	} else {
		m.Message = "No previous days recorded."
	}
	return m, nil, true
}

func handleNormalNextDay(m DashboardModel, _ string) (DashboardModel, tea.Cmd, bool) {
	nextID, _, err := m.db.GetAdjacentDay(m.ctx, m.day.ID, 1)
	if err == nil {
		m.refreshData(nextID)
	} else {
		m.Message = "No future days recorded."
	}
	return m, nil, true
}
