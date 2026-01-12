package tui

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	SprintDuration        = 90 * time.Minute
	BreakDuration         = 30 * time.Minute
	AutoLockAfter         = 10 * time.Minute
	passphraseMaxAttempts = 5
	passphraseLockout     = 30 * time.Second
)

var AppVersion = "0"

// View Modes
const (
	ViewModeAll     = 0
	ViewModeFocused = 1 // Hide Completed
	ViewModeMinimal = 2 // Hide Completed & Backlog
)

// --- Model ---
type DashboardModel struct {
	db                   *sql.DB
	day                  models.Day
	sprints              []models.Sprint
	workspaces           []models.Workspace
	activeWorkspaceIdx   int
	viewMode             int
	focusedColIdx        int
	focusedGoalIdx       int
	colScrollOffset      int
	goalScrollOffsets    map[int]int
	creatingGoal         bool
	editingGoal          bool
	editingGoalID        int64
	movingGoal           bool
	creatingWorkspace    bool
	initializingSprints  bool
	pendingWorkspaceID   int64
	tagging              bool
	tagInput             textinput.Model
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
	locked               bool
	lockMessage          string
	passphraseHash       string
	passphraseInput      textinput.Model
	lastInput            time.Time
	changingPassphrase   bool
	confirmingClearDB    bool
	clearDBNeedsPass     bool
	clearDBStatus        string
	passphraseStage      int
	passphraseStatus     string
	passphraseCurrent    textinput.Model
	passphraseNew        textinput.Model
	passphraseConfirm    textinput.Model
	passphraseAttempts   int
	passphraseLockUntil  time.Time
	journaling           bool
	journalEntries       []models.JournalEntry
	journalInput         textinput.Model
	searching            bool
	showAnalytics        bool
	searchResults        []models.Goal
	searchInput          textinput.Model
	searchCursor         int
	searchArchiveOnly    bool
	expandedState        map[int64]bool
	goalTreeCache        map[string][]models.Goal
	progress             progress.Model
	activeSprint         *models.Sprint
	activeTask           *models.Goal
	breakActive          bool
	breakStart           time.Time
	textInput            textinput.Model
	err                  error
	statusMessage        string
	statusIsError        bool
	Message              string
	width, height        int
}

type depOption struct {
	ID    int64
	Label string
}

