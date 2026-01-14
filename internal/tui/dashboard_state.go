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
	creatingGoal         bool
	editingGoal          bool
	editingGoalID        int64
	movingGoal           bool
	creatingWorkspace    bool
	initializingSprints  bool
	pendingWorkspaceID   int64
	tagging              bool
	tagCursor            int
	tagSelected          map[string]bool
	defaultTags          []string
	themeOrder           []string
	themePicking         bool
	themeCursor          int
	themeNames           []string
	depPicking           bool
	depCursor            int
	depOptions           []depOption
	depSelected          map[int64]bool
	settingRecurrence    bool
	recurrenceOptions    []string
	recurrenceCursor     int
	recurrenceMode       string
	weekdayOptions       []string
	monthOptions         []string
	recurrenceSelected   map[string]bool
	recurrenceFocus      string
	recurrenceItemCursor int
	recurrenceDayCursor  int
	monthDayOptions      []string
	confirmingDelete     bool
	confirmDeleteGoalID  int64
	journaling           bool
}

type ModalType string

const (
	ModalNone              ModalType = "none"
	ModalCreatingGoal      ModalType = "creating_goal"
	ModalEditingGoal       ModalType = "editing_goal"
	ModalMovingGoal        ModalType = "moving_goal"
	ModalCreatingWorkspace ModalType = "creating_workspace"
	ModalInitializing      ModalType = "initializing_sprints"
	ModalTagging           ModalType = "tagging"
	ModalThemePicking      ModalType = "theme_picking"
	ModalDependency        ModalType = "dependency"
	ModalRecurrence        ModalType = "recurrence"
	ModalConfirmDelete     ModalType = "confirm_delete"
	ModalJournaling        ModalType = "journaling"
)

func newModalManager() *ModalManager {
	return &ModalManager{
		tagSelected:        make(map[string]bool),
		depSelected:        make(map[int64]bool),
		recurrenceSelected: make(map[string]bool),
	}
}

func (m *ModalManager) InInputMode(security *SecurityManager, search SearchManager) bool {
	return security.changingPassphrase ||
		m.confirmingDelete ||
		security.confirmingClearDB ||
		m.creatingGoal ||
		m.editingGoal ||
		m.journaling ||
		search.Active ||
		m.creatingWorkspace ||
		m.initializingSprints ||
		m.tagging ||
		m.themePicking ||
		m.depPicking ||
		m.settingRecurrence
}

func (m *ModalManager) ActiveModal() ModalType {
	switch {
	case m.creatingGoal:
		return ModalCreatingGoal
	case m.editingGoal:
		return ModalEditingGoal
	case m.movingGoal:
		return ModalMovingGoal
	case m.creatingWorkspace:
		return ModalCreatingWorkspace
	case m.initializingSprints:
		return ModalInitializing
	case m.tagging:
		return ModalTagging
	case m.themePicking:
		return ModalThemePicking
	case m.depPicking:
		return ModalDependency
	case m.settingRecurrence:
		return ModalRecurrence
	case m.confirmingDelete:
		return ModalConfirmDelete
	case m.journaling:
		return ModalJournaling
	default:
		return ModalNone
	}
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
