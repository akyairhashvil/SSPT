package database

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// GetBacklogGoals retrieves goals that are not assigned to any sprint and belong to the workspace.
func (d *Database) GetBacklogGoals(ctx context.Context, workspaceID int64) ([]models.Goal, error) {
	query, args := NewGoalQuery().
		WhereBacklog().
		WhereWorkspace(workspaceID).
		Where("status != ?", "completed").
		Where("status != ?", "archived").
		OrderBy("rank ASC, created_at DESC").
		Build()
	return d.queryGoals(ctx, "backlog", query, args...)
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
func (d *Database) GetGoalsForSprint(ctx context.Context, sprintID int64) ([]models.Goal, error) {
	query, args := NewGoalQuery().
		WhereSprint(sprintID).
		Where("status != ?", "archived").
		OrderBy("rank ASC, created_at ASC").
		Build()
	return d.queryGoals(ctx, "list sprint", query, args...)
}

type ExistsCheckLevel int

const (
	ExistsCheckBasic ExistsCheckLevel = iota
	ExistsCheckDetailed
)

type ExistsResult struct {
	Exists     bool
	ExistingID int64
	MatchType  string
}

func (d *Database) CheckGoalExists(ctx context.Context, workspaceID int64, sprintID int64, parentID *int64, description string, level ExistsCheckLevel, seed *GoalSeed) (ExistsResult, error) {
	return withDBContextResult(d, ctx, func(ctx context.Context) (ExistsResult, error) {
		result := ExistsResult{MatchType: "none"}
		desc := strings.TrimSpace(description)
		if desc == "" {
			return result, nil
		}

		if level == ExistsCheckBasic {
			var row *sql.Row
			if parentID != nil {
				row = d.DB.QueryRowContext(ctx,
					"SELECT id FROM goals WHERE parent_id = ? AND description = ? LIMIT 1",
					*parentID, desc,
				)
			} else if sprintID > 0 {
				row = d.DB.QueryRowContext(ctx,
					"SELECT id FROM goals WHERE workspace_id = ? AND sprint_id = ? AND parent_id IS NULL AND description = ? LIMIT 1",
					workspaceID, sprintID, desc,
				)
			} else {
				row = d.DB.QueryRowContext(ctx,
					"SELECT id FROM goals WHERE workspace_id = ? AND sprint_id IS NULL AND parent_id IS NULL AND description = ? LIMIT 1",
					workspaceID, desc,
				)
			}

			var id int64
			if err := row.Scan(&id); err != nil {
				if err == sql.ErrNoRows {
					return result, nil
				}
				return result, wrapErr(EntityGoal, "exists", 0, err)
			}
			result.Exists = true
			result.ExistingID = id
			result.MatchType = "exact"
			return result, nil
		}

		if level != ExistsCheckDetailed {
			return result, fmt.Errorf("invalid exists check level: %d", level)
		}

		if seed == nil {
			return result, fmt.Errorf("goal seed is required for detailed exists check")
		}

		desc, priority, effort, tags, recurrence, notes, links := normalizeSeed(*seed)
		if desc == "" {
			return result, nil
		}

		var rows *sql.Rows
		var err error
		if parentID != nil {
			rows, err = d.DB.QueryContext(ctx,
				"SELECT id, description, priority, effort, tags, recurrence_rule, notes, links FROM goals WHERE parent_id = ? AND description = ?",
				*parentID, desc,
			)
		} else if sprintID > 0 {
			rows, err = d.DB.QueryContext(ctx,
				"SELECT id, description, priority, effort, tags, recurrence_rule, notes, links FROM goals WHERE workspace_id = ? AND sprint_id = ? AND parent_id IS NULL AND description = ?",
				workspaceID, sprintID, desc,
			)
		} else {
			rows, err = d.DB.QueryContext(ctx,
				"SELECT id, description, priority, effort, tags, recurrence_rule, notes, links FROM goals WHERE workspace_id = ? AND sprint_id IS NULL AND parent_id IS NULL AND description = ?",
				workspaceID, desc,
			)
		}
		if err != nil {
			return result, wrapErr(EntityGoal, "exists", 0, err)
		}
		defer rows.Close()

		for rows.Next() {
			var goal GoalSeed
			var id int64
			if err := rows.Scan(&id, &goal.Description, &goal.Priority, &goal.Effort, &goal.Tags, &goal.Recurrence, &goal.Notes, &goal.Links); err != nil {
				return result, wrapErr(EntityGoal, "exists", 0, err)
			}

			storedDesc, storedPriority, storedEffort, storedTags, storedRecurrence, storedNotes, storedLinks := normalizeSeed(goal)
			if desc == storedDesc && priority == storedPriority && effort == storedEffort && reflect.DeepEqual(tags, storedTags) && recurrence == storedRecurrence && notes == storedNotes && reflect.DeepEqual(links, storedLinks) {
				result.Exists = true
				result.ExistingID = id
				result.MatchType = "exact"
				return result, nil
			}
		}
		if err := rows.Err(); err != nil {
			return result, wrapErr(EntityGoal, "exists", 0, err)
		}
		return result, nil
	})
}

func (d *Database) GoalExists(ctx context.Context, workspaceID int64, sprintID int64, parentID *int64, description string) (bool, error) {
	result, err := d.CheckGoalExists(ctx, workspaceID, sprintID, parentID, description, ExistsCheckBasic, nil)
	return result.Exists, err
}

func (d *Database) GoalExistsDetailed(ctx context.Context, workspaceID int64, sprintID int64, parentID *int64, seed GoalSeed) (bool, error) {
	result, err := d.CheckGoalExists(ctx, workspaceID, sprintID, parentID, seed.Description, ExistsCheckDetailed, &seed)
	return result.Exists, err
}

// GetCompletedGoalsForDay retrieves all goals completed on a specific day and workspace across all sprints.
func (d *Database) GetCompletedGoalsForDay(ctx context.Context, dayID int64, workspaceID int64) ([]models.Goal, error) {
	dateStr, err := withDBContextResult(d, ctx, func(ctx context.Context) (string, error) {
		var dateStr string
		if err := d.DB.QueryRowContext(ctx, "SELECT date FROM days WHERE id = ?", dayID).Scan(&dateStr); err != nil {
			return "", wrapErr(EntityGoal, "completed list", 0, err)
		}
		return dateStr, nil
	})
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM goals
		WHERE status = 'completed' AND workspace_id = ?
		AND (
			sprint_id IN (SELECT id FROM sprints WHERE day_id = ?)
			OR (sprint_id IS NULL AND strftime('%%Y-%%m-%%d', completed_at) = ?)
		)
		ORDER BY completed_at DESC`, goalColumnsWithSprint)
	return d.queryGoals(ctx, "completed list", query, workspaceID, dayID, dateStr)
}

func (d *Database) GetActiveTask(ctx context.Context, workspaceID int64) (*models.Goal, error) {
	return withDBContextResult(d, ctx, func(ctx context.Context) (*models.Goal, error) {
		query := fmt.Sprintf(`
			SELECT %s
			FROM goals WHERE workspace_id = ? AND task_active = 1 LIMIT 1`, goalColumnsWithSprint)
		row := d.DB.QueryRowContext(ctx, query, workspaceID)
		g, err := scanGoalWithSprint(row)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, wrapErr(EntityGoal, "active task", 0, err)
		}
		return &g, nil
	})
}

func (d *Database) Search(ctx context.Context, query util.SearchQuery, workspaceID int64) ([]models.Goal, error) {
	builder := NewGoalQuery().WhereWorkspace(workspaceID)

	if len(query.Status) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(query.Status)), ",")
		statusArgs := make([]interface{}, 0, len(query.Status))
		for _, status := range query.Status {
			statusArgs = append(statusArgs, status)
		}
		builder.Where("status IN ("+placeholders+")", statusArgs...)
	}

	if len(query.Tags) > 0 {
		for _, t := range query.Tags {
			builder.Where("tags LIKE ?", "%"+t+"%")
		}
	}

	if len(query.Text) > 0 {
		for _, term := range query.Text {
			if strings.TrimSpace(term) == "" {
				continue
			}
			builder.Where("description LIKE ?", "%"+term+"%")
		}
	}

	sql, args := builder.OrderBy("created_at DESC").Limit(50).Build()

	return d.queryGoals(ctx, "search", sql, args...)
}

func (d *Database) queryGoals(ctx context.Context, op string, query string, args ...interface{}) ([]models.Goal, error) {
	return withDBContextResult(d, ctx, func(ctx context.Context) ([]models.Goal, error) {
		rows, err := d.DB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, wrapErr(EntityGoal, op, 0, err)
		}
		defer rows.Close()

		var goals []models.Goal
		for rows.Next() {
			g, err := scanGoalWithSprint(rows)
			if err != nil {
				return nil, wrapErr(EntityGoal, op, 0, err)
			}
			goals = append(goals, g)
		}
		if err := rows.Err(); err != nil {
			return nil, wrapErr(EntityGoal, op, 0, err)
		}
		return goals, nil
	})
}

func (d *Database) GetAllGoals(ctx context.Context) ([]models.Goal, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM goals
		ORDER BY rank ASC, created_at ASC`, goalColumnsWithSprint)
	return d.queryGoals(ctx, "list all", query)
}

// GetLastGoalID returns the highest goal ID or 0 when no goals exist.
func (d *Database) GetLastGoalID(ctx context.Context) (int64, error) {
	return withDBContextResult(d, ctx, func(ctx context.Context) (int64, error) {
		var id int64
		err := d.DB.QueryRowContext(ctx, "SELECT id FROM goals ORDER BY id DESC LIMIT 1").Scan(&id)
		if err == sql.ErrNoRows {
			return 0, nil
		}
		if err != nil {
			return 0, wrapErr(EntityGoal, "last id", 0, err)
		}
		return id, nil
	})
}

// GetArchivedGoals returns archived goals for a workspace.
func (d *Database) GetArchivedGoals(ctx context.Context, workspaceID int64) ([]models.Goal, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM goals
		WHERE status = 'archived' AND workspace_id = ?
		ORDER BY archived_at DESC`, goalColumnsWithSprint)
	return d.queryGoals(ctx, "archived list", query, workspaceID)
}
