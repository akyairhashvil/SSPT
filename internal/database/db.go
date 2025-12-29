package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB initializes the database connection and schema.
func InitDB(filepath string) {
	var err error
	DB, err = sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}

	createTables()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS days (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS sprints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			day_id INTEGER NOT NULL,
			sprint_number INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			start_time DATETIME,
			end_time DATETIME,
			FOREIGN KEY(day_id) REFERENCES days(id)
		);`,
		`CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sprint_id INTEGER,
			description TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY(sprint_id) REFERENCES sprints(id)
		);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Fatalf("Error creating table: %q: %s\n", err, query)
		}
	}
}

// CheckCurrentDay returns the Day ID if it exists for the current date.
func CheckCurrentDay() int64 {
	dateStr := time.Now().Format("2006-01-02")
	var id int64
	err := DB.QueryRow("SELECT id FROM days WHERE date = ?", dateStr).Scan(&id)
	if err == sql.ErrNoRows {
		return 0
	} else if err != nil {
		log.Printf("Error checking day: %v", err)
		return 0
	}
	return id
}

// BootstrapDay creates the day record and pre-allocates the chosen number of sprints.
func BootstrapDay(numSprints int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	dateStr := time.Now().Format("2006-01-02")
	res, err := tx.Exec("INSERT INTO days (date) VALUES (?)", dateStr)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert day: %w", err)
	}

	dayID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO sprints (day_id, sprint_number) VALUES (?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := 1; i <= numSprints; i++ {
		_, err = stmt.Exec(dayID, i)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert sprint %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// GetDay retrieves the full Day struct by ID.
func GetDay(id int64) (models.Day, error) {
	var d models.Day
	var dateStr string

	err := DB.QueryRow("SELECT id, date, started_at FROM days WHERE id = ?", id).
		Scan(&d.ID, &dateStr, &d.StartedAt)

	if err != nil {
		return d, err
	}
	d.Date = dateStr
	return d, nil
}

// GetSprints retrieves all sprints for a given day, ordered by number.
func GetSprints(dayID int64) ([]models.Sprint, error) {
	rows, err := DB.Query(`
		SELECT id, day_id, sprint_number, status, start_time, end_time 
		FROM sprints 
		WHERE day_id = ? 
		ORDER BY sprint_number ASC`, dayID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sprints []models.Sprint
	for rows.Next() {
		var s models.Sprint
		// We use NullTime to handle potential NULLs in the database gracefully
		err := rows.Scan(
			&s.ID,
			&s.DayID,
			&s.SprintNumber,
			&s.Status,
			&s.StartTime,
			&s.EndTime,
		)
		if err != nil {
			return nil, err
		}
		sprints = append(sprints, s)
	}
	return sprints, nil
}

// AddGoal inserts a new goal into the database.
// If sprintID is 0, it is treated as a Backlog item (NULL in DB).
func AddGoal(description string, sprintID int64) error {
	query := `INSERT INTO goals (description, sprint_id, status) VALUES (?, ?, 'pending')`

	var sprintIDArg interface{}
	if sprintID > 0 {
		sprintIDArg = sprintID
	} else {
		sprintIDArg = nil // SQL NULL
	}

	_, err := DB.Exec(query, description, sprintIDArg)
	return err
}

// GetBacklogGoals retrieves goals that are not assigned to any sprint (sprint_id IS NULL).
func GetBacklogGoals() ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, description, status, created_at 
		FROM goals 
		WHERE sprint_id IS NULL AND status != 'completed'
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.Description, &g.Status, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
// This is a helper to refresh data without reloading the whole day.
func GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, sprint_id, description, status, created_at 
		FROM goals 
		WHERE sprint_id = ? 
		ORDER BY created_at ASC`, sprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.SprintID, &g.Description, &g.Status, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// MoveGoal updates the sprint_id of a specific goal.
// Passing targetSprintID = 0 moves it to the Backlog (NULL).
func MoveGoal(goalID int64, targetSprintID int64) error {
	var sprintArg interface{}
	if targetSprintID == 0 {
		sprintArg = nil // SQL NULL for Backlog
	} else {
		sprintArg = targetSprintID
	}

	_, err := DB.Exec("UPDATE goals SET sprint_id = ? WHERE id = ?", sprintArg, goalID)
	return err
}

// StartSprint marks a sprint as active and records the current timestamp.
func StartSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'active', start_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

// CompleteSprint marks a sprint as finished.
func CompleteSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'completed', end_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

// UpdateGoalStatus toggles a goal's status in the database.
func UpdateGoalStatus(goalID int64, status string) error {
	_, err := DB.Exec("UPDATE goals SET status = ? WHERE id = ?", status, goalID)
	return err
}

// ResetSprint resets a sprint's status to 'pending' and clears the start time.
// This is used when stopping/aborting a timer.
func ResetSprint(sprintID int64) error {
	// We set start_time to NULL so the timer logic doesn't get confused if we restart it
	_, err := DB.Exec("UPDATE sprints SET status = 'pending', start_time = NULL WHERE id = ?", sprintID)
	return err
}
