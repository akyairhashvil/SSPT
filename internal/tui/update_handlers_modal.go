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
			entered := strings.TrimSpace(m.lock.PassphraseInput.Value())
			if limited, wait := m.passphraseRateLimited(); limited {
				remaining := wait.Round(time.Second)
				if remaining < time.Second {
					remaining = time.Second
				}
				m.lock.Message = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
				m.lock.PassphraseInput.Reset()
				m.lock.PassphraseInput.Focus()
				return m, nil
			}
			if m.lock.PassphraseHash == "" {
				if entered == "" {
					m.lock.Message = "Passphrase required"
					m.lock.PassphraseInput.Reset()
					m.lock.PassphraseInput.Focus()
					return m, nil
				}
				if err := util.ValidatePassphrase(entered); err != nil {
					m.lock.Message = err.Error()
					m.lock.PassphraseInput.Reset()
					m.lock.PassphraseInput.Focus()
					entered = ""
					return m, nil
				}
				encInfo := m.db.EncryptionStatus()
				encrypted := encInfo.DatabaseEncrypted
				encryptErr := error(nil)
				if !encrypted {
					if err := m.db.EncryptDatabase(m.ctx, entered); err != nil && err != database.ErrSQLCipherUnavailable {
						encryptErr = err
						if !m.db.DatabaseHasData(m.ctx) {
							if recErr := m.db.RecreateEncryptedDatabase(m.ctx, entered); recErr != nil {
								m.lock.Message = fmt.Sprintf("Failed to encrypt database: %v", recErr)
								m.lock.PassphraseInput.Reset()
								m.lock.PassphraseInput.Focus()
								entered = ""
								return m, nil
							}
						} else {
							m.lock.Message = fmt.Sprintf("Failed to encrypt database: %v", err)
							m.lock.PassphraseInput.Reset()
							m.lock.PassphraseInput.Focus()
							entered = ""
							return m, nil
						}
					} else if err == database.ErrSQLCipherUnavailable {
						encryptErr = err
						m.lock.Message = "SQLCipher unavailable; UI-only lock"
					}
				}
				m.lock.PassphraseHash = util.HashPassphrase(entered)
				if err := m.db.SetSetting(m.ctx, "passphrase_hash", m.lock.PassphraseHash); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
				}
				if encryptErr == database.ErrSQLCipherUnavailable {
					m.setStatusError("Encryption unavailable in this build; passphrase only locks the UI")
				} else if !m.db.EncryptionStatus().DatabaseEncrypted {
					m.setStatusError("Passphrase set but database remains unencrypted")
				}
				m.lock.Locked = false
				m.lock.Message = ""
				m.lock.PassphraseInput.Reset()
				m.clearPassphraseFailures()
				m.lock.LastInput = time.Now()
				entered = ""
				if m.day.ID > 0 {
					m.refreshData(m.day.ID)
				}
				return m, nil
			}
			if entered != "" && util.HashPassphrase(entered) == m.lock.PassphraseHash {
				m.lock.Locked = false
				m.lock.Message = ""
				m.lock.PassphraseInput.Reset()
				m.clearPassphraseFailures()
				m.lock.LastInput = time.Now()
				entered = ""
				return m, nil
			}
			m.recordPassphraseFailure()
			m.lock.Message = "Incorrect passphrase"
			m.lock.PassphraseInput.Reset()
			m.lock.PassphraseInput.Focus()
			entered = ""
			return m, nil
		}
	}
	m.lock.PassphraseInput, cmd = m.lock.PassphraseInput.Update(msg)
	return m, cmd
}

