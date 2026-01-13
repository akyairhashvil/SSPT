package tui

import (
	"strings"
	"testing"
)

func TestRenderHeaderIncludesStatus(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.Locked = false

	header := m.renderHeader()
	if !strings.Contains(header, "DB:") {
		t.Fatalf("expected header to include DB status, got %q", header)
	}
	if !strings.Contains(header, "Cipher:") {
		t.Fatalf("expected header to include cipher status, got %q", header)
	}
}
