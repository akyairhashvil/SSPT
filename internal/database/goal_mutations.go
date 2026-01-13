package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// AddGoal inserts a new goal into the database.
func (d *Database) AddGoal(ctx context.Context, workspaceID int64, description string, sprintID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var maxRank int
	var err error
	if sprintID > 0 {
		maxRank, err = d.getMaxGoalRank(ctx, sprintID)
	} else {
		maxRank, err = d.getMaxBacklogRank(ctx, workspaceID)
	}
	if err != nil {
		return wrapGoalErr("add", 0, err)
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	query := `INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags) VALUES (?, ?, ?, 'pending', ?, ?)`

	sprintIDArg := nullableInt64(sprintID)

	_, err = d.DB.ExecContext(ctx, query, workspaceID, description, sprintIDArg, maxRank+1, tags)
	return wrapGoalErr("add", 0, err)
}

func (d *Database) UpdateGoalPriority(ctx context.Context, goalID int64, priority int) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	if priority < 1 {
		priority = 1
	}
	if priority > 5 {
		priority = 5
	}
	_, err := d.DB.ExecContext(ctx, "UPDATE goals SET priority = ? WHERE id = ?", priority, goalID)
	return wrapGoalErr("update priority", goalID, err)
}

func (d *Database) AddGoalDetailed(ctx context.Context, workspaceID int64, sprintID int64, seed GoalSeed) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	seed.Description = strings.TrimSpace(seed.Description)
	if seed.Description == "" {
		return nil
	}

	var maxRank int
	var err error
	if sprintID > 0 {
		maxRank, err = d.getMaxGoalRank(ctx, sprintID)
	} else {
		maxRank, err = d.getMaxBacklogRank(ctx, workspaceID)
	}
	if err != nil {
		return wrapGoalErr("add detailed", 0, err)
	}

	tags := seed.Tags
	if len(tags) == 0 {
		tags = util.ExtractTags(seed.Description)
	}
	priority := normalizePriority(seed.Priority)
	effort := normalizeEffort(seed.Effort)
	tagsJSON := util.TagsToJSON(normalizeTagsFromSlice(tags))
	linksJSON, err := json.Marshal(seed.Links)
	if err != nil {
		return wrapGoalErr("add detailed", 0, err)
	}

	sprintIDArg := nullableInt64(sprintID)
	notesArg := nullableString("")
	if strings.TrimSpace(seed.Notes) != "" {
		notesArg = nullableString(seed.Notes)
	}
	recurrenceArg := nullableString("")
	if strings.TrimSpace(seed.Recurrence) != "" {
		recurrenceArg = nullableString(seed.Recurrence)
	}

	_, err = d.DB.ExecContext(ctx, `INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, priority, effort, notes, recurrence_rule, links)
		VALUES (?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)`,
		workspaceID, seed.Description, sprintIDArg, maxRank+1, tagsJSON, priority, effort, notesArg, recurrenceArg, string(linksJSON))
	return wrapGoalErr("add detailed", 0, err)
}

