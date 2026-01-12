package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

// GetAdjacentDay finds the previous (direction < 0) or next (direction > 0) day ID relative to the current one.
// Returns the new Day ID and its Date string.
func GetAdjacentDay(currentDayID int64, direction int) (int64, string, error) {
	d, err := getDefaultDB()
	if err != nil {
		return 0, "", err
	}
	return d.GetAdjacentDay(currentDayID, direction)
}

func (d *Database) GetAdjacentDay(currentDayID int64, direction int) (int64, string, error) {
	var query string
	if direction < 0 {
		query = "SELECT id, date FROM days WHERE id < ? ORDER BY id DESC LIMIT 1"
	} else {
		query = "SELECT id, date FROM days WHERE id > ? ORDER BY id ASC LIMIT 1"
	}

	var id int64
	var date string
	err := d.DB.QueryRow(query, currentDayID).Scan(&id, &date)
	if err != nil {
		return 0, "", err
	}
	return id, date, nil
}

// CheckCurrentDay returns the Day ID if it exists for the current date.
func CheckCurrentDay() int64 {
	d, err := getDefaultDB()
	if err != nil {
		return 0
	}
	return d.CheckCurrentDay()
}

// CheckCurrentDay returns the Day ID if it exists for the current date.
func (d *Database) CheckCurrentDay() int64 {
	dateStr := time.Now().Format("2006-01-02")
	var id int64
	err := d.DB.QueryRow("SELECT id FROM days WHERE date = ?", dateStr).Scan(&id)
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
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.BootstrapDay(workspaceID, numSprints)
}

// BootstrapDay creates the day record and pre-allocates the chosen number of sprints for a workspace.
func (d *Database) BootstrapDay(workspaceID int64, numSprints int) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	dateStr := time.Now().Format("2006-01-02")
	_, err = tx.Exec("INSERT OR IGNORE INTO days (date) VALUES (?)", dateStr)
	if err != nil {
		return rollbackWithLog(tx, fmt.Errorf("failed to ensure day: %w", err))
	}

	var dayID int64
	err = tx.QueryRow("SELECT id FROM days WHERE date = ?", dateStr).Scan(&dayID)
	if err != nil {
		return rollbackWithLog(tx, err)
	}

	stmt, err := tx.Prepare("INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)")
	if err != nil {
		return rollbackWithLog(tx, err)
	}
	defer stmt.Close()

	for i := 1; i <= numSprints; i++ {
		_, err = stmt.Exec(dayID, workspaceID, i)
		if err != nil {
			return rollbackWithLog(tx, fmt.Errorf("failed to insert sprint %d: %w", i, err))
		}
	}

	return tx.Commit()
}

// GetDay retrieves the full Day struct by ID.
func GetDay(id int64) (models.Day, error) {
	d, err := getDefaultDB()
	if err != nil {
		return models.Day{}, err
	}
	return d.GetDay(id)
}

// GetDay retrieves the full Day struct by ID.
func (d *Database) GetDay(id int64) (models.Day, error) {
	var day models.Day
	var dateStr string

	err := d.DB.QueryRow("SELECT id, date, started_at FROM days WHERE id = ?", id).
		Scan(&day.ID, &dateStr, &day.StartedAt)

	if err != nil {
		return day, err
	}
	day.Date = dateStr
	return day, nil
}

// GetSprints retrieves all sprints for a given day and workspace, ordered by number.
func GetSprints(dayID int64, workspaceID int64) ([]models.Sprint, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetSprints(dayID, workspaceID)
}

// GetSprints retrieves all sprints for a given day and workspace, ordered by number.
func (d *Database) GetSprints(dayID int64, workspaceID int64) ([]models.Sprint, error) {
	rows, err := d.DB.Query(`
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

// --- Sprint Lifecycle ---

func StartSprint(sprintID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.StartSprint(sprintID)
}

func (d *Database) StartSprint(sprintID int64) error {
	_, err := d.DB.Exec("UPDATE sprints SET status = 'active', start_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

func PauseSprint(sprintID int64, elapsedSeconds int) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.PauseSprint(sprintID, elapsedSeconds)
}

func (d *Database) PauseSprint(sprintID int64, elapsedSeconds int) error {
	_, err := d.DB.Exec(`
		UPDATE sprints 
		SET status = 'paused', 
		    elapsed_seconds = ?, 
		    last_paused_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, elapsedSeconds, sprintID)
	return err
}

