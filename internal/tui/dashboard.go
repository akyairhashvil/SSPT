// Package tui implements the terminal user interface for SSPT using
// the Bubble Tea framework (Elm architecture: Model-View-Update).
//
// The main entry point is DashboardModel which manages application state
// and handles keyboard input, timer updates, and screen rendering.
package tui

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const passphraseLockout = 30 * time.Second

var (
	AppVersion = "dev"
	GitCommit  = "unknown"
	BuildTime  = "unknown"
)

// View Modes
const (
	ViewModeAll     = config.ViewModeAll
	ViewModeFocused = config.ViewModeFocused // Hide Completed
	ViewModeMinimal = config.ViewModeMinimal // Hide Completed & Backlog
)

// --- Model ---
type DashboardModel struct {
	db                 Database
	ctx                context.Context
	day                models.Day
	sprints            []SprintView
	workspaces         []models.Workspace
	activeWorkspaceIdx int
	viewMode           int
	view               *ViewState
	modal              *ModalManager
	inputs             *InputState
	security           *SecurityManager
	journalEntries     []models.JournalEntry
	search             SearchManager
	showAnalytics      bool
	goalTreeCache      map[string][]GoalView
	progress           progress.Model
	timer              TimerManager
	theme              Theme
	err                error
	statusMessage      string
	statusIsError      bool
	Message            string
	width, height      int
}

type depOption struct {
	ID    int64
	Label string
}

func NewDashboardModel(ctx context.Context, db Database, dayID int64, theme Theme) DashboardModel {
	if ctx == nil {
		ctx = context.Background()
	}
	_, wsErr := db.EnsureDefaultWorkspace(ctx)
	inputs := newInputState()
	si := textinput.New()
	si.Placeholder = "Search..."
	si.Width = 30
	passInput := textinput.New()
	passInput.Placeholder = "Passphrase"
	passInput.EchoMode = textinput.EchoPassword
	passInput.Width = 30

	lock := NewLockModel(config.AutoLockAfter, passInput)
	search := NewSearchManager(si)
	view := newViewState()
	modal := newModalManager()
	security := newSecurityManager(lock)
	modal.defaultTags = []string{"urgent", "docs", "blocked", "waiting", "bug", "idea", "review", "focus", "later"}
	modal.themeOrder = []string{"default", "dracula", "cyberpunk", "solar"}
	modal.recurrenceOptions = []string{"none", "daily", "weekly", "monthly"}
	modal.recurrenceMode = "none"
	modal.weekdayOptions = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	modal.monthOptions = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
	modal.monthDayOptions = buildMonthDays()
	modal.recurrenceFocus = "mode"

	m := DashboardModel{
		db:                 db,
		ctx:                ctx,
		view:               view,
		modal:              modal,
		inputs:             inputs,
		security:           security,
		search:             search,
		timer:              NewTimerManager(),
		progress:           progress.New(progress.WithDefaultGradient()),
		activeWorkspaceIdx: 0,
		theme:              theme,
	}
	if wsErr != nil {
		m.setStatusError(fmt.Sprintf("Error ensuring default workspace: %v", wsErr))
	} else if err := m.loadWorkspaces(); err != nil {
		m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
	}
	if hash, ok := m.db.GetSetting(ctx, "passphrase_hash"); ok && hash != "" {
		m.security.lock.PassphraseHash = hash
		m.security.lock.Locked = true
		m.security.lock.Message = "Enter passphrase to unlock"
	} else {
		m.security.lock.Locked = true
		m.security.lock.Message = "Set passphrase to unlock"
	}
	m.security.lock.PassphraseInput.Focus()
	sort.Strings(m.modal.defaultTags)
	for name := range Themes {
		m.modal.themeNames = append(m.modal.themeNames, name)
	}
	sort.Strings(m.modal.themeNames)
	m.progress.Width = config.TargetTitleWidth
	m.refreshData(dayID)

	// Set initial focus
	if len(m.sprints) > 1 {
		for i := 1; i < len(m.sprints); i++ {
			if m.sprints[i].Status != models.StatusCompleted && m.sprints[i].SprintNumber > 0 {
				m.view.focusedColIdx = i
				break
			}
		}
	}
	return m
}

func (m DashboardModel) hasActiveSprint() bool {
	return m.timer.ActiveSprint != nil
}

func (m DashboardModel) validSprintIndex(idx int) bool {
	return idx >= 0 && idx < len(m.sprints)
}

func (m DashboardModel) currentSprint() *SprintView {
	if !m.validSprintIndex(m.view.focusedColIdx) {
		return nil
	}
	return &m.sprints[m.view.focusedColIdx]
}

func (m DashboardModel) canModifyGoals() bool {
	return !m.security.lock.Locked && !m.inInputMode()
}

func (m *DashboardModel) setStatusError(message string) {
	m.statusMessage = message
	m.statusIsError = true
}

func (m *DashboardModel) clearStatus() {
	m.statusMessage = ""
	m.statusIsError = false
}

