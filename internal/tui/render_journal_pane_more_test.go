package tui

import (
	"strings"
	"testing"
)

func TestRenderJournalPaneRecurrenceModal(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.modal.settingRecurrence = true
	m.modal.recurrenceFocus = "mode"
	m.modal.recurrenceOptions = []string{"none", "weekly"}
	m.modal.recurrenceCursor = 1
	out := m.renderJournalPane()
	if !strings.Contains(out, "Recurrence") {
		t.Fatalf("expected recurrence header")
	}
}

func TestRenderJournalPaneDepPicking(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.modal.depPicking = true
	m.modal.depOptions = []depOption{{ID: 1, Label: "Task #1"}}
	out := m.renderJournalPane()
	if !strings.Contains(out, "Dependencies") {
		t.Fatalf("expected dependencies header")
	}
	if !strings.Contains(out, "Task #1") {
		t.Fatalf("expected dependency option")
	}
}

func TestRenderJournalPaneTagging(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.modal.tagging = true
	m.modal.defaultTags = []string{"urgent"}
	out := m.renderJournalPane()
	if !strings.Contains(out, "Tags") {
		t.Fatalf("expected tags header")
	}
	if !strings.Contains(out, "Custom") {
		t.Fatalf("expected custom tag section")
	}
}

func TestRenderJournalPaneThemePicking(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.modal.themePicking = true
	m.modal.themeNames = []string{"default"}
	out := m.renderJournalPane()
	if !strings.Contains(out, "Themes") {
		t.Fatalf("expected themes header")
	}
}
