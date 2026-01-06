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
)

const (
	SprintDuration = 90 * time.Minute
	BreakDuration  = 30 * time.Minute
)

// --- Styles ---
var (
	docStyle = lipgloss.NewStyle().Margin(1, 1)

	baseColumnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	activeBorder = lipgloss.Color("63")

	headerStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Align(lipgloss.Center)
	goalStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	completedGoalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)

	breakStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	inputStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1).Width(50)

	urgentTagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	docsTagStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	blockedTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	defaultTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return TickMsg(t) })
}

type DashboardModel struct {
	db      *sql.DB
	day     models.Day
	sprints []models.Sprint

	workspaces         []models.Workspace
	activeWorkspaceIdx int

	focusedColIdx     int
	focusedGoalIdx    int
	colScrollOffset   int
	goalScrollOffsets map[int]int

	creatingGoal        bool
	editingGoal         bool
	editingGoalID       int64
	movingGoal          bool
	creatingWorkspace   bool
	initializingSprints bool
	pendingWorkspaceID  int64
	creatingTag         bool

	journaling     bool
	journalEntries []models.JournalEntry
	journalInput   textinput.Model

	searching      bool
	searchResults  []models.Goal
	searchInput    textinput.Model

	expandedState map[int64]bool

	progress     progress.Model
	activeSprint *models.Sprint
	breakActive  bool
	breakStart   time.Time

	textInput     textinput.Model
	err           error
	Message       string
	width, height int
}

func NewDashboardModel(db *sql.DB, dayID int64) DashboardModel {
	database.EnsureDefaultWorkspace()
	workspaces, _ := database.GetWorkspaces()

	ti := textinput.New()
	ti.Placeholder = "New Objective..."
	ti.CharLimit = 100
	ti.Width = 40

	ji := textinput.New()
	ji.Placeholder = "Log your thoughts..."
	ji.CharLimit = 200
	ji.Width = 50

	si := textinput.New()
	si.Placeholder = "Search history..."
	si.CharLimit = 50
	si.Width = 30

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 30

	m := DashboardModel{
		db:                 db,
		textInput:          ti,
		journalInput:       ji,
		searchInput:        si,
		progress:           prog,
		workspaces:         workspaces,
		activeWorkspaceIdx: 0,
		focusedColIdx:      1,
		goalScrollOffsets:  make(map[int]int),
		expandedState:      make(map[int64]bool),
	}
	m.refreshData(dayID)

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
	if len(m.workspaces) == 0 {
		return
	}
	activeWS := m.workspaces[m.activeWorkspaceIdx]
	day, _ := database.GetDay(dayID)
	rawSprints, _ := database.GetSprints(dayID, activeWS.ID)
	journalEntries, _ := database.GetJournalEntries(dayID, activeWS.ID)

	var fullList []models.Sprint

	completedGoals, _ := database.GetCompletedGoalsForDay(dayID, activeWS.ID)
	rootCompleted := BuildHierarchy(completedGoals)
	var trueCompletedRoots []models.Goal
	for _, g := range rootCompleted {
		if !g.ParentID.Valid {
			trueCompletedRoots = append(trueCompletedRoots, g)
		}
	}
	flatCompleted := Flatten(trueCompletedRoots, 0, m.expandedState)
	fullList = append(fullList, models.Sprint{ID: -1, SprintNumber: -1, Goals: flatCompleted})

	var pruneCompleted func([]models.Goal) []models.Goal
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

	backlogGoals, _ := database.GetBacklogGoals(activeWS.ID)
	rootBacklog := BuildHierarchy(backlogGoals)
	activeBacklogRoots := pruneCompleted(rootBacklog)
	flatBacklog := Flatten(activeBacklogRoots, 0, m.expandedState)
	fullList = append(fullList, models.Sprint{ID: 0, SprintNumber: 0, Goals: flatBacklog})

	m.activeSprint = nil
	for i := range rawSprints {
		goals, _ := database.GetGoalsForSprint(rawSprints[i].ID)
		rootGoals := BuildHierarchy(goals)
		activeSprintRoots := pruneCompleted(rootGoals)
		flatGoals := Flatten(activeSprintRoots, 0, m.expandedState)
		rawSprints[i].Goals = flatGoals
		fullList = append(fullList, rawSprints[i])
	}

	m.sprints = fullList
	m.day = day
	m.journalEntries = journalEntries

	m.activeSprint = nil
	for i := range m.sprints {
		if m.sprints[i].Status == "active" {
			m.activeSprint = &m.sprints[i]
			break
		}
	}
}

