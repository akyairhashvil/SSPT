package database

import (
	"context"
	"database/sql"
)

// AddGoalDependency creates a blocking relationship between two goals.
// The goal with dependsOnID must be completed before goalID can start.
//
// Constraints:
//   - Both goals must exist
//   - Goals must be in the same workspace
//   - Circular dependencies are not checked (caller responsibility)
//
// Returns OpError if either goal doesn't exist.
func (d *Database) AddGoalDependency(ctx context.Context, goalID, dependsOnID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	goalWS, ok := d.getGoalWorkspaceID(ctx, goalID)
	if !ok {
		return nil
	}
	depWS, ok := d.getGoalWorkspaceID(ctx, dependsOnID)
	if !ok || depWS != goalWS {
		return nil
	}
	_, err := d.DB.ExecContext(ctx, "INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id) VALUES (?, ?)", goalID, dependsOnID)
	return wrapGoalErr("add dependency", goalID, err)
}

func (d *Database) RemoveGoalDependency(ctx context.Context, goalID, dependsOnID int64) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "DELETE FROM task_deps WHERE goal_id = ? AND depends_on_id = ?", goalID, dependsOnID)
	return wrapGoalErr("remove dependency", goalID, err)
}

func (d *Database) GetGoalDependencies(ctx context.Context, goalID int64) (map[int64]bool, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, "SELECT depends_on_id FROM task_deps WHERE goal_id = ?", goalID)
	if err != nil {
		return nil, wrapGoalErr("get dependencies", goalID, err)
	}
	defer rows.Close()

	deps := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, wrapGoalErr("get dependencies", goalID, err)
		}
		deps[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, wrapGoalErr("get dependencies", goalID, err)
	}
	return deps, nil
}

func (d *Database) SetGoalDependencies(ctx context.Context, goalID int64, deps []int64) error {
	err := d.WithTx(ctx, func(tx *sql.Tx) error {
		goalWS, ok, err := getGoalWorkspaceIDTx(ctx, tx, goalID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM task_deps WHERE goal_id = ?", goalID); err != nil {
			return err
		}
		for _, id := range deps {
			if id == goalID {
				continue
			}
			depWS, ok, err := getGoalWorkspaceIDTx(ctx, tx, id)
			if err != nil {
				return err
			}
			if !ok || depWS != goalWS {
				continue
			}
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id) VALUES (?, ?)", goalID, id); err != nil {
				return err
			}
		}
		return nil
	})
	return wrapGoalErr("set dependencies", goalID, err)
}

func (d *Database) getGoalWorkspaceID(ctx context.Context, goalID int64) (int64, bool) {
	var wsID *int64
	err := d.DB.QueryRowContext(ctx, "SELECT workspace_id FROM goals WHERE id = ?", goalID).Scan(&wsID)
	if err != nil || wsID == nil {
		return 0, false
	}
	return *wsID, true
}

func getGoalWorkspaceIDTx(ctx context.Context, tx *sql.Tx, goalID int64) (int64, bool, error) {
	var wsID *int64
	if err := tx.QueryRowContext(ctx, "SELECT workspace_id FROM goals WHERE id = ?", goalID).Scan(&wsID); err != nil {
		return 0, false, err
	}
	if wsID == nil {
		return 0, false, nil
	}
	return *wsID, true, nil
}

// IsGoalBlocked checks if a goal has uncompleted dependencies.
// A goal is blocked if ANY of its dependencies have status != \"completed\".
//
// Returns:
//   - true if blocked by at least one incomplete dependency
//   - false if no dependencies or all dependencies completed
//   - error if goal doesn't exist or database error
func (d *Database) IsGoalBlocked(ctx context.Context, goalID int64) (bool, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var count int
	err := d.DB.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM task_deps td
		JOIN goals g ON td.depends_on_id = g.id
		WHERE td.goal_id = ? AND g.status != 'completed'`, goalID).Scan(&count)
	if err != nil {
		return false, wrapGoalErr("is blocked", goalID, err)
	}
	return count > 0, nil
}

func (d *Database) GetBlockedGoalIDs(ctx context.Context, workspaceID int64) (map[int64]bool, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `
		SELECT DISTINCT td.goal_id
		FROM task_deps td
		JOIN goals g ON td.depends_on_id = g.id
		JOIN goals gg ON td.goal_id = gg.id
		WHERE gg.workspace_id = ? AND g.status != 'completed'`, workspaceID)
	if err != nil {
		return nil, wrapGoalErr("list blocked", 0, err)
	}
	defer rows.Close()

	blocked := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, wrapGoalErr("list blocked", 0, err)
		}
		blocked[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, wrapGoalErr("list blocked", 0, err)
	}
	return blocked, nil
}
