package database

import "context"

// getMaxGoalRank returns the highest rank for goals in a sprint (top-level only).
func (d *Database) getMaxGoalRank(ctx context.Context, sprintID int64) (int, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()

	var maxRank int
	query := "SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ? AND parent_id IS NULL"
	err := d.DB.QueryRowContext(ctx, query, sprintID).Scan(&maxRank)
	return maxRank, err
}

// getMaxSubtaskRank returns the highest rank for subtasks under a parent goal.
func (d *Database) getMaxSubtaskRank(ctx context.Context, parentID int64) (int, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()

	var maxRank int
	query := "SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?"
	err := d.DB.QueryRowContext(ctx, query, parentID).Scan(&maxRank)
	return maxRank, err
}

// getMaxBacklogRank returns the highest rank for backlog goals (no sprint).
func (d *Database) getMaxBacklogRank(ctx context.Context, workspaceID int64) (int, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()

	var maxRank int
	query := "SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ? AND parent_id IS NULL"
	err := d.DB.QueryRowContext(ctx, query, workspaceID).Scan(&maxRank)
	return maxRank, err
}
