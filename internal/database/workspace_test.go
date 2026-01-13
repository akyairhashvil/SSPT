package database

import "testing"

func TestWorkspaceUpdates(t *testing.T) {
	db := setupTestDB(t)
	wsID, err := db.CreateWorkspace("Work", "work")
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	if err := db.UpdateWorkspaceTheme(wsID, "solarized"); err != nil {
		t.Fatalf("UpdateWorkspaceTheme failed: %v", err)
	}
	if err := db.UpdateWorkspaceViewMode(wsID, 2); err != nil {
		t.Fatalf("UpdateWorkspaceViewMode failed: %v", err)
	}
	if err := db.UpdateWorkspacePaneVisibility(wsID, true, false, true); err != nil {
		t.Fatalf("UpdateWorkspacePaneVisibility failed: %v", err)
	}

	workspaces, err := db.GetWorkspaces()
	if err != nil {
		t.Fatalf("GetWorkspaces failed: %v", err)
	}
	var found *struct {
		Theme         string
		ViewMode      int
		ShowCompleted bool
		ShowArchived  bool
	}
	for i := range workspaces {
		if workspaces[i].ID == wsID {
			found = &struct {
				Theme         string
				ViewMode      int
				ShowCompleted bool
				ShowArchived  bool
			}{
				Theme:         workspaces[i].Theme,
				ViewMode:      workspaces[i].ViewMode,
				ShowCompleted: workspaces[i].ShowCompleted,
				ShowArchived:  workspaces[i].ShowArchived,
			}
			break
		}
	}
	if found == nil {
		t.Fatalf("workspace %d not found", wsID)
	}
	if found.Theme != "solarized" {
		t.Fatalf("expected theme solarized, got %q", found.Theme)
	}
	if found.ViewMode != 2 {
		t.Fatalf("expected view mode 2, got %d", found.ViewMode)
	}
	if found.ShowCompleted {
		t.Fatalf("expected show completed to be false")
	}
	if !found.ShowArchived {
		t.Fatalf("expected show archived to be true")
	}
}