func (m DashboardModel) inInputMode() bool {
	return m.changingPassphrase || m.confirmingDelete || m.confirmingClearDB || m.creatingGoal || m.editingGoal || m.journaling || m.search.Active || m.creatingWorkspace || m.initializingSprints || m.tagging || m.themePicking || m.depPicking || m.settingRecurrence
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
	m.confirmingDelete = false
	m.confirmDeleteGoalID = 0
	m.confirmingClearDB = false
	m.clearDBNeedsPass = false
	m.clearDBStatus = ""
	m.lock.PassphraseInput.Reset()
	m.changingPassphrase = false
	m.passphraseStage = 0
	m.passphraseStatus = ""
	m.passphraseCurrent.Reset()
	m.passphraseNew.Reset()
	m.passphraseConfirm.Reset()
	m.creatingGoal, m.editingGoal, m.journaling, m.search.Active, m.creatingWorkspace, m.initializingSprints, m.tagging, m.themePicking, m.depPicking, m.settingRecurrence = false, false, false, false, false, false, false, false, false, false
	m.textInput.Reset()
	m.journalInput.Reset()
	m.search.Input.Reset()
	m.search.Cursor = 0
	m.search.ArchiveOnly = false
	m.tagInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalConfirm(msg tea.KeyMsg) (DashboardModel, tea.Cmd, bool) {
	if msg.Type != tea.KeyEnter {
		return m, nil, false
	}
	if m.confirmingDelete {
		if m.confirmDeleteGoalID > 0 {
			if err := m.db.DeleteGoal(m.ctx, m.confirmDeleteGoalID); err != nil {
				m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
			} else {
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
		}
		m.confirmingDelete = false
		m.confirmDeleteGoalID = 0
		return m, nil, true
	}
	if m.confirmingClearDB {
		if m.clearDBNeedsPass {
			entered := strings.TrimSpace(m.lock.PassphraseInput.Value())
			if limited, wait := m.passphraseRateLimited(); limited {
				remaining := wait.Round(time.Second)
				if remaining < time.Second {
					remaining = time.Second
				}
				m.clearDBStatus = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
				m.lock.PassphraseInput.Reset()
				m.lock.PassphraseInput.Focus()
				return m, nil, true
			}
			if entered == "" {
				m.clearDBStatus = "Passphrase required"
				m.lock.PassphraseInput.Reset()
				m.lock.PassphraseInput.Focus()
				return m, nil, true
			}
			if util.HashPassphrase(entered) != m.lock.PassphraseHash {
				m.recordPassphraseFailure()
				m.clearDBStatus = "Incorrect passphrase"
				m.lock.PassphraseInput.Reset()
				m.lock.PassphraseInput.Focus()
				entered = ""
				return m, nil, true
			}
			m.clearPassphraseFailures()
			entered = ""
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
				m.pendingWorkspaceID = wsID
				m.initializingSprints = true
				m.textInput.Placeholder = "How many sprints? (1-8)"
				m.textInput.Reset()
				m.textInput.Focus()
				m.lock.PassphraseHash = ""
				m.Message = "Database cleared. Set sprint count to start."
			}
		}
		m.confirmingClearDB = false
		m.clearDBNeedsPass = false
		m.clearDBStatus = ""
		m.lock.PassphraseInput.Reset()
		return m, nil, true
	}
	if m.changingPassphrase {
		switch m.passphraseStage {
		case 0:
			current := strings.TrimSpace(m.passphraseCurrent.Value())
			if limited, wait := m.passphraseRateLimited(); limited {
				remaining := wait.Round(time.Second)
				if remaining < time.Second {
					remaining = time.Second
				}
				m.passphraseStatus = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
				m.passphraseCurrent.Reset()
				m.passphraseCurrent.Focus()
				return m, nil, true
			}
			if m.lock.PassphraseHash != "" && util.HashPassphrase(current) != m.lock.PassphraseHash {
				m.recordPassphraseFailure()
				m.passphraseStatus = "Incorrect current passphrase"
				m.passphraseCurrent.Reset()
				m.passphraseCurrent.Focus()
				current = ""
				return m, nil, true
			}
			m.clearPassphraseFailures()
			current = ""
			m.passphraseStage = 1
			m.passphraseStatus = ""
			m.passphraseNew.Focus()
			return m, nil, true
		case 1:
			next := strings.TrimSpace(m.passphraseNew.Value())
			if next == "" {
				m.passphraseStatus = "New passphrase required"
				m.passphraseNew.Reset()
				m.passphraseNew.Focus()
				return m, nil, true
			}
			if err := util.ValidatePassphrase(next); err != nil {
				m.passphraseStatus = err.Error()
				m.passphraseNew.Reset()
				m.passphraseNew.Focus()
				next = ""
				return m, nil, true
			}
			m.passphraseStage = 2
			m.passphraseStatus = ""
			m.passphraseConfirm.Focus()
			return m, nil, true
		case 2:
			next := strings.TrimSpace(m.passphraseNew.Value())
			confirm := strings.TrimSpace(m.passphraseConfirm.Value())
			if next == "" {
				m.passphraseStatus = "New passphrase required"
				m.passphraseNew.Focus()
				next = ""
				confirm = ""
				return m, nil, true
			}
			if err := util.ValidatePassphrase(next); err != nil {
				m.passphraseStatus = err.Error()
				m.passphraseConfirm.Reset()
				m.passphraseConfirm.Focus()
				next = ""
				confirm = ""
				return m, nil, true
			}
			if next != confirm {
				m.passphraseStatus = "Passphrases do not match"
				m.passphraseConfirm.Reset()
				m.passphraseConfirm.Focus()
				next = ""
				confirm = ""
				return m, nil, true
			}
			if next == "" {
				m.passphraseStatus = "New passphrase required"
				m.passphraseNew.Reset()
				m.passphraseNew.Focus()
				next = ""
				confirm = ""
				return m, nil, true
			}
			encrypted := m.db.EncryptionStatus().DatabaseEncrypted
			encryptErr := error(nil)
			if encrypted {
				if err := m.db.RekeyDB(m.ctx, next); err != nil && err != database.ErrSQLCipherUnavailable {
					encryptErr = err
					m.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", err)
					m.passphraseConfirm.Reset()
					m.passphraseConfirm.Focus()
					return m, nil, true
				} else if err == database.ErrSQLCipherUnavailable {
					encryptErr = err
					m.passphraseStatus = "SQLCipher unavailable; UI-only lock"
				}
			} else {
				if err := m.db.EncryptDatabase(m.ctx, next); err != nil && err != database.ErrSQLCipherUnavailable {
					encryptErr = err
					if !m.db.DatabaseHasData(m.ctx) {
						if recErr := m.db.RecreateEncryptedDatabase(m.ctx, next); recErr != nil {
							m.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", recErr)
							m.passphraseConfirm.Reset()
							m.passphraseConfirm.Focus()
							return m, nil, true
						}
					} else {
						m.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", err)
						m.passphraseConfirm.Reset()
						m.passphraseConfirm.Focus()
						return m, nil, true
					}
				} else if err == database.ErrSQLCipherUnavailable {
					encryptErr = err
					m.passphraseStatus = "SQLCipher unavailable; UI-only lock"
				}
			}
			m.lock.PassphraseHash = util.HashPassphrase(next)
			if err := m.db.SetSetting(m.ctx, "passphrase_hash", m.lock.PassphraseHash); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
			}
			if encryptErr == database.ErrSQLCipherUnavailable {
				m.setStatusError("Encryption unavailable in this build; passphrase only locks the UI")
			} else if !m.db.EncryptionStatus().DatabaseEncrypted {
				m.setStatusError("Passphrase set but database remains unencrypted")
			}
			m.changingPassphrase = false
			m.passphraseStage = 0
			m.passphraseStatus = ""
			m.passphraseCurrent.Reset()
			m.passphraseNew.Reset()
			m.passphraseConfirm.Reset()
			m.clearPassphraseFailures()
			next = ""
			confirm = ""
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
	if m.journaling {
		text := m.journalInput.Value()
		if strings.TrimSpace(text) != "" {
			var sID, gID *int64
			if m.timer.ActiveSprint != nil {
				id := m.timer.ActiveSprint.ID
				sID = &id
			}
			if m.editingGoalID > 0 {
				id := m.editingGoalID
				gID = &id
			}
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			if err := m.db.AddJournalEntry(m.ctx, m.day.ID, activeWS.ID, sID, gID, text); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving journal entry: %v", err))
			} else {
				m.refreshData(m.day.ID)
			}
		}
		m.journaling, m.editingGoalID = false, 0
		m.journalInput.Reset()
		return m, nil, true
	}
	if m.creatingWorkspace {
		name := m.textInput.Value()
		if name != "" {
			newID, err := m.db.CreateWorkspace(m.ctx, name, strings.ToLower(name))
			if err == nil {
				m.pendingWorkspaceID, m.creatingWorkspace, m.initializingSprints = newID, false, true
				m.textInput.Placeholder = "How many sprints?"
				m.textInput.Reset()
			} else {
				m.err = err
				m.creatingWorkspace = false
			}
		}
		return m, nil, true
	}
	if m.initializingSprints {
		val := m.textInput.Value()
		if num, err := strconv.Atoi(val); err == nil && num > 0 && num <= 8 {
			if err := m.db.BootstrapDay(m.ctx, m.pendingWorkspaceID, num); err != nil {
				m.setStatusError(fmt.Sprintf("Error creating sprints: %v", err))
			} else if err := m.loadWorkspaces(); err != nil {
				m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
			} else {
				for i, ws := range m.workspaces {
					if ws.ID == m.pendingWorkspaceID {
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
		m.initializingSprints, m.pendingWorkspaceID = false, 0
		m.textInput.Reset()
		return m, nil, true
	}
	if m.tagging {
		raw := strings.Fields(m.tagInput.Value())
		tags := make(map[string]bool)
		for t, selected := range m.tagSelected {
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
		if err := m.db.SetGoalTags(m.ctx, m.editingGoalID, out); err != nil {
			m.setStatusError(fmt.Sprintf("Error saving tags: %v", err))
		} else {
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
		m.tagging, m.editingGoalID = false, 0
		m.tagInput.Reset()
		m.tagSelected = make(map[string]bool)
		m.tagCursor = 0
		return m, nil, true
	}
	if m.themePicking {
		if len(m.themeNames) > 0 && m.themeCursor < len(m.themeNames) {
			name := m.themeNames[m.themeCursor]
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			if err := m.db.UpdateWorkspaceTheme(m.ctx, activeWS.ID, name); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating workspace theme: %v", err))
			} else {
				m.workspaces[m.activeWorkspaceIdx].Theme = name
				SetTheme(name)
			}
		}
		m.themePicking = false
		return m, nil, true
	}
	if m.depPicking {
		var deps []int64
		for id, selected := range m.depSelected {
			if selected {
				deps = append(deps, id)
			}
		}
		if m.editingGoalID > 0 {
			if err := m.db.SetGoalDependencies(m.ctx, m.editingGoalID, deps); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving dependencies: %v", err))
			} else {
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
		}
		m.depPicking, m.editingGoalID = false, 0
		m.depSelected = make(map[int64]bool)
		return m, nil, true
	}
	if m.settingRecurrence {
		if m.editingGoalID > 0 {
			rule := m.recurrenceMode
			switch rule {
			case "none":
				if err := m.db.UpdateGoalRecurrence(m.ctx, m.editingGoalID, ""); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
				}
			case "daily":
				if err := m.db.UpdateGoalRecurrence(m.ctx, m.editingGoalID, "daily"); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
				}
			case "weekly":
				var days []string
				for _, d := range m.weekdayOptions {
					if m.recurrenceSelected[d] {
						days = append(days, d)
					}
				}
				if len(days) == 0 {
					m.Message = "Select at least one weekday."
				} else {
					if err := m.db.UpdateGoalRecurrence(m.ctx, m.editingGoalID, "weekly:"+strings.Join(days, ",")); err != nil {
						m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
					}
				}
			case "monthly":
				var months []string
				var days []string
				for _, mo := range m.monthOptions {
					if m.recurrenceSelected[mo] {
						months = append(months, mo)
					}
				}
				for _, d := range m.monthDayOptions {
					if m.recurrenceSelected["day:"+d] {
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
					if err := m.db.UpdateGoalRecurrence(m.ctx, m.editingGoalID, rule); err != nil {
						m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
					}
				}
			}
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		}
		m.settingRecurrence, m.editingGoalID = false, 0
		m.recurrenceSelected = make(map[string]bool)
		return m, nil, true
	}
	text := m.textInput.Value()
	if text != "" {
		if m.editingGoal {
			if err := m.db.EditGoal(m.ctx, m.editingGoalID, text); err != nil {
				m.setStatusError(fmt.Sprintf("Error updating goal: %v", err))
			}
		} else if m.editingGoalID > 0 {
			if err := m.db.AddSubtask(m.ctx, text, m.editingGoalID); err != nil {
				m.setStatusError(fmt.Sprintf("Error adding subtask: %v", err))
			} else {
				m.expandedState[m.editingGoalID] = true
			}
		} else {
			if err := m.db.AddGoal(m.ctx, m.workspaces[m.activeWorkspaceIdx].ID, text, m.sprints[m.focusedColIdx].ID); err != nil {
				m.setStatusError(fmt.Sprintf("Error adding goal: %v", err))
			}
		}
		m.invalidateGoalCache()
		m.refreshData(m.day.ID)
	}
	m.creatingGoal, m.editingGoal, m.editingGoalID = false, false, 0
	m.textInput.Reset()
	return m, nil, true
}

func (m DashboardModel) handleModalInput(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.confirmingClearDB && m.clearDBNeedsPass {
		m.lock.PassphraseInput, cmd = m.lock.PassphraseInput.Update(msg)
		return m, cmd
	}
	if m.changingPassphrase {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch m.passphraseStage {
			case 0:
				m.passphraseCurrent, cmd = m.passphraseCurrent.Update(msg)
			case 1:
				m.passphraseNew, cmd = m.passphraseNew.Update(msg)
			case 2:
				m.passphraseConfirm, cmd = m.passphraseConfirm.Update(msg)
			}
		}
		return m, cmd
	}
	if m.confirmingDelete {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "a":
				if m.confirmDeleteGoalID > 0 {
					if err := m.db.ArchiveGoal(m.ctx, m.confirmDeleteGoalID); err != nil {
						m.setStatusError(fmt.Sprintf("Error archiving goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
				}
				m.confirmingDelete = false
				m.confirmDeleteGoalID = 0
				return m, nil
			case "d", "backspace":
				if m.confirmDeleteGoalID > 0 {
					if err := m.db.DeleteGoal(m.ctx, m.confirmDeleteGoalID); err != nil {
						m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
				}
				m.confirmingDelete = false
				m.confirmDeleteGoalID = 0
				return m, nil
			}
		}
		return m, nil
	}
	if m.confirmingClearDB {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "c":
				if m.clearDBNeedsPass {
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
						m.pendingWorkspaceID = wsID
						m.initializingSprints = true
						m.textInput.Placeholder = "How many sprints? (1-8)"
						m.textInput.Reset()
						m.textInput.Focus()
						m.lock.PassphraseHash = ""
						m.Message = "Database cleared. Set sprint count to start."
					}
				}
				m.confirmingClearDB = false
				m.clearDBNeedsPass = false
				m.clearDBStatus = ""
				m.lock.PassphraseInput.Reset()
				return m, nil
			}
		}
		return m, nil
	}
	if m.settingRecurrence {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.recurrenceFocus == "mode" {
					if m.recurrenceCursor > 0 {
						m.recurrenceCursor--
					}
				} else if m.recurrenceFocus == "items" {
					if m.recurrenceItemCursor > 0 {
						m.recurrenceItemCursor--
					}
				} else if m.recurrenceFocus == "days" {
					if m.recurrenceDayCursor > 0 {
						m.recurrenceDayCursor--
					}
				}
				return m, nil
			case "down", "j":
				if m.recurrenceFocus == "mode" {
					if m.recurrenceCursor < len(m.recurrenceOptions)-1 {
						m.recurrenceCursor++
					}
				} else if m.recurrenceFocus == "items" {
					max := 0
					if m.recurrenceMode == "weekly" {
						max = len(m.weekdayOptions) - 1
					} else if m.recurrenceMode == "monthly" {
						max = len(m.monthOptions) - 1
					}
					if m.recurrenceItemCursor < max {
						m.recurrenceItemCursor++
					}
				} else if m.recurrenceFocus == "days" {
					maxDay := m.monthlyMaxDay()
					if maxDay <= 0 {
						return m, nil
					}
					if m.recurrenceDayCursor < maxDay-1 {
						m.recurrenceDayCursor++
					}
				}
				return m, nil
			case "tab":
				if m.recurrenceFocus == "items" && m.recurrenceMode == "monthly" {
					m.recurrenceFocus = "days"
				} else if m.recurrenceFocus == "days" {
					m.recurrenceFocus = "mode"
				} else if m.recurrenceFocus == "items" {
					m.recurrenceFocus = "mode"
				} else if len(m.recurrenceOptions) > 0 && m.recurrenceCursor < len(m.recurrenceOptions) {
					m.recurrenceMode = m.recurrenceOptions[m.recurrenceCursor]
					if m.recurrenceMode == "weekly" || m.recurrenceMode == "monthly" {
						m.recurrenceFocus = "items"
					} else {
						m.recurrenceFocus = "mode"
					}
				}
				return m, nil
			case " ":
				if m.recurrenceFocus == "items" {
					switch m.recurrenceMode {
					case "weekly":
						if m.recurrenceItemCursor < len(m.weekdayOptions) {
							key := m.weekdayOptions[m.recurrenceItemCursor]
							m.recurrenceSelected[key] = !m.recurrenceSelected[key]
						}
					case "monthly":
						if m.recurrenceItemCursor < len(m.monthOptions) {
							key := m.monthOptions[m.recurrenceItemCursor]
							m.recurrenceSelected[key] = !m.recurrenceSelected[key]
							m.pruneMonthlyDays(m.monthlyMaxDay())
						}
					}
				} else if m.recurrenceFocus == "days" {
					maxDay := m.monthlyMaxDay()
					if maxDay > 0 && m.recurrenceDayCursor < maxDay {
						key := "day:" + m.monthDayOptions[m.recurrenceDayCursor]
						m.recurrenceSelected[key] = !m.recurrenceSelected[key]
					}
				} else if m.recurrenceFocus == "mode" {
					if len(m.recurrenceOptions) > 0 && m.recurrenceCursor < len(m.recurrenceOptions) {
						m.recurrenceMode = m.recurrenceOptions[m.recurrenceCursor]
					}
				}
				return m, nil
			}
		}
		return m, nil
	}
	if m.depPicking {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.depCursor > 0 {
					m.depCursor--
				}
				return m, nil
			case "down", "j":
				if m.depCursor < len(m.depOptions)-1 {
					m.depCursor++
				}
				return m, nil
			case " ":
				if len(m.depOptions) > 0 && m.depCursor < len(m.depOptions) {
					id := m.depOptions[m.depCursor].ID
					m.depSelected[id] = !m.depSelected[id]
				}
				return m, nil
			}
		}
		return m, nil
	}
	if m.themePicking {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.themeCursor > 0 {
					m.themeCursor--
				}
				return m, nil
			case "down", "j":
				if m.themeCursor < len(m.themeNames)-1 {
					m.themeCursor++
				}
				return m, nil
			}
		}
		return m, nil
	}
	if m.tagging {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.tagCursor > 0 {
					m.tagCursor--
				}
				return m, nil
			case "down", "j":
				if m.tagCursor < len(m.defaultTags)-1 {
					m.tagCursor++
				}
				return m, nil
			case "tab":
				if len(m.defaultTags) > 0 && m.tagCursor < len(m.defaultTags) {
					tag := m.defaultTags[m.tagCursor]
					m.tagSelected[tag] = !m.tagSelected[tag]
				}
				return m, nil
			}
		}
		m.tagInput, cmd = m.tagInput.Update(msg)
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
	if m.journaling {
		m.journalInput, cmd = m.journalInput.Update(msg)
		return m, cmd
	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}
