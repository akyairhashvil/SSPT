package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
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
			workspace_id INTEGER,
			sprint_number INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			start_time DATETIME,
			end_time DATETIME,
			last_paused_at DATETIME,
			elapsed_seconds INTEGER DEFAULT 0,
			FOREIGN KEY(day_id) REFERENCES days(id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id)
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
			workspace_id INTEGER,
			sprint_id INTEGER,
			goal_id INTEGER,
			content TEXT NOT NULL,
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(day_id) REFERENCES days(id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id),
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
	
	// Sprints
	_, _ = DB.Exec("ALTER TABLE sprints ADD COLUMN workspace_id INTEGER")

	// Journal
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN goal_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN tags TEXT")
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN workspace_id INTEGER")
}

// --- Workspaces ---

func GetWorkspaces() ([]models.Workspace, error) {
	rows, err := DB.Query("SELECT id, name, slug FROM workspaces ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ws []models.Workspace
	for rows.Next() {
		var w models.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug); err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}
	return ws, nil
}

func EnsureDefaultWorkspace() (int64, error) {
	var id int64
	err := DB.QueryRow("SELECT id FROM workspaces WHERE slug = 'personal'").Scan(&id)
	if err == sql.ErrNoRows {
		res, err := DB.Exec("INSERT INTO workspaces (name, slug) VALUES ('Personal', 'personal')")
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func CreateWorkspace(name, slug string) (int64, error) {
	res, err := DB.Exec("INSERT INTO workspaces (name, slug) VALUES (?, ?)", name, slug)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// --- Day / Sprint Management ---

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

// BootstrapDay creates the day record and pre-allocates the chosen number of sprints for a workspace.
func BootstrapDay(workspaceID int64, numSprints int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	dateStr := time.Now().Format("2006-01-02")
	_, err = tx.Exec("INSERT OR IGNORE INTO days (date) VALUES (?)", dateStr)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to ensure day: %w", err)
	}

	var dayID int64
	err = tx.QueryRow("SELECT id FROM days WHERE date = ?", dateStr).Scan(&dayID)
	if err != nil {
		tx.Rollback()
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := 1; i <= numSprints; i++ {
		_, err = stmt.Exec(dayID, workspaceID, i)
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

// GetSprints retrieves all sprints for a given day and workspace, ordered by number.
func GetSprints(dayID int64, workspaceID int64) ([]models.Sprint, error) {
	rows, err := DB.Query(`
		SELECT id, day_id, workspace_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds
		FROM sprints 
		WHERE day_id = ? AND workspace_id = ?
		ORDER BY sprint_number ASC`, dayID, workspaceID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sprints []models.Sprint
	for rows.Next() {
		var s models.Sprint
		err := rows.Scan(
			&s.ID,
			&s.DayID,
			&s.WorkspaceID,
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

// --- Goals (Tasks) ---

// GetBacklogGoals retrieves goals that are not assigned to any sprint and belong to the workspace.
func GetBacklogGoals(workspaceID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, tags, created_at 
		FROM goals 
		WHERE sprint_id IS NULL AND status != 'completed' AND workspace_id = ?
		ORDER BY rank ASC, created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
func GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, created_at 
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
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// AddGoal inserts a new goal into the database.
func AddGoal(workspaceID int64, description string, sprintID int64) error {
	var maxRank int
	var err error
	if sprintID > 0 {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
	} else {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", workspaceID).Scan(&maxRank)
	}
	if err != nil {
		return err
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	query := `INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags) VALUES (?, ?, ?, 'pending', ?, ?)`

	var sprintIDArg interface{}
	if sprintID > 0 {
		sprintIDArg = sprintID
	} else {
		sprintIDArg = nil // SQL NULL
	}

	_, err = DB.Exec(query, workspaceID, description, sprintIDArg, maxRank+1, tags)
	return err
}

// GetCompletedGoalsForDay retrieves all goals completed on a specific day and workspace across all sprints.
func GetCompletedGoalsForDay(dayID int64, workspaceID int64) ([]models.Goal, error) {
	dateStr := ""
	err := DB.QueryRow("SELECT date FROM days WHERE id = ?", dayID).Scan(&dateStr)
	if err != nil {
		return nil, err
	}

	rows, err := DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, tags, created_at 
		FROM goals 
		WHERE status = 'completed' AND workspace_id = ?
		AND (
			sprint_id IN (SELECT id FROM sprints WHERE day_id = ?)
			OR (sprint_id IS NULL AND strftime('%Y-%m-%d', completed_at) = ?)
		)
		ORDER BY completed_at DESC`, workspaceID, dayID, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// AddSubtask inserts a new subtask linked to a parent goal.
func AddSubtask(description string, parentID int64) error {
	// Inherit sprint_id and workspace_id from parent
	var sprintID sql.NullInt64
	var workspaceID sql.NullInt64
	err := DB.QueryRow("SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return err
	}

	// Calculate rank among siblings
	var maxRank int
	err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?", parentID).Scan(&maxRank)
	if err != nil {
		return err
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	_, err = DB.Exec(`INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags) VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		description, parentID, sprintID, workspaceID, maxRank+1, tags)
	return err
}

// --- Sprint Lifecycle ---

func StartSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'active', start_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

func PauseSprint(sprintID int64, elapsedSeconds int) error {
	_, err := DB.Exec(`
		UPDATE sprints 
		SET status = 'paused', 
		    elapsed_seconds = ?, 
		    last_paused_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, elapsedSeconds, sprintID)
	return err
}

func CompleteSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'completed', end_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

func ResetSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'pending', start_time = NULL, elapsed_seconds = 0, last_paused_at = NULL WHERE id = ?", sprintID)
	return err
}

// --- Task Management ---

func UpdateGoalStatus(goalID int64, status string) error {
	completedAt := "NULL"
	if status == "completed" {
		completedAt = "CURRENT_TIMESTAMP"
	}
	query := fmt.Sprintf("UPDATE goals SET status = ?, completed_at = %s WHERE id = ?", completedAt)
	_, err := DB.Exec(query, status, goalID)
	return err
}

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

func AppendSprint(dayID int64, workspaceID int64) error {
	var lastSprintNum int
	err := DB.QueryRow("SELECT COALESCE(MAX(sprint_number), 0) FROM sprints WHERE day_id = ? AND workspace_id = ?", dayID, workspaceID).Scan(&lastSprintNum)
	if err != nil {
		return err
	}

	_, err = DB.Exec("INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)", dayID, workspaceID, lastSprintNum+1)
	return err
}

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

func EditGoal(goalID int64, newDescription string) error {
	tags := util.TagsToJSON(util.ExtractTags(newDescription))
	_, err := DB.Exec("UPDATE goals SET description = ?, tags = ? WHERE id = ?", newDescription, tags, goalID)
	return err
}

func DeleteGoal(goalID int64) error {
	_, err := DB.Exec("DELETE FROM goals WHERE id = ?", goalID)
	return err
}

// AddTagsToGoal appends new tags to a goal, avoiding duplicates.
func AddTagsToGoal(goalID int64, tagsToAdd []string) error {
	var currentTagsJSON sql.NullString
	err := DB.QueryRow("SELECT tags FROM goals WHERE id = ?", goalID).Scan(&currentTagsJSON)
	if err != nil {
		return err
	}

	currentTags := util.JSONToTags(currentTagsJSON.String)
	tagSet := make(map[string]bool)
	for _, t := range currentTags {
		tagSet[t] = true
	}
	for _, t := range tagsToAdd {
		tagSet[strings.ToLower(t)] = true
	}

	var newTags []string
	for t := range tagSet {
		newTags = append(newTags, t)
	}

	newTagsJSON := util.TagsToJSON(newTags)
	_, err = DB.Exec("UPDATE goals SET tags = ? WHERE id = ?", newTagsJSON, goalID)
	return err
}

// Search performs a dynamic search on goals based on structured query filters and workspace isolation.
func Search(query util.SearchQuery, workspaceID int64) ([]models.Goal, error) {
	var args []interface{}
	sql := `SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, created_at 
	        FROM goals WHERE workspace_id = ?`
	args = append(args, workspaceID)

	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			sql += ` AND EXISTS (SELECT 1 FROM json_each(COALESCE(tags, '[]')) WHERE value = ?)`
			args = append(args, tag)
		}
	}

	if len(query.Status) > 0 {
		placeholders := strings.Repeat(",?", len(query.Status)-1)
		sql += fmt.Sprintf(` AND status IN (?%s)`, placeholders)
		for _, s := range query.Status {
			args = append(args, s)
		}
	}

	if len(query.Text) > 0 {
		for _, text := range query.Text {
			sql += ` AND description LIKE ?`
			args = append(args, "%"+text+"%")
		}
	}

	sql += " ORDER BY created_at DESC LIMIT 50"

	return scanGoals(sql, args...)
}

func scanGoals(query string, args ...interface{}) ([]models.Goal, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

func GetAllGoals() ([]models.Goal, error) {
	return scanGoals(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, created_at 
		FROM goals 
		ORDER BY rank ASC, created_at ASC`)
}

func MovePendingToBacklog(sprintID int64) error {
	_, err := DB.Exec("UPDATE goals SET sprint_id = NULL WHERE sprint_id = ? AND status != 'completed'", sprintID)
	return err
}

// --- Journaling ---

func AddJournalEntry(dayID int64, workspaceID int64, sprintID sql.NullInt64, goalID sql.NullInt64, content string) error {
	_, err := DB.Exec("INSERT INTO journal_entries (day_id, workspace_id, sprint_id, goal_id, content) VALUES (?, ?, ?, ?, ?)", dayID, workspaceID, sprintID, goalID, content)
	return err
}

func GetJournalEntries(dayID int64, workspaceID int64) ([]models.JournalEntry, error) {
	rows, err := DB.Query(`
		SELECT id, day_id, workspace_id, sprint_id, goal_id, content, created_at 
		FROM journal_entries 
		WHERE day_id = ? AND workspace_id = ?
		ORDER BY created_at ASC`, dayID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.JournalEntry
	for rows.Next() {
		var e models.JournalEntry
		if err := rows.Scan(&e.ID, &e.DayID, &e.WorkspaceID, &e.SprintID, &e.GoalID, &e.Content, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
