package tui

import (
	"fmt"
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

func (m DashboardModel) handleModalConfirmPassphrase() (DashboardModel, tea.Cmd, bool) {
	if !m.security.changingPassphrase {
		return m, nil, false
	}
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
		if m.security.lock.PassphraseHash != "" {
			ok, upgradedHash := util.VerifyPassphraseWithUpgrade(m.security.lock.PassphraseHash, current)
			if !ok {
				m.recordPassphraseFailure()
				m.security.passphraseStatus = "Incorrect current passphrase"
				m.inputs.passphraseCurrent.Reset()
				m.inputs.passphraseCurrent.Focus()
				return m, nil, true
			}
			if upgradedHash != "" {
				if err := m.db.SetSetting(m.ctx, "passphrase_hash", upgradedHash); err != nil {
					m.setStatusError(fmt.Sprintf("Error saving passphrase: %v", err))
				} else {
					m.security.lock.PassphraseHash = upgradedHash
				}
			}
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
		if m.security.lock.PassphraseHash == "" {
			m.security.passphraseStatus = "Failed to update passphrase"
			m.inputs.passphraseConfirm.Reset()
			m.inputs.passphraseConfirm.Focus()
			return m, nil, true
		}
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
	return m, nil, false
}

func (m DashboardModel) handleModalInputPassphrase(msg tea.Msg) (DashboardModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	if m.security.confirmingClearDB && m.security.clearDBNeedsPass {
		m.security.lock.PassphraseInput, cmd = m.security.lock.PassphraseInput.Update(msg)
		return m, cmd, true
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
		return m, cmd, true
	}
	return m, nil, false
}
