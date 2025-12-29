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
	// Column dimensions
	ColWidth      = 26               // Fits nicely in standard terminals
	ColOuterWidth = ColWidth + 2 + 2 // Width + Padding(1*2) + Border(1*2)
)

// --- Styles ---
var (
	docStyle = lipgloss.NewStyle().Margin(1, 1)

	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(ColWidth).
			Height(20)

	activeColumnStyle = columnStyle.Copy().
				BorderForeground(lipgloss.Color("63")).
				BorderStyle(lipgloss.ThickBorder())

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Align(lipgloss.Center).
			Width(ColWidth)

	goalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(ColWidth)

	completedGoalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Strikethrough(true).
				Width(ColWidth)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1).
			Width(50)
)

// --- Messages ---
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// --- Model ---
type DashboardModel struct {
	db      *sql.DB
	day     models.Day
	sprints []models.Sprint

	// Navigation State
	focusedColIdx  int
	focusedGoalIdx int
	scrollOffset   int // Tracks left-most visible column

	// Mode State
	creatingGoal bool
	movingGoal   bool

	// Timer State
	progress     progress.Model
	activeSprint *models.Sprint

	// Components
	textInput textinput.Model
	err       error
	width     int
	height    int
}

func NewDashboardModel(db *sql.DB, dayID int64) DashboardModel {
	ti := textinput.New()
	ti.Placeholder = "New Objective..."
	ti.CharLimit = 100
	ti.Width = 40

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 30

	m := DashboardModel{
		db:            db,
		textInput:     ti,
		progress:      prog,
		focusedColIdx: 1, // Default to Sprint 1
		scrollOffset:  0,
	}

	m.refreshData(dayID)
	return m
}

func (m *DashboardModel) refreshData(dayID int64) {
	day, _ := database.GetDay(dayID)
	sprints, _ := database.GetSprints(dayID)

	// Fetch goals & check for active sprint
	for i := range sprints {
		goals, _ := database.GetGoalsForSprint(sprints[i].ID)
		sprints[i].Goals = goals

		if sprints[i].Status == "active" {
			m.activeSprint = &sprints[i]
		}
	}

	// Fetch Backlog
	backlogGoals, _ := database.GetBacklogGoals()
	backlogSprint := models.Sprint{ID: 0, SprintNumber: 0, Goals: backlogGoals}

	m.sprints = append([]models.Sprint{backlogSprint}, sprints...)
	m.day = day
}

