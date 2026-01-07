package tui

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SessionState defines the high-level mode of the application.
type SessionState int

const (
	StateInitializing SessionState = iota
	StateDashboard
)

// MainModel is the root bubbletea model that switches between sub-models.
type MainModel struct {
	state     SessionState
	db        *sql.DB
	textInput textinput.Model
	dashboard DashboardModel 
	err       error
	width     int // Store window dimensions
	height    int
}

func NewMainModel(db *sql.DB) MainModel {
	// ... (rest of function is fine)
	// Check if the day is already bootstrapped
	dayID := database.CheckCurrentDay()

	m := MainModel{
		db: db,
	}

	if dayID > 0 {
		m.state = StateDashboard
		m.dashboard = NewDashboardModel(db, dayID) // Load existing day
	} else {
		m.state = StateInitializing
		ti := textinput.New()
		ti.Placeholder = "1-8"
		ti.Focus()
		ti.CharLimit = 1
		ti.Width = 10
		m.textInput = ti
	}

	return m
}

func (m MainModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textinput.Blink) // Keep the cursor blinking

	// If we start directly in the dashboard (Day already exists),
	// we must fire the Dashboard's Init() to start the timer.
	if m.state == StateDashboard {
		cmds = append(cmds, m.dashboard.Init())
	}

	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to dashboard if active
		if m.state == StateDashboard {
			var newDash tea.Model
			newDash, cmd = m.dashboard.Update(msg)
			m.dashboard = newDash.(DashboardModel)
			return m, cmd
		}
	}

	// State-specific update logic
	switch m.state {
	case StateInitializing:
		return m.updateInitializing(msg)
	case StateDashboard:
		// We cast the return value back to DashboardModel to keep our state correct
		newDash, newCmd := m.dashboard.Update(msg)
		m.dashboard = newDash.(DashboardModel)
		return m, newCmd
	}

	return m, cmd
}

func (m MainModel) updateInitializing(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			val := m.textInput.Value()
			numSprints, err := strconv.Atoi(val)
			if err != nil || numSprints < 1 || numSprints > 8 {
				m.err = fmt.Errorf("please enter a valid number between 1 and 8")
				return m, nil
			}

			// EXECUTE THE BOOTSTRAP
			wsID, _ := database.EnsureDefaultWorkspace()
			if err := database.BootstrapDay(wsID, numSprints); err != nil {
				m.err = err
				return m, nil
			}

			// Transition state
			m.state = StateDashboard
			m.dashboard = NewDashboardModel(m.db, database.CheckCurrentDay()) // Load the new day
			m.dashboard.width = m.width
			m.dashboard.height = m.height
			return m, m.dashboard.Init()
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m MainModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress Ctrl+C to quit.", m.err)
	}

	switch m.state {
	case StateInitializing:
		return fmt.Sprintf(
			"\n  %s\n\n  %s\n\n  %s\n",
			"Salutations. Define your temporeal capacity.",
			"How many sprints will you execute today? (1-8)",
			m.textInput.View(),
		)
	case StateDashboard:
		return m.dashboard.View()
	}

	return ""
}
