package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleLockedState(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			return m, nil
		}
		if msg.Type == tea.KeyEnter {
			entered := strings.TrimSpace(m.security.lock.PassphraseInput.Value())
			if limited, wait := m.passphraseRateLimited(); limited {
				remaining := wait.Round(time.Second)
				if remaining < time.Second {
					remaining = time.Second
				}
				m.security.lock.Message = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
				m.security.lock.PassphraseInput.Reset()
				m.security.lock.PassphraseInput.Focus()
				return m, nil
			}
			auth := newAuthHandler(m.db, m.ctx)
			result := auth.ValidatePassphrase(entered, m.security.lock.PassphraseHash)
			if result.Error != nil {
				m.setStatusError(result.Error.Error())
				return m, nil
			}
			if !result.Success {
				m.recordPassphraseFailure()
				if result.Message != "" {
					m.security.lock.Message = result.Message
				} else {
					m.security.lock.Message = "Incorrect passphrase"
				}
				m.security.lock.PassphraseInput.Reset()
				m.security.lock.PassphraseInput.Focus()
				if !result.ShouldRetry {
					return m, tea.Quit
				}
				return m, nil
			}
			if result.PassphraseHash != "" {
				m.security.lock.PassphraseHash = result.PassphraseHash
			}
			if result.StatusError != "" {
				m.setStatusError(result.StatusError)
			}
			m.security.lock.Locked = false
			m.security.lock.Message = ""
			m.security.lock.PassphraseInput.Reset()
			m.clearPassphraseFailures()
			m.security.lock.LastInput = time.Now()
			if m.day.ID > 0 {
				m.refreshData(m.day.ID)
			}
			return m, nil
		}
	}
	m.security.lock.PassphraseInput, cmd = m.security.lock.PassphraseInput.Update(msg)
	return m, cmd
}

func (m DashboardModel) inInputMode() bool {
	return m.security.changingPassphrase || m.modal.confirmingDelete || m.security.confirmingClearDB || m.modal.creatingGoal || m.modal.editingGoal || m.modal.journaling || m.search.Active || m.modal.creatingWorkspace || m.modal.initializingSprints || m.modal.tagging || m.modal.themePicking || m.modal.depPicking || m.modal.settingRecurrence
}

func (m DashboardModel) handleModalState(msg tea.Msg) (DashboardModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if next, cmd, handled := m.handleModalCancel(keyMsg); handled {
			return next, cmd
		}
		if next, cmd, handled := m.handleModalConfirm(keyMsg); handled {
			return next, cmd
		}
	}
	return m.handleModalInput(msg)
}