// AddSubtask inserts a new subtask linked to a parent goal.
func (d *Database) AddSubtask(ctx context.Context, description string, parentID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var sprintID *int64
	var workspaceID *int64
	err := d.DB.QueryRowContext(ctx, "SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return wrapGoalErr("add subtask", parentID, err)
	}

	var maxRank int
	maxRank, err = d.getMaxSubtaskRank(ctx, parentID)
	if err != nil {
		return wrapGoalErr("add subtask", parentID, err)
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	_, err = d.DB.ExecContext(ctx, `INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags) VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		description, parentID, sprintID, workspaceID, maxRank+1, tags)
	return wrapGoalErr("add subtask", parentID, err)
}

func (d *Database) AddSubtaskDetailed(ctx context.Context, parentID int64, seed GoalSeed) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var sprintID *int64
	var workspaceID *int64
	err := d.DB.QueryRowContext(ctx, "SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return wrapGoalErr("add subtask detailed", parentID, err)
	}

	var maxRank int
	if maxRank, err = d.getMaxSubtaskRank(ctx, parentID); err != nil {
		return wrapGoalErr("add subtask detailed", parentID, err)
	}

	priority := normalizePriority(seed.Priority)
	effort := normalizeEffort(seed.Effort)
	tags := seed.Tags
	if len(tags) == 0 {
		tags = util.ExtractTags(seed.Description)
	}
	tagsJSON := util.TagsToJSON(normalizeTagsFromSlice(tags))
	linksJSON, err := json.Marshal(seed.Links)
	if err != nil {
		return wrapGoalErr("add subtask detailed", parentID, err)
	}

	notesArg := nullableString("")
	if strings.TrimSpace(seed.Notes) != "" {
		notesArg = nullableString(seed.Notes)
	}
	recurrenceArg := nullableString("")
	if strings.TrimSpace(seed.Recurrence) != "" {
		recurrenceArg = nullableString(seed.Recurrence)
	}

	_, err = d.DB.ExecContext(ctx, `INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags, priority, effort, notes, recurrence_rule, links)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)`,
		seed.Description, parentID, sprintID, workspaceID, maxRank+1, tagsJSON, priority, effort, notesArg, recurrenceArg, string(linksJSON))
	return wrapGoalErr("add subtask detailed", parentID, err)
}

func (d *Database) UpdateGoalStatus(ctx context.Context, goalID int64, status models.GoalStatus) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	if status == models.GoalStatusCompleted {
		var active int
		if err := d.DB.QueryRowContext(ctx, "SELECT task_active FROM goals WHERE id = ?", goalID).Scan(&active); err != nil {
			return wrapGoalErr("update status", goalID, err)
		}
		if active == 1 {
			if err := d.PauseTaskTimer(ctx, goalID); err != nil {
				return wrapGoalErr("update status", goalID, err)
			}
		}
	}
	var err error
	statusValue := string(status)
	if status == models.GoalStatusCompleted {
		_, err = d.DB.ExecContext(ctx, "UPDATE goals SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?", statusValue, goalID)
	} else {
		_, err = d.DB.ExecContext(ctx, "UPDATE goals SET status = ?, completed_at = NULL WHERE id = ?", statusValue, goalID)
	}
	if err != nil {
		return wrapGoalErr("update status", goalID, err)
	}
	if status == models.GoalStatusCompleted {
		return wrapGoalErr("regenerate", goalID, d.regenerateRecurringGoal(ctx, goalID))
	}
	return nil
}

func (d *Database) SwapGoalRanks(ctx context.Context, goalID1, goalID2 int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var rank1, rank2 int
	err := d.DB.QueryRowContext(ctx, "SELECT rank FROM goals WHERE id = ?", goalID1).Scan(&rank1)
	if err != nil {
		return wrapGoalErr("swap ranks", goalID1, err)
	}
	err = d.DB.QueryRowContext(ctx, "SELECT rank FROM goals WHERE id = ?", goalID2).Scan(&rank2)
	if err != nil {
		return wrapGoalErr("swap ranks", goalID2, err)
	}

	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return wrapGoalErr("swap ranks", 0, err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE goals SET rank = ? WHERE id = ?", rank2, goalID1)
	if err != nil {
		return wrapGoalErr("swap ranks", goalID1, rollbackWithLog(tx, err))
	}

	_, err = tx.ExecContext(ctx, "UPDATE goals SET rank = ? WHERE id = ?", rank1, goalID2)
	if err != nil {
		return wrapGoalErr("swap ranks", goalID2, rollbackWithLog(tx, err))
	}

	if err := tx.Commit(); err != nil {
		return wrapGoalErr("swap ranks", 0, err)
	}
	return nil
}

func (d *Database) StartTaskTimer(ctx context.Context, goalID int64) error {
	err := d.WithTx(ctx, func(tx *sql.Tx) error {
		var workspaceID *int64
		if err := tx.QueryRowContext(ctx, "SELECT workspace_id FROM goals WHERE id = ?", goalID).Scan(&workspaceID); err != nil {
			return err
		}
		if workspaceID == nil {
			return fmt.Errorf("workspace id missing for goal %d", goalID)
		}
		wsID := *workspaceID

		rows, err := tx.QueryContext(ctx, `SELECT id, task_started_at, task_elapsed_seconds FROM goals WHERE workspace_id = ? AND task_active = 1 AND id != ?`, wsID, goalID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var started *time.Time
			var elapsed int
			if err := rows.Scan(&id, &started, &elapsed); err != nil {
				return err
			}
			if started != nil {
				elapsed += int(time.Since(*started).Seconds())
			}
			if _, err := tx.ExecContext(ctx, "UPDATE goals SET task_active = 0, task_started_at = NULL, task_elapsed_seconds = ? WHERE id = ?", elapsed, id); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE goals SET task_active = 1, task_started_at = CURRENT_TIMESTAMP WHERE id = ?", goalID); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	})
	return wrapGoalErr("start task timer", goalID, err)
}

func (d *Database) PauseTaskTimer(ctx context.Context, goalID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var started *time.Time
	var elapsed int
	var active int
	if err := d.DB.QueryRowContext(ctx, "SELECT task_active, task_started_at, task_elapsed_seconds FROM goals WHERE id = ?", goalID).Scan(&active, &started, &elapsed); err != nil {
		return wrapGoalErr("pause task timer", goalID, err)
	}
	if active == 0 {
		return nil
	}
	if started != nil {
		elapsed += int(time.Since(*started).Seconds())
	}
	_, err := d.DB.ExecContext(ctx, "UPDATE goals SET task_active = 0, task_started_at = NULL, task_elapsed_seconds = ? WHERE id = ?", elapsed, goalID)
	return wrapGoalErr("pause task timer", goalID, err)
}

func (d *Database) MoveGoal(ctx context.Context, goalID int64, targetSprintID int64) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		sprintArg := nullableInt64(targetSprintID)
		_, err := d.DB.ExecContext(ctx, "UPDATE goals SET sprint_id = ? WHERE id = ?", sprintArg, goalID)
		return wrapGoalErr("move", goalID, err)
	})
}

func (d *Database) EditGoal(ctx context.Context, goalID int64, newDescription string) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		tags := util.TagsToJSON(util.ExtractTags(newDescription))
		_, err := d.DB.ExecContext(ctx, "UPDATE goals SET description = ?, tags = ? WHERE id = ?", newDescription, tags, goalID)
		return wrapGoalErr("edit", goalID, err)
	})
}

func (d *Database) DeleteGoal(ctx context.Context, goalID int64) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		_, err := d.DB.ExecContext(ctx, "DELETE FROM goals WHERE id = ?", goalID)
		return wrapGoalErr("delete", goalID, err)
	})
}

func (d *Database) ArchiveGoal(ctx context.Context, goalID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE goals SET status = 'archived', archived_at = CURRENT_TIMESTAMP WHERE id = ?", goalID)
	return wrapGoalErr("archive", goalID, err)
}

func (d *Database) UnarchiveGoal(ctx context.Context, goalID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE goals SET status = 'pending', archived_at = NULL, sprint_id = NULL WHERE id = ?", goalID)
	return wrapGoalErr("unarchive", goalID, err)
}
