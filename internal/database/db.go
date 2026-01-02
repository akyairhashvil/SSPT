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
		`CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT UNIQUE
		);`,
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
			last_paused_at DATETIME,
			elapsed_seconds INTEGER DEFAULT 0,
			FOREIGN KEY(day_id) REFERENCES days(id)
		);`,
		`CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			workspace_id INTEGER,
			sprint_id INTEGER,
			description TEXT NOT NULL,
			notes TEXT,
			status TEXT DEFAULT 'pending',
			priority INTEGER DEFAULT 3,
			effort TEXT DEFAULT 'M',
			tags TEXT,
			links TEXT,
			rank INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY(sprint_id) REFERENCES sprints(id),
			FOREIGN KEY(parent_id) REFERENCES goals(id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id)
		);`,
		`CREATE TABLE IF NOT EXISTS journal_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			day_id INTEGER NOT NULL,
			sprint_id INTEGER,
			goal_id INTEGER,
			content TEXT NOT NULL,
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(day_id) REFERENCES days(id),
			FOREIGN KEY(sprint_id) REFERENCES sprints(id),
			FOREIGN KEY(goal_id) REFERENCES goals(id)
		);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Fatalf("Error creating table: %q: %s\n", err, query)
		}
	}

	// Migrations for existing databases
	migrate()
}

func migrate() {
	// Add last_paused_at to sprints
	_, _ = DB.Exec("ALTER TABLE sprints ADD COLUMN last_paused_at DATETIME")
	// Add elapsed_seconds to sprints
	_, _ = DB.Exec("ALTER TABLE sprints ADD COLUMN elapsed_seconds INTEGER DEFAULT 0")
	// Add rank to goals
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN rank INTEGER DEFAULT 0")
	
	// V3 Migrations
	// Goals
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN parent_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN workspace_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN notes TEXT")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN priority INTEGER DEFAULT 3")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN effort TEXT DEFAULT 'M'")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN tags TEXT")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN links TEXT")
	
	// Journal
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN goal_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN tags TEXT")
}

// ... (Rest of file) ...

// GetBacklogGoals retrieves goals that are not assigned to any sprint (sprint_id IS NULL).
func GetBacklogGoals() ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, created_at 
		FROM goals 
		WHERE sprint_id IS NULL AND status != 'completed'
		ORDER BY rank ASC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		// Scan basics
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
func GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, created_at 
		FROM goals 
		WHERE sprint_id = ? 
		ORDER BY rank ASC, created_at ASC`, sprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
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
		SELECT id, day_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds
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
			&s.LastPausedAt,
			&s.ElapsedSeconds,
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
	var maxRank int
	var err error
	if sprintID > 0 {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
	} else {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL").Scan(&maxRank)
	}
	if err != nil {
		return err
	}

	query := `INSERT INTO goals (description, sprint_id, status, rank) VALUES (?, ?, 'pending', ?)`

	var sprintIDArg interface{}
	if sprintID > 0 {
		sprintIDArg = sprintID
	} else {
		sprintIDArg = nil // SQL NULL
	}

	_, err = DB.Exec(query, description, sprintIDArg, maxRank+1)
	return err
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

// PauseSprint saves the elapsed time and marks the sprint as paused.
func PauseSprint(sprintID int64, elapsedSeconds int) error {
	_, err := DB.Exec(`
		UPDATE sprints 
		SET status = 'paused', 
		    elapsed_seconds = ?, 
		    last_paused_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, elapsedSeconds, sprintID)
	return err
}

// CompleteSprint marks a sprint as finished.
func CompleteSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'completed', end_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

// UpdateGoalStatus toggles a goal's status in the database.
func UpdateGoalStatus(goalID int64, status string) error {
	completedAt := "NULL"
	if status == "completed" {
		completedAt = "CURRENT_TIMESTAMP"
	}
	query := fmt.Sprintf("UPDATE goals SET status = ?, completed_at = %s WHERE id = ?", completedAt)
	_, err := DB.Exec(query, status, goalID)
	return err
}

// ResetSprint resets a sprint's status to 'pending' and clears the start time and elapsed seconds.
func ResetSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'pending', start_time = NULL, elapsed_seconds = 0, last_paused_at = NULL WHERE id = ?", sprintID)
	return err
}