func (m *DashboardModel) passphraseRateLimited() (bool, time.Duration) {
	if m.security.lock.LockUntil.IsZero() {
		return false, 0
	}
	if time.Now().Before(m.security.lock.LockUntil) {
		return true, time.Until(m.security.lock.LockUntil)
	}
	m.security.lock.LockUntil = time.Time{}
	return false, 0
}

func (m *DashboardModel) recordPassphraseFailure() {
	m.security.lock.Attempts++
	if m.security.lock.Attempts >= config.MaxPassphraseAttempts {
		m.security.lock.Attempts = 0
		m.security.lock.LockUntil = time.Now().Add(passphraseLockout)
	}
}

func (m *DashboardModel) clearPassphraseFailures() {
	m.security.lock.Attempts = 0
	m.security.lock.LockUntil = time.Time{}
}

func (m *DashboardModel) invalidateGoalCache() {
	m.goalTreeCache = make(map[string][]GoalView)
}

func (m *DashboardModel) getGoalTree(key string, fetch func() ([]models.Goal, error)) ([]GoalView, error) {
	if m.goalTreeCache == nil {
		m.goalTreeCache = make(map[string][]GoalView)
	}
	if tree, ok := m.goalTreeCache[key]; ok {
		return tree, nil
	}
	goals, err := fetch()
	if err != nil {
		return nil, err
	}
	tree := BuildHierarchy(goals)
	m.goalTreeCache[key] = tree
	return tree, nil
}

func cloneGoals(goals []GoalView) []GoalView {
	if len(goals) == 0 {
		return nil
	}
	out := make([]GoalView, len(goals))
	for i, g := range goals {
		g.Subtasks = cloneGoals(g.Subtasks)
		out[i] = g
	}
	return out
}

func (m *DashboardModel) loadWorkspaces() error {
	workspaces, err := m.db.GetWorkspaces(m.ctx)
	if err != nil {
		return err
	}
	m.workspaces = workspaces
	return nil
}

