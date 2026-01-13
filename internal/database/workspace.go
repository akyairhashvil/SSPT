package database

import (
	"database/sql"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func (d *Database) GetWorkspaces() ([]models.Workspace, error) {
	rows, err := d.DB.Query("SELECT id, name, slug, view_mode, theme, show_backlog, show_completed, show_archived FROM workspaces ORDER BY id ASC")
	if err != nil {
		return nil, &WorkspaceError{Op: "list", Err: err}
	}
	defer rows.Close()

	var ws []models.Workspace
	for rows.Next() {
		var w models.Workspace
		var viewMode *int64
		var theme *string
		var showBacklog, showCompleted, showArchived *int64

		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &viewMode, &theme, &showBacklog, &showCompleted, &showArchived); err != nil {
			return nil, &WorkspaceError{Op: "list", Err: err}
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
		return nil, &WorkspaceError{Op: "list", Err: err}
	}
	return ws, nil
}

func (d *Database) UpdateWorkspaceViewMode(workspaceID int64, mode int) error {
	_, err := d.DB.Exec("UPDATE workspaces SET view_mode = ? WHERE id = ?", mode, workspaceID)
	if err != nil {
		return &WorkspaceError{Op: "update view_mode", ID: workspaceID, Err: err}
	}
	return nil
}

func (d *Database) UpdateWorkspaceTheme(workspaceID int64, theme string) error {
	_, err := d.DB.Exec("UPDATE workspaces SET theme = ? WHERE id = ?", theme, workspaceID)
	if err != nil {
		return &WorkspaceError{Op: "update theme", ID: workspaceID, Err: err}
	}
	return nil
}

func (d *Database) UpdateWorkspacePaneVisibility(workspaceID int64, showBacklog, showCompleted, showArchived bool) error {
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
	_, err := d.DB.Exec("UPDATE workspaces SET show_backlog = ?, show_completed = ?, show_archived = ? WHERE id = ?", backlog, completed, archived, workspaceID)
	if err != nil {
		return &WorkspaceError{Op: "update panes", ID: workspaceID, Err: err}
	}
	return nil
}

func (d *Database) EnsureDefaultWorkspace() (int64, error) {
	var id int64
	err := d.DB.QueryRow("SELECT id FROM workspaces WHERE slug = 'personal'").Scan(&id)
	if err == sql.ErrNoRows {
		res, err := d.DB.Exec("INSERT INTO workspaces (name, slug) VALUES ('Personal', 'personal')")
		if err != nil {
			return 0, &WorkspaceError{Op: "ensure default", Err: err}
		}
		return res.LastInsertId()
	}
	if err != nil {
		return 0, &WorkspaceError{Op: "ensure default", Err: err}
	}
	return id, nil
}

func (d *Database) CreateWorkspace(name, slug string) (int64, error) {
	res, err := d.DB.Exec("INSERT INTO workspaces (name, slug) VALUES (?, ?)", name, slug)
	if err != nil {
		return 0, &WorkspaceError{Op: "create", Err: err}
	}
	return res.LastInsertId()
}

func (d *Database) GetWorkspaceIDBySlug(slug string) (int64, bool, error) {
	var id int64
	err := d.DB.QueryRow("SELECT id FROM workspaces WHERE slug = ?", slug).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, &WorkspaceError{Op: "get by slug", Err: err}
	}
	return id, true, nil
}
