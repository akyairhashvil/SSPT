package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// GetAdjacentDay finds the previous (direction < 0) or next (direction > 0) day ID relative to the current one.
// Returns the new Day ID and its Date string.

func (d *Database) GetAdjacentDay(ctx context.Context, currentDayID int64, direction int) (int64, string, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var query string
	if direction < 0 {
		query = "SELECT id, date FROM days WHERE id < ? ORDER BY id DESC LIMIT 1"
	} else {
		query = "SELECT id, date FROM days WHERE id > ? ORDER BY id ASC LIMIT 1"
	}

	var id int64
	var date string
	err := d.DB.QueryRowContext(ctx, query, currentDayID).Scan(&id, &date)
	if err != nil {
		return 0, "", err
	}
	return id, date, nil
}

// CheckCurrentDay returns the Day ID if it exists for the current date.

// CheckCurrentDay returns the Day ID if it exists for the current date.
func (d *Database) CheckCurrentDay(ctx context.Context) int64 {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	dateStr := time.Now().Format("2006-01-02")
	var id int64
	err := d.DB.QueryRowContext(ctx, "SELECT id FROM days WHERE date = ?", dateStr).Scan(&id)
	if err == sql.ErrNoRows {
		return 0
	} else if err != nil {
		util.LogError("check day", err)
		return 0
	}
	return id
}

// BootstrapDay creates the day record and pre-allocates the chosen number of sprints for a workspace.

// BootstrapDay creates the day record and pre-allocates the chosen number of sprints for a workspace.
func (d *Database) BootstrapDay(ctx context.Context, workspaceID int64, numSprints int) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return wrapSprintErr("bootstrap", 0, err)
	}

	dateStr := time.Now().Format("2006-01-02")
	_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO days (date) VALUES (?)", dateStr)
	if err != nil {
		return wrapSprintErr("bootstrap", 0, rollbackWithLog(tx, fmt.Errorf("failed to ensure day: %w", err)))
	}

	var dayID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM days WHERE date = ?", dateStr).Scan(&dayID)
	if err != nil {
		return wrapSprintErr("bootstrap", 0, rollbackWithLog(tx, err))
	}

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)")
	if err != nil {
		return wrapSprintErr("bootstrap", 0, rollbackWithLog(tx, err))
	}
	defer stmt.Close()

	for i := 1; i <= numSprints; i++ {
		_, err = stmt.ExecContext(ctx, dayID, workspaceID, i)
		if err != nil {
			return wrapSprintErr("bootstrap", 0, rollbackWithLog(tx, fmt.Errorf("failed to insert sprint %d: %w", i, err)))
		}
	}

	if err := tx.Commit(); err != nil {
		return wrapSprintErr("bootstrap", 0, err)
	}
	return nil
}

// GetDay retrieves the full Day struct by ID.

// GetDay retrieves the full Day struct by ID.
func (d *Database) GetDay(ctx context.Context, id int64) (models.Day, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var day models.Day
	var dateStr string

	err := d.DB.QueryRowContext(ctx, "SELECT id, date, started_at FROM days WHERE id = ?", id).
		Scan(&day.ID, &dateStr, &day.StartedAt)

	if err != nil {
		return day, err
	}
	day.Date = dateStr
	return day, nil
}

// GetSprints retrieves all sprints for a given day and workspace, ordered by number.

// GetSprints retrieves all sprints for a given day and workspace, ordered by number.
func (d *Database) GetSprints(ctx context.Context, dayID int64, workspaceID int64) ([]models.Sprint, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `
		SELECT id, day_id, workspace_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds
		FROM sprints 
		WHERE day_id = ? AND workspace_id = ?
		ORDER BY sprint_number ASC`, dayID, workspaceID)

	if err != nil {
		return nil, wrapSprintErr("list", 0, err)
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
			return nil, wrapSprintErr("list", 0, err)
		}
		sprints = append(sprints, s)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapSprintErr("list", 0, err)
	}
	return sprints, nil
}

// --- Sprint Lifecycle ---

