package tui

import (
	"testing"

	"github.com/akyairhashvil/SSPT/internal/config"
)

func TestHandleWorkspaceSwitch(t *testing.T) {
	m := setupTestDashboard(t)
	if _, err := m.db.CreateWorkspace(m.ctx, "Other", "other"); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}
	if err := m.loadWorkspaces(); err != nil {
		t.Fatalf("loadWorkspaces failed: %v", err)
	}
	m.activeWorkspaceIdx = 0

	next, _, handled := m.handleWorkspaceSwitch("w")
	if !handled {
		t.Fatalf("expected handler to handle switch")
	}
	if next.activeWorkspaceIdx == 0 {
		t.Fatalf("expected active workspace to change")
	}
	if next.view.focusedColIdx != config.DefaultFocusColumn {
		t.Fatalf("expected focus column reset")
	}
}

func TestHandleWorkspaceViewMode(t *testing.T) {
	m := setupTestDashboard(t)
	m.viewMode = ViewModeAll
	next, _, handled := m.handleWorkspaceViewMode("v")
	if !handled {
		t.Fatalf("expected view mode handler to handle")
	}
	if next.viewMode == ViewModeAll {
		t.Fatalf("expected view mode to change")
	}
	if len(next.workspaces) == 0 {
		t.Fatalf("expected workspaces to be present")
	}
	if next.workspaces[next.activeWorkspaceIdx].ViewMode != next.viewMode {
		t.Fatalf("expected workspace view mode to be updated")
	}
}