func CompleteSprint(sprintID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.CompleteSprint(sprintID)
}

func (d *Database) CompleteSprint(sprintID int64) error {
	_, err := d.DB.Exec("UPDATE sprints SET status = 'completed', end_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

func ResetSprint(sprintID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.ResetSprint(sprintID)
}

func (d *Database) ResetSprint(sprintID int64) error {
	_, err := d.DB.Exec("UPDATE sprints SET status = 'pending', start_time = NULL, elapsed_seconds = 0, last_paused_at = NULL WHERE id = ?", sprintID)
	return err
}

func AppendSprint(dayID int64, workspaceID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AppendSprint(dayID, workspaceID)
}

func (d *Database) AppendSprint(dayID int64, workspaceID int64) error {
	var lastSprintNum int
	err := d.DB.QueryRow("SELECT COALESCE(MAX(sprint_number), 0) FROM sprints WHERE day_id = ? AND workspace_id = ?", dayID, workspaceID).Scan(&lastSprintNum)
	if err != nil {
		return err
	}
	if lastSprintNum >= 8 {
		return fmt.Errorf("max sprints reached (8)")
	}

	_, err = d.DB.Exec("INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)", dayID, workspaceID, lastSprintNum+1)
	return err
}

func RemoveLastSprint(dayID int64, workspaceID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.RemoveLastSprint(dayID, workspaceID)
}

func (d *Database) RemoveLastSprint(dayID int64, workspaceID int64) error {
	var count int
	if err := d.DB.QueryRow(
		"SELECT COUNT(1) FROM sprints WHERE day_id = ? AND workspace_id = ? AND sprint_number > 0",
		dayID, workspaceID,
	).Scan(&count); err != nil {
		return err
	}
	if count <= 1 {
		return fmt.Errorf("cannot remove last sprint")
	}

	var sprintID int64
	var status string
	err := d.DB.QueryRow(
		"SELECT id, status FROM sprints WHERE day_id = ? AND workspace_id = ? AND sprint_number > 0 ORDER BY sprint_number DESC LIMIT 1",
		dayID, workspaceID,
	).Scan(&sprintID, &status)
	if err != nil {
		return err
	}
	if status == "active" || status == "paused" {
		return fmt.Errorf("cannot remove active sprint")
	}
	if _, err := d.DB.Exec("UPDATE goals SET sprint_id = NULL WHERE sprint_id = ?", sprintID); err != nil {
		return err
	}
	_, err = d.DB.Exec("DELETE FROM sprints WHERE id = ?", sprintID)
	return err
}

func GetSprintGoalCounts(sprintID int64) (int, int, error) {
	d, err := getDefaultDB()
	if err != nil {
		return 0, 0, err
	}
	return d.GetSprintGoalCounts(sprintID)
}

func (d *Database) GetSprintGoalCounts(sprintID int64) (int, int, error) {
	var total int
	if err := d.DB.QueryRow("SELECT COUNT(1) FROM goals WHERE sprint_id = ? AND status != 'archived'", sprintID).Scan(&total); err != nil {
		return 0, 0, err
	}
	var completed int
	if err := d.DB.QueryRow("SELECT COUNT(1) FROM goals WHERE sprint_id = ? AND status = 'completed'", sprintID).Scan(&completed); err != nil {
		return 0, 0, err
	}
	return total, completed, nil
}

func MovePendingToBacklog(sprintID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.MovePendingToBacklog(sprintID)
}

func (d *Database) MovePendingToBacklog(sprintID int64) error {
	_, err := d.DB.Exec("UPDATE goals SET sprint_id = NULL WHERE sprint_id = ? AND status != 'completed'", sprintID)
	return err
}
