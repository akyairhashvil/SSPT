package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) handleModalConfirmClearDB() (DashboardModel, tea.Cmd, bool) {
	if !m.security.confirmingClearDB {
		return m, nil, false
	}
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
		ok, upgradedHash := util.VerifyPassphraseWithUpgrade(m.security.lock.PassphraseHash, entered)
		if !ok {
			m.recordPassphraseFailure()
			m.security.clearDBStatus = "Incorrect passphrase"
			m.security.lock.PassphraseInput.Reset()
			m.security.lock.PassphraseInput.Focus()
			return m, nil, true
		}
		if upgradedHash != "" {
			if err := m.db.SetSetting(m.ctx, "passphrase_hash", upgradedHash); err != nil {
				m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
			} else {
				m.security.lock.PassphraseHash = upgradedHash
			}
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
			m.modal.Open(&WorkspaceInitState{WorkspaceID: wsID})
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

func (m DashboardModel) handleModalInputClearDB(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	if !m.security.confirmingClearDB {
		return m, nil, false
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "c":
			if m.security.clearDBNeedsPass {
				return m, nil, true
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
					m.modal.Open(&WorkspaceInitState{WorkspaceID: wsID})
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
	}
	return m, nil, true
}
