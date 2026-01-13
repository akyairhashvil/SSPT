package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestRenderJournalPaneAnalytics(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.showAnalytics = true
	out := m.renderJournalPane()
	if !strings.Contains(out, "Burndown") {
		t.Fatalf("expected analytics header")
	}
}

func TestRenderJournalPanePassphraseChange(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.showAnalytics = false
	m.security.changingPassphrase = true
	m.security.passphraseStage = 1
	m.security.lock.PassphraseHash = "hash"
	out := m.renderJournalPane()
	if !strings.Contains(out, "Change Passphrase") {
		t.Fatalf("expected passphrase header")
	}
	if !strings.Contains(out, "Current") {
		t.Fatalf("expected current passphrase section")
	}
}

func TestRenderJournalPaneSearch(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	m.search.Active = true
	m.search.Results = []models.Goal{{Description: "Find me"}}
	out := m.renderJournalPane()
	if !strings.Contains(out, "Search Results") {
		t.Fatalf("expected search header")
	}
	if !strings.Contains(out, "Find me") {
		t.Fatalf("expected search result")
	}
}

func TestRenderJournalPaneJournalEntries(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.height = 24
	now := time.Now()
	m.journalEntries = []models.JournalEntry{
		{Content: "Hello", CreatedAt: now},
	}
	out := m.renderJournalPane()
	if !strings.Contains(out, "Journal") {
		t.Fatalf("expected journal header")
	}
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected journal content")
	}
}
