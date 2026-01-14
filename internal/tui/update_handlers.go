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

func wrapKeyHandler(fn func(DashboardModel, string) (DashboardModel, bool)) KeyHandler {
	return func(m DashboardModel, key string) (DashboardModel, tea.Cmd, bool) {
		next, handled := fn(m, key)
		return next, nil, handled
	}
}

var normalModeRegistry = buildNormalModeRegistry()

func buildNormalModeRegistry() *HandlerRegistry {
	reg := NewHandlerRegistry()

	register := func(key string, handler KeyHandler, desc string, priority int) {
		reg.Register(KeyBinding{
			Key:         key,
			Handler:     handler,
			Description: desc,
			Priority:    priority,
		})
	}

	// Navigation and scrolling.
	register("tab", wrapKeyHandler(DashboardModel.handleTabFocus), "", 0)
	register("right", wrapKeyHandler(DashboardModel.handleTabFocus), "", 0)
	register("l", wrapKeyHandler(DashboardModel.handleTabFocus), "", 0)
	register("shift+tab", wrapKeyHandler(DashboardModel.handleTabFocus), "", 0)
	register("left", wrapKeyHandler(DashboardModel.handleTabFocus), "", 0)
	register("h", wrapKeyHandler(DashboardModel.handleTabFocus), "", 0)
	register("up", wrapKeyHandler(DashboardModel.handleArrowKeys), "", 0)
	register("k", wrapKeyHandler(DashboardModel.handleArrowKeys), "", 0)
	register("down", wrapKeyHandler(DashboardModel.handleArrowKeys), "", 0)
	register("j", wrapKeyHandler(DashboardModel.handleArrowKeys), "", 0)
	register("G", wrapKeyHandler(DashboardModel.handleScrolling), "Graph", 0)

	// Goal operations.
	register("n", DashboardModel.handleGoalCreate, "New", 0)
	register("N", DashboardModel.handleGoalCreate, "Sub", 0)
	register("e", DashboardModel.handleGoalEdit, "Edit", 0)
	register("d", DashboardModel.handleGoalDelete, "Delete", 0)
	register("backspace", DashboardModel.handleGoalDelete, "", 0)
	register("m", DashboardModel.handleGoalMove, "Move", 0)
	register("z", DashboardModel.handleGoalExpandCollapse, "Toggle", 0)
	register("T", DashboardModel.handleGoalTaskTimer, "Task", 0)
	register("P", DashboardModel.handleGoalPriority, "Priority", 0)
	register("J", DashboardModel.handleGoalJournalStart, "Journal", 0)
	register("ctrl+j", DashboardModel.handleGoalJournalStart, "", 0)
	register("A", DashboardModel.handleGoalArchive, "Archive", 0)
	register("u", DashboardModel.handleGoalArchive, "Unarchive", 0)
	register("D", DashboardModel.handleGoalDependencyPicker, "Deps", 0)
	register("R", DashboardModel.handleGoalRecurrencePicker, "Repeat", 0)
	register(" ", DashboardModel.handleGoalStatusToggle, "", 0)
	register("t", DashboardModel.handleGoalTagging, "Tag", 0)

	// Sprint operations.
	register("s", DashboardModel.handleSprintPause, "", 10)
	register("s", DashboardModel.handleSprintStart, "", 5)
	register("x", DashboardModel.handleSprintReset, "", 0)

	// Workspace operations.
	register("+", DashboardModel.handleWorkspaceSprintCount, "Sprint", 0)
	register("-", DashboardModel.handleWorkspaceSprintCount, "Sprint", 0)
	register("w", DashboardModel.handleWorkspaceSwitch, "Cycle", 0)
	register("W", DashboardModel.handleWorkspaceCreate, "New WS", 0)
	register("b", DashboardModel.handleWorkspaceVisibility, "Backlog", 0)
	register("c", DashboardModel.handleWorkspaceVisibility, "Completed", 0)
	register("a", DashboardModel.handleWorkspaceVisibility, "Archived", 0)
	register("v", DashboardModel.handleWorkspaceViewMode, "View", 0)
	register("Y", DashboardModel.handleWorkspaceTheme, "Theme", 0)
	register("I", DashboardModel.handleWorkspaceSeedImport, "Import", 0)
	register("ctrl+r", DashboardModel.handleWorkspaceReport, "Report", 0)

	// Global controls.
	register("q", handleNormalQuit, "Quit", 0)
	register("ctrl+c", handleNormalQuit, "", 0)
	register("L", handleNormalLock, "Lock", 0)
	register("ctrl+e", handleNormalExport, "Export", 0)
	register("/", handleNormalSearch, "Search", 0)
	register("C", handleNormalClearDB, "Clear DB", 0)
	register("p", handleNormalPassphrase, "Passphrase", 0)
	register("<", handleNormalPrevDay, "Prev", 0)
	register(">", handleNormalNextDay, "Next", 0)

	return reg
}

func (m DashboardModel) handleNormalMode(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	key := msg.String()
	if next, cmd, handled := normalModeRegistry.Handle(m, key); handled {
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
