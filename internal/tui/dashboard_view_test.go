package tui

import (
	"strings"
	"testing"
)

func TestDashboardViewLocked(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 20
	m.security.lock.Locked = true
	m.security.lock.PassphraseHash = "hash"
	output := m.View()
	if !strings.Contains(output, "Locked") {
		t.Fatalf("expected locked view to include Locked title")
	}
}

func TestDashboardViewNormal(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 20
	m.security.lock.Locked = false
	output := m.View()
	if output == "" {
		t.Fatalf("expected view output")
	}
	if !strings.Contains(output, "Sprint") {
		t.Fatalf("expected view output to include sprint header")
	}
}

func TestDashboardViewAnalytics(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 20
	m.security.lock.Locked = false
	m.showAnalytics = true
	output := m.View()
	if !strings.Contains(output, "Burndown") {
		t.Fatalf("expected analytics view to include Burndown header")
	}
}
