package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) inInputMode() bool {
	return m.modal.InInputMode(m.security, m.search)
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
	handlers := []func(DashboardModel) (DashboardModel, tea.Cmd, bool){
		DashboardModel.handleModalConfirmDelete,
		DashboardModel.handleModalConfirmClearDB,
		DashboardModel.handleModalConfirmPassphrase,
		DashboardModel.handleModalConfirmSearch,
		DashboardModel.handleModalConfirmJournaling,
		DashboardModel.handleModalConfirmWorkspaceCreate,
		DashboardModel.handleModalConfirmInitializeSprints,
		DashboardModel.handleModalConfirmTagging,
		DashboardModel.handleModalConfirmTheme,
		DashboardModel.handleModalConfirmDependencies,
		DashboardModel.handleModalConfirmRecurrence,
		DashboardModel.handleModalConfirmGoalEdit,
	}
	for _, handler := range handlers {
		if next, cmd, handled := handler(m); handled {
			return next, cmd, true
		}
	}
	return m, nil, false
}

func (m DashboardModel) handleModalInput(msg tea.Msg) (DashboardModel, tea.Cmd) {
	handlers := []func(DashboardModel, tea.Msg) (DashboardModel, tea.Cmd, bool){
		DashboardModel.handleModalInputPassphrase,
		DashboardModel.handleModalInputConfirmDelete,
		DashboardModel.handleModalInputClearDB,
		DashboardModel.handleModalInputRecurrence,
		DashboardModel.handleModalInputDependencies,
		DashboardModel.handleModalInputTheme,
		DashboardModel.handleModalInputTagging,
		DashboardModel.handleModalInputSearch,
		DashboardModel.handleModalInputJournaling,
		DashboardModel.handleModalInputGoalText,
	}
	for _, handler := range handlers {
		if next, cmd, handled := handler(m, msg); handled {
			return next, cmd
		}
	}
	return m, nil
}
