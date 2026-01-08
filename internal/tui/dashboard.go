package tui

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	SprintDuration = 90 * time.Minute
	BreakDuration  = 30 * time.Minute
	AutoLockAfter  = 10 * time.Minute
)

var AppVersion = "0"

// View Modes
const (
	ViewModeAll     = 0
	ViewModeFocused = 1 // Hide Completed
	ViewModeMinimal = 2 // Hide Completed & Backlog
)

// --- Messages ---
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return TickMsg(t) })
}

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
	passphraseStage      int
	passphraseStatus     string
	passphraseCurrent    textinput.Model
	passphraseNew        textinput.Model
	passphraseConfirm    textinput.Model
	journaling           bool
	journalEntries       []models.JournalEntry
	journalInput         textinput.Model
	searching            bool
	searchResults        []models.Goal
	searchInput          textinput.Model
	searchCursor         int
	searchArchiveOnly    bool
	expandedState        map[int64]bool
	progress             progress.Model
	activeSprint         *models.Sprint
	breakActive          bool
	breakStart           time.Time
	textInput            textinput.Model
	err                  error
	Message              string
	width, height        int
}

type depOption struct {
	ID    int64
	Label string
}

func NewDashboardModel(db *sql.DB, dayID int64) DashboardModel {
	database.EnsureDefaultWorkspace()
	workspaces, _ := database.GetWorkspaces()
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
		workspaces:         workspaces,
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

func (m *DashboardModel) refreshData(dayID int64) {
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
	blockedIDs, _ := database.GetBlockedGoalIDs(activeWS.ID)

	day, _ := database.GetDay(dayID)
	rawSprints, _ := database.GetSprints(dayID, activeWS.ID)
	journalEntries, _ := database.GetJournalEntries(dayID, activeWS.ID)

	var fullList []models.Sprint

	// Archived Column (first)
	if activeWS.ShowArchived {
		archivedGoals, _ := database.GetArchivedGoals(activeWS.ID)
		flatArchived := Flatten(BuildHierarchy(archivedGoals), 0, m.expandedState)
		fullList = append(fullList, models.Sprint{ID: -2, SprintNumber: -2, Goals: flatArchived})
	}

	// Completed Column
	if activeWS.ShowCompleted {
		completedGoals, _ := database.GetCompletedGoalsForDay(dayID, activeWS.ID)
		flatCompleted := Flatten(BuildHierarchy(completedGoals), 0, m.expandedState)
		fullList = append(fullList, models.Sprint{ID: -1, SprintNumber: -1, Goals: flatCompleted})
	}

	var pruneCompleted = func(goals []models.Goal) []models.Goal {
		var out []models.Goal
		for _, g := range goals {
			if g.Status != "completed" {
				g.Subtasks = append([]models.Goal{}, g.Subtasks...)
				out = append(out, g)
			}
		}
		return out
	}

	var applyBlocked func(goals []models.Goal) []models.Goal
	applyBlocked = func(goals []models.Goal) []models.Goal {
		for i := range goals {
			if blockedIDs[goals[i].ID] {
				goals[i].Blocked = true
			}
			if len(goals[i].Subtasks) > 0 {
				goals[i].Subtasks = applyBlocked(goals[i].Subtasks)
			}
		}
		return goals
	}

	// Backlog Column
	if activeWS.ShowBacklog {
		backlogGoals, _ := database.GetBacklogGoals(activeWS.ID)
		backlogTree := applyBlocked(pruneCompleted(BuildHierarchy(backlogGoals)))
		flatBacklog := Flatten(backlogTree, 0, m.expandedState)
		fullList = append(fullList, models.Sprint{ID: 0, SprintNumber: 0, Goals: flatBacklog})
	}

	// Sprints
	for i := range rawSprints {
		goals, _ := database.GetGoalsForSprint(rawSprints[i].ID)
		sprintTree := applyBlocked(pruneCompleted(BuildHierarchy(goals)))
		rawSprints[i].Goals = Flatten(sprintTree, 0, m.expandedState)
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

func hashPassphrase(pass string) string {
	sum := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(sum[:])
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
				database.CompleteSprint(m.activeSprint.ID)
				database.MovePendingToBacklog(m.activeSprint.ID)
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
				if m.passphraseHash == "" {
					if entered == "" {
						m.lockMessage = "Passphrase required"
						m.passphraseInput.Reset()
						m.passphraseInput.Focus()
						return m, nil
					}
					m.passphraseHash = hashPassphrase(entered)
					_ = database.SetSetting("passphrase_hash", m.passphraseHash)
					m.locked = false
					m.lockMessage = ""
					m.passphraseInput.Reset()
					m.lastInput = time.Now()
					return m, nil
				}
				if entered != "" && hashPassphrase(entered) == m.passphraseHash {
					m.locked = false
					m.lockMessage = ""
					m.passphraseInput.Reset()
					m.lastInput = time.Now()
					return m, nil
				}
				m.lockMessage = "Incorrect passphrase"
				m.passphraseInput.Reset()
				m.passphraseInput.Focus()
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
	if m.changingPassphrase || m.confirmingDelete || m.creatingGoal || m.editingGoal || m.journaling || m.searching || m.creatingWorkspace || m.initializingSprints || m.tagging || m.themePicking || m.depPicking || m.settingRecurrence {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.confirmingDelete = false
				m.confirmDeleteGoalID = 0
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
						database.DeleteGoal(m.confirmDeleteGoalID)
						m.refreshData(m.day.ID)
					}
					m.confirmingDelete = false
					m.confirmDeleteGoalID = 0
				} else if m.changingPassphrase {
					switch m.passphraseStage {
					case 0:
						current := strings.TrimSpace(m.passphraseCurrent.Value())
						if m.passphraseHash != "" && hashPassphrase(current) != m.passphraseHash {
							m.passphraseStatus = "Incorrect current passphrase"
							m.passphraseCurrent.Reset()
							m.passphraseCurrent.Focus()
							return m, nil
						}
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
							return m, nil
						}
						if next != confirm {
							m.passphraseStatus = "Passphrases do not match"
							m.passphraseConfirm.Reset()
							m.passphraseConfirm.Focus()
							return m, nil
						}
						m.passphraseHash = hashPassphrase(next)
						_ = database.SetSetting("passphrase_hash", m.passphraseHash)
						m.changingPassphrase = false
						m.passphraseStage = 0
						m.passphraseStatus = ""
						m.passphraseCurrent.Reset()
						m.passphraseNew.Reset()
						m.passphraseConfirm.Reset()
						m.Message = "Passphrase updated."
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
						database.AddJournalEntry(m.day.ID, activeWS.ID, sID, gID, text)
						m.refreshData(m.day.ID)
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
						database.BootstrapDay(m.pendingWorkspaceID, num)
						m.workspaces, _ = database.GetWorkspaces()
						for i, ws := range m.workspaces {
							if ws.ID == m.pendingWorkspaceID {
								m.activeWorkspaceIdx = i
								break
							}
						}
						m.refreshData(m.day.ID)
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
					_ = database.SetGoalTags(m.editingGoalID, out)
					m.refreshData(m.day.ID)
					m.tagging, m.editingGoalID = false, 0
					m.tagInput.Reset()
					m.tagSelected = make(map[string]bool)
					m.tagCursor = 0
				} else if m.themePicking {
					if len(m.themeNames) > 0 && m.themeCursor < len(m.themeNames) {
						name := m.themeNames[m.themeCursor]
						activeWS := m.workspaces[m.activeWorkspaceIdx]
						database.UpdateWorkspaceTheme(activeWS.ID, name)
						m.workspaces[m.activeWorkspaceIdx].Theme = name
						SetTheme(name)
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
						_ = database.SetGoalDependencies(m.editingGoalID, deps)
						m.refreshData(m.day.ID)
					}
					m.depPicking, m.editingGoalID = false, 0
					m.depSelected = make(map[int64]bool)
				} else if m.settingRecurrence {
					if m.editingGoalID > 0 {
						rule := m.recurrenceMode
						switch rule {
						case "none":
							_ = database.UpdateGoalRecurrence(m.editingGoalID, "")
						case "daily":
							_ = database.UpdateGoalRecurrence(m.editingGoalID, "daily")
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
								_ = database.UpdateGoalRecurrence(m.editingGoalID, "weekly:"+strings.Join(days, ","))
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
								_ = database.UpdateGoalRecurrence(m.editingGoalID, rule)
							}
						}
						m.refreshData(m.day.ID)
					}
					m.settingRecurrence, m.editingGoalID = false, 0
					m.recurrenceSelected = make(map[string]bool)
				} else {
					text := m.textInput.Value()
					if text != "" {
						if m.editingGoal {
							database.EditGoal(m.editingGoalID, text)
						} else if m.editingGoalID > 0 {
							database.AddSubtask(text, m.editingGoalID)
							m.expandedState[m.editingGoalID] = true
						} else {
							database.AddGoal(m.workspaces[m.activeWorkspaceIdx].ID, text, m.sprints[m.focusedColIdx].ID)
						}
						m.refreshData(m.day.ID)
					}
					m.creatingGoal, m.editingGoal, m.editingGoalID = false, false, 0
					m.textInput.Reset()
				}
				return m, nil
			}
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
						_ = database.ArchiveGoal(m.confirmDeleteGoalID)
						m.refreshData(m.day.ID)
					}
					m.confirmingDelete = false
					m.confirmDeleteGoalID = 0
					return m, nil
				case "d", "backspace":
					if m.confirmDeleteGoalID > 0 {
						database.DeleteGoal(m.confirmDeleteGoalID)
						m.refreshData(m.day.ID)
					}
					m.confirmingDelete = false
					m.confirmDeleteGoalID = 0
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
						_ = database.UnarchiveGoal(target.ID)
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
						database.MoveGoal(goal.ID, targetID)
						m.refreshData(m.day.ID)
						if m.focusedGoalIdx > 0 {
							m.focusedGoalIdx--
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
		case "ctrl+j":
			m.journaling, m.editingGoalID = true, 0
			m.journalInput.Placeholder = "Log your thoughts..."
			m.journalInput.Focus()
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
					database.ArchiveGoal(sprint.Goals[m.focusedGoalIdx].ID)
					m.refreshData(m.day.ID)
					if m.focusedGoalIdx > 0 {
						m.focusedGoalIdx--
					}
				}
			}
		case "u":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				sprint := m.sprints[m.focusedColIdx]
				if sprint.SprintNumber == -2 {
					database.UnarchiveGoal(sprint.Goals[m.focusedGoalIdx].ID)
					m.refreshData(m.day.ID)
					if m.focusedGoalIdx > 0 {
						m.focusedGoalIdx--
					}
				}
			}
		case "b":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				activeWS.ShowBacklog = !activeWS.ShowBacklog
				_ = database.UpdateWorkspacePaneVisibility(activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived)
				m.workspaces[m.activeWorkspaceIdx].ShowBacklog = activeWS.ShowBacklog
				m.refreshData(m.day.ID)
			}
		case "c":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				activeWS.ShowCompleted = !activeWS.ShowCompleted
				_ = database.UpdateWorkspacePaneVisibility(activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived)
				m.workspaces[m.activeWorkspaceIdx].ShowCompleted = activeWS.ShowCompleted
				m.refreshData(m.day.ID)
			}
		case "a":
			if len(m.workspaces) > 0 {
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				activeWS.ShowArchived = !activeWS.ShowArchived
				_ = database.UpdateWorkspacePaneVisibility(activeWS.ID, activeWS.ShowBacklog, activeWS.ShowCompleted, activeWS.ShowArchived)
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
				m.depSelected, _ = database.GetGoalDependencies(target.ID)
				m.depCursor = 0
				return m, nil
			}
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
				if blocked, _ := database.IsGoalBlocked(goal.ID); blocked {
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
					database.UpdateGoalStatus(goal.ID, newStatus)
					m.refreshData(m.day.ID)
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
						database.PauseSprint(target.ID, int(elapsed))
						m.refreshData(m.day.ID)
					} else if m.activeSprint == nil && (target.Status == "pending" || target.Status == "paused") {
						database.StartSprint(target.ID)
						m.refreshData(m.day.ID)
						return m, tickCmd()
					}
				}
			}
		case "x":
			if m.activeSprint != nil {
				database.ResetSprint(m.activeSprint.ID)
				m.activeSprint = nil
				m.refreshData(m.day.ID)
			}
		case "+":
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			database.AppendSprint(m.day.ID, activeWS.ID)
			m.refreshData(m.day.ID)
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
				database.UpdateWorkspaceViewMode(activeWS.ID, m.viewMode)
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
		case "T":
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
			GeneratePDFReport(m.day.ID, activeWS.ID)
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

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	if m.locked {
		var lockContent strings.Builder
		title := "Locked"
		if m.passphraseHash == "" {
			title = "Set Passphrase"
		}
		lockContent.WriteString(CurrentTheme.Focused.Render(title) + "\n\n")
		if m.lockMessage != "" {
			lockContent.WriteString(CurrentTheme.Dim.Render(m.lockMessage) + "\n")
		}
		lockContent.WriteString(CurrentTheme.Focused.Render("> ") + m.passphraseInput.View())

		lockFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(1, 2)
		lockBox := lockFrame.Render(lockContent.String())
		return "\x1b[H\x1b[2J" + lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, lockBox)
	}

	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress any key to continue.", m.err)
	}
	if m.Message != "" {
		return CurrentTheme.Break.Copy().Foreground(lipgloss.Color("208")).Render(m.Message)
	}

	// 1. Determine Timer Content
	var timerContent string
	var timerColor lipgloss.Style

	if m.breakActive {
		elapsed := time.Since(m.breakStart)
		rem := BreakDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timerContent = fmt.Sprintf("☕ BREAK TIME: %02d:%02d REMAINING", int(rem.Minutes()), int(rem.Seconds())%60)
		timerColor = CurrentTheme.Break
	} else if m.activeSprint != nil {
		elapsed := time.Since(m.activeSprint.StartTime.Time) + (time.Duration(m.activeSprint.ElapsedSeconds) * time.Second)
		rem := SprintDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(SprintDuration))
		timerContent = fmt.Sprintf("ACTIVE SPRINT: %d  |  %s  |  %s remaining", m.activeSprint.SprintNumber, barView, timeStr)
		timerColor = CurrentTheme.Focused
	} else {
		if len(m.workspaces) > 0 {
			// Safety index check
			idx := m.activeWorkspaceIdx
			if idx >= len(m.workspaces) {
				idx = 0
			}
			timerContent = fmt.Sprintf("[%s | %s] Select Sprint & Press 's' to Start", m.workspaces[idx].Name, m.day.Date)
		} else {
			timerContent = "No workspaces found."
		}
		timerColor = CurrentTheme.Dim

		if m.focusedColIdx < len(m.sprints) {
			target := m.sprints[m.focusedColIdx]
			if target.Status == "paused" {
				elapsed := time.Duration(target.ElapsedSeconds) * time.Second
				rem := SprintDuration - elapsed
				timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
				timerContent = fmt.Sprintf("PAUSED SPRINT: %d  |  %s remaining  |  [s] to Resume", target.SprintNumber, timeStr)
				timerColor = CurrentTheme.Break
			}
		}
	}

	if timerContent == "" {
		timerContent = "SSPT - Ready"
		timerColor = CurrentTheme.Dim
	}
	timerContent = fmt.Sprintf("%s  |  v%s", timerContent, AppVersion)

	// 2. Render Header (Timer Box)
	headerFrame := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(CurrentTheme.Border).
		Padding(0, 1)
	headerExtra := lipgloss.Width(headerFrame.Render(""))
	headerWidth := m.width - headerExtra
	if headerWidth < 1 {
		headerWidth = 1
	}
	timerBox := headerFrame.Width(headerWidth).Render(timerColor.Render(timerContent))

	// 3. Render Footer
	var footer string
	var footerContent string
	var footerHelpLines []string
	var rawFooter string
	if m.creatingGoal || m.editingGoal || m.creatingWorkspace || m.initializingSprints {
		footerContent = CurrentTheme.Input.Render(m.textInput.View())
	} else if m.tagging {
		footerContent = CurrentTheme.Dim.Render("[Tab] Toggle Tag | [Enter] Save | [Esc] Cancel")
	} else if m.themePicking {
		footerContent = CurrentTheme.Dim.Render("[Enter] Apply Theme | [Esc] Cancel")
	} else if m.depPicking {
		footerContent = CurrentTheme.Dim.Render("[Space] Toggle | [Enter] Save | [Esc] Cancel")
	} else if m.settingRecurrence {
		footerContent = CurrentTheme.Dim.Render("[Tab] Next | [Space] Toggle | [Enter] Save | [Esc] Cancel")
	} else if m.confirmingDelete {
		footerContent = CurrentTheme.Focused.Render("Delete task? [d] Delete | [a] Archive | [Esc] Cancel")
	} else if m.changingPassphrase {
		footerContent = CurrentTheme.Dim.Render("[Enter] Next | [Esc] Cancel")
	} else if m.journaling {
		// Only render journaling input in the journal pane, avoid duplicate
		// footer = fmt.Sprintf("%s", CurrentTheme.Input.Render(m.journalInput.View()))
		footerContent = CurrentTheme.Dim.Render("[Enter] to Save Log | [Esc] Cancel")
	} else if m.movingGoal {
		footerContent = CurrentTheme.Focused.Render("MOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		baseHelp := "[n]New|[N]Sub|[e]Edit|[z]Toggle|[w]Cycle|[W]New WS|[t]Tag|[m]Move|[D]Deps|[R]Repeat|[/]Search|[J]Journal|[p]Passphrase|[d]Delete|[A]Archive|[u]Unarchive|[L]Lock|[b]Backlog|[c]Completed|[a]Archived|[v]View|[T]Theme"
		var timerHelp string
		if m.activeSprint != nil {
			timerHelp = "|[s]PAUSE|[x]STOP"
		} else {
			timerHelp = "|[s]Start"
		}
		fullHelp := baseHelp + timerHelp + "|[ctrl+r]Report|[q]Quit"
		rawFooter = fullHelp
		footerContent = CurrentTheme.Dim.Render(fullHelp)
	}
	if footerContent != "" {
		boxed := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		innerWidth := m.width - lipgloss.Width(boxed.Render(""))
		if innerWidth < 1 {
			innerWidth = 1
		}
		content := footerContent
		if !m.creatingGoal && !m.editingGoal && !m.creatingWorkspace && !m.initializingSprints && !m.tagging && !m.themePicking && !m.depPicking && !m.settingRecurrence && !m.confirmingDelete && !m.changingPassphrase {
			tokens := strings.Split(rawFooter, "|")
			var lines []string
			var current string
			for _, token := range tokens {
				token = strings.TrimSpace(token)
				if token == "" {
					continue
				}
				if current == "" {
					current = token
					continue
				}
				candidate := current + " | " + token
				if ansi.StringWidth(candidate) > innerWidth {
					lines = append(lines, current)
					current = token
				} else {
					current = candidate
				}
			}
			if current != "" {
				lines = append(lines, current)
			}
			for _, line := range lines {
				footerHelpLines = append(footerHelpLines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, CurrentTheme.Dim.Render(line)))
			}
			content = lipgloss.JoinVertical(lipgloss.Left, footerHelpLines...)
		} else if !m.confirmingDelete && !m.changingPassphrase && (m.creatingGoal || m.editingGoal || m.creatingWorkspace || m.initializingSprints || m.tagging || m.themePicking || m.depPicking || m.settingRecurrence) {
			content = footerContent
		} else if m.changingPassphrase {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if m.confirmingDelete {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		}
		footer = boxed.Width(innerWidth).Render(content)
	}

	splitLines := func(s string) []string {
		if s == "" {
			return nil
		}
		return strings.Split(s, "\n")
	}
	trimLines := func(s string, max int) string {
		if max <= 0 || s == "" {
			return ""
		}
		lines := splitLines(s)
		if len(lines) <= max {
			return s
		}
		lines = lines[:max]
		return strings.Join(lines, "\n")
	}

	// 4. Render Journal/Search Pane
	var journalPane string
	journalHeight := 0
	if m.changingPassphrase {
		var passContent strings.Builder
		passContent.WriteString(CurrentTheme.Focused.Render("Change Passphrase") + "\n\n")
		if m.passphraseStatus != "" {
			passContent.WriteString(CurrentTheme.Break.Render(m.passphraseStatus) + "\n")
		}
		currentCursor := "  "
		newCursor := "  "
		confirmCursor := "  "
		switch m.passphraseStage {
		case 0:
			currentCursor = "> "
		case 1:
			newCursor = "> "
		case 2:
			confirmCursor = "> "
		}
		if m.passphraseHash != "" {
			passContent.WriteString(CurrentTheme.Dim.Render("Current") + "\n")
			passContent.WriteString(currentCursor + m.passphraseCurrent.View() + "\n")
		}
		passContent.WriteString(CurrentTheme.Dim.Render("New") + "\n")
		passContent.WriteString(newCursor + m.passphraseNew.View() + "\n")
		passContent.WriteString(CurrentTheme.Dim.Render("Confirm") + "\n")
		passContent.WriteString(confirmCursor + m.passphraseConfirm.View())

		passFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		passExtraWidth := lipgloss.Width(passFrame.Render(""))
		passWidth := m.width - passExtraWidth
		if passWidth < 1 {
			passWidth = 1
		}
		journalPane = passFrame.Width(passWidth).Render(passContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.settingRecurrence {
		var recContent strings.Builder
		recContent.WriteString(CurrentTheme.Focused.Render("Recurrence") + "\n")
		recContent.WriteString(CurrentTheme.Dim.Render("Tab next step | Space toggle | Enter save") + "\n\n")

		if m.recurrenceFocus == "mode" {
			recContent.WriteString(CurrentTheme.Focused.Render("Frequency") + "\n")
			for i, opt := range m.recurrenceOptions {
				cursor := "  "
				if i == m.recurrenceCursor {
					cursor = "> "
				}
				marker := " "
				if opt == m.recurrenceMode {
					marker = "*"
				}
				recContent.WriteString(fmt.Sprintf("%s[%s] %s\n", cursor, marker, opt))
			}
		} else if m.recurrenceMode == "weekly" {
			recContent.WriteString(CurrentTheme.Dim.Render("Frequency: weekly") + "\n\n")
			recContent.WriteString(CurrentTheme.Focused.Render("Weekdays") + "\n")
			for i, d := range m.weekdayOptions {
				cursor := "  "
				if m.recurrenceFocus == "items" && i == m.recurrenceItemCursor {
					cursor = "> "
				}
				check := "[ ]"
				if m.recurrenceSelected[d] {
					check = "[x]"
				}
				recContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, d))
			}
		} else if m.recurrenceMode == "monthly" {
			recContent.WriteString(CurrentTheme.Dim.Render("Frequency: monthly") + "\n\n")
			if m.recurrenceFocus == "items" {
				recContent.WriteString(CurrentTheme.Focused.Render("Months") + "\n")
				for i, mo := range m.monthOptions {
					cursor := "  "
					if m.recurrenceFocus == "items" && i == m.recurrenceItemCursor {
						cursor = "> "
					}
					check := "[ ]"
					if m.recurrenceSelected[mo] {
						check = "[x]"
					}
					recContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, mo))
				}
			} else if m.recurrenceFocus == "days" {
				recContent.WriteString(CurrentTheme.Focused.Render("Days") + "\n")
				maxDay := m.monthlyMaxDay()
				if maxDay <= 0 {
					recContent.WriteString(CurrentTheme.Dim.Render("  (select month(s) first)"))
				} else {
					if m.recurrenceDayCursor > maxDay-1 {
						m.recurrenceDayCursor = maxDay - 1
					}
					var entries []string
					for i := 0; i < maxDay; i++ {
						d := m.monthDayOptions[i]
						cursor := "  "
						if m.recurrenceFocus == "days" && i == m.recurrenceDayCursor {
							cursor = "> "
						}
						check := "[ ]"
						if m.recurrenceSelected["day:"+d] {
							check = "[x]"
						}
						entries = append(entries, fmt.Sprintf("%s%s %2s", cursor, check, d))
					}
					colWidth := 0
					for _, entry := range entries {
						w := ansi.StringWidth(entry)
						if w > colWidth {
							colWidth = w
						}
					}
					rows := (len(entries) + 1) / 2
					for i := 0; i < rows; i++ {
						left := entries[i]
						right := ""
						if i+rows < len(entries) {
							right = entries[i+rows]
						}
						padding := colWidth - ansi.StringWidth(left)
						if padding < 0 {
							padding = 0
						}
						line := left + strings.Repeat(" ", padding+2) + right
						recContent.WriteString(line + "\n")
					}
				}
			}
		}

		recFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		recExtraWidth := lipgloss.Width(recFrame.Render(""))
		recWidth := m.width - recExtraWidth
		if recWidth < 1 {
			recWidth = 1
		}
		journalPane = recFrame.Width(recWidth).Render(recContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.depPicking {
		var depContent strings.Builder
		depContent.WriteString(CurrentTheme.Focused.Render("Dependencies") + "\n")
		depContent.WriteString(CurrentTheme.Dim.Render("Space to toggle, Enter to save") + "\n\n")
		if len(m.depOptions) == 0 {
			depContent.WriteString(CurrentTheme.Dim.Render("  (no tasks)\n"))
		} else {
			maxLines := m.height / 2
			if maxLines < 6 {
				maxLines = 6
			}
			start := 0
			if m.depCursor >= maxLines {
				start = m.depCursor - maxLines + 1
			}
			end := start + maxLines
			if end > len(m.depOptions) {
				end = len(m.depOptions)
			}
			if start > 0 {
				depContent.WriteString(CurrentTheme.Dim.Render("  ...\n"))
			}
			for i := start; i < end; i++ {
				opt := m.depOptions[i]
				cursor := "  "
				if i == m.depCursor {
					cursor = "> "
				}
				check := "[ ]"
				if m.depSelected[opt.ID] {
					check = "[x]"
				}
				depContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, opt.Label))
			}
			if end < len(m.depOptions) {
				depContent.WriteString(CurrentTheme.Dim.Render("  ...\n"))
			}
		}

		depFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		depExtraWidth := lipgloss.Width(depFrame.Render(""))
		depWidth := m.width - depExtraWidth
		if depWidth < 1 {
			depWidth = 1
		}
		journalPane = depFrame.Width(depWidth).Render(depContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.themePicking {
		var themeContent strings.Builder
		themeContent.WriteString(CurrentTheme.Focused.Render("Themes") + "\n")
		themeContent.WriteString(CurrentTheme.Dim.Render("Use ↑/↓ to select, Enter to apply") + "\n\n")
		if len(m.themeNames) == 0 {
			themeContent.WriteString(CurrentTheme.Dim.Render("  (no themes)\n"))
		} else {
			for i, name := range m.themeNames {
				cursor := "  "
				if i == m.themeCursor {
					cursor = "> "
				}
				themeContent.WriteString(fmt.Sprintf("%s%s\n", cursor, name))
			}
		}
		themeFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		themeExtraWidth := lipgloss.Width(themeFrame.Render(""))
		themeWidth := m.width - themeExtraWidth
		if themeWidth < 1 {
			themeWidth = 1
		}
		journalPane = themeFrame.Width(themeWidth).Render(themeContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.tagging {
		var tagContent strings.Builder
		tagContent.WriteString(CurrentTheme.Focused.Render("Tags") + "\n")
		tagContent.WriteString(CurrentTheme.Dim.Render("Use ↑/↓ to select, Tab to toggle, Enter to save") + "\n\n")
		for i, tag := range m.defaultTags {
			cursor := "  "
			if i == m.tagCursor {
				cursor = "> "
			}
			check := "[ ]"
			if m.tagSelected[tag] {
				check = "[x]"
			}
			tagContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, tag))
		}
		if len(m.defaultTags) == 0 {
			tagContent.WriteString(CurrentTheme.Dim.Render("  (no default tags)\n"))
		}
		tagContent.WriteString("\n" + CurrentTheme.Focused.Render("Custom") + "\n")
		tagContent.WriteString(CurrentTheme.Focused.Render("> ") + m.tagInput.View())

		tagFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		tagExtraWidth := lipgloss.Width(tagFrame.Render(""))
		tagWidth := m.width - tagExtraWidth
		if tagWidth < 1 {
			tagWidth = 1
		}
		journalPane = tagFrame.Width(tagWidth).Render(tagContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.searching {
		var searchContent strings.Builder
		header := "Search Results"
		if m.searchArchiveOnly {
			header = "Search Archived"
		}
		searchContent.WriteString(CurrentTheme.Focused.Render(header) + "\n")
		searchContent.WriteString(CurrentTheme.Focused.Render("/ ") + m.searchInput.View() + "\n\n")
		if len(m.searchResults) == 0 {
			searchContent.WriteString(CurrentTheme.Dim.Render("  (no results)"))
		} else {
			for i, g := range m.searchResults {
				status := g.Status
				if status == "" {
					status = "pending"
				}
				prefix := "  "
				style := CurrentTheme.Goal
				if i == m.searchCursor {
					prefix = "> "
					style = CurrentTheme.Focused
				}
				line := fmt.Sprintf("%s %s", CurrentTheme.Dim.Render(status), g.Description)
				searchContent.WriteString(prefix + style.Render(line) + "\n")
			}
		}
		journalFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(searchContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if len(m.journalEntries) > 0 || m.journaling {
		var journalContent strings.Builder
		journalContent.WriteString(CurrentTheme.Focused.Render("Journal") + "\n\n")
		start := len(m.journalEntries) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(m.journalEntries); i++ {
			entry := m.journalEntries[i]
			var labels []string
			if entry.SprintID.Valid {
				for _, s := range m.sprints {
					if s.ID == entry.SprintID.Int64 {
						labels = append(labels, fmt.Sprintf("S%d", s.SprintNumber))
						break
					}
				}
			}
			if entry.GoalID.Valid {
				labels = append(labels, fmt.Sprintf("TASK:%d", entry.GoalID.Int64))
			}
			labelStr := ""
			if len(labels) > 0 {
				labelStr = fmt.Sprintf("[%s] ", strings.Join(labels, "|"))
			}
			line := fmt.Sprintf("%s %s%s",
				CurrentTheme.Dim.Render(entry.CreatedAt.Format("15:04")),
				CurrentTheme.Highlight.Render(labelStr),
				entry.Content)
			journalContent.WriteString(line + "\n")
		}
		if m.journaling {
			journalContent.WriteString("\n" + CurrentTheme.Focused.Render("> ") + m.journalInput.View())
		}
		journalFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(journalContent.String())
		journalHeight = lipgloss.Height(journalPane)
	}

	// 5. Calculate Layout Dimensions
	footerGap := 0
	if footer != "" {
		footerGap = 1
	}
	headerLines := splitLines(timerBox)
	footerSplit := splitLines(footer)
	availableLines := m.height - len(headerLines) - len(footerSplit) - footerGap
	if availableLines < 0 {
		availableLines = 0
	}
	minBoardHeight := 3
	if journalPane != "" {
		journalCap := availableLines - minBoardHeight
		if journalCap < 0 {
			journalCap = 0
		}
		journalPane = trimLines(journalPane, journalCap)
		journalHeight = len(splitLines(journalPane))
	}
	columnHeight := availableLines - journalHeight
	if columnHeight < 0 {
		columnHeight = 0
	}

	// Determine visible columns based on ViewMode
	var scrollableIndices []int
	showBacklog := true
	showCompleted := true
	showArchived := false
	if len(m.workspaces) > 0 && m.activeWorkspaceIdx < len(m.workspaces) {
		showBacklog = m.workspaces[m.activeWorkspaceIdx].ShowBacklog
		showCompleted = m.workspaces[m.activeWorkspaceIdx].ShowCompleted
		showArchived = m.workspaces[m.activeWorkspaceIdx].ShowArchived
	}
	for i := 0; i < len(m.sprints); i++ {
		sprint := m.sprints[i]
		if sprint.Status == "completed" && sprint.SprintNumber > 0 {
			continue
		}
		if sprint.SprintNumber == -1 && (!showCompleted || m.viewMode == ViewModeFocused) {
			continue
		}
		if sprint.SprintNumber == 0 && (!showBacklog || m.viewMode == ViewModeMinimal) {
			continue
		}
		if sprint.SprintNumber == -2 && !showArchived {
			continue
		}
		if m.viewMode == ViewModeMinimal && sprint.SprintNumber < 0 {
			continue
		}
		scrollableIndices = append(scrollableIndices, i)
	}

	displayCount := 4
	if m.viewMode == ViewModeMinimal {
		displayCount = 3
	}

	colFrame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(CurrentTheme.Dim.GetForeground()).
		Padding(0, 1)
	colExtra := lipgloss.Width(colFrame.Render(""))
	// Dynamic Width Calculation
	availableWidth := m.width // Total available width across the terminal
	availablePerCol := availableWidth / displayCount
	colContentWidth := availablePerCol - colExtra
	if colContentWidth < 0 {
		colContentWidth = 0
	}
	colExtraHeight := lipgloss.Height(colFrame.Render(""))
	// Scroll Logic
	if m.colScrollOffset > len(scrollableIndices)-displayCount {
		m.colScrollOffset = len(scrollableIndices) - displayCount
	}
	if m.colScrollOffset < 0 {
		m.colScrollOffset = 0
	}

	var visibleIndices []int
	for i := 0; i < displayCount; i++ {
		idx := m.colScrollOffset + i
		if idx < len(scrollableIndices) {
			visibleIndices = append(visibleIndices, scrollableIndices[idx])
		}
	}

	renderBoard := func(height int) string {
		if height <= 0 {
			return ""
		}
		contentHeightLocal := height - colExtraHeight
		if contentHeightLocal < 0 {
			contentHeightLocal = 0
		}
		dynColStyleLocal := colFrame.Copy().
			Width(colContentWidth).
			Height(contentHeightLocal).
			MaxHeight(contentHeightLocal)
		dynActiveColStyleLocal := dynColStyleLocal.Copy().
			BorderForeground(CurrentTheme.Border).
			BorderStyle(lipgloss.ThickBorder())

		var renderedCols []string
		if height > 4 { // Only render if we have minimal space
			for _, realIdx := range visibleIndices {
				sprint := m.sprints[realIdx]
				style := dynColStyleLocal
				if realIdx == m.focusedColIdx {
					style = dynActiveColStyleLocal
				}

				var title string
				switch sprint.SprintNumber {
				case -1:
					title = "Completed"
				case 0:
					title = "Backlog"
				case -2:
					title = "Archived"
				default:
					title = fmt.Sprintf("Sprint %d", sprint.SprintNumber)
				}

				if m.activeSprint != nil && sprint.ID == m.activeSprint.ID {
					title = "▶ " + title
				} else if sprint.Status == "paused" {
					title = "⏸ " + title
				}

				header := CurrentTheme.Header.Copy().Width(colContentWidth).Render(title)
				headerHeight := lipgloss.Height(header)

				// Render Goals
				visibleHeight := contentHeightLocal - headerHeight
				if visibleHeight < 0 {
					visibleHeight = 0
				}
				type goalRange struct {
					start int
					end   int
				}
				var lines []string
				var ranges []goalRange
				if len(sprint.Goals) == 0 {
					lines = []string{CurrentTheme.Dim.Render("  (empty)")}
				} else {
					ranges = make([]goalRange, len(sprint.Goals))
					isArchivedColumn := sprint.SprintNumber == -2
					lastArchiveDate := ""
					for j, g := range sprint.Goals {
						if isArchivedColumn {
							archiveDate := "Unknown"
							if g.ArchivedAt.Valid {
								archiveDate = g.ArchivedAt.Time.Format("2006-01-02")
							}
							if archiveDate != lastArchiveDate {
								lastArchiveDate = archiveDate
								lines = append(lines, CurrentTheme.Dim.Render(" "+archiveDate))
							}
						}
						start := len(lines)

						// Tags
						var tags []string
						var tagView string
						if g.Tags.Valid && g.Tags.String != "" && g.Tags.String != "[]" {
							tags = util.JSONToTags(g.Tags.String)
							sort.Strings(tags)
							for _, t := range tags {
								st := CurrentTheme.TagDefault
								switch t {
								case "urgent":
									st = CurrentTheme.TagUrgent
								case "docs":
									st = CurrentTheme.TagDocs
								case "blocked":
									st = CurrentTheme.TagBlocked
								case "bug":
									st = CurrentTheme.TagBug
								case "idea":
									st = CurrentTheme.TagIdea
								case "review":
									st = CurrentTheme.TagReview
								case "focus":
									st = CurrentTheme.TagFocus
								case "later":
									st = CurrentTheme.TagLater
								}
								tagView += " " + st.Render("#"+t)
							}
						}

						// Indentation & Icon
						indicator := "•"
						if len(g.Subtasks) > 0 {
							indicator = "▶"
							if g.Expanded {
								indicator = "▼"
							}
						}
						var icons []string
						for _, t := range tags {
							if icon, ok := tagIcon(t); ok {
								icons = append(icons, icon)
							}
						}
						if g.Blocked {
							icons = append(icons, "⛔")
						}
						if g.RecurrenceRule.Valid && strings.TrimSpace(g.RecurrenceRule.String) != "" {
							icons = append(icons, "↻")
						}
						prefix := indicator
						if len(icons) > 0 {
							prefix = indicator + " " + strings.Join(icons, "")
						}
						prefix = fmt.Sprintf("%s%s ", strings.Repeat("  ", g.Level), prefix)

						// Goal Description
						rawLine := fmt.Sprintf("%s%s #%d", prefix, g.Description, g.ID)
						isFocused := realIdx == m.focusedColIdx && j == m.focusedGoalIdx
						lead := "  "
						base := CurrentTheme.Goal.Copy()
						if g.Status == "completed" {
							base = CurrentTheme.CompletedGoal.Copy()
						}
						if isFocused {
							base = CurrentTheme.Focused.Copy()
							lead = "> "
						}

						leadWidth := ansi.StringWidth(lead)
						contentWidth := colContentWidth - leadWidth
						if contentWidth < 1 {
							contentWidth = 1
						}
						combined := base.Render(rawLine) + tagView
						wrapped := ansi.Wrap(combined, contentWidth, "")
						goalLines := strings.Split(wrapped, "\n")
						if len(goalLines) == 0 {
							goalLines = []string{""}
						}
						indent := strings.Repeat(" ", leadWidth)
						for i := range goalLines {
							if i == 0 {
								goalLines[i] = lead + goalLines[i]
							} else {
								goalLines[i] = indent + goalLines[i]
							}
						}

						lines = append(lines, goalLines...)
						ranges[j] = goalRange{start: start, end: len(lines)}
					}
				}

				scrollStart := 0
				if len(ranges) > 0 && realIdx == m.focusedColIdx && m.focusedGoalIdx < len(ranges) {
					r := ranges[m.focusedGoalIdx]
					if visibleHeight > 0 {
						if r.end-r.start >= visibleHeight {
							scrollStart = r.start
						} else {
							if r.start < scrollStart {
								scrollStart = r.start
							}
							if r.end > scrollStart+visibleHeight {
								scrollStart = r.end - visibleHeight
							}
						}
					}
				}
				maxStart := len(lines) - visibleHeight
				if maxStart < 0 {
					maxStart = 0
				}
				if scrollStart > maxStart {
					scrollStart = maxStart
				}

				var visibleLines []string
				if visibleHeight > 0 {
					end := scrollStart + visibleHeight
					if end > len(lines) {
						end = len(lines)
					}
					visibleLines = append(visibleLines, lines[scrollStart:end]...)
				}
				for len(visibleLines) < visibleHeight {
					visibleLines = append(visibleLines, "")
				}
				if visibleHeight > 0 && len(lines) > visibleHeight {
					if scrollStart > 0 {
						visibleLines[0] = CurrentTheme.Dim.Render("  ...")
					}
					if scrollStart+visibleHeight < len(lines) {
						visibleLines[len(visibleLines)-1] = CurrentTheme.Dim.Render("  ...")
					}
				}

				goalContent := strings.Join(visibleLines, "\n")

				var colBody string
				if goalContent == "" {
					colBody = header
				} else {
					colBody = lipgloss.JoinVertical(lipgloss.Left, header, goalContent)
				}
				renderedCols = append(renderedCols, style.Render(colBody))
			}
		}

		if len(renderedCols) == 0 {
			return ""
		}
		board := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
		board = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, board)
		return lipgloss.NewStyle().
			Height(height).
			MaxHeight(height).
			Render(board)
	}

	// 7. Assemble Final View
	// Use explicit block rendering to ensure order
	var finalView string

	// Header Block
	finalView = lipgloss.JoinVertical(lipgloss.Left, finalView, timerBox)

	boardHeight := columnHeight
	if footer != "" && boardHeight > 0 {
		boardHeight--
	}
	var board string
	if m.height > 0 {
		for boardHeight > 0 {
			board = strings.TrimRight(renderBoard(boardHeight), "\n")
			boardLines := len(splitLines(board))
			journalLines := len(splitLines(journalPane))
			total := len(headerLines) + boardLines + journalLines + footerGap + len(footerSplit)
			if total <= m.height {
				break
			}
			boardHeight--
		}
	} else {
		board = renderBoard(boardHeight)
	}
	if boardHeight == 0 {
		board = renderBoard(0)
	}

	var lines []string
	lines = append(lines, headerLines...)
	if board != "" {
		lines = append(lines, splitLines(board)...)
	} else {
		lines = append(lines, "  (Window too small)")
	}
	if journalPane != "" {
		lines = append(lines, splitLines(journalPane)...)
	}
	if footer != "" && footerGap > 0 {
		lines = append(lines, "")
	}
	if footer != "" {
		lines = append(lines, footerSplit...)
	}
	if m.height > 0 {
		if len(lines) > m.height {
			lines = lines[:m.height]
		} else if len(lines) < m.height {
			lines = append(lines, make([]string, m.height-len(lines))...)
		}
	}
	return "\x1b[H\x1b[2J" + strings.Join(lines, "\n")
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

func tagIcon(tag string) (string, bool) {
	switch tag {
	case "urgent":
		return "⚡", true
	case "docs":
		return "📄", true
	case "blocked":
		return "⛔", true
	case "waiting":
		return "⏳", true
	case "bug":
		return "🐞", true
	case "idea":
		return "💡", true
	case "review":
		return "🔎", true
	case "focus":
		return "🎯", true
	case "later":
		return "💤", true
	default:
		return "", false
	}
}
