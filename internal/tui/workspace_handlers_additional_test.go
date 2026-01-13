package tui

import (
	"strings"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkspaceVisibilityToggles(t *testing.T) {
	m := setupTestDashboard(t)
	active := m.workspaces[m.activeWorkspaceIdx]

	m, _, handled := m.handleWorkspaceVisibility("b")
	if !handled {
		t.Fatalf("expected handled for backlog toggle")
	}
	if m.workspaces[m.activeWorkspaceIdx].ShowBacklog == active.ShowBacklog {
		t.Fatalf("expected backlog visibility toggled")
	}

	prevCompleted := m.workspaces[m.activeWorkspaceIdx].ShowCompleted
	m, _, handled = m.handleWorkspaceVisibility("c")
	if !handled {
		t.Fatalf("expected handled for completed toggle")
	}
	if m.workspaces[m.activeWorkspaceIdx].ShowCompleted == prevCompleted {
		t.Fatalf("expected completed visibility toggled")
	}

	prevArchived := m.workspaces[m.activeWorkspaceIdx].ShowArchived
	m, _, handled = m.handleWorkspaceVisibility("a")
	if !handled {
		t.Fatalf("expected handled for archived toggle")
	}
	if m.workspaces[m.activeWorkspaceIdx].ShowArchived == prevArchived {
		t.Fatalf("expected archived visibility toggled")
	}
}

func TestWorkspaceThemePicker(t *testing.T) {
	m := setupTestDashboard(t)
	m.modal.themeNames = []string{m.workspaces[m.activeWorkspaceIdx].Theme, "other"}
	m, _, handled := m.handleWorkspaceTheme("Y")
	if !handled {
		t.Fatalf("expected handled for theme picker")
	}
	if !m.modal.themePicking {
		t.Fatalf("expected themePicking to be true")
	}
	if m.modal.themeCursor != 0 {
		t.Fatalf("expected themeCursor to match active theme")
	}
}

func TestWorkspaceSeedImportMessage(t *testing.T) {
	m := setupTestDashboard(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, _, handled := m.handleWorkspaceSeedImport("I")
	if !handled {
		t.Fatalf("expected handled for seed import")
	}
	if strings.TrimSpace(m.Message) == "" {
		t.Fatalf("expected import message")
	}
}

func TestWorkspaceSprintCountAdjustments(t *testing.T) {
	m := setupTestDashboard(t)
	before := len(m.sprints)
	m, _, handled := m.handleWorkspaceSprintCount("+")
	if !handled {
		t.Fatalf("expected handled for append sprint")
	}
	if len(m.sprints) <= before {
		t.Fatalf("expected sprints to grow")
	}

	m, _, handled = m.handleWorkspaceSprintCount("-")
	if !handled {
		t.Fatalf("expected handled for remove sprint")
	}
}

func TestWorkspaceReport(t *testing.T) {
	m := setupTestDashboard(t)
	t.Setenv("XDG_DOCUMENTS_DIR", t.TempDir())
	_, cmd, handled := m.handleWorkspaceReport("ctrl+r")
	if !handled {
		t.Fatalf("expected handled for report")
	}
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
}

func TestWorkspaceViewModeSwitch(t *testing.T) {
	m := setupTestDashboard(t)
	m.view.focusedColIdx = config.DefaultFocusColumn
	m, _, handled := m.handleWorkspaceViewMode("v")
	if !handled {
		t.Fatalf("expected handled for view mode")
	}
	if m.viewMode == ViewModeFocused && m.sprints[m.view.focusedColIdx].SprintNumber == -1 {
		t.Fatalf("unexpected focused on completed column")
	}
}

func TestWorkspaceSwitchMessage(t *testing.T) {
	m := setupTestDashboard(t)
	m.workspaces = m.workspaces[:1]
	m, _, handled := m.handleWorkspaceSwitch("w")
	if !handled {
		t.Fatalf("expected handled for workspace switch")
	}
	if !strings.Contains(m.Message, "No other workspaces") {
		t.Fatalf("expected no workspace message")
	}
}

func TestWorkspaceCreate(t *testing.T) {
	m := setupTestDashboard(t)
	m, _, handled := m.handleWorkspaceCreate("W")
	if !handled {
		t.Fatalf("expected handled for workspace create")
	}
	if !m.modal.creatingWorkspace {
		t.Fatalf("expected creatingWorkspace true")
	}
	if !m.inputs.textInput.Focused() {
		t.Fatalf("expected text input focused")
	}
}

func TestWorkspaceReportCmd(t *testing.T) {
	m := setupTestDashboard(t)
	t.Setenv("XDG_DOCUMENTS_DIR", t.TempDir())
	_, cmd, _ := m.handleWorkspaceReport("ctrl+r")
	if cmd == nil {
		t.Fatalf("expected command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit msg")
	}
}
