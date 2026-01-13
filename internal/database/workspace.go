package database

import (
	"context"
	"database/sql"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func (d *Database) GetWorkspaces(ctx context.Context) ([]models.Workspace, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, "SELECT id, name, slug, view_mode, theme, show_backlog, show_completed, show_archived FROM workspaces ORDER BY id ASC")
	if err != nil {
		return nil, wrapWorkspaceErr("list", 0, err)
	}
	defer rows.Close()

	var ws []models.Workspace
	for rows.Next() {
		var w models.Workspace
		var viewMode *int64
		var theme *string
		var showBacklog, showCompleted, showArchived *int64

		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &viewMode, &theme, &showBacklog, &showCompleted, &showArchived); err != nil {
			return nil, wrapWorkspaceErr("list", 0, err)
		}

		if viewMode != nil {
			w.ViewMode = int(*viewMode)
		} else {
			w.ViewMode = 0
		}

		if theme != nil {
			w.Theme = *theme
		} else {
			w.Theme = "default"
		}
		if showBacklog != nil {
			w.ShowBacklog = *showBacklog != 0
		} else {
			w.ShowBacklog = true
		}
		if showCompleted != nil {
			w.ShowCompleted = *showCompleted != 0
		} else {
			w.ShowCompleted = true
		}
		if showArchived != nil {
			w.ShowArchived = *showArchived != 0
		} else {
			w.ShowArchived = false
		}

		ws = append(ws, w)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapWorkspaceErr("list", 0, err)
	}
	return ws, nil
}

func (d *Database) UpdateWorkspaceViewMode(ctx context.Context, workspaceID int64, mode int) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE workspaces SET view_mode = ? WHERE id = ?", mode, workspaceID)
	if err != nil {
		return wrapWorkspaceErr("update view_mode", workspaceID, err)
	}
	return nil
}

func (d *Database) UpdateWorkspaceTheme(ctx context.Context, workspaceID int64, theme string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "UPDATE workspaces SET theme = ? WHERE id = ?", theme, workspaceID)
	if err != nil {
		return wrapWorkspaceErr("update theme", workspaceID, err)
	}
	return nil
}

func (d *Database) UpdateWorkspacePaneVisibility(ctx context.Context, workspaceID int64, showBacklog, showCompleted, showArchived bool) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	backlog := 0
	completed := 0
	archived := 0
	if showBacklog {
		backlog = 1
	}
	if showCompleted {
		completed = 1
	}
	if showArchived {
		archived = 1
	}
	_, err := d.DB.ExecContext(ctx, "UPDATE workspaces SET show_backlog = ?, show_completed = ?, show_archived = ? WHERE id = ?", backlog, completed, archived, workspaceID)
	if err != nil {
		return wrapWorkspaceErr("update panes", workspaceID, err)
	}
	return nil
}

func (d *Database) EnsureDefaultWorkspace(ctx context.Context) (int64, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var id int64
	err := d.DB.QueryRowContext(ctx, "SELECT id FROM workspaces WHERE slug = 'personal'").Scan(&id)
	if err == sql.ErrNoRows {
		res, err := d.DB.ExecContext(ctx, "INSERT INTO workspaces (name, slug) VALUES ('Personal', 'personal')")
		if err != nil {
			return 0, wrapWorkspaceErr("ensure default", 0, err)
		}
		return res.LastInsertId()
	}
	if err != nil {
		return 0, wrapWorkspaceErr("ensure default", 0, err)
	}
	return id, nil
}

func (d *Database) CreateWorkspace(ctx context.Context, name, slug string) (int64, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	res, err := d.DB.ExecContext(ctx, "INSERT INTO workspaces (name, slug) VALUES (?, ?)", name, slug)
	if err != nil {
		return 0, wrapWorkspaceErr("create", 0, err)
	}
	return res.LastInsertId()
}

func (d *Database) GetWorkspaceIDBySlug(ctx context.Context, slug string) (int64, bool, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var id int64
	err := d.DB.QueryRowContext(ctx, "SELECT id FROM workspaces WHERE slug = ?", slug).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, wrapWorkspaceErr("get by slug", 0, err)
	}
	return id, true, nil
}
