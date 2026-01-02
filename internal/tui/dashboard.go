package tui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
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
)

type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return TickMsg(t) })
}

type DashboardModel struct {
	db      *sql.DB
	day     models.Day
	sprints []models.Sprint

	focusedColIdx     int
	focusedGoalIdx    int
	colScrollOffset   int
	goalScrollOffsets map[int]int

	creatingGoal  bool
	editingGoal   bool
	editingGoalID int64
	movingGoal    bool

	journaling     bool
	journalEntries []models.JournalEntry
	journalInput   textinput.Model

	searching      bool
	searchResults  []models.Goal
	searchInput    textinput.Model

	expandedState map[int64]bool // Tracks collapsed/expanded state of tasks

	progress     progress.Model
	activeSprint *models.Sprint
	breakActive  bool
	breakStart   time.Time

	textInput     textinput.Model
	err           error
	width, height int
}

func NewDashboardModel(db *sql.DB, dayID int64) DashboardModel {
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
		db:                db,
		textInput:         ti,
		journalInput:      ji,
		searchInput:       si,
		progress:          prog,
		focusedColIdx:     1, // Start focused on Backlog (now at index 1)
		goalScrollOffsets: make(map[int]int),
		expandedState:     make(map[int64]bool),
	}
	m.refreshData(dayID)

	// Adjust focus if necessary
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
	day, _ := database.GetDay(dayID)
	rawSprints, _ := database.GetSprints(dayID)
	journalEntries, _ := database.GetJournalEntries(dayID)

	var fullList []models.Sprint
	
	// Column 0: Completed Goals
	completedGoals, _ := database.GetCompletedGoalsForDay(dayID)
	rootCompleted := BuildHierarchy(completedGoals)
	flatCompleted := Flatten(rootCompleted, 0, m.expandedState)
	fullList = append(fullList, models.Sprint{ID: -1, SprintNumber: -1, Goals: flatCompleted})

	// Column 1: Backlog
	backlogGoals, _ := database.GetBacklogGoals()
	rootBacklog := BuildHierarchy(backlogGoals)
	var activeBacklog []models.Goal
	for _, g := range rootBacklog {
		if g.Status != "completed" {
			activeBacklog = append(activeBacklog, g)
		}
	}
	flatBacklog := Flatten(activeBacklog, 0, m.expandedState)
	fullList = append(fullList, models.Sprint{ID: 0, SprintNumber: 0, Goals: flatBacklog})

	m.activeSprint = nil
	for i := range rawSprints {
		goals, _ := database.GetGoalsForSprint(rawSprints[i].ID)
		rootGoals := BuildHierarchy(goals)
		var activeRoots []models.Goal
		for _, g := range rootGoals {
			if g.Status != "completed" {
				activeRoots = append(activeRoots, g)
			}
		}
		flatGoals := Flatten(activeRoots, 0, m.expandedState)
		
		rawSprints[i].Goals = flatGoals
		fullList = append(fullList, rawSprints[i])
	}

	m.sprints = fullList
	m.day = day
	m.journalEntries = journalEntries

	// Re-assign activeSprint to point to the stable element in m.sprints
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
			elapsed := time.Since(m.breakStart)
			if elapsed >= BreakDuration {
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

	// --- INPUT MODE ---
	if m.creatingGoal || m.editingGoal || m.journaling || m.searching {
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.creatingGoal = false
				m.editingGoal = false
				m.journaling = false
				m.searching = false
				m.textInput.Reset()
				m.journalInput.Reset()
				m.searchInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				if m.searching {
					m.searching = false
					m.searchInput.Reset()
					return m, nil
				}
				if m.journaling {
					text := m.journalInput.Value()
					if strings.TrimSpace(text) != "" {
						var sprintID sql.NullInt64
						if m.activeSprint != nil {
							sprintID = sql.NullInt64{Int64: m.activeSprint.ID, Valid: true}
						}
						database.AddJournalEntry(m.day.ID, sprintID, text)
						m.refreshData(m.day.ID)
					}
					m.journaling = false
					m.journalInput.Reset()
				} else {
					text := m.textInput.Value()
					if strings.TrimSpace(text) != "" {
						if m.editingGoal {
							database.EditGoal(m.editingGoalID, text)
						} else {
							// creatingGoal is true
							if m.editingGoalID > 0 {
								// Subtask creation mode (Parent ID stored in editingGoalID)
								database.AddSubtask(text, m.editingGoalID)
								// Expand parent so we see the new child
								m.expandedState[m.editingGoalID] = true
							} else {
								targetSprint := m.sprints[m.focusedColIdx]
								database.AddGoal(text, targetSprint.ID)
							}
						}
						m.refreshData(m.day.ID)
					}
					m.creatingGoal = false
					m.editingGoal = false
					m.editingGoalID = 0
					m.textInput.Reset()
				}
				return m, nil
			}
		}
		if m.searching {
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.searchResults, _ = database.SearchGoals(m.searchInput.Value())
		} else if m.journaling {
			m.journalInput, cmd = m.journalInput.Update(msg)
		} else {
			m.textInput, cmd = m.textInput.Update(msg)
		}
		return m, cmd
	}

	// --- MOVE MODE ---
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

					var targetID int64
					found := false
					if targetNum == 0 {
						targetID = 0
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
						// Cursor safety
						if m.focusedGoalIdx >= len(currentSprint.Goals)-1 && m.focusedGoalIdx > 0 {
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

	// --- STANDARD NAVIGATION ---
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab", "right", "l":
			// 1. Determine next candidate index
			nextIdx := -1
			if m.focusedColIdx < 1 {
				nextIdx = m.focusedColIdx + 1
			} else {
				// Search for next non-completed sprint starting after current
				for i := m.focusedColIdx + 1; i < len(m.sprints); i++ {
					if m.sprints[i].Status != "completed" {
						nextIdx = i
						break
					}
				}
			}

			if nextIdx != -1 {
				m.focusedColIdx = nextIdx
				m.focusedGoalIdx = 0
				
				// 2. Handle Scrolling if we moved into scrollable area (idx >= 2)
				if m.focusedColIdx >= 2 {
					// Build visible scrollable list to determine position
					var scrollableIndices []int
					for i := 2; i < len(m.sprints); i++ {
						if m.sprints[i].Status != "completed" {
							scrollableIndices = append(scrollableIndices, i)
						}
					}
					// Find position 'k'
					k := -1
					for idx, realIdx := range scrollableIndices {
						if realIdx == m.focusedColIdx {
							k = idx
							break
						}
					}
					// Scroll right if needed
					// Window is size 2. Visible indices: offset, offset+1
					if k >= m.colScrollOffset + 2 {
						m.colScrollOffset = k - 1 // Make it the last visible
					}
				}
			}

		case "shift+tab", "left", "h":
			prevIdx := -1
			if m.focusedColIdx <= 1 {
				prevIdx = m.focusedColIdx - 1
			} else {
				// Search backwards
				for i := m.focusedColIdx - 1; i >= 2; i-- {
					if m.sprints[i].Status != "completed" {
						prevIdx = i
						break
					}
				}
				if prevIdx == -1 {
					prevIdx = 1 // Fallback to Backlog
				}
			}

			if prevIdx >= 0 {
				m.focusedColIdx = prevIdx
				m.focusedGoalIdx = 0

				// Scroll check
				if m.focusedColIdx >= 2 {
					var scrollableIndices []int
					for i := 2; i < len(m.sprints); i++ {
						if m.sprints[i].Status != "completed" {
							scrollableIndices = append(scrollableIndices, i)
						}
					}
					k := -1
					for idx, realIdx := range scrollableIndices {
						if realIdx == m.focusedColIdx {
							k = idx
							break
						}
					}
					if k < m.colScrollOffset {
						m.colScrollOffset = k
					}
				}
			}

		case "up", "k":
			if m.focusedGoalIdx > 0 {
				m.focusedGoalIdx--
				if m.focusedGoalIdx < m.goalScrollOffsets[m.focusedColIdx] {
					m.goalScrollOffsets[m.focusedColIdx]--
				}
			}
		case "down", "j":
			goalsLen := len(m.sprints[m.focusedColIdx].Goals)
			if m.focusedGoalIdx < goalsLen-1 {
				m.focusedGoalIdx++
				if m.focusedGoalIdx >= m.goalScrollOffsets[m.focusedColIdx]+10 {
					m.goalScrollOffsets[m.focusedColIdx]++
				}
			}

		case "shift+up", "K":
			currentSprint := m.sprints[m.focusedColIdx]
			if m.focusedGoalIdx > 0 && len(currentSprint.Goals) > 1 {
				g1 := currentSprint.Goals[m.focusedGoalIdx]
				g2 := currentSprint.Goals[m.focusedGoalIdx-1]
				database.SwapGoalRanks(g1.ID, g2.ID)
				m.refreshData(m.day.ID)
				m.focusedGoalIdx--
			}

		case "shift+down", "J":
			currentSprint := m.sprints[m.focusedColIdx]
			if m.focusedGoalIdx < len(currentSprint.Goals)-1 && len(currentSprint.Goals) > 1 {
				g1 := currentSprint.Goals[m.focusedGoalIdx]
				g2 := currentSprint.Goals[m.focusedGoalIdx+1]
				database.SwapGoalRanks(g1.ID, g2.ID)
				m.refreshData(m.day.ID)
				m.focusedGoalIdx++
			}

		case "n":
			if m.focusedColIdx == 0 { // Cannot add goals to 'Completed' column
				return m, nil
			}
			m.creatingGoal = true
			m.textInput.Placeholder = "New Objective..."
			m.textInput.Focus()
			return m, nil

		case "N": // Shift+n
			if m.focusedColIdx == 0 { return m, nil }
			currentSprint := m.sprints[m.focusedColIdx]
			if len(currentSprint.Goals) > m.focusedGoalIdx {
				parent := currentSprint.Goals[m.focusedGoalIdx]
				m.creatingGoal = true
				m.editingGoalID = parent.ID // Hack: store parent ID here temporarily or add new field
				// Better to use a specific flag or field.
				// But creatingGoal is boolean.
				// I'll add a 'parentID' field to DashboardModel if needed, 
				// OR just use editingGoalID as "ContextID" since I'm not editing.
				// Let's use a new state: creatingSubtask
				// I'll add `creatingSubtask bool` and `parentGoalID int64` to struct later?
				// For now, I'll piggyback.
				// Wait, I can't easily change struct in REPLACE without replacing whole struct def.
				// I'll assume I can use `editingGoalID` as "ParentID" when `creatingGoal` is true.
				// But `creatingGoal` usually implies "New Root".
				// I need to distinguish "New Root" vs "New Child".
				// I'll use `movingGoal` as a flag? No.
				// I'll add `creatingSubtask` bool to DashboardModel in a separate step if I can.
				// Or... I can check `editingGoalID` != 0 when `creatingGoal` is true.
				// Normally `creatingGoal` -> `AddGoal(text, sprintID)`.
				// If I set `editingGoalID` to parentID, I can check that.
				m.editingGoalID = parent.ID
				m.textInput.Placeholder = "New Subtask..."
				m.textInput.Focus()
				return m, nil
			}

		case "z":
			currentSprint := m.sprints[m.focusedColIdx]
			if len(currentSprint.Goals) > m.focusedGoalIdx {
				target := currentSprint.Goals[m.focusedGoalIdx]
				// Toggle in map
				if m.expandedState[target.ID] {
					delete(m.expandedState, target.ID)
				} else {
					m.expandedState[target.ID] = true
				}
				m.refreshData(m.day.ID)
			}
			return m, nil

		case "ctrl+j":
			m.journaling = true
			m.journalInput.Focus()
			return m, nil

		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, nil

		case "e":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				m.editingGoal = true
				m.editingGoalID = target.ID
				m.textInput.SetValue(target.Description)
				m.textInput.Focus()
				return m, nil
			}

		case "d", "backspace":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				target := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				database.DeleteGoal(target.ID)
				m.refreshData(m.day.ID)
				if m.focusedGoalIdx > 0 {
					m.focusedGoalIdx--
				}
			}

		case "m":
			if m.focusedColIdx == 0 { // Cannot move from 'Completed'
				return m, nil
			}
			if len(m.sprints[m.focusedColIdx].Goals) > 0 {
				m.movingGoal = true
			}

		case " ":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				goal := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				
				// Block completion if subtasks are pending
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
				}
			}

		case "s":
			if m.breakActive {
				return m, nil
			}
			target := m.sprints[m.focusedColIdx]
			if target.SprintNumber > 0 {
				if m.activeSprint != nil && m.activeSprint.ID == target.ID {
					// Toggle Pause if already active
					elapsed := time.Since(m.activeSprint.StartTime.Time).Seconds() + float64(m.activeSprint.ElapsedSeconds)
					database.PauseSprint(target.ID, int(elapsed))
					m.refreshData(m.day.ID)
				} else if m.activeSprint == nil && (target.Status == "pending" || target.Status == "paused") {
					database.StartSprint(target.ID)
					m.refreshData(m.day.ID)
					return m, tickCmd()
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
			database.AppendSprint(m.day.ID)
			m.refreshData(m.day.ID)

		case "ctrl+r":
			GeneratePDFReport(m.day.ID)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DashboardModel) View() string {
	// --- 1. PRE-CALCULATE SECTIONS ---
	
	// A. Header / Timer
	var timerContent string
	var timerColor lipgloss.Style

	if m.breakActive {
		elapsed := time.Since(m.breakStart)
		rem := BreakDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		timerContent = fmt.Sprintf("☕ BREAK TIME: %s REMAINING", timeStr)
		timerColor = breakStyle
	} else if m.activeSprint != nil {
		elapsed := time.Since(m.activeSprint.StartTime.Time) + (time.Duration(m.activeSprint.ElapsedSeconds) * time.Second)
		rem := SprintDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(SprintDuration))
		timerContent = fmt.Sprintf("ACTIVE SPRINT: %d  |  %s  |  %s remaining", m.activeSprint.SprintNumber, barView, timeStr)
		timerColor = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	} else {
		// Check if focused sprint is paused
		target := m.sprints[m.focusedColIdx]
		if target.Status == "paused" {
			elapsed := time.Duration(target.ElapsedSeconds) * time.Second
			rem := SprintDuration - elapsed
			timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
			timerContent = fmt.Sprintf("PAUSED SPRINT: %d  |  %s remaining  |  [s] to Resume", target.SprintNumber, timeStr)
			timerColor = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		} else {
			timerContent = "Select Sprint & Press 's' to Start"
			timerColor = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		}
	}

	// Render Timer Box
	// Width = Window - Margins(4) - Border(2) - Padding(2) = Window - 8
	boxWidth := m.width - 8
	if boxWidth < 20 {
		boxWidth = 20
	}

	timerBox := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Margin(1, 2). // Top/Bottom: 1, Left/Right: 2
		Render(timerColor.Render(timerContent))

	// B. Footer
	var footer string
	if m.creatingGoal || m.editingGoal {
		footer = fmt.Sprintf("\n\n%s", inputStyle.Render(m.textInput.View()))
	} else if m.movingGoal {
		footer = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(
			"\n\nMOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		baseHelp := "[n] New | [Shift+N] Subtask | [e] Edit | [d] Delete | [z] Toggle | [Space] Status | [m] Move | [/] Search | [Ctrl+J] Journal | "
		var timerHelp string
		if m.activeSprint != nil {
			timerHelp = "[s] PAUSE | [x] STOP | "
		} else {
			timerHelp = "[s] Start | "
		}
		fullHelp := baseHelp + timerHelp + "[Ctrl+R] Report | [q] Quit"
		footer = "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fullHelp)
	}

	// C. Journal / Search Pane (Calculate height first)
	var journalPane string
	journalHeight := 0

	if m.searching {
		var searchContent strings.Builder
		searchContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Search Results") + "\n")
		searchContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("/ ") + m.searchInput.View() + "\n\n")
		
		if len(m.searchResults) == 0 {
			searchContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  (no results)"))
		} else {
			for _, g := range m.searchResults {
				status := "[ ]"
				if g.Status == "completed" {
					status = "[x]"
				}
				searchContent.WriteString(fmt.Sprintf("  %s %s\n", 
					lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(status),
					g.Description))
			}
		}
		journalPane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(m.width - 4).
			Render(searchContent.String())
		
		journalHeight = lipgloss.Height(journalPane)

	} else if len(m.journalEntries) > 0 || m.journaling {
		var journalContent strings.Builder
		// Title styling consistent with columns
		journalContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Journal") + "\n\n")
		
		// Show last 3 entries
		start := len(m.journalEntries) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(m.journalEntries); i++ {
			entry := m.journalEntries[i]
			timeStr := entry.CreatedAt.Format("15:04")
			sprintLabel := ""
			if entry.SprintID.Valid {
				// Find sprint number
				for _, s := range m.sprints {
					if s.ID == entry.SprintID.Int64 {
						sprintLabel = fmt.Sprintf("[S%d] ", s.SprintNumber)
						break
					}
				}
			}
			// Fix spacing/offset
			line := fmt.Sprintf("%s %s%s", 
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(timeStr),
				lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Render(sprintLabel),
				entry.Content)
			journalContent.WriteString(line + "\n")
		}
		
		if m.journaling {
			journalContent.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> ") + m.journalInput.View())
		}
		
		journalPane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(m.width - 4).
			Render(journalContent.String())
			
		journalHeight = lipgloss.Height(journalPane)
	}

	// --- 2. LAYOUT CALCULATION ---
	
	// Vertical Space
	topHeight := lipgloss.Height(timerBox)
	footerHeight := lipgloss.Height(footer)
	
	// Available height for sprints = Total - Top - Footer - Journal - Margins
	columnHeight := m.height - topHeight - footerHeight - journalHeight - 2 // -2 for main margins
	if columnHeight < 10 {
		columnHeight = 10 // Minimum safe height
	}

	// Horizontal Space
	// Enforce exactly 4 visible columns
	// Subtract extra space (14) for borders, padding, and document margins.
	// Formula: (TotalWidth - DocMargin(2) - Borders(4*2) - Buffer(4)) / 4
	dynamicColWidth := (m.width - 14) / 4
	if dynamicColWidth < 20 {
		dynamicColWidth = 20
	}
	contentWidth := dynamicColWidth - 4

	dynColStyle := baseColumnStyle.Copy().
		Width(dynamicColWidth).
		Height(columnHeight)

	dynActiveColStyle := dynColStyle.Copy().
		BorderForeground(activeBorder).
		BorderStyle(lipgloss.ThickBorder())


	// --- 3. BUILD RENDER LIST (FIXED + SCROLLABLE) ---
	// Always show: Completed (0), Backlog (1)
	// Scrollable: 2 Sprints
	
	var scrollableIndices []int
	for i := 2; i < len(m.sprints); i++ {
		if m.sprints[i].Status != "completed" {
			scrollableIndices = append(scrollableIndices, i)
		}
	}
	
	// Clamp Offset
	if m.colScrollOffset > len(scrollableIndices) - 2 {
		m.colScrollOffset = len(scrollableIndices) - 2
	}
	if m.colScrollOffset < 0 {
		m.colScrollOffset = 0
	}

	var visibleIndices []int
	visibleIndices = append(visibleIndices, 0)
	visibleIndices = append(visibleIndices, 1)

	// Add 2 visible scrollable sprints
	for i := 0; i < 2; i++ {
		idx := m.colScrollOffset + i
		if idx < len(scrollableIndices) {
			visibleIndices = append(visibleIndices, scrollableIndices[idx])
		}
	}

	var renderedCols []string

	// --- 4. RENDER COLUMNS ---
	for _, realIdx := range visibleIndices {
		sprint := m.sprints[realIdx]
		
		style := dynColStyle
		if realIdx == m.focusedColIdx {
			style = dynActiveColStyle
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
				
				// Visual Hierarchy
				indent := strings.Repeat("  ", g.Level)
				icon := "• "
				// We check if this goal was originally a parent. 
				// Since we flat-mapped, we can check if it has subtasks populated.
				// Note: 'g' is a value copy from the slice.
				if len(g.Subtasks) > 0 {
					if g.Expanded {
						icon = "▼ "
					} else {
						icon = "▶ "
					}
				}
				
				prefix := fmt.Sprintf("%s%s", indent, icon)

				rawLine := fmt.Sprintf("%s%s", prefix, g.Description)
				var styledLine string

				if realIdx == m.focusedColIdx && j == m.focusedGoalIdx {
					base := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Width(contentWidth)
					if m.movingGoal {
						styledLine = base.Render("> " + rawLine + " [Target?]")
					} else {
						styledLine = base.Render("> " + rawLine)
					}
				} else {
					base := goalStyle.Copy().Width(contentWidth)
					if g.Status == "completed" {
						base = completedGoalStyle.Copy().Width(contentWidth)
					}
					styledLine = base.Render("  " + rawLine)
				}

				lineHeight := lipgloss.Height(styledLine)

				if currentLines+lineHeight > goalViewportHeight {
					goalContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n  ... (more)"))
					break
				}

				goalContent.WriteString(styledLine + "\n")
				currentLines += lineHeight
			}
		}

		fillLines := goalViewportHeight - currentLines
		if fillLines > 0 {
			goalContent.WriteString(strings.Repeat("\n", fillLines))
		}

		renderedCols = append(renderedCols, style.Render(lipgloss.JoinVertical(lipgloss.Left, header, goalContent.String())))
	}

	board := docStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...))

	// --- 5. ASSEMBLE ---
	// Stack: TopBar -> Board -> Journal -> Footer
	
	mainContent := lipgloss.JoinVertical(lipgloss.Left, timerBox, board)
	if journalPane != "" {
		mainContent = lipgloss.JoinVertical(lipgloss.Left, mainContent, journalPane)
	}
	mainContent = lipgloss.JoinVertical(lipgloss.Left, mainContent, footer)
	
	return mainContent
}
