package tui

import tea "github.com/charmbracelet/bubbletea"

type ModalType int

const (
	ModalNone ModalType = iota
	ModalGoalCreate
	ModalGoalEdit
	ModalGoalDelete
	ModalGoalMove
	ModalWorkspaceCreate
	ModalWorkspaceInit
	ModalDependency
	ModalTagging
	ModalRecurrence
	ModalTheme
	ModalJournaling
	ModalPassphrase
	ModalSearch
	ModalClearDB
)

type ModalState interface {
	Type() ModalType
	HandleKey(key string) (ModalState, tea.Cmd)
}

type GoalCreateState struct {
	ParentID int64
}

func (s *GoalCreateState) Type() ModalType { return ModalGoalCreate }
func (s *GoalCreateState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type GoalEditState struct {
	GoalID int64
}

func (s *GoalEditState) Type() ModalType { return ModalGoalEdit }
func (s *GoalEditState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type GoalDeleteState struct {
	GoalID int64
}

func (s *GoalDeleteState) Type() ModalType { return ModalGoalDelete }
func (s *GoalDeleteState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type GoalMoveState struct{}

func (s *GoalMoveState) Type() ModalType { return ModalGoalMove }
func (s *GoalMoveState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type WorkspaceCreateState struct{}

func (s *WorkspaceCreateState) Type() ModalType { return ModalWorkspaceCreate }
func (s *WorkspaceCreateState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type WorkspaceInitState struct {
	WorkspaceID int64
}

func (s *WorkspaceInitState) Type() ModalType { return ModalWorkspaceInit }
func (s *WorkspaceInitState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type TaggingState struct {
	GoalID   int64
	Cursor   int
	Selected map[string]bool
}

func (s *TaggingState) Type() ModalType { return ModalTagging }
func (s *TaggingState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type ThemeState struct {
	Cursor int
}

func (s *ThemeState) Type() ModalType { return ModalTheme }
func (s *ThemeState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type DependencyState struct {
	GoalID   int64
	Cursor   int
	Options  []depOption
	Selected map[int64]bool
}

func (s *DependencyState) Type() ModalType { return ModalDependency }
func (s *DependencyState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type RecurrenceState struct {
	GoalID          int64
	Options         []string
	Cursor          int
	Mode            string
	WeekdayOptions  []string
	MonthOptions    []string
	Selected        map[string]bool
	Focus           string
	ItemCursor      int
	DayCursor       int
	MonthDayOptions []string
}

func (s *RecurrenceState) Type() ModalType { return ModalRecurrence }
func (s *RecurrenceState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}

type JournalState struct {
	GoalID int64
}

func (s *JournalState) Type() ModalType { return ModalJournaling }
func (s *JournalState) HandleKey(key string) (ModalState, tea.Cmd) {
	return s, nil
}
