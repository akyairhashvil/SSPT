package database

import (
	"context"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

func (d *Database) regenerateRecurringGoal(ctx context.Context, goalID int64) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		var g models.Goal
		err := d.DB.QueryRowContext(ctx, `
			SELECT id, description, workspace_id, sprint_id, notes, priority, effort, tags, recurrence_rule
			FROM goals WHERE id = ?`, goalID).Scan(
			&g.ID, &g.Description, &g.WorkspaceID, &g.SprintID, &g.Notes, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule,
		)
		if err != nil {
			return wrapErr(EntityGoal, "recurrence", goalID, err)
		}
		if g.RecurrenceRule == nil || strings.TrimSpace(*g.RecurrenceRule) == "" {
			return nil
		}
		rule := strings.ToLower(strings.TrimSpace(*g.RecurrenceRule))
		if rule != "daily" && !strings.HasPrefix(rule, "weekly:") && !strings.HasPrefix(rule, "monthly:") {
			return nil
		}

		var maxRank int
		if g.WorkspaceID != nil {
			var err error
			maxRank, err = d.getMaxBacklogRank(ctx, *g.WorkspaceID)
			if err != nil {
				util.LogError("recurring goal rank lookup failed, defaulting to 0", err)
				maxRank = 0
			}
		}
		wsID := toNullableArg(g.WorkspaceID)
		_, err = d.DB.ExecContext(ctx, `INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, notes, priority, effort, recurrence_rule)
			VALUES (?, ?, NULL, 'pending', ?, ?, ?, ?, ?, ?)`,
			wsID, g.Description, maxRank+1, g.Tags, g.Notes, g.Priority, g.Effort, g.RecurrenceRule,
		)
		return wrapErr(EntityGoal, "recurrence", goalID, err)
	})
}

func (d *Database) UpdateGoalRecurrence(ctx context.Context, goalID int64, rule string) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		value := nullableStringIf(rule)
		_, err := d.DB.ExecContext(ctx, "UPDATE goals SET recurrence_rule = ? WHERE id = ?", value, goalID)
		return wrapErr(EntityGoal, "update recurrence", goalID, err)
	})
}
