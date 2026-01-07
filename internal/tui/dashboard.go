package tui

import (
	"database/sql"
	"fmt"
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
	db                  *sql.DB
	day                 models.Day
	sprints             []models.Sprint
	workspaces          []models.Workspace
	activeWorkspaceIdx  int
	viewMode            int
	focusedColIdx       int
	focusedGoalIdx      int
	colScrollOffset     int
	goalScrollOffsets   map[int]int
	creatingGoal        bool
	editingGoal         bool
	editingGoalID       int64
	movingGoal          bool
	creatingWorkspace   bool
	initializingSprints bool
	pendingWorkspaceID  int64
	creatingTag         bool
	journaling          bool
	journalEntries      []models.JournalEntry
	journalInput        textinput.Model
	searching           bool
	searchResults       []models.Goal
	searchInput         textinput.Model
	expandedState       map[int64]bool
	progress            progress.Model
	activeSprint        *models.Sprint
	breakActive         bool
	breakStart          time.Time
	textInput           textinput.Model
	err                 error
	Message             string
	width, height       int
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

	m := DashboardModel{
		db:                 db,
		textInput:          ti,
		journalInput:       ji,
		searchInput:        si,
		progress:           progress.New(progress.WithDefaultGradient()),
		workspaces:         workspaces,
		activeWorkspaceIdx: 0,
		focusedColIdx:      1,
		goalScrollOffsets:  make(map[int]int),
		expandedState:      make(map[int64]bool),
	}
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

	day, _ := database.GetDay(dayID)
	rawSprints, _ := database.GetSprints(dayID, activeWS.ID)
	journalEntries, _ := database.GetJournalEntries(dayID, activeWS.ID)

	var fullList []models.Sprint

	// Completed Column
	completedGoals, _ := database.GetCompletedGoalsForDay(dayID, activeWS.ID)
	flatCompleted := Flatten(BuildHierarchy(completedGoals), 0, m.expandedState)
	fullList = append(fullList, models.Sprint{ID: -1, SprintNumber: -1, Goals: flatCompleted})

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

	// Backlog Column
	backlogGoals, _ := database.GetBacklogGoals(activeWS.ID)
	flatBacklog := Flatten(pruneCompleted(BuildHierarchy(backlogGoals)), 0, m.expandedState)
	fullList = append(fullList, models.Sprint{ID: 0, SprintNumber: 0, Goals: flatBacklog})

	// Sprints
	for i := range rawSprints {
		goals, _ := database.GetGoalsForSprint(rawSprints[i].ID)
		rawSprints[i].Goals = Flatten(pruneCompleted(BuildHierarchy(goals)), 0, m.expandedState)
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

func (m DashboardModel) Init() tea.Cmd { return textinput.Blink }

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
	}

	// Input Modes
	if m.creatingGoal || m.editingGoal || m.journaling || m.searching || m.creatingWorkspace || m.initializingSprints || m.creatingTag {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.creatingGoal, m.editingGoal, m.journaling, m.searching, m.creatingWorkspace, m.initializingSprints, m.creatingTag = false, false, false, false, false, false, false
				m.textInput.Reset()
				m.journalInput.Reset()
				m.searchInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				if m.searching {
					m.searching = false
					m.searchInput.Reset()
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
				} else if m.creatingTag {
					tags := strings.Split(m.textInput.Value(), " ")
					if len(tags) > 0 {
						database.AddTagsToGoal(m.editingGoalID, tags)
						m.refreshData(m.day.ID)
					}
					m.creatingTag, m.editingGoalID = false, 0
					m.textInput.Reset()
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
		if m.searching {
			m.searchInput, cmd = m.searchInput.Update(msg)
			if _, ok := msg.(tea.KeyMsg); ok && len(m.workspaces) > 0 {
				m.searchResults, m.err = database.Search(util.ParseSearchQuery(m.searchInput.Value()), m.workspaces[m.activeWorkspaceIdx].ID)
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
			m.searchInput.Focus()
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
				database.DeleteGoal(m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx].ID)
				m.refreshData(m.day.ID)
				if m.focusedGoalIdx > 0 {
					m.focusedGoalIdx--
				}
			}
		case "m":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				m.movingGoal = true
				return m, nil
			}
		case " ":
			if m.focusedColIdx < len(m.sprints) && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				goal := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
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
				m.creatingTag, m.editingGoalID = true, target.ID
				m.textInput.Placeholder = "Add tags (space-separated)..."
				m.textInput.Focus()
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
				activeWS := m.workspaces[m.activeWorkspaceIdx]
				newTheme := "default"
				if activeWS.Theme == "default" {
					newTheme = "dracula"
				}

				database.UpdateWorkspaceTheme(activeWS.ID, newTheme)
				m.workspaces[m.activeWorkspaceIdx].Theme = newTheme
				SetTheme(newTheme)
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
	if m.creatingGoal || m.editingGoal || m.creatingWorkspace || m.initializingSprints || m.creatingTag {
		footerContent = CurrentTheme.Input.Render(m.textInput.View())
	} else if m.journaling {
		// Only render journaling input in the journal pane, avoid duplicate
		// footer = fmt.Sprintf("%s", CurrentTheme.Input.Render(m.journalInput.View()))
		footerContent = CurrentTheme.Dim.Render("[Enter] to Save Log | [Esc] Cancel")
	} else if m.movingGoal {
		footerContent = CurrentTheme.Focused.Render("MOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		baseHelp := "[n]New|[N]Sub|[e]Edit|[z]Toggle|[w]Cycle|[W]New WS|[t]Tag|[m]Move|[/]Search|[J]Journal|[v]View|[T]Theme"
		var timerHelp string
		if m.activeSprint != nil {
			timerHelp = "|[s]PAUSE|[x]STOP"
		} else {
			timerHelp = "|[s]Start"
		}
		fullHelp := baseHelp + timerHelp + "|[ctrl+r]Report|[q]Quit"
		footerContent = CurrentTheme.Dim.Render(fullHelp)
	}
	if footerContent != "" {
		if !m.creatingGoal && !m.editingGoal && !m.creatingWorkspace && !m.initializingSprints && !m.creatingTag {
			footer = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, footerContent)
		} else {
			footer = footerContent
		}
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
	if m.searching {
		var searchContent strings.Builder
		searchContent.WriteString(CurrentTheme.Focused.Render("Search Results") + "\n")
		searchContent.WriteString(CurrentTheme.Focused.Render("/ ") + m.searchInput.View() + "\n\n")
		if len(m.searchResults) == 0 {
			searchContent.WriteString(CurrentTheme.Dim.Render("  (no results)"))
		} else {
			for _, g := range m.searchResults {
				status := "[ ]"
				if g.Status == "completed" {
					status = "[x]"
				}
				searchContent.WriteString(fmt.Sprintf("  %s %s\n", CurrentTheme.Dim.Render(status), g.Description))
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
	footerLines := splitLines(footer)
	availableLines := m.height - len(headerLines) - len(footerLines) - footerGap
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
	for i := 0; i < len(m.sprints); i++ {
		sprint := m.sprints[i]
		if sprint.Status == "completed" && sprint.SprintNumber > 0 {
			continue
		}
		if m.viewMode == ViewModeFocused && sprint.SprintNumber == -1 {
			continue
		}
		if m.viewMode == ViewModeMinimal && sprint.SprintNumber <= 0 {
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
					for j, g := range sprint.Goals {
						start := len(lines)

						// Indentation & Icon
						prefix := fmt.Sprintf("%s%s", strings.Repeat("  ", g.Level), "• ")
						if len(g.Subtasks) > 0 {
							prefix = fmt.Sprintf("%s%s", strings.Repeat("  ", g.Level), "▶ ")
							if g.Expanded {
								prefix = fmt.Sprintf("%s%s", strings.Repeat("  ", g.Level), "▼ ")
							}
						}

						// Tags
						var tagView string
						if g.Tags.Valid && g.Tags.String != "" && g.Tags.String != "[]" {
							tags := util.JSONToTags(g.Tags.String)
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

						// Goal Description
						rawLine := fmt.Sprintf("%s%s", prefix, g.Description)
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
			total := len(headerLines) + boardLines + journalLines + footerGap + len(footerLines)
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
		lines = append(lines, footerLines...)
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
