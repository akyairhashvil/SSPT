package tui

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/util"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
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
	case TickMsg:
		if !m.locked && m.passphraseHash != "" && time.Since(m.lastInput) >= AutoLockAfter {
			m.locked = true
			m.lockMessage = "Session locked (idle)"
			m.passphraseInput.Reset()
			m.passphraseInput.Focus()
			return m, nil
		}
		if m.breakActive {
			if time.Since(m.breakStart) >= BreakDuration {
				m.breakActive = false
			}
			return m, tickCmd()
		}
		if m.activeSprint != nil {
			elapsed := time.Since(m.activeSprint.StartTime.Time) + (time.Duration(m.activeSprint.ElapsedSeconds) * time.Second)
			if elapsed >= SprintDuration {
				if err := database.CompleteSprint(m.activeSprint.ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error completing sprint: %v", err))
					return m, tickCmd()
				}
				if err := database.MovePendingToBacklog(m.activeSprint.ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error moving pending tasks: %v", err))
				}
				m.activeSprint, m.breakActive, m.breakStart = nil, true, time.Now()
				m.refreshData(m.day.ID)
				return m, tickCmd()
			}
			newProg, _ := m.progress.Update(msg)
			m.progress = newProg.(progress.Model)
			return m, tickCmd()
		}
		return m, tickCmd()
	}

	if m.locked {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				entered := strings.TrimSpace(m.passphraseInput.Value())
				if limited, wait := m.passphraseRateLimited(); limited {
					remaining := wait.Round(time.Second)
					if remaining < time.Second {
						remaining = time.Second
					}
					m.lockMessage = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
					m.passphraseInput.Reset()
					m.passphraseInput.Focus()
					return m, nil
				}
				if m.passphraseHash == "" {
					if entered == "" {
						m.lockMessage = "Passphrase required"
						m.passphraseInput.Reset()
						m.passphraseInput.Focus()
						return m, nil
					}
					if err := util.ValidatePassphrase(entered); err != nil {
						m.lockMessage = err.Error()
						m.passphraseInput.Reset()
						m.passphraseInput.Focus()
						entered = ""
						return m, nil
					}
					_, encrypted, _ := database.EncryptionStatus()
					encryptErr := error(nil)
					if !encrypted {
						if err := database.EncryptDatabase(entered); err != nil && err != database.ErrSQLCipherUnavailable {
							encryptErr = err
							if !database.DatabaseHasData() {
								if recErr := database.RecreateEncryptedDatabase(entered); recErr != nil {
									m.lockMessage = fmt.Sprintf("Failed to encrypt database: %v", recErr)
									m.passphraseInput.Reset()
									m.passphraseInput.Focus()
									entered = ""
									return m, nil
								}
							} else {
								m.lockMessage = fmt.Sprintf("Failed to encrypt database: %v", err)
								m.passphraseInput.Reset()
								m.passphraseInput.Focus()
								entered = ""
								return m, nil
							}
						} else if err == database.ErrSQLCipherUnavailable {
							encryptErr = err
							m.lockMessage = "SQLCipher unavailable; UI-only lock"
						}
					}
					m.passphraseHash = util.HashPassphrase(entered)
					if err := database.SetSetting("passphrase_hash", m.passphraseHash); err != nil {
						m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
					}
					if encryptErr == database.ErrSQLCipherUnavailable {
						m.setStatusError("Encryption unavailable in this build; passphrase only locks the UI")
					} else if _, encrypted, _ := database.EncryptionStatus(); !encrypted {
						m.setStatusError("Passphrase set but database remains unencrypted")
					}
					m.locked = false
					m.lockMessage = ""
					m.passphraseInput.Reset()
					m.clearPassphraseFailures()
					m.lastInput = time.Now()
					entered = ""
					if m.day.ID > 0 {
						m.refreshData(m.day.ID)
					}
					return m, nil
				}
				if entered != "" && util.HashPassphrase(entered) == m.passphraseHash {
					m.locked = false
					m.lockMessage = ""
					m.passphraseInput.Reset()
					m.clearPassphraseFailures()
					m.lastInput = time.Now()
					entered = ""
					return m, nil
				}
				m.recordPassphraseFailure()
				m.lockMessage = "Incorrect passphrase"
				m.passphraseInput.Reset()
				m.passphraseInput.Focus()
				entered = ""
				return m, nil
			}
		}
		m.passphraseInput, cmd = m.passphraseInput.Update(msg)
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type != tea.KeyNull {
			m.lastInput = time.Now()
		}
	}

	// Input Modes
	if m.changingPassphrase || m.confirmingDelete || m.confirmingClearDB || m.creatingGoal || m.editingGoal || m.journaling || m.searching || m.creatingWorkspace || m.initializingSprints || m.tagging || m.themePicking || m.depPicking || m.settingRecurrence {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.confirmingDelete = false
				m.confirmDeleteGoalID = 0
				m.confirmingClearDB = false
				m.clearDBNeedsPass = false
				m.clearDBStatus = ""
				m.passphraseInput.Reset()
				m.changingPassphrase = false
				m.passphraseStage = 0
				m.passphraseStatus = ""
				m.passphraseCurrent.Reset()
				m.passphraseNew.Reset()
				m.passphraseConfirm.Reset()
				m.creatingGoal, m.editingGoal, m.journaling, m.searching, m.creatingWorkspace, m.initializingSprints, m.tagging, m.themePicking, m.depPicking, m.settingRecurrence = false, false, false, false, false, false, false, false, false, false
				m.textInput.Reset()
				m.journalInput.Reset()
				m.searchInput.Reset()
				m.searchCursor = 0
				m.searchArchiveOnly = false
				m.tagInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				if m.confirmingDelete {
					if m.confirmDeleteGoalID > 0 {
						if err := database.DeleteGoal(m.confirmDeleteGoalID); err != nil {
							m.setStatusError(fmt.Sprintf("Error deleting goal: %v", err))
						} else {
							m.invalidateGoalCache()
							m.refreshData(m.day.ID)
						}
					}
					m.confirmingDelete = false
					m.confirmDeleteGoalID = 0
				} else if m.confirmingClearDB {
					if m.clearDBNeedsPass {
						entered := strings.TrimSpace(m.passphraseInput.Value())
						if limited, wait := m.passphraseRateLimited(); limited {
							remaining := wait.Round(time.Second)
							if remaining < time.Second {
								remaining = time.Second
							}
							m.clearDBStatus = fmt.Sprintf("Too many attempts. Try again in %s", remaining)
							m.passphraseInput.Reset()
							m.passphraseInput.Focus()
							return m, nil
						}
						if entered == "" {
							m.clearDBStatus = "Passphrase required"
							m.passphraseInput.Reset()
							m.passphraseInput.Focus()
							return m, nil
						}
						if util.HashPassphrase(entered) != m.passphraseHash {
							m.recordPassphraseFailure()
							m.clearDBStatus = "Incorrect passphrase"
							m.passphraseInput.Reset()
							m.passphraseInput.Focus()
							entered = ""
							return m, nil
						}
						m.clearPassphraseFailures()
						entered = ""
					}
					if err := database.ClearDatabase(); err != nil {
						m.err = err
					} else {
						wsID, wsErr := database.EnsureDefaultWorkspace()
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
							m.passphraseHash = ""
							m.Message = "Database cleared. Set sprint count to start."
						}
					}
					m.confirmingClearDB = false
					m.clearDBNeedsPass = false
					m.clearDBStatus = ""
					m.passphraseInput.Reset()
				} else if m.changingPassphrase {
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
							return m, nil
						}
						if m.passphraseHash != "" && util.HashPassphrase(current) != m.passphraseHash {
							m.recordPassphraseFailure()
							m.passphraseStatus = "Incorrect current passphrase"
							m.passphraseCurrent.Reset()
							m.passphraseCurrent.Focus()
							current = ""
							return m, nil
						}
						m.clearPassphraseFailures()
						current = ""
						m.passphraseStage = 1
						m.passphraseStatus = ""
						m.passphraseNew.Focus()
						return m, nil
					case 1:
						next := strings.TrimSpace(m.passphraseNew.Value())
						if next == "" {
							m.passphraseStatus = "New passphrase required"
							m.passphraseNew.Reset()
							m.passphraseNew.Focus()
							return m, nil
						}
						if err := util.ValidatePassphrase(next); err != nil {
							m.passphraseStatus = err.Error()
							m.passphraseNew.Reset()
							m.passphraseNew.Focus()
							next = ""
							return m, nil
						}
						m.passphraseStage = 2
						m.passphraseStatus = ""
						m.passphraseConfirm.Focus()
						return m, nil
					case 2:
						next := strings.TrimSpace(m.passphraseNew.Value())
						confirm := strings.TrimSpace(m.passphraseConfirm.Value())
						if next == "" {
							m.passphraseStatus = "New passphrase required"
							m.passphraseNew.Focus()
							next = ""
							confirm = ""
							return m, nil
						}
						if err := util.ValidatePassphrase(next); err != nil {
							m.passphraseStatus = err.Error()
							m.passphraseConfirm.Reset()
							m.passphraseConfirm.Focus()
							next = ""
							confirm = ""
							return m, nil
						}
						if next != confirm {
							m.passphraseStatus = "Passphrases do not match"
							m.passphraseConfirm.Reset()
							m.passphraseConfirm.Focus()
							next = ""
							confirm = ""
							return m, nil
						}
						_, encrypted, _ := database.EncryptionStatus()
						encryptErr := error(nil)
						if encrypted {
							if err := database.RekeyDB(next); err != nil && err != database.ErrSQLCipherUnavailable {
								encryptErr = err
								m.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", err)
								m.passphraseConfirm.Reset()
								m.passphraseConfirm.Focus()
								return m, nil
							} else if err == database.ErrSQLCipherUnavailable {
								encryptErr = err
								m.passphraseStatus = "SQLCipher unavailable; UI-only lock"
							}
						} else {
							if err := database.EncryptDatabase(next); err != nil && err != database.ErrSQLCipherUnavailable {
								encryptErr = err
								if !database.DatabaseHasData() {
									if recErr := database.RecreateEncryptedDatabase(next); recErr != nil {
										m.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", recErr)
										m.passphraseConfirm.Reset()
										m.passphraseConfirm.Focus()
										return m, nil
									}
								} else {
									m.passphraseStatus = fmt.Sprintf("Failed to update DB encryption: %v", err)
									m.passphraseConfirm.Reset()
									m.passphraseConfirm.Focus()
									return m, nil
								}
							} else if err == database.ErrSQLCipherUnavailable {
								encryptErr = err
								m.passphraseStatus = "SQLCipher unavailable; UI-only lock"
							}
						}
						m.passphraseHash = util.HashPassphrase(next)
						if err := database.SetSetting("passphrase_hash", m.passphraseHash); err != nil {
							m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
						}
						if encryptErr == database.ErrSQLCipherUnavailable {
							m.setStatusError("Encryption unavailable in this build; passphrase only locks the UI")
						} else if _, encrypted, _ := database.EncryptionStatus(); !encrypted {
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
						return m, nil
					}
				} else if m.searching {
					m.searching = false
					m.searchInput.Reset()
					m.searchCursor = 0
					m.searchArchiveOnly = false
				} else if m.journaling {
					text := m.journalInput.Value()
					if strings.TrimSpace(text) != "" {
						var sID, gID sql.NullInt64
						if m.activeSprint != nil {
							sID = sql.NullInt64{Int64: m.activeSprint.ID, Valid: true}
						}
						if m.editingGoalID > 0 {
							gID = sql.NullInt64{Int64: m.editingGoalID, Valid: true}
						}
						activeWS := m.workspaces[m.activeWorkspaceIdx]
						if err := database.AddJournalEntry(m.day.ID, activeWS.ID, sID, gID, text); err != nil {
							m.setStatusError(fmt.Sprintf("Error saving journal entry: %v", err))
						} else {
							m.refreshData(m.day.ID)
						}
					}
					m.journaling, m.editingGoalID = false, 0
					m.journalInput.Reset()
				} else if m.creatingWorkspace {
					name := m.textInput.Value()
					if name != "" {
						newID, err := database.CreateWorkspace(name, strings.ToLower(name))
						if err == nil {
							m.pendingWorkspaceID, m.creatingWorkspace, m.initializingSprints = newID, false, true
							m.textInput.Placeholder = "How many sprints?"
							m.textInput.Reset()
						} else {
							m.err = err
							m.creatingWorkspace = false
						}
					}
				} else if m.initializingSprints {
					val := m.textInput.Value()
					if num, err := strconv.Atoi(val); err == nil && num > 0 && num <= 8 {
						if err := database.BootstrapDay(m.pendingWorkspaceID, num); err != nil {
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
							if dayID := database.CheckCurrentDay(); dayID > 0 {
								if day, err := database.GetDay(dayID); err == nil {
									m.day = day
								}
								m.refreshData(dayID)
							}
						}
					}
					m.initializingSprints, m.pendingWorkspaceID = false, 0
					m.textInput.Reset()
				} else if m.tagging {
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
					if err := database.SetGoalTags(m.editingGoalID, out); err != nil {
						m.setStatusError(fmt.Sprintf("Error saving tags: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
					m.tagging, m.editingGoalID = false, 0
					m.tagInput.Reset()
					m.tagSelected = make(map[string]bool)
					m.tagCursor = 0
				} else if m.themePicking {
					if len(m.themeNames) > 0 && m.themeCursor < len(m.themeNames) {
						name := m.themeNames[m.themeCursor]
						activeWS := m.workspaces[m.activeWorkspaceIdx]
						if err := database.UpdateWorkspaceTheme(activeWS.ID, name); err != nil {
							m.setStatusError(fmt.Sprintf("Error updating workspace theme: %v", err))
						} else {
							m.workspaces[m.activeWorkspaceIdx].Theme = name
							SetTheme(name)
						}
					}
					m.themePicking = false
				} else if m.depPicking {
					var deps []int64
					for id, selected := range m.depSelected {
						if selected {
							deps = append(deps, id)
						}
					}
					if m.editingGoalID > 0 {
						if err := database.SetGoalDependencies(m.editingGoalID, deps); err != nil {
							m.setStatusError(fmt.Sprintf("Error saving dependencies: %v", err))
						} else {
							m.invalidateGoalCache()
							m.refreshData(m.day.ID)
						}
					}
					m.depPicking, m.editingGoalID = false, 0
					m.depSelected = make(map[int64]bool)
				} else if m.settingRecurrence {
					if m.editingGoalID > 0 {
						rule := m.recurrenceMode
						switch rule {
						case "none":
							if err := database.UpdateGoalRecurrence(m.editingGoalID, ""); err != nil {
								m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
							}
						case "daily":
							if err := database.UpdateGoalRecurrence(m.editingGoalID, "daily"); err != nil {
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
								if err := database.UpdateGoalRecurrence(m.editingGoalID, "weekly:"+strings.Join(days, ",")); err != nil {
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
								if err := database.UpdateGoalRecurrence(m.editingGoalID, rule); err != nil {
									m.setStatusError(fmt.Sprintf("Error saving recurrence: %v", err))
								}
							}
						}
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
					m.settingRecurrence, m.editingGoalID = false, 0
					m.recurrenceSelected = make(map[string]bool)
				} else {
					text := m.textInput.Value()
					if text != "" {
						if m.editingGoal {
							if err := database.EditGoal(m.editingGoalID, text); err != nil {
								m.setStatusError(fmt.Sprintf("Error updating goal: %v", err))
							}
						} else if m.editingGoalID > 0 {
							if err := database.AddSubtask(text, m.editingGoalID); err != nil {
								m.setStatusError(fmt.Sprintf("Error adding subtask: %v", err))
							} else {
								m.expandedState[m.editingGoalID] = true
							}
						} else {
							if err := database.AddGoal(m.workspaces[m.activeWorkspaceIdx].ID, text, m.sprints[m.focusedColIdx].ID); err != nil {
								m.setStatusError(fmt.Sprintf("Error adding goal: %v", err))
							}
						}
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
					m.creatingGoal, m.editingGoal, m.editingGoalID = false, false, 0
					m.textInput.Reset()
				}
				return m, nil
			}
		}
		if m.confirmingClearDB && m.clearDBNeedsPass {
			m.passphraseInput, cmd = m.passphraseInput.Update(msg)
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
		} else if m.confirmingDelete {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "a":
					if m.confirmDeleteGoalID > 0 {
						if err := database.ArchiveGoal(m.confirmDeleteGoalID); err != nil {
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
						if err := database.DeleteGoal(m.confirmDeleteGoalID); err != nil {
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
		} else if m.confirmingClearDB {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "c":
					if m.clearDBNeedsPass {
						return m, nil
					}
					if err := database.ClearDatabase(); err != nil {
						m.err = err
					} else {
						wsID, wsErr := database.EnsureDefaultWorkspace()
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
							m.passphraseHash = ""
							m.Message = "Database cleared. Set sprint count to start."
						}
					}
					m.confirmingClearDB = false
					m.clearDBNeedsPass = false
					m.clearDBStatus = ""
					m.passphraseInput.Reset()
					return m, nil
				}
			}
		} else if m.settingRecurrence {
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
		} else if m.depPicking {
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
		} else if m.themePicking {
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
		} else if m.tagging {
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
		} else if m.searching {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "up", "k":
					if m.searchCursor > 0 {
						m.searchCursor--
					}
					return m, nil
				case "down", "j":
					if m.searchCursor < len(m.searchResults)-1 {
						m.searchCursor++
					}
					return m, nil
				case "u":
					if m.searchArchiveOnly && len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults) {
						target := m.searchResults[m.searchCursor]
						if err := database.UnarchiveGoal(target.ID); err != nil {
							m.setStatusError(fmt.Sprintf("Error unarchiving goal: %v", err))
						} else {
							m.invalidateGoalCache()
							m.refreshData(m.day.ID)
							query := util.ParseSearchQuery(m.searchInput.Value())
							query.Status = []string{"archived"}
							m.searchResults, m.err = database.Search(query, m.workspaces[m.activeWorkspaceIdx].ID)
							if m.searchCursor >= len(m.searchResults) {
								m.searchCursor = len(m.searchResults) - 1
							}
							if m.searchCursor < 0 {
								m.searchCursor = 0
							}
						}
					}
					return m, nil
				}
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			if _, ok := msg.(tea.KeyMsg); ok && len(m.workspaces) > 0 {
				query := util.ParseSearchQuery(m.searchInput.Value())
				if m.searchArchiveOnly {
					query.Status = []string{"archived"}
				}
				m.searchResults, m.err = database.Search(query, m.workspaces[m.activeWorkspaceIdx].ID)
				if m.searchCursor >= len(m.searchResults) {
					m.searchCursor = len(m.searchResults) - 1
				}
				if m.searchCursor < 0 {
					m.searchCursor = 0
				}
			}
		} else if m.journaling {
			m.journalInput, cmd = m.journalInput.Update(msg)
		} else {
			m.textInput, cmd = m.textInput.Update(msg)
		}
		return m, cmd
	}

	// Move Mode
	if m.movingGoal {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.movingGoal = false
				return m, nil
			}
			if len(msg.String()) == 1 && strings.Contains("012345678", msg.String()) {
				targetNum := int(msg.String()[0] - '0')
				currentSprint := m.sprints[m.focusedColIdx]
				if len(currentSprint.Goals) > m.focusedGoalIdx {
					goal := currentSprint.Goals[m.focusedGoalIdx]
					var targetID int64 = 0 // Default to Backlog
					found := false
					if targetNum == 0 {
						found = true
					} else {
						for _, s := range m.sprints {
							if s.SprintNumber == targetNum {
								targetID = s.ID
								found = true
								break
							}
						}
					}
					if found {
						if err := database.MoveGoal(goal.ID, targetID); err != nil {
							m.setStatusError(fmt.Sprintf("Error moving goal: %v", err))
						} else {
							m.invalidateGoalCache()
							m.refreshData(m.day.ID)
							if m.focusedGoalIdx > 0 {
								m.focusedGoalIdx--
							}
						}
					}
				}
				m.movingGoal = false
				return m, nil
			}
		}
		return m, nil
	}

	// Normal Mode
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "L":
			if m.passphraseHash == "" {
				m.lockMessage = "Set passphrase to unlock"
			} else {
				m.lockMessage = "Enter passphrase to unlock"
			}
			m.locked = true
			m.passphraseInput.Reset()
			m.passphraseInput.Focus()
			return m, nil
		case "tab", "right", "l":
			nextIdx := -1
			for i := m.focusedColIdx + 1; i < len(m.sprints); i++ {
				if m.sprints[i].Status != "completed" || i < 2 {
					nextIdx = i
					break
				}
			}
			if nextIdx != -1 {
				m.focusedColIdx, m.focusedGoalIdx = nextIdx, 0
				if m.focusedColIdx >= 2 {
					m.colScrollOffset++
				}
			}
		case "shift+tab", "left", "h":
			if m.focusedColIdx > 0 {
				m.focusedColIdx--
				m.focusedGoalIdx = 0
				if m.colScrollOffset > 0 {
					m.colScrollOffset--
				}
			}
		case "up", "k":
			if m.focusedGoalIdx > 0 {
				m.focusedGoalIdx--
			}
		case "down", "j":
			if m.focusedColIdx < len(m.sprints) && m.focusedGoalIdx < len(m.sprints[m.focusedColIdx].Goals)-1 {
				m.focusedGoalIdx++
			}
		case "G":
			m.showAnalytics = !m.showAnalytics
			m.searching = false
			m.journaling = false
			return m, nil
		case "n":
			m.creatingGoal, m.editingGoalID = true, 0
			m.textInput.Placeholder = "New Objective..."
			m.textInput.Focus()
			return m, nil
		case "N":
			if m.focusedColIdx < len(m.sprints) && m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				parent := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.creatingGoal, m.editingGoalID = true, parent.ID
				m.textInput.Placeholder = "New Subtask..."
				m.textInput.Focus()
				return m, nil
			}
		case "z":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.expandedState[target.ID] = !m.expandedState[target.ID]
				m.refreshData(m.day.ID)
			}
		case "T":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				if target.TaskActive {
					if err := database.PauseTaskTimer(target.ID); err != nil {
						m.setStatusError(fmt.Sprintf("Error pausing task timer: %v", err))
					} else {
						m.Message = "Task timer paused."
					}
				} else {
					if err := database.StartTaskTimer(target.ID); err != nil {
						m.setStatusError(fmt.Sprintf("Error starting task timer: %v", err))
					} else {
						m.Message = "Task timer started."
					}
				}
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
		case "P":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				next := target.Priority + 1
				if next < 1 || next > 5 {
					next = 1
				}
				if err := database.UpdateGoalPriority(target.ID, next); err != nil {
					m.setStatusError(fmt.Sprintf("Error updating priority: %v", err))
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
				}
			}
		case "ctrl+j":
			m.journaling, m.editingGoalID = true, 0
			m.journalInput.Placeholder = "Log your thoughts..."
			m.journalInput.Focus()
			return m, nil
		case "ctrl+e":
			path, err := ExportVault(m.passphraseHash)
			if err != nil {
				m.Message = fmt.Sprintf("Export failed: %v", err)
			} else {
				m.Message = fmt.Sprintf("Export saved: %s", path)
			}
			return m, nil
		case "J":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.journaling, m.editingGoalID = true, target.ID
				m.journalInput.Placeholder = fmt.Sprintf("Log for: %s", target.Description)
				m.journalInput.Focus()
				return m, nil
			}
		case "/":
			m.searching = true
			m.searchArchiveOnly = m.focusedColIdx < len(m.sprints) && m.sprints[m.focusedColIdx].SprintNumber == -2
			m.searchCursor = 0
			m.searchInput.Focus()
			if m.searchArchiveOnly && len(m.workspaces) > 0 {
				query := util.ParseSearchQuery(m.searchInput.Value())
				query.Status = []string{"archived"}
				m.searchResults, m.err = database.Search(query, m.workspaces[m.activeWorkspaceIdx].ID)
			}
			return m, nil
		case "e":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.editingGoal, m.editingGoalID = true, target.ID
				m.textInput.SetValue(target.Description)
				m.textInput.Focus()
				return m, nil
			}
		case "d", "backspace":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				m.confirmingDelete = true
				m.confirmDeleteGoalID = m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx].ID
			}
		case "A":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				sprint := m.sprints[m.focusedColIdx]
				if sprint.SprintNumber != -2 {
					if err := database.ArchiveGoal(sprint.Goals[m.focusedGoalIdx].ID); err != nil {
						m.setStatusError(fmt.Sprintf("Error archiving goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
						if m.focusedGoalIdx > 0 {
							m.focusedGoalIdx--
						}
					}
				}
			}
		case "u":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				sprint := m.sprints[m.focusedColIdx]
				if sprint.SprintNumber == -2 {
					if err := database.UnarchiveGoal(sprint.Goals[m.focusedGoalIdx].ID); err != nil {
						m.setStatusError(fmt.Sprintf("Error unarchiving goal: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
						if m.focusedGoalIdx > 0 {
							m.focusedGoalIdx--
						}
					}
				}
			}
		case "b":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				activeWS.ShowBacklog = !activeWS.ShowBacklog
				if err := database.UpdateWorkspacePaneVisibility(activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived); err != nil {
					m.setStatusError(fmt.Sprintf("Error updating workspace view: %v", err))
				}
				m.workspaces[m.activeWorkspaceIdx].ShowBacklog = activeWS.ShowBacklog
				m.refreshData(m.day.ID)
			}
		case "c":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				activeWS.ShowCompleted = !activeWS.ShowCompleted
				if err := database.UpdateWorkspacePaneVisibility(activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived); err != nil {
					m.setStatusError(fmt.Sprintf("Error updating workspace view: %v", err))
				}
				m.workspaces[m.activeWorkspaceIdx].ShowCompleted = activeWS.ShowCompleted
				m.refreshData(m.day.ID)
			}
		case "a":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				activeWS.ShowArchived = !activeWS.ShowArchived
				if err := database.UpdateWorkspacePaneVisibility(activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived); err != nil {
					m.setStatusError(fmt.Sprintf("Error updating workspace view: %v", err))
				}
				m.workspaces[m.activeWorkspaceIdx].ShowArchived = activeWS.ShowArchived
				m.refreshData(m.day.ID)
			}
		case "m":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				m.movingGoal = true
				return m, nil
			}
		case "D":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.depPicking, m.editingGoalID = true, target.ID
				m.depOptions = m.buildDepOptions(target.ID)
				deps, err := database.GetGoalDependencies(target.ID)
				if err != nil {
					m.setStatusError(fmt.Sprintf("Error loading dependencies: %v", err))
					m.depSelected = make(map[int64]bool)
				} else {
					m.depSelected = deps
				}
				m.depCursor = 0
				return m, nil
			}
		case "C":
			m.confirmingClearDB = true
			m.clearDBNeedsPass = m.passphraseHash != ""
			m.clearDBStatus = ""
			m.passphraseInput.Reset()
			m.passphraseInput.Placeholder = "Passphrase"
			m.passphraseInput.Focus()
			return m, nil
		case "I":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				seedPath, err := EnsureSeedFile()
				if err != nil {
					m.Message = fmt.Sprintf("Seed file error: %v", err)
					return m, nil
				}
				count, _, backlogFallback, err := ImportSeed(seedPath, activeWS.ID, m.day.ID)
				if err != nil {
					m.Message = fmt.Sprintf("Seed import failed: %v", err)
					return m, nil
				}
				if count == 0 {
					if backlogFallback > 0 {
						m.Message = "Seed import complete. Some sprint tasks moved to backlog (max 8)."
					} else {
						m.Message = "Seed already imported."
					}
				} else {
					if backlogFallback > 0 {
						m.Message = fmt.Sprintf("Imported %d tasks (some moved to backlog: max 8).", count)
					} else {
						m.Message = fmt.Sprintf("Imported %d tasks from seed.", count)
					}
				}
				m.invalidateGoalCache()
				m.refreshData(m.day.ID)
			}
			return m, nil
		case "p":
			m.changingPassphrase = true
			m.passphraseStatus = ""
			m.passphraseStage = 0
			m.passphraseCurrent.Reset()
			m.passphraseNew.Reset()
			m.passphraseConfirm.Reset()
			if m.passphraseHash == "" {
				m.passphraseStage = 1
				m.passphraseNew.Focus()
			} else {
				m.passphraseCurrent.Focus()
			}
			return m, nil
		case "R":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.settingRecurrence, m.editingGoalID = true, target.ID
				m.recurrenceCursor = 0
				m.recurrenceMode = "none"
				m.recurrenceSelected = make(map[string]bool)
				m.recurrenceFocus = "mode"
				m.recurrenceItemCursor = 0
				m.recurrenceDayCursor = 0
				if target.RecurrenceRule.Valid {
					rule := strings.ToLower(strings.TrimSpace(target.RecurrenceRule.String))
					switch {
					case rule == "daily":
						m.recurrenceMode = "daily"
					case strings.HasPrefix(rule, "weekly:"):
						m.recurrenceMode = "weekly"
						parts := strings.Split(strings.TrimPrefix(rule, "weekly:"), ",")
						for _, p := range parts {
							p = strings.TrimSpace(p)
							if p != "" {
								m.recurrenceSelected[p] = true
							}
						}
						for i, d := range m.weekdayOptions {
							if m.recurrenceSelected[d] {
								m.recurrenceItemCursor = i
								break
							}
						}
					case strings.HasPrefix(rule, "monthly:"):
						m.recurrenceMode = "monthly"
						payload := strings.TrimPrefix(rule, "monthly:")
						var months []string
						var days []string
						if strings.Contains(payload, "months=") || strings.Contains(payload, "days=") {
							chunks := strings.Split(payload, ";")
							for _, chunk := range chunks {
								chunk = strings.TrimSpace(chunk)
								switch {
								case strings.HasPrefix(chunk, "months="):
									months = strings.Split(strings.TrimPrefix(chunk, "months="), ",")
								case strings.HasPrefix(chunk, "days="):
									days = strings.Split(strings.TrimPrefix(chunk, "days="), ",")
								}
							}
						} else if payload != "" {
							months = strings.Split(payload, ",")
						}
						for _, mo := range months {
							mo = strings.TrimSpace(mo)
							if mo != "" {
								m.recurrenceSelected[mo] = true
							}
						}
						if len(days) == 0 {
							days = []string{"1"}
						}
						for _, d := range days {
							d = strings.TrimSpace(d)
							if d != "" {
								m.recurrenceSelected["day:"+d] = true
							}
						}
						for i, mo := range m.monthOptions {
							if m.recurrenceSelected[mo] {
								m.recurrenceItemCursor = i
								break
							}
						}
						for i, d := range m.monthDayOptions {
							if m.recurrenceSelected["day:"+d] {
								m.recurrenceDayCursor = i
								break
							}
						}
					}
				}
				for i, opt := range m.recurrenceOptions {
					if opt == m.recurrenceMode {
						m.recurrenceCursor = i
						break
					}
				}
				return m, nil
			}
		case " ":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				goal := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				blocked, err := database.IsGoalBlocked(goal.ID)
				if err != nil {
					m.setStatusError(fmt.Sprintf("Error checking dependencies: %v", err))
				} else if blocked {
					m.Message = "Blocked by dependency. Complete dependencies first."
					return m, nil
				}
				canToggle := true
				if goal.Status == "pending" {
					for _, sub := range goal.Subtasks {
						if sub.Status != "completed" {
							canToggle = false
							break
						}
					}
				}
				if canToggle {
					newStatus := "pending"
					if goal.Status == "pending" {
						newStatus = "completed"
					}
					if err := database.UpdateGoalStatus(goal.ID, newStatus); err != nil {
						m.setStatusError(fmt.Sprintf("Error updating goal status: %v", err))
					} else {
						m.invalidateGoalCache()
						m.refreshData(m.day.ID)
					}
				} else {
					m.Message = "Cannot complete task with pending subtasks!"
				}
			}
		case "s":
			if !m.breakActive && m.focusedColIdx < len(m.sprints) {
				target := m.sprints[m.focusedColIdx]
				if target.SprintNumber > 0 {
					if m.activeSprint != nil && m.activeSprint.ID == target.ID {
						elapsed := int(time.Since(m.activeSprint.StartTime.Time).Seconds()) + m.activeSprint.ElapsedSeconds
						if err := database.PauseSprint(target.ID, int(elapsed)); err != nil {
							m.setStatusError(fmt.Sprintf("Error pausing sprint: %v", err))
						} else {
							m.refreshData(m.day.ID)
						}
					} else if m.activeSprint == nil && (target.Status == "pending" || target.Status == "paused") {
						if err := database.StartSprint(target.ID); err != nil {
							m.setStatusError(fmt.Sprintf("Error starting sprint: %v", err))
						} else {
							m.refreshData(m.day.ID)
							return m, tickCmd()
						}
					}
				}
			}
		case "x":
			if m.activeSprint != nil {
				if err := database.ResetSprint(m.activeSprint.ID); err != nil {
					m.setStatusError(fmt.Sprintf("Error resetting sprint: %v", err))
				} else {
					m.activeSprint = nil
					m.refreshData(m.day.ID)
				}
			}
		case "+":
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			if err := database.AppendSprint(m.day.ID, activeWS.ID); err != nil {
				m.Message = fmt.Sprintf("Add sprint failed: %v", err)
				return m, nil
			}
			m.invalidateGoalCache()
			m.refreshData(m.day.ID)
		case "-":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				if err := database.RemoveLastSprint(m.day.ID, activeWS.ID); err != nil {
					m.Message = fmt.Sprintf("Remove sprint failed: %v", err)
				} else {
					m.invalidateGoalCache()
					m.refreshData(m.day.ID)
					if m.focusedColIdx >= len(m.sprints) {
						m.focusedColIdx = len(m.sprints) - 1
					}
				}
			}
		case "w":
			if len(m.workspaces) > 1 {
				m.activeWorkspaceIdx = (m.activeWorkspaceIdx + 1) % len(m.workspaces)
				m.refreshData(m.day.ID)
				m.focusedColIdx = 1
			} else {
				m.Message = "No other workspaces. Use Shift+W to create a new one."
			}
		case "W":
			m.creatingWorkspace = true
			m.textInput.Focus()
			return m, nil
		case "t":
			if m.focusedColIdx < len(m.sprints) && m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.tagging, m.editingGoalID = true, target.ID
				m.tagInput.Focus()
				m.tagInput.SetValue("")
				m.tagSelected = make(map[string]bool)
				var customTags []string
				for _, t := range util.JSONToTags(target.Tags.String) {
					if containsTag(m.defaultTags, t) {
						m.tagSelected[t] = true
					} else {
						customTags = append(customTags, t)
					}
				}
				if len(customTags) > 0 {
					sort.Strings(customTags)
					m.tagInput.SetValue(strings.Join(customTags, " "))
				}
				m.tagCursor = 0
				return m, nil
			}
		case "v":
			m.viewMode = (m.viewMode + 1) % 3
			// Persist choice
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				if err := database.UpdateWorkspaceViewMode(activeWS.ID, m.viewMode); err != nil {
					m.setStatusError(fmt.Sprintf("Error updating view mode: %v", err))
				}
				// Update local cache so refreshData doesn't overwrite it immediately if called
				m.workspaces[m.activeWorkspaceIdx].ViewMode = m.viewMode
			}

			// Reset focus if current column becomes hidden
			if m.viewMode == ViewModeFocused && m.sprints[m.focusedColIdx].SprintNumber == -1 {
				m.focusedColIdx = 1 // Jump to Backlog or first Sprint
			} else if m.viewMode == ViewModeMinimal && m.sprints[m.focusedColIdx].SprintNumber <= 0 {
				m.focusedColIdx = 2 // Jump to first Sprint
				if len(m.sprints) <= 2 {
					m.focusedColIdx = 0
				} // Fallback
			}
		case "Y":
			if len(m.workspaces) > 0 {
				m.themePicking = true
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				for i, t := range m.themeNames {
					if t == activeWS.Theme {
						m.themeCursor = i
						break
					}
				}
				return m, nil
			}
		case "ctrl+r":
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			path, err := GeneratePDFReport(m.day.ID, activeWS.ID)
			if err != nil {
				m.setStatusError(fmt.Sprintf("Report failed: %v", err))
				return m, nil
			}
			fmt.Printf("\nPDF Report generated: %s\n", path)
			return m, tea.Quit
		case "<":
			prevID, _, err := database.GetAdjacentDay(m.day.ID, -1)
			if err == nil {
				m.refreshData(prevID)
			} else {
				m.Message = "No previous days recorded."
			}
		case ">":
			nextID, _, err := database.GetAdjacentDay(m.day.ID, 1)
			if err == nil {
				m.refreshData(nextID)
			} else {
				m.Message = "No future days recorded."
			}
		}
	}
	return m, nil
}