func (m DashboardModel) handleModalCancel(msg tea.KeyMsg) (DashboardModel, tea.Cmd, bool) {
	if msg.Type != tea.KeyEsc {
		return m, nil, false
	}
	m.modal.confirmingDelete = false
	m.modal.confirmDeleteGoalID = 0
	m.security.confirmingClearDB = false
	m.security.clearDBNeedsPass = false
	m.security.clearDBStatus = ""
	m.security.lock.PassphraseInput.Reset()
	m.security.changingPassphrase = false
	m.security.passphraseStage = 0
	m.security.passphraseStatus = ""
	m.inputs.passphraseCurrent.Reset()
	m.inputs.passphraseNew.Reset()
	m.inputs.passphraseConfirm.Reset()
	m.modal.creatingGoal, m.modal.editingGoal, m.modal.journaling, m.search.Active, m.modal.creatingWorkspace, m.modal.initializingSprints, m.modal.tagging, m.modal.themePicking, m.modal.depPicking, m.modal.settingRecurrence = false, false, false, false, false, false, false, false, false, false
	m.inputs.textInput.Reset()
	m.inputs.journalInput.Reset()
	m.search.Input.Reset()
	m.search.Cursor = 0
	m.search.ArchiveOnly = false
	m.inputs.tagInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalConfirm(msg tea.KeyMsg) (DashboardModel, tea.Cmd, bool) {
	if msg.Type != tea.KeyEnter {
		return m, nil, false
	}
	if m.modal.confirmingDelete {
		if m.modal.confirmDeleteGoalID > 0 {
			if err := m.db.DeleteGoal(m.ctx, m.modal.confirmDeleteGoalID); err != nil {
				m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
			} else {
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
		}
		m.modal.confirmingDelete = false
		m.modal.confirmDeleteGoalID = 0
		return m, nil, true
	}
	if m.security.confirmingClearDB {
		if m.security.clearDBNeedsPass {
			entered := strings.TrimSpace(m.security.lock.PassphraseInput.Value())
			if limited, wait := m.passphraseRateLimited(); limited {
				remaining := wait.Round(time.Second)
				if remaining < time.Second {
					remaining = time.Second
				}
				m.security.clearDBStatus = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
				m.security.lock.PassphraseInput.Reset()
				m.security.lock.PassphraseInput.Focus()
				return m, nil, true
			}
			if entered == "" {
				m.security.clearDBStatus = "Passphrase required"
				m.security.lock.PassphraseInput.Reset()
				m.security.lock.PassphraseInput.Focus()
				return m, nil, true
			}
			if util.HashPassphrase(entered) != m.security.lock.PassphraseHash {
				m.recordPassphraseFailure()
				m.security.clearDBStatus = "Incorrect passphrase"
				m.security.lock.PassphraseInput.Reset()
				m.security.lock.PassphraseInput.Focus()
				return m, nil, true
			}
			m.clearPassphraseFailures()
		}
		if err := m.db.ClearDatabase(m.ctx); err != nil {
			m.err = err
		} else {
			wsID, wsErr := m.db.EnsureDefaultWorkspace(m.ctx)
			if wsErr != nil {
				m.setStatusError(fmt.Sprintf("Error ensuring default workspace: %v", wsErr))
			} else if err := m.loadWorkspaces(); err != nil {
				m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
			} else {
				m.activeWorkspaceIdx = 0
				m.modal.pendingWorkspaceID = wsID
				m.modal.initializingSprints = true
				m.inputs.textInput.Placeholder = "How many sprints? (1-8)"
				m.inputs.textInput.Reset()
				m.inputs.textInput.Focus()
				m.security.lock.PassphraseHash = ""
				m.Message = "Database cleared. Set sprint count to start."
			}
		}
		m.security.confirmingClearDB = false
		m.security.clearDBNeedsPass = false
		m.security.clearDBStatus = ""
		m.security.lock.PassphraseInput.Reset()
		return m, nil, true
	}
	if m.security.changingPassphrase {
		switch m.security.passphraseStage {
		case 0:
			current := strings.TrimSpace(m.inputs.passphraseCurrent.Value())
			if limited, wait := m.passphraseRateLimited(); limited {
				remaining := wait.Round(time.Second)
				if remaining < time.Second {
					remaining = time.Second
				}
				m.security.passphraseStatus = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
				m.inputs.passphraseCurrent.Reset()
				m.inputs.passphraseCurrent.Focus()
				return m, nil, true
			}
			if m.security.lock.PassphraseHash != "" && util.HashPassphrase(current) != m.security.lock.PassphraseHash {
				m.recordPassphraseFailure()
				m.security.passphraseStatus = "Incorrect current passphrase"
				m.inputs.passphraseCurrent.Reset()
				m.inputs.passphraseCurrent.Focus()
				return m, nil, true
			}
			m.clearPassphraseFailures()
			m.security.passphraseStage = 1
			m.security.passphraseStatus = ""
			m.inputs.passphraseNew.Focus()
			return m, nil, true
		case 1:
			next := strings.TrimSpace(m.inputs.passphraseNew.Value())
			if next == "" {
				m.security.passphraseStatus = "New passphrase required"
				m.inputs.passphraseNew.Reset()
				m.inputs.passphraseNew.Focus()
				return m, nil, true
			}
			if err := util.ValidatePassphrase(next); err != nil {
				m.security.passphraseStatus = err.Error()
				m.inputs.passphraseNew.Reset()
				m.inputs.passphraseNew.Focus()
				return m, nil, true
			}
			m.security.passphraseStage = 2
			m.security.passphraseStatus = ""
			m.inputs.passphraseConfirm.Focus()
			return m, nil, true
		case 2:
			next := strings.TrimSpace(m.inputs.passphraseNew.Value())
			confirm := strings.TrimSpace(m.inputs.passphraseConfirm.Value())
			if next == "" {
				m.security.passphraseStatus = "New passphrase required"
				m.inputs.passphraseNew.Focus()
				return m, nil, true
			}
			if err := util.ValidatePassphrase(next); err != nil {
				m.security.passphraseStatus = err.Error()
				m.inputs.passphraseConfirm.Reset()
				m.inputs.passphraseConfirm.Focus()
				return m, nil, true
			}
			if next != confirm {
				m.security.passphraseStatus = "Passphrases do not match"
				m.inputs.passphraseConfirm.Reset()
				m.inputs.passphraseConfirm.Focus()
				return m, nil, true
			}
			if next == "" {
				m.security.passphraseStatus = "New passphrase required"
				m.inputs.passphraseNew.Reset()
				m.inputs.passphraseNew.Focus()
				return m, nil, true
			}
			encrypted := m.db.EncryptionStatus().DatabaseEncrypted
			encryptErr := error(nil)
			if encrypted {
				if err := m.db.RekeyDB(m.ctx, next); err != nil && err != database.ErrSQLCipherUnavailable {
					m.security.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", err)
					m.inputs.passphraseConfirm.Reset()
					m.inputs.passphraseConfirm.Focus()
					return m, nil, true
				} else if err == database.ErrSQLCipherUnavailable {
					encryptErr = err
					m.security.passphraseStatus = "SQLCipher unavailable; UI-only lock"
				}
			} else {
				if err := m.db.EncryptDatabase(m.ctx, next); err != nil && err != database.ErrSQLCipherUnavailable {
					if !m.db.DatabaseHasData(m.ctx) {
						if recErr := m.db.RecreateEncryptedDatabase(m.ctx, next); recErr != nil {
							m.security.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", recErr)
							m.inputs.passphraseConfirm.Reset()
							m.inputs.passphraseConfirm.Focus()
							return m, nil, true
						}
					} else {
						m.security.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", err)
						m.inputs.passphraseConfirm.Reset()
						m.inputs.passphraseConfirm.Focus()
						return m, nil, true
					}
				} else if err == database.ErrSQLCipherUnavailable {
					encryptErr = err
					m.security.passphraseStatus = "SQLCipher unavailable; UI-only lock"
				}
			}
			m.security.lock.PassphraseHash = util.HashPassphrase(next)
			if err := m.db.SetSetting(m.ctx, "passphrase_hash", m.security.lock.PassphraseHash); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
			}
			if encryptErr == database.ErrSQLCipherUnavailable {
				m.setStatusError("Encryption unavailable in this build; passphrase only locks the UI")
			} else if !m.db.EncryptionStatus().DatabaseEncrypted {
				m.setStatusError("Passphrase set but database remains unencrypted")
			}
			m.security.changingPassphrase = false
			m.security.passphraseStage = 0
			m.security.passphraseStatus = ""
			m.inputs.passphraseCurrent.Reset()
			m.inputs.passphraseNew.Reset()
			m.inputs.passphraseConfirm.Reset()
			m.clearPassphraseFailures()
			m.Message = "Passphrase updated."
			if m.day.ID > 0 {
				m.refreshData(m.day.ID)
			}
			return m, nil, true
		}
	}
	if m.search.Active {
		m.search.Active = false
		m.search.Input.Reset()
		m.search.Cursor = 0
		m.search.ArchiveOnly = false
		return m, nil, true
	}
	if m.modal.journaling {
		text := m.inputs.journalInput.Value()
		if strings.TrimSpace(text) != "" {
			var sID, gID *int64
			if m.timer.ActiveSprint != nil {
				id := m.timer.ActiveSprint.ID
				sID = &id
			}
			if m.modal.editingGoalID > 0 {
				id := m.modal.editingGoalID
				gID = &id
			}
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			if err := m.db.AddJournalEntry(m.ctx, m.day.ID, activeWS.ID, sID, gID, text); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving journal entry: %v", err))
			} else {
				m.refreshData(m.day.ID)
			}
		}
		m.modal.journaling, m.modal.editingGoalID = false, 0
		m.inputs.journalInput.Reset()
		return m, nil, true
	}
	if m.modal.creatingWorkspace {
		name := m.inputs.textInput.Value()
		if name != "" {
			newID, err := m.db.CreateWorkspace(m.ctx, name, strings.ToLower(name))
			if err == nil {
				m.modal.pendingWorkspaceID, m.modal.creatingWorkspace, m.modal.initializingSprints = newID, false, true
				m.inputs.textInput.Placeholder = "How many sprints?"
				m.inputs.textInput.Reset()
			} else {
				m.err = err
				m.modal.creatingWorkspace = false
			}
		}
		return m, nil, true
	}
	if m.modal.initializingSprints {
		val := m.inputs.textInput.Value()
		if num, err := strconv.Atoi(val); err == nil && num > 0 && num <= 8 {
			if err := m.db.BootstrapDay(m.ctx, m.modal.pendingWorkspaceID, num); err != nil {
				m.setStatusError(fmt.Sprintf("Error creating sprints: %v", err))
			} else if err := m.loadWorkspaces(); err != nil {
				m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
			} else {
				for i, ws := range m.workspaces {
					if ws.ID == m.modal.pendingWorkspaceID {
						m.activeWorkspaceIdx = i
						break
					}
				}
				if dayID := m.db.CheckCurrentDay(m.ctx); dayID > 0 {
					if day, err := m.db.GetDay(m.ctx, dayID); err == nil {
						m.day = day
					}
					m.refreshData(dayID)
				}
			}
		}
		m.modal.initializingSprints, m.modal.pendingWorkspaceID = false, 0
		m.inputs.textInput.Reset()
		return m, nil, true
	}
	if m.modal.tagging {
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
	if m.modal.themePicking {
		if len(m.modal.themeNames) > 0 && m.modal.themeCursor < len(m.modal.themeNames) {
			name := m.modal.themeNames[m.modal.themeCursor]
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			if err := m.db.UpdateWorkspaceTheme(m.ctx, activeWS.ID, name); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating workspace theme: %v", err))
			} else {
				m.workspaces[m.activeWorkspaceIdx].Theme = name
				SetTheme(name)
			}
		}
		m.modal.themePicking = false
		return m, nil, true
	}
	if m.modal.depPicking {
		var deps []int64
		for id, selected := range m.modal.depSelected {
			if selected {
				deps = append(deps, id)
			}
		}
		if m.modal.editingGoalID > 0 {
			if err := m.db.SetGoalDependencies(m.ctx, m.modal.editingGoalID, deps); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving dependencies: %v", err))
			} else {
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
		}
		m.modal.depPicking, m.modal.editingGoalID = false, 0
		m.modal.depSelected = make(map[int64]bool)
		return m, nil, true
	}
	if m.modal.settingRecurrence {
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
	text := m.inputs.textInput.Value()
	if text != "" {
		if m.modal.editingGoal {
			if err := m.db.EditGoal(m.ctx, m.modal.editingGoalID, text); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating goal: %v", err))
			}
		} else if m.modal.editingGoalID > 0 {
			if err := m.db.AddSubtask(m.ctx, text, m.modal.editingGoalID); err != nil {
				m.setStatusError(fmt.Sprintf("Error adding subtask: %v", err))
			} else {
				m.view.expandedState[m.modal.editingGoalID] = true
			}
		} else {
			if err := m.db.AddGoal(m.ctx, m.workspaces[m.activeWorkspaceIdx].ID, text, m.sprints[m.view.focusedColIdx].ID); err != nil {
				m.setStatusError(fmt.Sprintf("Error adding goal: %v", err))
			}
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	m.modal.creatingGoal, m.modal.editingGoal, m.modal.editingGoalID = false, false, 0
	m.inputs.textInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalInput(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.security.confirmingClearDB && m.security.clearDBNeedsPass {
		m.security.lock.PassphraseInput, cmd = m.security.lock.PassphraseInput.Update(msg)
		return m, cmd
	}
	if m.security.changingPassphrase {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch m.security.passphraseStage {
			case 0:
				m.inputs.passphraseCurrent, cmd = m.inputs.passphraseCurrent.Update(msg)
			case 1:
				m.inputs.passphraseNew, cmd = m.inputs.passphraseNew.Update(msg)
			case 2:
				m.inputs.passphraseConfirm, cmd = m.inputs.passphraseConfirm.Update(msg)
			}
		}
		return m, cmd
	}
	if m.modal.confirmingDelete {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "a":
				if m.modal.confirmDeleteGoalID > 0 {
					if err := m.db.ArchiveGoal(m.ctx, m.modal.confirmDeleteGoalID); err != nil {
						m.setStatusError(fmt.Sprintf("Error archiving goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
				}
				m.modal.confirmingDelete = false
				m.modal.confirmDeleteGoalID = 0
				return m, nil
			case "d", "backspace":
				if m.modal.confirmDeleteGoalID > 0 {
					if err := m.db.DeleteGoal(m.ctx, m.modal.confirmDeleteGoalID); err != nil {
						m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
				}
				m.modal.confirmingDelete = false
				m.modal.confirmDeleteGoalID = 0
				return m, nil
			}
		}
		return m, nil
	}
	if m.security.confirmingClearDB {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "c":
				if m.security.clearDBNeedsPass {
					return m, nil
				}
				if err := m.db.ClearDatabase(m.ctx); err != nil {
					m.err = err
				} else {
					wsID, wsErr := m.db.EnsureDefaultWorkspace(m.ctx)
					if wsErr != nil {
						m.setStatusError(fmt.Sprintf("Error ensuring default workspace: %v", wsErr))
					} else if err := m.loadWorkspaces(); err != nil {
						m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
					} else {
						m.activeWorkspaceIdx = 0
						m.modal.pendingWorkspaceID = wsID
						m.modal.initializingSprints = true
						m.inputs.textInput.Placeholder = "How many sprints? (1-8)"
						m.inputs.textInput.Reset()
						m.inputs.textInput.Focus()
						m.security.lock.PassphraseHash = ""
						m.Message = "Database cleared. Set sprint count to start."
					}
				}
				m.security.confirmingClearDB = false
				m.security.clearDBNeedsPass = false
				m.security.clearDBStatus = ""
				m.security.lock.PassphraseInput.Reset()
				return m, nil
			}
		}
		return m, nil
	}
	if m.modal.settingRecurrence {
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
				return m, nil
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
						return m, nil
					}
					if m.modal.recurrenceDayCursor < maxDay-1 {
						m.modal.recurrenceDayCursor++
					}
				}
				return m, nil
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
				return m, nil
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
				return m, nil
			}
		}
		return m, nil
	}
	if m.modal.depPicking {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.modal.depCursor > 0 {
					m.modal.depCursor--
				}
				return m, nil
			case "down", "j":
				if m.modal.depCursor < len(m.modal.depOptions)-1 {
					m.modal.depCursor++
				}
				return m, nil
			case " ":
				if len(m.modal.depOptions) > 0 && m.modal.depCursor < len(m.modal.depOptions) {
					id := m.modal.depOptions[m.modal.depCursor].ID
					m.modal.depSelected[id] = !m.modal.depSelected[id]
				}
				return m, nil
			}
		}
		return m, nil
	}
	if m.modal.themePicking {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.modal.themeCursor > 0 {
					m.modal.themeCursor--
				}
				return m, nil
			case "down", "j":
				if m.modal.themeCursor < len(m.modal.themeNames)-1 {
					m.modal.themeCursor++
				}
				return m, nil
			}
		}
		return m, nil
	}
	if m.modal.tagging {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.modal.tagCursor > 0 {
					m.modal.tagCursor--
				}
				return m, nil
			case "down", "j":
				if m.modal.tagCursor < len(m.modal.defaultTags)-1 {
					m.modal.tagCursor++
				}
				return m, nil
			case "tab":
				if len(m.modal.defaultTags) > 0 && m.modal.tagCursor < len(m.modal.defaultTags) {
					tag := m.modal.defaultTags[m.modal.tagCursor]
					m.modal.tagSelected[tag] = !m.modal.tagSelected[tag]
				}
				return m, nil
			}
		}
		m.inputs.tagInput, cmd = m.inputs.tagInput.Update(msg)
		return m, cmd
	}
	if m.search.Active {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up", "k":
				if m.search.Cursor > 0 {
					m.search.Cursor--
				}
				return m, nil
			case "down", "j":
				if m.search.Cursor < len(m.search.Results)-1 {
					m.search.Cursor++
				}
				return m, nil
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
				return m, nil
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
		return m, cmd
	}
	if m.modal.journaling {
		m.inputs.journalInput, cmd = m.inputs.journalInput.Update(msg)
		return m, cmd
	}
	m.inputs.textInput, cmd = m.inputs.textInput.Update(msg)
	return m, cmd
}