func NewDashboardModel(db *sql.DB, dayID int64) DashboardModel {
	_, wsErr := database.EnsureDefaultWorkspace()
	ti := textinput.New()
	ti.Placeholder = "New Objective..."
	ti.CharLimit = 100
	ti.Width = 40
	ji := textinput.New()
	ji.Placeholder = "Log thoughts..."
	ji.Width = 50
	si := textinput.New()
	si.Placeholder = "Search..."
	si.Width = 30
	tagInput := textinput.New()
	tagInput.Placeholder = "Add custom tags (space-separated)"
	tagInput.Width = 50
	passInput := textinput.New()
	passInput.Placeholder = "Passphrase"
	passInput.EchoMode = textinput.EchoPassword
	passInput.Width = 30
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

	m := DashboardModel{
		db:                 db,
		textInput:          ti,
		journalInput:       ji,
		searchInput:        si,
		tagInput:           tagInput,
		progress:           progress.New(progress.WithDefaultGradient()),
		activeWorkspaceIdx: 0,
		focusedColIdx:      1,
		goalScrollOffsets:  make(map[int]int),
		expandedState:      make(map[int64]bool),
		tagSelected:        make(map[string]bool),
		defaultTags:        []string{"urgent", "docs", "blocked", "waiting", "bug", "idea", "review", "focus", "later"},
		themeOrder:         []string{"default", "dracula", "cyberpunk", "solar"},
		depSelected:        make(map[int64]bool),
		recurrenceOptions:  []string{"none", "daily", "weekly", "monthly"},
		recurrenceMode:     "none",
		weekdayOptions:     []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
		monthOptions:       []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"},
		monthDayOptions:    buildMonthDays(),
		recurrenceSelected: make(map[string]bool),
		recurrenceFocus:    "mode",
		passphraseInput:    passInput,
		passphraseCurrent:  passCurrent,
		passphraseNew:      passNew,
		passphraseConfirm:  passConfirm,
		lastInput:          time.Now(),
	}
	if wsErr != nil {
		m.setStatusError(fmt.Sprintf("Error ensuring default workspace: %v", wsErr))
	} else if err := m.loadWorkspaces(); err != nil {
		m.setStatusError(fmt.Sprintf("Error loading workspaces: %v", err))
	}
	if hash, ok := database.GetSetting("passphrase_hash"); ok && hash != "" {
		m.passphraseHash = hash
		m.locked = true
		m.lockMessage = "Enter passphrase to unlock"
	} else {
		m.locked = true
		m.lockMessage = "Set passphrase to unlock"
	}
	m.passphraseInput.Focus()
	sort.Strings(m.defaultTags)
	for name := range Themes {
		m.themeNames = append(m.themeNames, name)
	}
	sort.Strings(m.themeNames)
	m.progress.Width = 30
	m.refreshData(dayID)

	// Set initial focus
	if len(m.sprints) > 1 {
		for i := 1; i < len(m.sprints); i++ {
			if m.sprints[i].Status != "completed" && m.sprints[i].SprintNumber > 0 {
				m.focusedColIdx = i
				break
			}
		}
	}
	return m
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
	if m.passphraseLockUntil.IsZero() {
		return false, 0
	}
	if time.Now().Before(m.passphraseLockUntil) {
		return true, time.Until(m.passphraseLockUntil)
	}
	m.passphraseLockUntil = time.Time{}
	return false, 0
}

func (m *DashboardModel) recordPassphraseFailure() {
	m.passphraseAttempts++
	if m.passphraseAttempts >= passphraseMaxAttempts {
		m.passphraseAttempts = 0
		m.passphraseLockUntil = time.Now().Add(passphraseLockout)
	}
}

func (m *DashboardModel) clearPassphraseFailures() {
	m.passphraseAttempts = 0
	m.passphraseLockUntil = time.Time{}
}

func (m *DashboardModel) invalidateGoalCache() {
	m.goalTreeCache = make(map[string][]models.Goal)
}

