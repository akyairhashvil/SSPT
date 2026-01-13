package database

import (
	"context"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

func (d *Database) regenerateRecurringGoal(ctx context.Context, goalID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var g models.Goal
	err := d.DB.QueryRowContext(ctx, `
		SELECT id, description, workspace_id, sprint_id, notes, priority, effort, tags, recurrence_rule
		FROM goals WHERE id = ?`, goalID).Scan(
		&g.ID, &g.Description, &g.WorkspaceID, &g.SprintID, &g.Notes, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule,
	)
	if err != nil {
		return err
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
		if err := d.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", *g.WorkspaceID).Scan(&maxRank); err != nil {
			util.LogError("recurring goal rank lookup failed, defaulting to 0", err)
			maxRank = 0
		}
	}
	var wsID interface{} = nil
	if g.WorkspaceID != nil {
		wsID = *g.WorkspaceID
	}
	_, err = d.DB.ExecContext(ctx, `INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, notes, priority, effort, recurrence_rule)
		VALUES (?, ?, NULL, 'pending', ?, ?, ?, ?, ?, ?)`,
		wsID, g.Description, maxRank+1, g.Tags, g.Notes, g.Priority, g.Effort, g.RecurrenceRule,
	)
	return err
}

func (d *Database) UpdateGoalRecurrence(ctx context.Context, goalID int64, rule string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var value interface{} = nil
	if strings.TrimSpace(rule) != "" {
		value = rule
	}
	_, err := d.DB.ExecContext(ctx, "UPDATE goals SET recurrence_rule = ? WHERE id = ?", value, goalID)
	return wrapGoalErr("update recurrence", goalID, err)
}