func (m *DashboardModel) refreshData(dayID int64) {
	m.clearStatus()
	// Initialize with empty placeholders to prevent panics
	m.sprints = []SprintView{
		{Sprint: models.Sprint{ID: -1, SprintNumber: -1}, Goals: []GoalView{}},
		{Sprint: models.Sprint{ID: 0, SprintNumber: 0}, Goals: []GoalView{}},
	}

	if len(m.workspaces) == 0 {
		m.Message = "No workspaces found. Please create one."
		return
	}
	activeWS := m.workspaces[m.activeWorkspaceIdx]
	m.viewMode = activeWS.ViewMode
	m.theme = ResolveTheme(activeWS.Theme)
	blockedIDs, err := m.db.GetBlockedGoalIDs(m.ctx, activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading blocked goals: %v", err))
		return
	}
	m.timer.ActiveTask = nil
	if task, err := m.db.GetActiveTask(m.ctx, activeWS.ID); err == nil {
		m.timer.ActiveTask = task
	}

	day, err := m.db.GetDay(m.ctx, dayID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading day: %v", err))
		return
	}
	rawSprints, err := m.db.GetSprints(m.ctx, dayID, activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading sprints: %v", err))
		return
	}
	journalEntries, err := m.db.GetJournalEntries(m.ctx, dayID, activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading journal entries: %v", err))
		return
	}

	var fullList []SprintView

	// Archived Column (first)
	if activeWS.ShowArchived {
		archivedKey := fmt.Sprintf("archived:%d", activeWS.ID)
		archivedGoals, err := m.getGoalTree(archivedKey, func() ([]models.Goal, error) {
			return m.db.GetArchivedGoals(m.ctx, activeWS.ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading archived goals: %v", err))
			return
		}
		flatArchived := Flatten(archivedGoals, 0, m.view.expandedState, 0)
		fullList = append(fullList, SprintView{Sprint: models.Sprint{ID: -2, SprintNumber: -2}, Goals: flatArchived})
	}

	// Completed Column
	if activeWS.ShowCompleted {
		completedKey := fmt.Sprintf("completed:%d:%d", activeWS.ID, dayID)
		completedGoals, err := m.getGoalTree(completedKey, func() ([]models.Goal, error) {
			return m.db.GetCompletedGoalsForDay(m.ctx, dayID, activeWS.ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading completed goals: %v", err))
			return
		}
		flatCompleted := Flatten(completedGoals, 0, m.view.expandedState, 0)
		fullList = append(fullList, SprintView{Sprint: models.Sprint{ID: -1, SprintNumber: -1}, Goals: flatCompleted})
	}

	var pruneCompleted func(goals []GoalView) []GoalView
	pruneCompleted = func(goals []GoalView) []GoalView {
		var out []GoalView
		for _, g := range goals {
			if g.Status != models.GoalStatusCompleted {
				g.Subtasks = pruneCompleted(g.Subtasks)
				out = append(out, g)
			}
		}
		return out
	}

	var applyBlocked func(goals []GoalView, depth int) []GoalView
	warned := false
	applyBlocked = func(goals []GoalView, depth int) []GoalView {
		if depth >= goalTreeMaxDepthDefault {
			if !warned {
				log.Printf("goal tree depth exceeds %d; truncating blocked propagation", goalTreeWarnDepth)
				warned = true
			}
			return goals
		}
		if depth >= goalTreeWarnDepth && !warned {
			log.Printf("goal tree depth exceeds %d; truncating blocked propagation", goalTreeWarnDepth)
			warned = true
		}
		for i := range goals {
			if blockedIDs[goals[i].ID] {
				goals[i].Blocked = true
			}
			if len(goals[i].Subtasks) > 0 {
				goals[i].Subtasks = applyBlocked(goals[i].Subtasks, depth+1)
			}
		}
		return goals
	}

	// Backlog Column
	if activeWS.ShowBacklog {
		backlogKey := fmt.Sprintf("backlog:%d", activeWS.ID)
		backlogGoals, err := m.getGoalTree(backlogKey, func() ([]models.Goal, error) {
			return m.db.GetBacklogGoals(m.ctx, activeWS.ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading backlog goals: %v", err))
			return
		}
		backlogTree := applyBlocked(pruneCompleted(cloneGoals(backlogGoals)), 0)
		flatBacklog := Flatten(backlogTree, 0, m.view.expandedState, 0)
		fullList = append(fullList, SprintView{Sprint: models.Sprint{ID: 0, SprintNumber: 0}, Goals: flatBacklog})
	}

	// Sprints
	for i := range rawSprints {
		sprintKey := fmt.Sprintf("sprint:%d", rawSprints[i].ID)
		goals, err := m.getGoalTree(sprintKey, func() ([]models.Goal, error) {
			return m.db.GetGoalsForSprint(m.ctx, rawSprints[i].ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading sprint goals: %v", err))
			return
		}
		sprintTree := applyBlocked(pruneCompleted(cloneGoals(goals)), 0)
		sprintView := SprintView{Sprint: rawSprints[i], Goals: Flatten(sprintTree, 0, m.view.expandedState, 0)}
		fullList = append(fullList, sprintView)
	}

	m.sprints = fullList
	m.day, m.journalEntries = day, journalEntries
	m.timer.ActiveSprint = nil
	for i := range m.sprints {
		if m.sprints[i].Status == models.StatusActive {
			m.timer.ActiveSprint = &m.sprints[i]
			break
		}
	}
}

func (m *DashboardModel) buildDepOptions(targetID int64) []depOption {
	var opts []depOption
	for _, sprint := range m.sprints {
		if sprint.SprintNumber == -2 {
			continue
		}
		var title string
		switch sprint.SprintNumber {
		case -1:
			title = "Completed"
		case 0:
			title = "Backlog"
		default:
			title = fmt.Sprintf("Sprint %d", sprint.SprintNumber)
		}
		for _, g := range sprint.Goals {
			if g.ID == targetID {
				continue
			}
			indent := strings.Repeat("  ", g.Level)
			label := fmt.Sprintf("[%s] %s#%d %s", title, indent, g.ID, g.Description)
			opts = append(opts, depOption{ID: g.ID, Label: label})
		}
	}
	return opts
}

func buildMonthDays() []string {
	var days []string
	for i := 1; i <= 31; i++ {
		days = append(days, strconv.Itoa(i))
	}
	return days
}

func parseYear(dateStr string) int {
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			return t.Year()
		}
	}
	return time.Now().Year()
}

func monthDayLimit(month string, year int) int {
	switch month {
	case "apr", "jun", "sep", "nov":
		return 30
	case "feb":
		if year%400 == 0 || (year%4 == 0 && year%100 != 0) {
			return 29
		}
		return 28
	default:
		return 31
	}
}

func (m *DashboardModel) selectedMonths() []string {
	var out []string
	for _, mo := range m.modal.monthOptions {
		if m.modal.recurrenceSelected[mo] {
			out = append(out, mo)
		}
	}
	return out
}

func (m *DashboardModel) monthlyMaxDay() int {
	months := m.selectedMonths()
	if len(months) == 0 {
		return 0
	}
	year := parseYear(m.day.Date)
	maxDay := 31
	for _, mo := range months {
		days := monthDayLimit(mo, year)
		if days < maxDay {
			maxDay = days
		}
	}
	return maxDay
}

func (m *DashboardModel) pruneMonthlyDays(maxDay int) {
	for key := range m.modal.recurrenceSelected {
		if strings.HasPrefix(key, "day:") {
			val := strings.TrimPrefix(key, "day:")
			if day, err := strconv.Atoi(val); err == nil && day > maxDay {
				delete(m.modal.recurrenceSelected, key)
			}
		}
	}
	if maxDay <= 0 {
		m.modal.recurrenceDayCursor = 0
		return
	}
	if m.modal.recurrenceDayCursor > maxDay-1 {
		m.modal.recurrenceDayCursor = maxDay - 1
	}
}

func (m DashboardModel) Init() tea.Cmd { return tea.Batch(textinput.Blink, tickCmd()) }