// SwapGoalRanks swaps the rank of two goals to allow reordering.
func SwapGoalRanks(goalID1, goalID2 int64) error {
	var rank1, rank2 int
	err := DB.QueryRow("SELECT rank FROM goals WHERE id = ?", goalID1).Scan(&rank1)
	if err != nil {
		return err
	}
	err = DB.QueryRow("SELECT rank FROM goals WHERE id = ?", goalID2).Scan(&rank2)
	if err != nil {
		return err
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE goals SET rank = ? WHERE id = ?", rank2, goalID1)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("UPDATE goals SET rank = ? WHERE id = ?", rank1, goalID2)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// AppendSprint adds a new sprint to the specified day.
func AppendSprint(dayID int64) error {
	var lastSprintNum int
	err := DB.QueryRow("SELECT COALESCE(MAX(sprint_number), 0) FROM sprints WHERE day_id = ?", dayID).Scan(&lastSprintNum)
	if err != nil {
		return err
	}

	_, err = DB.Exec("INSERT INTO sprints (day_id, sprint_number) VALUES (?, ?)", dayID, lastSprintNum+1)
	return err
}

// GetCompletedGoalsForDay retrieves all goals completed on a specific day across all sprints.
func GetCompletedGoalsForDay(dayID int64) ([]models.Goal, error) {
	dateStr := ""
	err := DB.QueryRow("SELECT date FROM days WHERE id = ?", dayID).Scan(&dateStr)
	if err != nil {
		return nil, err
	}

	rows, err := DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, created_at 
		FROM goals 
		WHERE status = 'completed' 
		AND (
			sprint_id IN (SELECT id FROM sprints WHERE day_id = ?)
			OR (sprint_id IS NULL AND strftime('%Y-%m-%d', completed_at) = ?)
		)
		ORDER BY completed_at DESC`, dayID, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// AddSubtask inserts a new subtask linked to a parent goal.
func AddSubtask(description string, parentID int64) error {
	// Inherit sprint_id from parent
	var sprintID sql.NullInt64
	err := DB.QueryRow("SELECT sprint_id FROM goals WHERE id = ?", parentID).Scan(&sprintID)
	if err != nil {
		return err
	}

	// Calculate rank among siblings
	var maxRank int
	err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?", parentID).Scan(&maxRank)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`INSERT INTO goals (description, parent_id, sprint_id, status, rank) VALUES (?, ?, ?, 'pending', ?)`,
		description, parentID, sprintID, maxRank+1)
	return err
}

// --- Task Management ---

func EditGoal(goalID int64, newDescription string) error {
	_, err := DB.Exec("UPDATE goals SET description = ? WHERE id = ?", newDescription, goalID)
	return err
}

func DeleteGoal(goalID int64) error {
	_, err := DB.Exec("DELETE FROM goals WHERE id = ?", goalID)
	return err
}

// SearchGoals finds tasks matching a string across all history (or limit to current day if preferred).
func SearchGoals(query string) ([]models.Goal, error) {
	likeQuery := "%" + query + "%"
	rows, err := DB.Query(`
		SELECT id, sprint_id, description, status, created_at 
		FROM goals WHERE description LIKE ? ORDER BY created_at DESC LIMIT 20`, likeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		rows.Scan(&g.ID, &g.SprintID, &g.Description, &g.Status, &g.CreatedAt)
		goals = append(goals, g)
	}
	return goals, nil
}

// --- Sprint Lifecycle Logic ---

// MovePendingToBacklog transfers all unfinished tasks from a specific sprint to the Backlog (Sprint ID 0/NULL).
func MovePendingToBacklog(sprintID int64) error {
	_, err := DB.Exec("UPDATE goals SET sprint_id = NULL WHERE sprint_id = ? AND status != 'completed'", sprintID)
	return err
}

// --- Journaling ---

// AddJournalEntry inserts a new journal entry.
func AddJournalEntry(dayID int64, sprintID sql.NullInt64, content string) error {
	_, err := DB.Exec("INSERT INTO journal_entries (day_id, sprint_id, content) VALUES (?, ?, ?)", dayID, sprintID, content)
	return err
}

// GetJournalEntries retrieves all journal entries for a given day.
func GetJournalEntries(dayID int64) ([]models.JournalEntry, error) {
	rows, err := DB.Query(`
		SELECT id, day_id, sprint_id, content, created_at 
		FROM journal_entries 
		WHERE day_id = ? 
		ORDER BY created_at ASC`, dayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.JournalEntry
	for rows.Next() {
		var e models.JournalEntry
		if err := rows.Scan(&e.ID, &e.DayID, &e.SprintID, &e.Content, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
