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

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 30

	m := DashboardModel{
		db:                db,
		textInput:         ti,
		progress:          prog,
		focusedColIdx:     0,
		goalScrollOffsets: make(map[int]int),
	}
	m.refreshData(dayID)

	for i, s := range m.sprints {
		if s.Status != "completed" {
			m.focusedColIdx = i
			break
		}
	}

	return m
}

func (m *DashboardModel) refreshData(dayID int64) {
	day, _ := database.GetDay(dayID)
	rawSprints, _ := database.GetSprints(dayID)

	var fullList []models.Sprint
	backlogGoals, _ := database.GetBacklogGoals()
	fullList = append(fullList, models.Sprint{ID: 0, SprintNumber: 0, Goals: backlogGoals})

	for i := range rawSprints {
		goals, _ := database.GetGoalsForSprint(rawSprints[i].ID)
		rawSprints[i].Goals = goals
		fullList = append(fullList, rawSprints[i])
		if rawSprints[i].Status == "active" {
			m.activeSprint = &rawSprints[i]
		}
	}

	m.sprints = fullList
	m.day = day
}

func (m DashboardModel) Init() tea.Cmd {
	if m.activeSprint != nil || m.breakActive {
		return tea.Batch(textinput.Blink, tickCmd())
	}
	return textinput.Blink
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// --- Dynamic Constants ---
	estColWidth := 30
	if m.width > 0 {
		avail := m.width - 4
		count := avail / 30
		if count < 1 {
			count = 1
		}
		estColWidth = avail / count
	}

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
			elapsed := time.Since(m.activeSprint.StartTime.Time)
			if elapsed >= SprintDuration {
				database.CompleteSprint(m.activeSprint.ID)
				database.MovePendingToBacklog(m.activeSprint.ID)
				m.activeSprint = nil
				m.breakActive = true
				m.breakStart = time.Now()
				m.refreshData(m.day.ID)
				for i, s := range m.sprints {
					if s.Status != "completed" {
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
	if m.creatingGoal || m.editingGoal {
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.creatingGoal = false
				m.editingGoal = false
				m.textInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				text := m.textInput.Value()
				if strings.TrimSpace(text) != "" {
					if m.editingGoal {
						database.EditGoal(m.editingGoalID, text)
					} else {
						targetSprint := m.sprints[m.focusedColIdx]
						database.AddGoal(text, targetSprint.ID)
					}
					m.refreshData(m.day.ID)
				}
				m.creatingGoal = false
				m.editingGoal = false
				m.textInput.Reset()
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
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
			nextIdx := m.focusedColIdx + 1
			for nextIdx < len(m.sprints) {
				if m.sprints[nextIdx].Status != "completed" {
					m.focusedColIdx = nextIdx
					m.focusedGoalIdx = 0
					break
				}
				nextIdx++
			}
			maxCols := (m.width - 4) / estColWidth
			if maxCols < 1 {
				maxCols = 1
			}
			if m.focusedColIdx >= m.colScrollOffset+maxCols {
				m.colScrollOffset++
			}

		case "shift+tab", "left", "h":
			prevIdx := m.focusedColIdx - 1
			for prevIdx >= 0 {
				if m.sprints[prevIdx].Status != "completed" {
					m.focusedColIdx = prevIdx
					m.focusedGoalIdx = 0
					break
				}
				prevIdx--
			}
			if m.focusedColIdx < m.colScrollOffset {
				m.colScrollOffset--
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

		case "n":
			m.creatingGoal = true
			m.textInput.Placeholder = "New Objective..."
			m.textInput.Focus()
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
			if len(m.sprints[m.focusedColIdx].Goals) > 0 {
				m.movingGoal = true
			}

		case " ":
			if len(m.sprints[m.focusedColIdx].Goals) > m.focusedGoalIdx {
				goal := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				newStatus := "pending"
				if goal.Status == "pending" {
					newStatus = "completed"
				}
				database.UpdateGoalStatus(goal.ID, newStatus)
				m.refreshData(m.day.ID)
			}

		case "s":
			if m.breakActive {
				return m, nil
			}
			target := m.sprints[m.focusedColIdx]
			if target.SprintNumber > 0 && m.activeSprint == nil && target.Status == "pending" {
				database.StartSprint(target.ID)
				m.refreshData(m.day.ID)
				return m, tickCmd()
			}

		case "x": // <--- STOP (ABORT) LOGIC RESTORED
			if m.activeSprint != nil {
				database.ResetSprint(m.activeSprint.ID)
				m.activeSprint = nil
				m.refreshData(m.day.ID)
				return m, nil
			}

		case "p":
			GeneratePDFReport(m.day.ID)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DashboardModel) View() string {
	// --- 1. DYNAMIC DIMENSIONS ---
	availableWidth := m.width - 4
	if availableWidth < 20 {
		availableWidth = 20
	}

	minColWidth := 35
	maxVisibleCols := availableWidth / minColWidth
	if maxVisibleCols < 1 {
		maxVisibleCols = 1
	}

	dynamicColWidth := availableWidth / maxVisibleCols
	contentWidth := dynamicColWidth - 4

	columnHeight := m.height - 9
	if columnHeight < 10 {
		columnHeight = 10
	}

	dynColStyle := baseColumnStyle.Copy().
		Width(dynamicColWidth).
		Height(columnHeight)

	dynActiveColStyle := dynColStyle.Copy().
		BorderForeground(activeBorder).
		BorderStyle(lipgloss.ThickBorder())

	// --- 2. HEADER / TIMER ---
	var topBar string
	if m.breakActive {
		elapsed := time.Since(m.breakStart)
		rem := BreakDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		topBar = breakStyle.Render(fmt.Sprintf("\n  ☕ BREAK TIME: %s REMAINING \n", timeStr))
	} else if m.activeSprint != nil {
		elapsed := time.Since(m.activeSprint.StartTime.Time)
		rem := SprintDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(SprintDuration))
		topBar = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(
			fmt.Sprintf("\nACTIVE SPRINT: %d  |  %s  |  %s remaining\n", m.activeSprint.SprintNumber, barView, timeStr))
	} else {
		topBar = "\n[ Select Sprint & Press 's' to Start ]\n"
	}

	// --- 3. FILTER / SLICE COLUMNS ---
	var visibleColumns []models.Sprint
	for _, s := range m.sprints {
		if s.Status == "completed" {
			continue
		}
		visibleColumns = append(visibleColumns, s)
	}

	if m.colScrollOffset > len(visibleColumns)-maxVisibleCols {
		m.colScrollOffset = len(visibleColumns) - maxVisibleCols
	}
	if m.colScrollOffset < 0 {
		m.colScrollOffset = 0
	}
	endIdx := m.colScrollOffset + maxVisibleCols
	if endIdx > len(visibleColumns) {
		endIdx = len(visibleColumns)
	}

	viewportSlice := visibleColumns[m.colScrollOffset:endIdx]
	var renderedCols []string

	// --- 4. RENDER LOOP ---
	for _, sprint := range viewportSlice {
		realIdx := -1
		for idx, s := range m.sprints {
			if s.ID == sprint.ID {
				realIdx = idx
				break
			}
		}

		style := dynColStyle
		if realIdx == m.focusedColIdx {
			style = dynActiveColStyle
		}

		title := fmt.Sprintf("SPRINT %d", sprint.SprintNumber)
		if sprint.SprintNumber == 0 {
			title = "BACKLOG"
		}
		if m.activeSprint != nil && sprint.ID == m.activeSprint.ID {
			title = "▶ " + title
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
				prefix := fmt.Sprintf("%d. ", j+1)

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

	// --- 5. FOOTER ---
	var footer string
	if m.creatingGoal || m.editingGoal {
		footer = fmt.Sprintf("\n\n%s", inputStyle.Render(m.textInput.View()))
	} else if m.movingGoal {
		footer = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(
			"\n\nMOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		// Context-aware Footer
		baseHelp := "[n] New | [e] Edit | [d] Delete | [Space] Toggle | [m] Move | "
		var timerHelp string
		if m.activeSprint != nil {
			timerHelp = "[x] STOP Timer | " // <--- SHOWS STOP OPTION
		} else {
			timerHelp = "[s] Start | "
		}

		fullHelp := baseHelp + timerHelp + "[p] PDF Report | [q] Quit"
		footer = "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fullHelp)
	}

	return lipgloss.JoinVertical(lipgloss.Left, topBar, board, footer)
}