func (m DashboardModel) Init() tea.Cmd {
	if m.activeSprint != nil || m.breakActive {
		return tea.Batch(textinput.Blink, tickCmd())
	}
	return textinput.Blink
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case TickMsg:
		if m.breakActive {
			if time.Since(m.breakStart) >= BreakDuration {
				m.breakActive = false
				return m, nil
			}
			return m, tickCmd()
		}
		if m.activeSprint != nil {
			elapsed := time.Since(m.activeSprint.StartTime.Time) + (time.Duration(m.activeSprint.ElapsedSeconds) * time.Second)
			if elapsed >= SprintDuration {
				database.CompleteSprint(m.activeSprint.ID)
				database.MovePendingToBacklog(m.activeSprint.ID)
				m.activeSprint = nil
				m.breakActive = true
				m.breakStart = time.Now()
				m.refreshData(m.day.ID)
				for i, s := range m.sprints {
					if s.Status != "completed" && s.SprintNumber > 0 {
						m.focusedColIdx = i
						break
					}
				}
				return m, tickCmd()
			}
			newProg, progCmd := m.progress.Update(msg)
			m.progress = newProg.(progress.Model)
			return m, tea.Batch(progCmd, tickCmd())
		}
	}

	if m.Message != "" {
		m.Message = ""
	}

	if m.creatingGoal || m.editingGoal || m.journaling || m.searching || m.creatingWorkspace || m.initializingSprints || m.creatingTag {
		var cmd tea.Cmd
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
						var sprintID, goalID sql.NullInt64
						if m.activeSprint != nil {
							sprintID = sql.NullInt64{Int64: m.activeSprint.ID, Valid: true}
						}
						if m.editingGoalID > 0 {
							goalID = sql.NullInt64{Int64: m.editingGoalID, Valid: true}
						}
						activeWS := m.workspaces[m.activeWorkspaceIdx]
						database.AddJournalEntry(m.day.ID, activeWS.ID, sprintID, goalID, text)
						m.refreshData(m.day.ID)
					}
					m.journaling, m.editingGoalID = false, 0
					m.journalInput.Reset()
				} else if m.creatingWorkspace {
					name := m.textInput.Value()
					if strings.TrimSpace(name) != "" {
						slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
						newID, err := database.CreateWorkspace(name, slug)
						if err == nil {
							m.pendingWorkspaceID = newID
							m.creatingWorkspace, m.initializingSprints = false, true
							m.textInput.Placeholder = "How many sprints for today? (1-8)"
							m.textInput.Reset()
						} else {
							m.Message = fmt.Sprintf("Error creating workspace: %v", err)
							m.creatingWorkspace = false
							m.textInput.Reset()
						}
					}
				} else if m.initializingSprints {
					val := m.textInput.Value()
					numSprints, err := strconv.Atoi(val)
					if err == nil && numSprints > 0 && numSprints <= 8 {
						database.BootstrapDay(m.pendingWorkspaceID, numSprints)
						m.workspaces, _ = database.GetWorkspaces()
						for i, ws := range m.workspaces {
							if ws.ID == m.pendingWorkspaceID {
								m.activeWorkspaceIdx = i
								break
							}
						}
						m.refreshData(m.day.ID)
						m.focusedColIdx = 1
					}
					m.initializingSprints, m.pendingWorkspaceID = false, 0
					m.textInput.Reset()
				} else if m.creatingTag {
					tagsToAdd := strings.Split(m.textInput.Value(), " ")
					if len(tagsToAdd) > 0 {
						database.AddTagsToGoal(m.editingGoalID, tagsToAdd)
						m.refreshData(m.day.ID)
					}
					m.creatingTag, m.editingGoalID = false, 0
					m.textInput.Reset()
				} else {
					text := m.textInput.Value()
					if strings.TrimSpace(text) != "" {
						if m.editingGoal {
							database.EditGoal(m.editingGoalID, text)
						} else if m.editingGoalID > 0 {
							database.AddSubtask(text, m.editingGoalID)
							m.expandedState[m.editingGoalID] = true
						} else {
							activeWS := m.workspaces[m.activeWorkspaceIdx]
							targetSprint := m.sprints[m.focusedColIdx]
							database.AddGoal(activeWS.ID, text, targetSprint.ID)
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
		} else if m.journaling {
			m.journalInput, cmd = m.journalInput.Update(msg)
		} else {
			m.textInput, cmd = m.textInput.Update(msg)
		}
		return m, cmd
	}

	if m.movingGoal {
		// ...
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			// ...
		case "shift+tab", "left", "h":
			// ...
		case "up", "k":
			// ...
		case "down", "j":
			// ...
		case "shift+up":
			// ...
		case "shift+down":
			// ...
		case "n":
			m.creatingGoal, m.editingGoalID = true, 0
			m.textInput.Placeholder = "New Objective..."
			m.textInput.Focus()
			return m, nil
		case "N":
			if m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				parent := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.creatingGoal, m.editingGoalID = true, parent.ID
				m.textInput.Placeholder = "New Subtask..."
				m.textInput.Focus()
				return m, nil
			}
		case "z":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.expandedState[target.ID] = !m.expandedState[target.ID]
				m.refreshData(m.day.ID)
			}
			return m, nil
		case "ctrl+j":
			m.journaling, m.editingGoalID = true, 0
			m.journalInput.Placeholder = "Log your thoughts..."
			m.journalInput.Focus()
			return m, nil
		case "J":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
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
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.editingGoal, m.editingGoalID = true, target.ID
				m.textInput.SetValue(target.Description)
				m.textInput.Focus()
				return m, nil
			}
		case "d", "backspace":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				database.DeleteGoal(m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx].ID)
				m.refreshData(m.day.ID)
				if m.focusedGoalIdx > 0 {
					m.focusedGoalIdx--
				}
			}
		case "m":
			if m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > 0 {
				m.movingGoal = true
			}
		case " ":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
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
			if !m.breakActive {
				target := m.sprints[m.focusedColIdx]
				if target.SprintNumber > 0 {
					if m.activeSprint != nil && m.activeSprint.ID == target.ID {
						elapsed := time.Since(m.activeSprint.StartTime.Time).Seconds() + float64(m.activeSprint.ElapsedSeconds)
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
				return m, nil
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
			return m, nil
		case "W":
			m.creatingWorkspace = true
			m.textInput.Placeholder = "New Workspace Name..."
			m.textInput.Focus()
			return m, nil
		case "t":
			if m.focusedColIdx > 0 && len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.creatingTag, m.editingGoalID = true, target.ID
				m.textInput.Placeholder = "Add tags (space-separated)..."
				m.textInput.Focus()
				return m, nil
			}
		case "ctrl+r":
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			GeneratePDFReport(m.day.ID, activeWS.ID)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress any key to continue.", m.err)
	}
	if m.Message != "" {
		return lipgloss.NewStyle().Padding(1, 2).Foreground(lipgloss.Color("208")).Render(m.Message)
	}

	var timerContent string
	var timerColor lipgloss.Style

	if m.breakActive {
		elapsed := time.Since(m.breakStart)
		rem := BreakDuration - elapsed
		if rem < 0 { rem = 0 }
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		timerContent = fmt.Sprintf("☕ BREAK TIME: %s REMAINING", timeStr)
		timerColor = breakStyle
	} else if m.activeSprint != nil {
		elapsed := time.Since(m.activeSprint.StartTime.Time) + (time.Duration(m.activeSprint.ElapsedSeconds) * time.Second)
		rem := SprintDuration - elapsed
		if rem < 0 { rem = 0 }
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(SprintDuration))
		timerContent = fmt.Sprintf("ACTIVE SPRINT: %d  |  %s  |  %s remaining", m.activeSprint.SprintNumber, barView, timeStr)
		timerColor = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	} else {
		if m.focusedColIdx < len(m.sprints) {
			target := m.sprints[m.focusedColIdx]
			if target.Status == "paused" {
				elapsed := time.Duration(target.ElapsedSeconds) * time.Second
				rem := SprintDuration - elapsed
				timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
				timerContent = fmt.Sprintf("PAUSED SPRINT: %d  |  %s remaining  |  [s] to Resume", target.SprintNumber, timeStr)
				timerColor = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
			} else {
				if len(m.workspaces) > 0 {
					timerContent = fmt.Sprintf("[%s] Select Sprint & Press 's' to Start", m.workspaces[m.activeWorkspaceIdx].Name)
				} else {
					timerContent = "No workspaces found."
				}
				timerColor = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			}
		}
	}

	boxWidth := m.width - 8
	if boxWidth < 20 { boxWidth = 20 }
	timerBox := lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1).Margin(1, 2).Render(timerColor.Render(timerContent))

	var footer string
	if m.creatingGoal || m.editingGoal || m.creatingWorkspace || m.initializingSprints || m.creatingTag {
		footer = fmt.Sprintf("\n\n%s", inputStyle.Render(m.textInput.View()))
	} else if m.journaling {
		footer = fmt.Sprintf("\n\n%s", inputStyle.Render(m.journalInput.View()))
	} else if m.movingGoal {
		footer = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("\n\nMOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		baseHelp := "[n]New|[N]Sub|[e]Edit|[z]Toggle|[w]Cycle|[W]New WS|[t]Tag|[m]Move|[/]Search|[J]Journal"
		var timerHelp string
		if m.activeSprint != nil {
			timerHelp = "|[s]PAUSE|[x]STOP"
		} else {
			timerHelp = "|[s]Start"
		}
		fullHelp := baseHelp + timerHelp + "|[ctrl+r]Report|[q]Quit"
		footer = "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fullHelp)
	}

	var journalPane string
	journalHeight := 0
	if m.searching {
		// ... search pane logic ...
	} else if len(m.journalEntries) > 0 || m.journaling {
		// ... journal pane logic ...
	}

	topHeight := lipgloss.Height(timerBox)
	footerHeight := lipgloss.Height(footer)
	columnHeight := m.height - topHeight - footerHeight - journalHeight - 2
	if columnHeight < 10 { columnHeight = 10 }

	dynamicColWidth := (m.width - 14) / 4
	if dynamicColWidth < 20 { dynamicColWidth = 20 }
	contentWidth := dynamicColWidth - 4

	dynColStyle := baseColumnStyle.Copy().Width(dynamicColWidth).Height(columnHeight)
	dynActiveColStyle := dynColStyle.Copy().BorderForeground(activeBorder).BorderStyle(lipgloss.ThickBorder())

	var scrollableIndices []int
	for i := 2; i < len(m.sprints); i++ {
		if m.sprints[i].Status != "completed" {
			scrollableIndices = append(scrollableIndices, i)
		}
	}
	
	if m.colScrollOffset > len(scrollableIndices)-2 {
		m.colScrollOffset = len(scrollableIndices) - 2
	}
	if m.colScrollOffset < 0 { m.colScrollOffset = 0 }

	var visibleIndices []int
	visibleIndices = append(visibleIndices, 0, 1)
	for i := 0; i < 2; i++ {
		idx := m.colScrollOffset + i
		if idx < len(scrollableIndices) {
			visibleIndices = append(visibleIndices, scrollableIndices[idx])
		}
	}

	var renderedCols []string
	for _, realIdx := range visibleIndices {
		sprint := m.sprints[realIdx]
		style := dynColStyle
		if realIdx == m.focusedColIdx {
			style = dynActiveColStyle
		}

		var title string
		switch sprint.SprintNumber {
		case -1: title = "Completed"
		case 0: title = "Backlog"
		default: title = fmt.Sprintf("Sprint %d", sprint.SprintNumber)
		}

		if m.activeSprint != nil && sprint.ID == m.activeSprint.ID {
			title = "▶ " + title
		} else if sprint.Status == "paused" {
			title = "⏸ " + title
		}

		header := headerStyle.Copy().Width(contentWidth).Render(title)
		goalViewportHeight := columnHeight - 2
		var goalContent strings.Builder
		goalContent.WriteString("\n")
		currentLines := 1
		scrollStart := m.goalScrollOffsets[realIdx]

		if len(sprint.Goals) == 0 {
			goalContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  (empty)"))
		} else {
			for j := scrollStart; j < len(sprint.Goals); j++ {
				g := sprint.Goals[j]
				indent := strings.Repeat("  ", g.Level)
				icon := "• "
				if len(g.Subtasks) > 0 {
					icon = "▶ "
					if g.Expanded {
						icon = "▼ "
					}
				}
				
				prefix := fmt.Sprintf("%s%s", indent, icon)
				
				var tagView string
				var tagWidth int
				if g.Tags != "" && g.Tags != "[]" {
					tags := util.JSONToTags(g.Tags)
					for _, t := range tags {
						style := defaultTagStyle
						switch t {
						case "urgent": style = urgentTagStyle
						case "docs": style = docsTagStyle
						case "blocked": style = blockedTagStyle
						}
						tagView += " " + style.Render("#"+t)
					}
					tagWidth = lipgloss.Width(tagView)
				}
				
				rawLine := fmt.Sprintf("%s%s", prefix, g.Description)
				
				availableWidth := contentWidth - tagWidth
				if availableWidth < 10 {
					availableWidth = 10 
				}

				var styledLine string
				if realIdx == m.focusedColIdx && j == m.focusedGoalIdx {
					base := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
					lineContent := ""
					if m.movingGoal {
						lineContent = base.Copy().Width(availableWidth).Render("> " + rawLine + " [Target?]")
					} else {
						lineContent = base.Copy().Width(availableWidth).Render("> " + rawLine)
					}
					styledLine = lipgloss.JoinHorizontal(lipgloss.Top, lineContent, tagView)
				} else {
					base := goalStyle.Copy()
					if g.Status == "completed" {
						base = completedGoalStyle.Copy()
					}
					lineContent := base.Width(availableWidth).Render("  " + rawLine)
					styledLine = lipgloss.JoinHorizontal(lipgloss.Top, lineContent, tagView)
				}

				lh := lipgloss.Height(styledLine)
				if currentLines+lh > goalViewportHeight {
					goalContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n  ... (more)"))
					break
				}
				goalContent.WriteString(styledLine + "\n")
				currentLines += lh
			}
		}

		fillLines := goalViewportHeight - currentLines
		if fillLines > 0 {
			goalContent.WriteString(strings.Repeat("\n", fillLines))
		}
		renderedCols = append(renderedCols, style.Render(lipgloss.JoinVertical(lipgloss.Left, header, goalContent.String())))
	}

	board := docStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...))
	
	mainContent := lipgloss.JoinVertical(lipgloss.Left, timerBox, board)
	if journalPane != "" {
		mainContent = lipgloss.JoinVertical(lipgloss.Left, mainContent, journalPane)
	}
	mainContent = lipgloss.JoinVertical(lipgloss.Left, mainContent, footer)
	
	return mainContent
}