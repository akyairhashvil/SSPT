package tui

import (
	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
)

// ViewState tracks cursor focus and scroll positions.
type ViewState struct {
	focusedColIdx     int
	focusedGoalIdx    int
	colScrollOffset   int
	goalScrollOffsets map[int]int
	expandedState     map[int64]bool
}

func newViewState() *ViewState {
	return &ViewState{
		focusedColIdx:     config.DefaultFocusColumn,
		goalScrollOffsets: make(map[int]int),
		expandedState:     make(map[int64]bool),
	}
}

// ModalManager tracks modal-related UI state and selections.
type ModalManager struct {
	current           ModalState
	defaultTags       []string
	themeOrder        []string
	themeNames        []string
	recurrenceOptions []string
	weekdayOptions    []string
	monthOptions      []string
	monthDayOptions   []string
}

func newModalManager() *ModalManager {
	return &ModalManager{}
}

func (m *ModalManager) InInputMode(security *SecurityManager, search SearchManager) bool {
	if security.changingPassphrase || security.confirmingClearDB || search.Active {
		return true
	}
	return m.current != nil && m.current.Type() != ModalGoalMove
}

func (m *ModalManager) ActiveModal() ModalType {
	if m.current == nil {
		return ModalNone
	}
	return m.current.Type()
}

func (m *ModalManager) IsOpen() bool {
	return m.current != nil
}

func (m *ModalManager) Current() ModalState {
	return m.current
}

func (m *ModalManager) Open(state ModalState) {
	m.current = state
}

func (m *ModalManager) Close() {
	m.current = nil
}

func (m *ModalManager) Is(t ModalType) bool {
	return m.current != nil && m.current.Type() == t
}

func (m *ModalManager) GoalCreateState() (*GoalCreateState, bool) {
	state, ok := m.current.(*GoalCreateState)
	return state, ok
}

func (m *ModalManager) GoalEditState() (*GoalEditState, bool) {
	state, ok := m.current.(*GoalEditState)
	return state, ok
}

func (m *ModalManager) GoalDeleteState() (*GoalDeleteState, bool) {
	state, ok := m.current.(*GoalDeleteState)
	return state, ok
}

func (m *ModalManager) GoalMoveState() (*GoalMoveState, bool) {
	state, ok := m.current.(*GoalMoveState)
	return state, ok
}

func (m *ModalManager) WorkspaceCreateState() (*WorkspaceCreateState, bool) {
	state, ok := m.current.(*WorkspaceCreateState)
	return state, ok
}

func (m *ModalManager) WorkspaceInitState() (*WorkspaceInitState, bool) {
	state, ok := m.current.(*WorkspaceInitState)
	return state, ok
}

func (m *ModalManager) TaggingState() (*TaggingState, bool) {
	state, ok := m.current.(*TaggingState)
	return state, ok
}

func (m *ModalManager) ThemeState() (*ThemeState, bool) {
	state, ok := m.current.(*ThemeState)
	return state, ok
}

func (m *ModalManager) DependencyState() (*DependencyState, bool) {
	state, ok := m.current.(*DependencyState)
	return state, ok
}

func (m *ModalManager) RecurrenceState() (*RecurrenceState, bool) {
	state, ok := m.current.(*RecurrenceState)
	return state, ok
}

func (m *ModalManager) JournalState() (*JournalState, bool) {
	state, ok := m.current.(*JournalState)
	return state, ok
}

// InputState stores all text input models.
type InputState struct {
	textInput         textinput.Model
	journalInput      textinput.Model
	tagInput          textinput.Model
	passphraseCurrent textinput.Model
	passphraseNew     textinput.Model
	passphraseConfirm textinput.Model
}

func newInputState() *InputState {
	ti := textinput.New()
	ti.Placeholder = "New Objective..."
	ti.CharLimit = config.MaxTitleLength
	ti.Width = 40

	ji := textinput.New()
	ji.Placeholder = "Log thoughts..."
	ji.Width = 50

	tagInput := textinput.New()
	tagInput.Placeholder = "Add custom tags (space-separated)"
	tagInput.Width = 50

	passCurrent := textinput.New()
	passCurrent.Placeholder = "Current passphrase"
	passCurrent.EchoMode = textinput.EchoPassword
	passCurrent.Width = 30

	passNew := textinput.New()
	passNew.Placeholder = "New passphrase"
	passNew.EchoMode = textinput.EchoPassword
	passNew.Width = 30

	passConfirm := textinput.New()
	passConfirm.Placeholder = "Confirm passphrase"
	passConfirm.EchoMode = textinput.EchoPassword
	passConfirm.Width = 30

	return &InputState{
		textInput:         ti,
		journalInput:      ji,
		tagInput:          tagInput,
		passphraseCurrent: passCurrent,
		passphraseNew:     passNew,
		passphraseConfirm: passConfirm,
	}
}

// SecurityManager tracks authentication and destructive operation flags.
type SecurityManager struct {
	lock               LockModel
	changingPassphrase bool
	confirmingClearDB  bool
	clearDBNeedsPass   bool
	clearDBStatus      string
	passphraseStage    int
	passphraseStatus   string
}

func newSecurityManager(lock LockModel) *SecurityManager {
	return &SecurityManager{lock: lock}
}