func (d *Database) StartSprint(ctx context.Context, sprintID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE sprints SET status = 'active', start_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	if err != nil {
		return wrapSprintErr("start", sprintID, err)
	}
	return nil
}

func (d *Database) PauseSprint(ctx context.Context, sprintID int64, elapsedSeconds int) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, `
		UPDATE sprints 
		SET status = 'paused', 
		    elapsed_seconds = ?, 
		    last_paused_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, elapsedSeconds, sprintID)
	if err != nil {
		return wrapSprintErr("pause", sprintID, err)
	}
	return nil
}

func (d *Database) CompleteSprint(ctx context.Context, sprintID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE sprints SET status = 'completed', end_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	if err != nil {
		return wrapSprintErr("complete", sprintID, err)
	}
	return nil
}

func (d *Database) ResetSprint(ctx context.Context, sprintID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE sprints SET status = 'pending', start_time = NULL, elapsed_seconds = 0, last_paused_at = NULL WHERE id = ?", sprintID)
	if err != nil {
		return wrapSprintErr("reset", sprintID, err)
	}
	return nil
}

func (d *Database) AppendSprint(ctx context.Context, dayID int64, workspaceID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var lastSprintNum int
	err := d.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(sprint_number), 0) FROM sprints WHERE day_id = ? AND workspace_id = ?", dayID, workspaceID).Scan(&lastSprintNum)
	if err != nil {
		return wrapSprintErr("append", 0, err)
	}
	if lastSprintNum >= 8 {
		return wrapSprintErr("append", 0, fmt.Errorf("max sprints reached (8)"))
	}

	_, err = d.DB.ExecContext(ctx, "INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)", dayID, workspaceID, lastSprintNum+1)
	if err != nil {
		return wrapSprintErr("append", 0, err)
	}
	return nil
}

func (d *Database) RemoveLastSprint(ctx context.Context, dayID int64, workspaceID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var count int
	if err := d.DB.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM sprints WHERE day_id = ? AND workspace_id = ? AND sprint_number > 0",
		dayID, workspaceID,
	).Scan(&count); err != nil {
		return wrapSprintErr("remove", 0, err)
	}
	if count <= 1 {
		return wrapSprintErr("remove", 0, fmt.Errorf("cannot remove last sprint"))
	}

	var sprintID int64
	var status string
	err := d.DB.QueryRowContext(ctx,
		"SELECT id, status FROM sprints WHERE day_id = ? AND workspace_id = ? AND sprint_number > 0 ORDER BY sprint_number DESC LIMIT 1",
		dayID, workspaceID,
	).Scan(&sprintID, &status)
	if err != nil {
		return wrapSprintErr("remove", 0, err)
	}
	if status == "active" || status == "paused" {
		return wrapSprintErr("remove", sprintID, fmt.Errorf("cannot remove active sprint"))
	}
	if _, err := d.DB.ExecContext(ctx, "UPDATE goals SET sprint_id = NULL WHERE sprint_id = ?", sprintID); err != nil {
		return wrapSprintErr("remove", sprintID, err)
	}
	_, err = d.DB.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprintID)
	if err != nil {
		return wrapSprintErr("remove", sprintID, err)
	}
	return nil
}

func (d *Database) GetSprintGoalCounts(ctx context.Context, sprintID int64) (int, int, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var total int
	if err := d.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM goals WHERE sprint_id = ? AND status != 'archived'", sprintID).Scan(&total); err != nil {
		return 0, 0, wrapSprintErr("counts", sprintID, err)
	}
	var completed int
	if err := d.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM goals WHERE sprint_id = ? AND status = 'completed'", sprintID).Scan(&completed); err != nil {
		return 0, 0, wrapSprintErr("counts", sprintID, err)
	}
	return total, completed, nil
}

func (d *Database) MovePendingToBacklog(ctx context.Context, sprintID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE goals SET sprint_id = NULL WHERE sprint_id = ? AND status != 'completed'", sprintID)
	if err != nil {
		return wrapSprintErr("move pending", sprintID, err)
	}
	return nil
}
