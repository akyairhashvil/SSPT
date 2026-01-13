package database

import (
	"context"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func (d *Database) AddJournalEntry(ctx context.Context, dayID int64, workspaceID int64, sprintID *int64, goalID *int64, content string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "INSERT INTO journal_entries (day_id, workspace_id, sprint_id, goal_id, content) VALUES (?, ?, ?, ?, ?)", dayID, workspaceID, sprintID, goalID, content)
	return err
}

func (d *Database) GetJournalEntries(ctx context.Context, dayID int64, workspaceID int64) ([]models.JournalEntry, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `
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