func (m *DashboardModel) getGoalTree(key string, fetch func() ([]models.Goal, error)) ([]models.Goal, error) {
	if m.goalTreeCache == nil {
		m.goalTreeCache = make(map[string][]models.Goal)
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

func cloneGoals(goals []models.Goal) []models.Goal {
	if len(goals) == 0 {
		return nil
	}
	out := make([]models.Goal, len(goals))
	for i, g := range goals {
		g.Subtasks = cloneGoals(g.Subtasks)
		out[i] = g
	}
	return out
}

func (m *DashboardModel) loadWorkspaces() error {
	workspaces, err := database.GetWorkspaces()
	if err != nil {
		return err
	}
	m.workspaces = workspaces
	return nil
}

func (m *DashboardModel) refreshData(dayID int64) {
	m.clearStatus()
	// Initialize with empty placeholders to prevent panics
	m.sprints = []models.Sprint{
		{ID: -1, SprintNumber: -1, Goals: []models.Goal{}},
		{ID: 0, SprintNumber: 0, Goals: []models.Goal{}},
	}

	if len(m.workspaces) == 0 {
		m.Message = "No workspaces found. Please create one."
		return
	}
	activeWS := m.workspaces[m.activeWorkspaceIdx]
	m.viewMode = activeWS.ViewMode
	SetTheme(activeWS.Theme)
	blockedIDs, err := database.GetBlockedGoalIDs(activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading blocked goals: %v", err))
		return
	}
	m.activeTask = nil
	if task, err := database.GetActiveTask(activeWS.ID); err == nil {
		m.activeTask = task
	}

	day, err := database.GetDay(dayID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading day: %v", err))
		return
	}
	rawSprints, err := database.GetSprints(dayID, activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading sprints: %v", err))
		return
	}
	journalEntries, err := database.GetJournalEntries(dayID, activeWS.ID)
	if err != nil {
		m.setStatusError(fmt.Sprintf("Error loading journal entries: %v", err))
		return
	}

	var fullList []models.Sprint

	// Archived Column (first)
	if activeWS.ShowArchived {
		archivedKey := fmt.Sprintf("archived:%d", activeWS.ID)
		archivedGoals, err := m.getGoalTree(archivedKey, func() ([]models.Goal, error) {
			return database.GetArchivedGoals(activeWS.ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading archived goals: %v", err))
			return
		}
		flatArchived := Flatten(archivedGoals, 0, m.expandedState, 0)
		fullList = append(fullList, models.Sprint{ID: -2, SprintNumber: -2, Goals: flatArchived})
	}

	// Completed Column
	if activeWS.ShowCompleted {
		completedKey := fmt.Sprintf("completed:%d:%d", activeWS.ID, dayID)
		completedGoals, err := m.getGoalTree(completedKey, func() ([]models.Goal, error) {
			return database.GetCompletedGoalsForDay(dayID, activeWS.ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading completed goals: %v", err))
			return
		}
		flatCompleted := Flatten(completedGoals, 0, m.expandedState, 0)
		fullList = append(fullList, models.Sprint{ID: -1, SprintNumber: -1, Goals: flatCompleted})
	}

	var pruneCompleted func(goals []models.Goal) []models.Goal
	pruneCompleted = func(goals []models.Goal) []models.Goal {
		var out []models.Goal
		for _, g := range goals {
			if g.Status != "completed" {
				g.Subtasks = pruneCompleted(g.Subtasks)
				out = append(out, g)
			}
		}
		return out
	}

	var applyBlocked func(goals []models.Goal, depth int) []models.Goal
	warned := false
	applyBlocked = func(goals []models.Goal, depth int) []models.Goal {
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
			return database.GetBacklogGoals(activeWS.ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading backlog goals: %v", err))
			return
		}
		backlogTree := applyBlocked(pruneCompleted(cloneGoals(backlogGoals)), 0)
		flatBacklog := Flatten(backlogTree, 0, m.expandedState, 0)
		fullList = append(fullList, models.Sprint{ID: 0, SprintNumber: 0, Goals: flatBacklog})
	}

	// Sprints
	for i := range rawSprints {
		sprintKey := fmt.Sprintf("sprint:%d", rawSprints[i].ID)
		goals, err := m.getGoalTree(sprintKey, func() ([]models.Goal, error) {
			return database.GetGoalsForSprint(rawSprints[i].ID)
		})
		if err != nil {
			m.setStatusError(fmt.Sprintf("Error loading sprint goals: %v", err))
			return
		}
		sprintTree := applyBlocked(pruneCompleted(cloneGoals(goals)), 0)
		rawSprints[i].Goals = Flatten(sprintTree, 0, m.expandedState, 0)
		fullList = append(fullList, rawSprints[i])
	}

	m.sprints = fullList
	m.day, m.journalEntries = day, journalEntries
	m.activeSprint = nil
	for i := range m.sprints {
		if m.sprints[i].Status == "active" {
			m.activeSprint = &m.sprints[i]
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
	for _, mo := range m.monthOptions {
		if m.recurrenceSelected[mo] {
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
	for key := range m.recurrenceSelected {
		if strings.HasPrefix(key, "day:") {
			val := strings.TrimPrefix(key, "day:")
			if day, err := strconv.Atoi(val); err == nil && day > maxDay {
				delete(m.recurrenceSelected, key)
			}
		}
	}
	if maxDay <= 0 {
		m.recurrenceDayCursor = 0
		return
	}
	if m.recurrenceDayCursor > maxDay-1 {
		m.recurrenceDayCursor = maxDay - 1
	}
}

func (m DashboardModel) Init() tea.Cmd { return tea.Batch(textinput.Blink, tickCmd()) }
