package database

import (
	"database/sql"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func GetWorkspaces() ([]models.Workspace, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetWorkspaces()
}

func (d *Database) GetWorkspaces() ([]models.Workspace, error) {
	rows, err := d.DB.Query("SELECT id, name, slug, view_mode, theme, show_backlog, show_completed, show_archived FROM workspaces ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ws []models.Workspace
	for rows.Next() {
		var w models.Workspace
		var viewMode sql.NullInt64
		var theme sql.NullString
		var showBacklog, showCompleted, showArchived sql.NullInt64

		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &viewMode, &theme, &showBacklog, &showCompleted, &showArchived); err != nil {
			return nil, err
		}

		if viewMode.Valid {
			w.ViewMode = int(viewMode.Int64)
		} else {
			w.ViewMode = 0
		}

		if theme.Valid {
			w.Theme = theme.String
		} else {
			w.Theme = "default"
		}
		if showBacklog.Valid {
			w.ShowBacklog = showBacklog.Int64 != 0
		} else {
			w.ShowBacklog = true
		}
		if showCompleted.Valid {
			w.ShowCompleted = showCompleted.Int64 != 0
		} else {
			w.ShowCompleted = true
		}
		if showArchived.Valid {
			w.ShowArchived = showArchived.Int64 != 0
		} else {
			w.ShowArchived = false
		}

		ws = append(ws, w)
	}
	return ws, nil
}

func UpdateWorkspaceViewMode(workspaceID int64, mode int) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UpdateWorkspaceViewMode(workspaceID, mode)
}

func (d *Database) UpdateWorkspaceViewMode(workspaceID int64, mode int) error {
	_, err := d.DB.Exec("UPDATE workspaces SET view_mode = ? WHERE id = ?", mode, workspaceID)
	return err
}

func UpdateWorkspaceTheme(workspaceID int64, theme string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UpdateWorkspaceTheme(workspaceID, theme)
}

func (d *Database) UpdateWorkspaceTheme(workspaceID int64, theme string) error {
	_, err := d.DB.Exec("UPDATE workspaces SET theme = ? WHERE id = ?", theme, workspaceID)
	return err
}

func UpdateWorkspacePaneVisibility(workspaceID int64, showBacklog, showCompleted, showArchived bool) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UpdateWorkspacePaneVisibility(workspaceID, showBacklog, showCompleted, showArchived)
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
	return err
}

func EnsureDefaultWorkspace() (int64, error) {
	d, err := getDefaultDB()
	if err != nil {
		return 0, err
	}
	return d.EnsureDefaultWorkspace()
}

func (d *Database) EnsureDefaultWorkspace() (int64, error) {
	var id int64
	err := d.DB.QueryRow("SELECT id FROM workspaces WHERE slug = 'personal'").Scan(&id)
	if err == sql.ErrNoRows {
		res, err := d.DB.Exec("INSERT INTO workspaces (name, slug) VALUES ('Personal', 'personal')")
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func CreateWorkspace(name, slug string) (int64, error) {
	d, err := getDefaultDB()
	if err != nil {
		return 0, err
	}
	return d.CreateWorkspace(name, slug)
}

func (d *Database) CreateWorkspace(name, slug string) (int64, error) {
	res, err := d.DB.Exec("INSERT INTO workspaces (name, slug) VALUES (?, ?)", name, slug)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetWorkspaceIDBySlug(slug string) (int64, bool, error) {
	d, err := getDefaultDB()
	if err != nil {
		return 0, false, err
	}
	return d.GetWorkspaceIDBySlug(slug)
}

func (d *Database) GetWorkspaceIDBySlug(slug string) (int64, bool, error) {
	var id int64
	err := d.DB.QueryRow("SELECT id FROM workspaces WHERE slug = ?", slug).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}