func (m DashboardModel) Init() tea.Cmd {
	if m.activeSprint != nil {
		return tea.Batch(textinput.Blink, tickCmd())
	}
	return textinput.Blink
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	// --- TIMER LOGIC ---
	if _, ok := msg.(TickMsg); ok {
		if m.activeSprint != nil {
			elapsed := time.Since(m.activeSprint.StartTime.Time)

			// Check for Finish
			if elapsed >= SprintDuration {
				database.CompleteSprint(m.activeSprint.ID)
				m.activeSprint = nil
				m.refreshData(m.day.ID)
				return m, nil
			}

			// Update Progress (Type Assertion Fix)
			newProg, progCmd := m.progress.Update(msg)
			m.progress = newProg.(progress.Model)
			return m, tea.Batch(progCmd, tickCmd())
		}
		return m, nil
	}

	// --- INPUT MODE ---
	if m.creatingGoal {
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.creatingGoal = false
				m.textInput.Reset()
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				desc := m.textInput.Value()
				if strings.TrimSpace(desc) != "" {
					targetSprint := m.sprints[m.focusedColIdx]
					if err := database.AddGoal(desc, targetSprint.ID); err != nil {
						m.err = err
					} else {
						m.refreshData(m.day.ID)
					}
				}
				m.creatingGoal = false
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
			// Check for digits 0-8
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
						m.focusedGoalIdx = 0
					}
				}
				m.movingGoal = false
				return m, nil
			}
		}
		return m, nil
	}

	// --- NAVIGATION & ACTIONS ---
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		// QUIT
		case "q", "ctrl+c":
			return m, tea.Quit

		// NAVIGATION
		case "tab", "right", "l":
			if m.focusedColIdx < len(m.sprints)-1 {
				m.focusedColIdx++
				m.focusedGoalIdx = 0

				// Viewport Scroll Right
				maxVisibleCols := (m.width - 4) / ColOuterWidth
				if maxVisibleCols < 1 {
					maxVisibleCols = 1
				}
				if m.focusedColIdx >= m.scrollOffset+maxVisibleCols {
					m.scrollOffset++
				}
			}
		case "shift+tab", "left", "h":
			if m.focusedColIdx > 0 {
				m.focusedColIdx--
				m.focusedGoalIdx = 0

				// Viewport Scroll Left
				if m.focusedColIdx < m.scrollOffset {
					m.scrollOffset--
				}
			}
		case "up", "k":
			if m.focusedGoalIdx > 0 {
				m.focusedGoalIdx--
			}
		case "down", "j":
			if m.focusedGoalIdx < len(m.sprints[m.focusedColIdx].Goals)-1 {
				m.focusedGoalIdx++
			}

		// ACTIONS
		case "n":
			m.creatingGoal = true
			m.textInput.Focus()
			return m, nil

		case "m":
			if len(m.sprints[m.focusedColIdx].Goals) > 0 {
				m.movingGoal = true
			}

		case " ": // Spacebar Toggle
			if len(m.sprints[m.focusedColIdx].Goals) > 0 {
				goal := m.sprints[m.focusedColIdx].Goals[m.focusedGoalIdx]
				newStatus := "completed"
				if goal.Status == "completed" {
					newStatus = "pending"
				}
				database.UpdateGoalStatus(goal.ID, newStatus)
				m.refreshData(m.day.ID)
			}

		case "s": // START Timer
			targetSprint := m.sprints[m.focusedColIdx]
			if targetSprint.SprintNumber > 0 && m.activeSprint == nil && targetSprint.Status == "pending" {
				database.StartSprint(targetSprint.ID)
				m.refreshData(m.day.ID)
				return m, tickCmd()
			}

		case "x": // STOP (Abort) Timer
			if m.activeSprint != nil {
				database.ResetSprint(m.activeSprint.ID)
				m.activeSprint = nil
				m.refreshData(m.day.ID)
				return m, nil
			}

		case "p": // Print Report
			GenerateReport(m.day.ID)
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m DashboardModel) View() string {
	// 1. Timer Overlay
	var timerView string
	if m.activeSprint != nil {
		elapsed := time.Since(m.activeSprint.StartTime.Time)
		remaining := SprintDuration - elapsed
		if remaining < 0 {
			remaining = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(remaining.Minutes()), int(remaining.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(SprintDuration))

		timerView = fmt.Sprintf("\nACTIVE SPRINT: %d  |  %s  |  %s remaining\n",
			m.activeSprint.SprintNumber, barView, timeStr)
		timerView = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(timerView)
	} else {
		timerView = "\n[ Select Sprint & Press 's' to Start ]\n"
	}

	// 2. Render Columns (VIEWPORT SLICING)
	var columns []string

	availableWidth := m.width - 4
	maxVisibleCols := availableWidth / ColOuterWidth
	if maxVisibleCols < 1 {
		maxVisibleCols = 1
	}

	endIdx := m.scrollOffset + maxVisibleCols
	if endIdx > len(m.sprints) {
		endIdx = len(m.sprints)
	}
	visibleSprints := m.sprints[m.scrollOffset:endIdx]

	for i, sprint := range visibleSprints {
		trueIndex := m.scrollOffset + i

		style := columnStyle
		if trueIndex == m.focusedColIdx {
			style = activeColumnStyle
		}

		var title string
		if sprint.SprintNumber == 0 {
			title = "BACKLOG"
		} else {
			title = fmt.Sprintf("SPRINT %d", sprint.SprintNumber)
		}

		if m.activeSprint != nil && sprint.ID == m.activeSprint.ID {
			title = "▶ " + title
		}

		header := headerStyle.Render(title)

		var goalContent strings.Builder
		goalContent.WriteString("\n")

		if len(sprint.Goals) == 0 {
			goalContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  (empty)"))
		} else {
			for idx, g := range sprint.Goals {
				prefix := fmt.Sprintf("%d. ", idx+1)
				// Natural wrapping via lipgloss width constraint
				desc := g.Description

				line := fmt.Sprintf("%s%s", prefix, desc)

				if trueIndex == m.focusedColIdx && idx == m.focusedGoalIdx {
					line = "> " + line
					if m.movingGoal {
						line = line + " (?)"
					}
					line = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(line)
				} else {
					line = "  " + line
					if g.Status == "completed" {
						line = completedGoalStyle.Render(line)
					} else {
						line = goalStyle.Render(line)
					}
				}
				goalContent.WriteString(line + "\n")
			}
		}

		colView := style.Render(lipgloss.JoinVertical(lipgloss.Left, header, goalContent.String()))
		columns = append(columns, colView)
	}

	board := docStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, columns...))

	// 3. Footer
	var footer string
	if m.creatingGoal {
		footer = fmt.Sprintf("\n\n%s", inputStyle.Render(m.textInput.View()))
	} else if m.movingGoal {
		footer = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(
			"\n\nMove to Sprint? (0=Backlog, 1-8=Sprint #)")
	} else {
		pageInfo := fmt.Sprintf(" [Cols %d-%d of %d]", m.scrollOffset+1, endIdx, len(m.sprints))

		baseHelp := "[n] New | [Space] Toggle | [m] Move | "
		var timerHelp string

		if m.activeSprint != nil {
			timerHelp = "[x] STOP Timer | "
		} else {
			timerHelp = "[s] Start | "
		}

		fullHelp := baseHelp + timerHelp + "[p] Report | [q] Quit" + pageInfo
		footer = "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fullHelp)
	}

	return lipgloss.JoinVertical(lipgloss.Left, timerView, board, footer)
}
